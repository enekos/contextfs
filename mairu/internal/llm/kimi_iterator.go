package llm

// kimiHistoryTrackingIterator wraps a Kimi stream iterator and updates the
// provider's history with the user prompt and assistant response once the
// stream finishes. This is required because Kimi's history is managed manually
// in the provider, unlike Gemini which manages it inside the chat session.
type kimiHistoryTrackingIterator struct {
	inner            ChatStreamIterator
	provider         *KimiProvider
	prompt           string
	content          string
	reasoningContent string
	toolCalls        []ToolCall
	committed        bool
	skipUserCommit   bool
}

func (k *kimiHistoryTrackingIterator) Next() (ChatStreamChunk, error) {
	chunk, err := k.inner.Next()
	if err != nil {
		k.commit()
		return chunk, err
	}

	k.content += chunk.Content
	k.reasoningContent += chunk.ReasoningContent
	if len(chunk.ToolCalls) > 0 {
		k.toolCalls = append(k.toolCalls, chunk.ToolCalls...)
	}

	if chunk.FinishReason == "stop" || chunk.FinishReason == "length" {
		k.commit()
	}

	return chunk, nil
}

func (k *kimiHistoryTrackingIterator) Done() bool {
	return k.inner.Done()
}

func (k *kimiHistoryTrackingIterator) commit() {
	if k.committed {
		return
	}
	k.committed = true

	if !k.skipUserCommit {
		// Record the user turn
		k.provider.history = append(k.provider.history, Message{Role: "user", Content: k.prompt})
	}

	// Record the assistant turn
	k.provider.history = append(k.provider.history, Message{
		Role:             "assistant",
		Content:          k.content,
		ReasoningContent: k.reasoningContent,
		ToolCalls:        k.toolCalls,
	})
}
