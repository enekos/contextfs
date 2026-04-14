package llm

import (
	"errors"
	"testing"
)

// mockStreamIterator simulates a streaming response for tests.
type mockStreamIterator struct {
	chunks []ChatStreamChunk
	idx    int
	done   bool
}

func (m *mockStreamIterator) Next() (ChatStreamChunk, error) {
	if m.idx >= len(m.chunks) {
		m.done = true
		return ChatStreamChunk{}, errors.New("EOF")
	}
	chunk := m.chunks[m.idx]
	m.idx++
	if chunk.FinishReason == "stop" || chunk.FinishReason == "length" {
		m.done = true
	}
	return chunk, nil
}

func (m *mockStreamIterator) Done() bool {
	return m.done
}

func TestKimiHistoryTrackingIterator_RecordsTurnOnCompletion(t *testing.T) {
	provider := &KimiProvider{
		history:      []Message{{Role: "user", Content: "hello"}},
		isNewSession: false,
	}

	inner := &mockStreamIterator{
		chunks: []ChatStreamChunk{
			{Content: "Hello "},
			{Content: "world!"},
			{FinishReason: "stop"},
		},
	}

	iter := &kimiHistoryTrackingIterator{
		inner:    inner,
		provider: provider,
		prompt:   "say hi",
	}

	// Consume all chunks
	for !iter.Done() {
		_, err := iter.Next()
		if err != nil {
			break
		}
	}

	if len(provider.history) != 3 {
		t.Fatalf("expected 3 history messages, got %d: %+v", len(provider.history), provider.history)
	}

	if provider.history[1].Role != "user" || provider.history[1].Content != "say hi" {
		t.Fatalf("expected user message 'say hi', got %+v", provider.history[1])
	}

	if provider.history[2].Role != "assistant" || provider.history[2].Content != "Hello world!" {
		t.Fatalf("expected assistant message 'Hello world!', got %+v", provider.history[2])
	}
}

func TestKimiHistoryTrackingIterator_RecordsToolCalls(t *testing.T) {
	provider := &KimiProvider{
		history:      []Message{},
		isNewSession: true,
	}

	inner := &mockStreamIterator{
		chunks: []ChatStreamChunk{
			{ToolCalls: []ToolCall{{ID: "1", Name: "bash", Arguments: map[string]any{"command": "ls"}}}},
			{FinishReason: "stop"},
		},
	}

	iter := &kimiHistoryTrackingIterator{
		inner:    inner,
		provider: provider,
		prompt:   "list files",
	}

	for !iter.Done() {
		_, err := iter.Next()
		if err != nil {
			break
		}
	}

	if len(provider.history) != 2 {
		t.Fatalf("expected 2 history messages, got %d", len(provider.history))
	}

	assistant := provider.history[1]
	if assistant.Role != "assistant" {
		t.Fatalf("expected assistant role, got %s", assistant.Role)
	}
	if len(assistant.ToolCalls) != 1 || assistant.ToolCalls[0].Name != "bash" {
		t.Fatalf("expected bash tool call, got %+v", assistant.ToolCalls)
	}
}

func TestKimiSendFunctionResponsesStream_AppendsToolMessages(t *testing.T) {
	provider := &KimiProvider{
		history: []Message{
			{Role: "user", Content: "list files"},
			{Role: "assistant", ToolCalls: []ToolCall{{ID: "1", Name: "bash", Arguments: map[string]any{"command": "ls"}}}},
		},
		isNewSession: false,
		model:        "kimi-k2.5",
	}

	// Simulate what SendFunctionResponsesStream does before building messages
	provider.history = append(provider.history, Message{
		Role:       "tool",
		Content:    `{"output":"file.txt"}`,
		ToolCallID: "bash",
	})

	if len(provider.history) != 3 {
		t.Fatalf("expected 3 history messages after tool response, got %d: %+v", len(provider.history), provider.history)
	}

	if provider.history[2].Role != "tool" || provider.history[2].ToolCallID != "bash" {
		t.Fatalf("expected tool message, got %+v", provider.history[2])
	}

	// Verify buildMessagesFromHistory produces a valid request
	msgs := provider.buildMessagesFromHistory()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages from history, got %d: %+v", len(msgs), msgs)
	}
	if msgs[2].Role != "tool" {
		t.Fatalf("expected third message role 'tool', got %s", msgs[2].Role)
	}
}
