package llm

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestKimiProvider_buildMessages(t *testing.T) {
	k := &KimiProvider{
		systemPrompt: "You are a helpful assistant.",
		history: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
	}

	msgs := k.buildMessages("How are you?")

	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "You are a helpful assistant." {
		t.Errorf("unexpected system message: %+v", msgs[0])
	}
	if msgs[1].Role != "user" || msgs[1].Content != "Hello" {
		t.Errorf("unexpected history message 1: %+v", msgs[1])
	}
	if msgs[2].Role != "assistant" || msgs[2].Content != "Hi there" {
		t.Errorf("unexpected history message 2: %+v", msgs[2])
	}
	if msgs[3].Role != "user" || msgs[3].Content != "How are you?" {
		t.Errorf("unexpected user message: %+v", msgs[3])
	}
}

func TestKimiProvider_buildMessages_noSystem(t *testing.T) {
	k := &KimiProvider{history: []Message{{Role: "user", Content: "Hello"}}}
	msgs := k.buildMessages("World")

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestKimiProvider_buildMessages_withToolCalls(t *testing.T) {
	k := &KimiProvider{
		history: []Message{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "read_file", Arguments: map[string]any{"file_path": "foo.go"}},
				},
			},
			{Role: "user", ToolCallID: "call_1", Content: "file content"},
		},
	}

	msgs := k.buildMessages("Next")

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if len(msgs[0].ToolCalls) != 1 || msgs[0].ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("unexpected tool calls in message 0: %+v", msgs[0].ToolCalls)
	}
	if msgs[1].ToolCallID != "call_1" {
		t.Errorf("unexpected tool call ID in message 1: %s", msgs[1].ToolCallID)
	}
}

func TestKimiProvider_buildMessagesFromHistory(t *testing.T) {
	k := &KimiProvider{
		systemPrompt: "sys",
		history: []Message{
			{Role: "user", Content: "q1"},
			{Role: "model", Content: "a1"},
		},
	}

	msgs := k.buildMessagesFromHistory()

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("expected system role, got %s", msgs[0].Role)
	}
	if msgs[2].Role != "assistant" {
		t.Errorf("expected assistant role for model, got %s", msgs[2].Role)
	}
}

func TestKimiProvider_buildKimiTools(t *testing.T) {
	k := &KimiProvider{
		tools: []Tool{
			{Name: "bash", Description: "run shell"},
		},
		dynamicTools: []Tool{
			{Name: "custom", Description: "custom tool"},
		},
	}

	tools := k.buildKimiTools()

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Type != "function" || tools[0].Function.Name != "bash" {
		t.Errorf("unexpected tool 0: %+v", tools[0])
	}
	if tools[1].Function.Name != "custom" {
		t.Errorf("unexpected tool 1: %+v", tools[1])
	}
}

func TestKimiProvider_convertKimiToolCalls(t *testing.T) {
	k := &KimiProvider{}
	calls := []KimiToolCall{
		{
			ID:   "c1",
			Type: "function",
			Function: KimiFunctionCall{
				Name:      "read_file",
				Arguments: `{"file_path": "main.go"}`,
			},
		},
		{
			ID:   "c2",
			Type: "function",
			Function: KimiFunctionCall{
				Name:      "bad",
				Arguments: `not json`,
			},
		},
	}

	result := k.convertKimiToolCalls(calls)

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result[0].Name != "read_file" {
		t.Errorf("unexpected name: %s", result[0].Name)
	}
	if result[0].Arguments["file_path"] != "main.go" {
		t.Errorf("unexpected args: %v", result[0].Arguments)
	}
	// second call should have nil args because of unmarshal error
	if result[1].Arguments != nil {
		t.Errorf("expected nil args for invalid json, got %v", result[1].Arguments)
	}
}

func TestKimiProvider_convertToolCallsToKimi(t *testing.T) {
	k := &KimiProvider{}
	calls := []ToolCall{
		{ID: "c1", Name: "bash", Arguments: map[string]any{"command": "ls"}},
	}

	result := k.convertToolCallsToKimi(calls)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Function.Name != "bash" {
		t.Errorf("unexpected name: %s", result[0].Function.Name)
	}
	if result[0].Function.Arguments != `{"command":"ls"}` {
		t.Errorf("unexpected args: %s", result[0].Function.Arguments)
	}
}

func TestKimiProvider_buildMessages_roundTrip(t *testing.T) {
	k := &KimiProvider{
		history: []Message{
			{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Name: "bash", Arguments: map[string]any{"cmd": "ls"}}}},
			{Role: "user", ToolCallID: "c1", Content: "output"},
		},
	}

	msgs := k.buildMessages("Next")
	if len(msgs[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msgs[0].ToolCalls))
	}

	// Verify the round-trip conversion produces valid JSON arguments
	reconverted := k.convertKimiToolCalls(msgs[0].ToolCalls)
	if diff := cmp.Diff(k.history[0].ToolCalls, reconverted); diff != "" {
		t.Errorf("tool call round-trip mismatch (-want +got):\n%s", diff)
	}
}
