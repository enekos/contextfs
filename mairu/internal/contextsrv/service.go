package contextsrv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

var ErrModerationRejected = errors.New("content rejected by moderation policy")

type Service interface {
	Health() map[string]any
	CreateMemory(input MemoryCreateInput) (Memory, error)
	ListMemories(project string, limit int) ([]Memory, error)
	UpdateMemory(input MemoryUpdateInput) (Memory, error)
	DeleteMemory(id string) error
	CreateSkill(input SkillCreateInput) (Skill, error)
	ListSkills(project string, limit int) ([]Skill, error)
	UpdateSkill(input SkillUpdateInput) (Skill, error)
	DeleteSkill(id string) error
	CreateContextNode(input ContextCreateInput) (ContextNode, error)
	ListContextNodes(project string, parentURI *string, limit int) ([]ContextNode, error)
	UpdateContextNode(input ContextUpdateInput) (ContextNode, error)
	DeleteContextNode(uri string) error
	Search(opts SearchOptions) (map[string]any, error)
	Dashboard(limit int, project string) (map[string]any, error)
	ClusterStats() map[string]any
	VibeQuery(prompt, project string, topK int) (VibeQueryResult, error)
	PlanVibeMutation(prompt, project string, topK int) (VibeMutationPlan, error)
	ExecuteVibeMutation(ops []VibeMutationOp, project string) ([]map[string]any, error)
	ListModerationQueue(limit int) ([]ModerationEvent, error)
	ReviewModeration(input ModerationReviewInput) error
}

type Repository interface {
	CreateMemory(ctx context.Context, input MemoryCreateInput) (Memory, error)
	ListMemories(ctx context.Context, project string, limit int) ([]Memory, error)
	UpdateMemory(ctx context.Context, input MemoryUpdateInput) (Memory, error)
	DeleteMemory(ctx context.Context, id string) error
	CreateSkill(ctx context.Context, input SkillCreateInput) (Skill, error)
	ListSkills(ctx context.Context, project string, limit int) ([]Skill, error)
	UpdateSkill(ctx context.Context, input SkillUpdateInput) (Skill, error)
	DeleteSkill(ctx context.Context, id string) error
	CreateContextNode(ctx context.Context, input ContextCreateInput) (ContextNode, error)
	ListContextNodes(ctx context.Context, project string, parentURI *string, limit int) ([]ContextNode, error)
	UpdateContextNode(ctx context.Context, input ContextUpdateInput) (ContextNode, error)
	DeleteContextNode(ctx context.Context, uri string) error
	SearchText(ctx context.Context, opts SearchOptions) (map[string]any, error)
	ListModerationQueue(ctx context.Context, limit int) ([]ModerationEvent, error)
	ReviewModeration(ctx context.Context, input ModerationReviewInput) error
	EnqueueOutbox(ctx context.Context, entityType, entityID, opType string, payload any) error
}

type AppService struct {
	repo Repository
}

func NewService(repo Repository) *AppService {
	return &AppService{repo: repo}
}

func (s *AppService) Health() map[string]any {
	return map[string]any{"ok": true, "service": "contextsrv"}
}

func (s *AppService) CreateMemory(input MemoryCreateInput) (Memory, error) {
	if strings.TrimSpace(input.Content) == "" {
		return Memory{}, fmt.Errorf("content is required")
	}
	if input.Importance <= 0 {
		input.Importance = 1
	}
	if input.Category == "" {
		input.Category = "observation"
	}
	if input.Owner == "" {
		input.Owner = "agent"
	}
	m := ModerateContent(input.Content)
	input.ModerationStatus = m.Status
	input.ModerationReasons = m.Reasons
	input.ReviewRequired = m.Status == ModerationStatusFlaggedSoft
	if m.Status == ModerationStatusRejectHard {
		return Memory{}, fmt.Errorf("%w: %s", ErrModerationRejected, strings.Join(m.Reasons, ", "))
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	out, err := s.repo.CreateMemory(context.Background(), input)
	if err != nil {
		return Memory{}, err
	}
	_ = s.repo.EnqueueOutbox(context.Background(), "memory", out.ID, "upsert", out)
	return out, nil
}

func (s *AppService) ListMemories(project string, limit int) ([]Memory, error) {
	return s.repo.ListMemories(context.Background(), project, limit)
}

func (s *AppService) UpdateMemory(input MemoryUpdateInput) (Memory, error) {
	return s.repo.UpdateMemory(context.Background(), input)
}

func (s *AppService) DeleteMemory(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("id is required")
	}
	return s.repo.DeleteMemory(context.Background(), id)
}

func (s *AppService) CreateSkill(input SkillCreateInput) (Skill, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" {
		return Skill{}, fmt.Errorf("name and description are required")
	}
	m := ModerateContent(input.Name + ": " + input.Description)
	input.ModerationStatus = m.Status
	input.ModerationReasons = m.Reasons
	input.ReviewRequired = m.Status == ModerationStatusFlaggedSoft
	if m.Status == ModerationStatusRejectHard {
		return Skill{}, fmt.Errorf("%w: %s", ErrModerationRejected, strings.Join(m.Reasons, ", "))
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	out, err := s.repo.CreateSkill(context.Background(), input)
	if err != nil {
		return Skill{}, err
	}
	_ = s.repo.EnqueueOutbox(context.Background(), "skill", out.ID, "upsert", out)
	return out, nil
}

