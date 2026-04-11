package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/google/generative-ai-go/genai"
)

type mockRouterClient struct {
	generate func(ctx context.Context, out any) error
}

func (m mockRouterClient) GenerateJSON(ctx context.Context, _ string, _ string, _ *genai.Schema, out any) error {
	if m.generate == nil {
		return nil
	}
	return m.generate(ctx, out)
}

func TestDecideMemoryAction_FallsBackOnInvalidUpdate(t *testing.T) {
	client := mockRouterClient{
		generate: func(_ context.Context, out any) error {
			decision := out.(*RouterAction)
			decision.Action = "update"
			decision.Reason = "merge"
			return nil
		},
	}

	action, err := DecideMemoryAction(context.Background(), client, "new", []RouterCandidate{
		{ID: "m1", Content: "old", Score: 0.9},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action.Action != "create" {
		t.Fatalf("expected fallback create action, got %q", action.Action)
	}
	if action.Reason == "" {
		t.Fatalf("expected fallback reason to be populated")
	}
}

func TestDecideMemoryAction_ReportsClientError(t *testing.T) {
	wantErr := errors.New("json decode failed")
	client := mockRouterClient{
		generate: func(_ context.Context, _ any) error {
			return wantErr
		},
	}
	action, err := DecideMemoryAction(context.Background(), client, "new", []RouterCandidate{
		{ID: "m1", Content: "old", Score: 0.9},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped error %v, got %v", wantErr, err)
	}
	if action.Action != "create" {
		t.Fatalf("expected create fallback, got %q", action.Action)
	}
}

func TestDecideContextAction_UnknownActionFallsBackToCreate(t *testing.T) {
	client := mockRouterClient{
		generate: func(_ context.Context, out any) error {
			decision := out.(*RouterAction)
			decision.Action = "merge"
			decision.Reason = "bad action"
			return nil
		},
	}
	action, err := DecideContextAction(context.Background(), client, "context://p/a", "A", "new", []RouterCandidate{
		{ID: "context://p/b", Content: "existing", Score: 0.95},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action.Action != "create" {
		t.Fatalf("expected fallback create action, got %q", action.Action)
	}
	if action.Reason == "" {
		t.Fatalf("expected fallback reason for unknown action")
	}
}
