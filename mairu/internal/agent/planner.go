package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"mairu/internal/llm"
	"mairu/internal/prompts"
)

// PlanResult contains the output of the planner.
type PlanResult struct {
	ToolNames []string `json:"tools"`
}

// Planner decides which tools are relevant for a given prompt.
type Planner struct {
	provider llm.Provider
}

// NewPlanner creates a planner for the given provider.
func NewPlanner(provider llm.Provider) *Planner {
	return &Planner{provider: provider}
}

var (
	strongComplexityIndicators = []*regexp.Regexp{
		regexp.MustCompile(`\b(and then|after that|first|second|third|finally|next|subsequently|before|then)\b`),
		regexp.MustCompile(`\b(refactor|implement|build)\b`),
		regexp.MustCompile(`\b(search for|find all|update every)\b`),
	}
	weakComplexityIndicators = []*regexp.Regexp{
		regexp.MustCompile(`\b(create a|write a)\b`),
	}
)

// isComplexPrompt uses a lightweight heuristic to decide whether planning is worthwhile.
// It scores the prompt on length and presence of complexity indicators.
func isComplexPrompt(prompt string) bool {
	lower := strings.ToLower(prompt)
	score := 0

	if len(strings.Fields(prompt)) > 50 {
		score += 2
	}

	for _, re := range strongComplexityIndicators {
		if re.MatchString(lower) {
			score += 2
		}
	}
	for _, re := range weakComplexityIndicators {
		if re.MatchString(lower) {
			score += 1
		}
	}

	return score >= 2
}

// Plan analyzes the prompt and returns a subset of relevant tools.
// For simple prompts it returns nil, signalling that the full tool set should be used.
func (p *Planner) Plan(ctx context.Context, prompt string) (*PlanResult, error) {
	if !isComplexPrompt(prompt) {
		return nil, nil
	}

	tools := p.provider.GetTools()
	if len(tools) == 0 {
		return nil, nil
	}

	var toolDescriptions strings.Builder
	for _, t := range tools {
		fmt.Fprintf(&toolDescriptions, "- %s: %s\n", t.Name, t.Description)
	}

	plannerPrompt, err := prompts.Get("tool_planner", struct {
		ToolDescriptions string
		Prompt           string
	}{
		ToolDescriptions: toolDescriptions.String(),
		Prompt:           prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render planner prompt: %w", err)
	}

	planCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	raw, err := p.provider.GenerateContent(planCtx, p.provider.GetModelName(), plannerPrompt)
	if err != nil {
		return nil, fmt.Errorf("planner generation failed: %w", err)
	}

	raw = extractJSONBlock(raw)
	var plan PlanResult
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, fmt.Errorf("planner failed to parse JSON: %w", err)
	}

	// Validate tool names against available tools
	available := make(map[string]bool, len(tools))
	for _, t := range tools {
		available[t.Name] = true
	}

	var valid []string
	for _, name := range plan.ToolNames {
		if available[name] {
			valid = append(valid, name)
		}
	}
	plan.ToolNames = valid
	return &plan, nil
}

func extractJSONBlock(s string) string {
	if idx := strings.Index(s, "```json"); idx != -1 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end != -1 {
			s = s[:end]
		}
	} else if idx := strings.Index(s, "```"); idx != -1 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end != -1 {
			s = s[:end]
		}
	}
	return strings.TrimSpace(s)
}