func (s *AppService) ListSkills(project string, limit int) ([]Skill, error) {
	return s.repo.ListSkills(context.Background(), project, limit)
}

func (s *AppService) UpdateSkill(input SkillUpdateInput) (Skill, error) {
	return s.repo.UpdateSkill(context.Background(), input)
}

func (s *AppService) DeleteSkill(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("id is required")
	}
	return s.repo.DeleteSkill(context.Background(), id)
}

func (s *AppService) CreateContextNode(input ContextCreateInput) (ContextNode, error) {
	if strings.TrimSpace(input.URI) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Abstract) == "" {
		return ContextNode{}, fmt.Errorf("uri, name, and abstract are required")
	}
	m := ModerateContent(input.Name + ": " + input.Abstract + "\n" + input.Content)
	input.ModerationStatus = m.Status
	input.ModerationReasons = m.Reasons
	input.ReviewRequired = m.Status == ModerationStatusFlaggedSoft
	if m.Status == ModerationStatusRejectHard {
		return ContextNode{}, fmt.Errorf("%w: %s", ErrModerationRejected, strings.Join(m.Reasons, ", "))
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	out, err := s.repo.CreateContextNode(context.Background(), input)
	if err != nil {
		return ContextNode{}, err
	}
	_ = s.repo.EnqueueOutbox(context.Background(), "context_node", out.URI, "upsert", out)
	return out, nil
}

func (s *AppService) ListContextNodes(project string, parentURI *string, limit int) ([]ContextNode, error) {
	return s.repo.ListContextNodes(context.Background(), project, parentURI, limit)
}

func (s *AppService) UpdateContextNode(input ContextUpdateInput) (ContextNode, error) {
	return s.repo.UpdateContextNode(context.Background(), input)
}

func (s *AppService) DeleteContextNode(uri string) error {
	if strings.TrimSpace(uri) == "" {
		return fmt.Errorf("uri is required")
	}
	return s.repo.DeleteContextNode(context.Background(), uri)
}

func (s *AppService) Search(opts SearchOptions) (map[string]any, error) {
	return s.repo.SearchText(context.Background(), opts)
}

func (s *AppService) Dashboard(limit int, project string) (map[string]any, error) {
	memories, err := s.repo.ListMemories(context.Background(), project, limit)
	if err != nil {
		return nil, err
	}
	skills, err := s.repo.ListSkills(context.Background(), project, limit)
	if err != nil {
		return nil, err
	}
	contextNodes, err := s.repo.ListContextNodes(context.Background(), project, nil, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"counts": map[string]int{
			"skills":       len(skills),
			"memories":     len(memories),
			"contextNodes": len(contextNodes),
		},
		"skills":       skills,
		"memories":     memories,
		"contextNodes": contextNodes,
	}, nil
}

func (s *AppService) ClusterStats() map[string]any {
	return map[string]any{
		"ok":      true,
		"service": "contextsrv",
		"indexes": []string{"contextfs_memories", "contextfs_skills", "contextfs_context_nodes"},
	}
}

func (s *AppService) VibeQuery(prompt, project string, topK int) (VibeQueryResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return VibeQueryResult{}, fmt.Errorf("prompt is required")
	}
	search, err := s.Search(SearchOptions{
		Query:   prompt,
		Project: project,
		Store:   "all",
		TopK:    topK,
	})
	if err != nil {
		return VibeQueryResult{}, err
	}
	return VibeQueryResult{
		Reasoning: "Queried memories, skills, and context nodes with the same prompt for broad recall.",
		Results: []VibeSearchGroup{
			{Store: "memory", Query: prompt, Items: toAnyMapSlice(search["memories"])},
			{Store: "skill", Query: prompt, Items: toAnyMapSlice(search["skills"])},
			{Store: "node", Query: prompt, Items: toAnyMapSlice(search["contextNodes"])},
		},
	}, nil
}

