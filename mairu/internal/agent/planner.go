package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mairu/internal/llm"
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

// isComplexPrompt uses a lightweight heuristic to decide whether planning is worthwhile.
func isComplexPrompt(prompt string) bool {
	words := strings.Fields(prompt)
	if len(words) > 50 {
		return true
	}
	lower := strings.ToLower(prompt)
	keywords := []string{
		"and then", "after that", "first ", "second ", "third ",
		"finally", "next ", "subsequently", "before ", "then ",
		"search for", "find all", "update every", "refactor",
		"implement", "create a", "build a", "write a",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
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

	plannerPrompt := fmt.Sprintf(`You are a tool-routing planner. Given the user request below, select the MINIMAL set of tools needed to complete the task.

Available tools:
%s
User request: %s

Respond with ONLY a JSON object in this exact format:
{"tools": ["tool_name_1", "tool_name_2"]}

If the task is conversational and requires no tools, respond with: {"tools": []}
`, toolDescriptions.String(), prompt)

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