func (s *AppService) PlanVibeMutation(prompt, project string, topK int) (VibeMutationPlan, error) {
	if strings.TrimSpace(prompt) == "" {
		return VibeMutationPlan{}, fmt.Errorf("prompt is required")
	}
	if topK <= 0 {
		topK = 5
	}
	plan := VibeMutationPlan{
		Reasoning: "Generated a conservative mutation plan from plain-English intent.",
	}
	search, err := s.Search(SearchOptions{
		Query:   prompt,
		Project: project,
		Store:   "memories",
		TopK:    topK,
	})
	if err == nil {
		if existing, ok := bestMemoryMatch(search["memories"], prompt); ok {
			plan.Reasoning = "Existing memory is highly similar, so route to update instead of duplicate create."
			plan.Operations = append(plan.Operations, VibeMutationOp{
				Op:          "update_memory",
				Target:      existing.ID,
				Description: "Update the closest matching memory with refined content.",
				Data: map[string]any{
					"id":       existing.ID,
					"content":  prompt,
					"category": "observation",
					"owner":    "agent",
				},
			})
			return plan, nil
		}
	}

	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "remember"):
		plan.Operations = append(plan.Operations, VibeMutationOp{
			Op:          "create_memory",
			Description: "Store the statement as a durable memory.",
			Data: map[string]any{
				"content":    prompt,
				"category":   "observation",
				"owner":      "agent",
				"importance": 5,
				"project":    project,
			},
		})
	case strings.Contains(lower, "skill"):
		plan.Operations = append(plan.Operations, VibeMutationOp{
			Op:          "create_skill",
			Description: "Create a skill from the prompt.",
			Data: map[string]any{
				"name":        "Derived Skill",
				"description": prompt,
				"project":     project,
			},
		})
	default:
		plan.Operations = append(plan.Operations, VibeMutationOp{
			Op:          "create_memory",
			Description: "Default to storing prompt as an observation memory.",
			Data: map[string]any{
				"content":    prompt,
				"category":   "observation",
				"owner":      "agent",
				"importance": 4,
				"project":    project,
			},
		})
	}
	return plan, nil
}

func (s *AppService) ExecuteVibeMutation(ops []VibeMutationOp, project string) ([]map[string]any, error) {
	results := make([]map[string]any, 0, len(ops))
	for _, op := range ops {
		switch op.Op {
		case "create_memory":
			content, _ := op.Data["content"].(string)
			if content == "" {
				results = append(results, map[string]any{"op": op.Op, "error": "missing content"})
				continue
			}
			importance := intFromAny(op.Data["importance"], 5)
			mem, err := s.CreateMemory(MemoryCreateInput{
				Project:    firstString(op.Data["project"], project),
				Content:    content,
				Category:   firstString(op.Data["category"], "observation"),
				Owner:      firstString(op.Data["owner"], "agent"),
				Importance: importance,
			})
			if err != nil {
				results = append(results, map[string]any{"op": op.Op, "error": err.Error()})
				continue
			}
			results = append(results, map[string]any{"op": op.Op, "result": "created memory " + mem.ID})
		case "update_memory":
			id := firstString(op.Data["id"], op.Target)
			if id == "" {
				results = append(results, map[string]any{"op": op.Op, "error": "missing id"})
				continue
			}
			updated, err := s.UpdateMemory(MemoryUpdateInput{
				ID:         id,
				Content:    firstString(op.Data["content"], ""),
				Category:   firstString(op.Data["category"], ""),
				Owner:      firstString(op.Data["owner"], ""),
				Importance: intFromAny(op.Data["importance"], 0),
			})
			if err != nil {
				results = append(results, map[string]any{"op": op.Op, "error": err.Error()})
				continue
			}
			results = append(results, map[string]any{"op": op.Op, "result": "updated memory " + updated.ID})
		case "create_skill":
			skill, err := s.CreateSkill(SkillCreateInput{
				Project:     firstString(op.Data["project"], project),
				Name:        firstString(op.Data["name"], "Derived Skill"),
				Description: firstString(op.Data["description"], ""),
			})
			if err != nil {
				results = append(results, map[string]any{"op": op.Op, "error": err.Error()})
				continue
			}
			results = append(results, map[string]any{"op": op.Op, "result": "created skill " + skill.ID})
		default:
			results = append(results, map[string]any{"op": op.Op, "error": "unsupported op"})
		}
	}
	return results, nil
}

type memorySearchHit struct {
	ID      string
	Content string
}

func bestMemoryMatch(raw any, prompt string) (memorySearchHit, bool) {
	items := toAnyMapSlice(raw)
	best := memorySearchHit{}
	bestScore := 0.0
	for _, item := range items {
		id, _ := item["id"].(string)
		content, _ := item["content"].(string)
		if id == "" || strings.TrimSpace(content) == "" {
			continue
		}
		score := textSimilarity(content, prompt)
		if score > bestScore {
			bestScore = score
			best = memorySearchHit{ID: id, Content: content}
		}
	}
	return best, bestScore >= 0.85
}

func textSimilarity(a, b string) float64 {
	setA := tokenSet(a)
	setB := tokenSet(b)
	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}
	inter := 0
	for token := range setA {
		if _, ok := setB[token]; ok {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / math.Max(float64(union), 1)
}

func tokenSet(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range strings.Fields(strings.ToLower(s)) {
		token = strings.Trim(token, ".,:;!?()[]{}\"'`")
		if token == "" {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

func (s *AppService) ListModerationQueue(limit int) ([]ModerationEvent, error) {
	return s.repo.ListModerationQueue(context.Background(), limit)
}

func (s *AppService) ReviewModeration(input ModerationReviewInput) error {
	return s.repo.ReviewModeration(context.Background(), input)
}

func toAnyMapSlice(v any) []map[string]any {
	if v == nil {
		return []map[string]any{}
	}
	if direct, ok := v.([]map[string]any); ok {
		return direct
	}
	out := []map[string]any{}
	switch arr := v.(type) {
	case []any:
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func firstString(v any, fallback string) string {
	if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}

func intFromAny(v any, fallback int) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return fallback
	}
}
