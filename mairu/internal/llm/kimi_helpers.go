package llm

import (
	"encoding/json"
	"log/slog"
)

func (k *KimiProvider) buildMessages(userPrompt string) []KimiMessage {
	var messages []KimiMessage

	// Add system prompt if set
	if k.systemPrompt != "" {
		messages = append(messages, KimiMessage{
			Role:    "system",
			Content: k.systemPrompt,
		})
	}

	// Add history
	for _, msg := range k.history {
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		}

		kMsg := KimiMessage{
			Role:    role,
			Content: msg.Content,
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			kMsg.ToolCalls = k.convertToolCallsToKimi(msg.ToolCalls)
		}

		// Handle tool responses
		if msg.ToolCallID != "" {
			kMsg.ToolCallID = msg.ToolCallID
		}

		messages = append(messages, kMsg)
	}

	// Add user message
	messages = append(messages, KimiMessage{
		Role:    "user",
		Content: userPrompt,
	})

	return messages
}

func (k *KimiProvider) buildMessagesFromHistory() []KimiMessage {
	var messages []KimiMessage

	// Add system prompt if set
	if k.systemPrompt != "" {
		messages = append(messages, KimiMessage{
			Role:    "system",
			Content: k.systemPrompt,
		})
	}

	// Add all history
	for _, msg := range k.history {
		role := msg.Role
		if role == "model" || role == "assistant" {
			role = "assistant"
		}

		kMsg := KimiMessage{
			Role:    role,
			Content: msg.Content,
		}

		if len(msg.ToolCalls) > 0 {
			kMsg.ToolCalls = k.convertToolCallsToKimi(msg.ToolCalls)
		}

		if msg.ToolCallID != "" {
			kMsg.ToolCallID = msg.ToolCallID
		}

		messages = append(messages, kMsg)
	}

	return messages
}

func (k *KimiProvider) buildKimiTools() []KimiTool {
	allTools := append(k.tools, k.dynamicTools...)
	kimiTools := make([]KimiTool, 0, len(allTools))

	for _, tool := range allTools {
		kimiTools = append(kimiTools, KimiTool{
			Type:     "function",
			Function: KimiFunctionDef(tool),
		})
	}

	return kimiTools
}

func (k *KimiProvider) convertKimiToolCalls(calls []KimiToolCall) []ToolCall {
	toolCalls := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		var args map[string]any
		if call.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
				slog.Error("failed to unmarshal tool call arguments", "error", err)
			}
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: args,
		})
	}
	return toolCalls
}

func (k *KimiProvider) convertToolCallsToKimi(calls []ToolCall) []KimiToolCall {
	kimiCalls := make([]KimiToolCall, 0, len(calls))
	for _, call := range calls {
		argsJSON, err := json.Marshal(call.Arguments)
		if err != nil {
			slog.Error("failed to marshal tool call arguments", "error", err)
			argsJSON = []byte("{}")
		}
		kimiCalls = append(kimiCalls, KimiToolCall{
			ID:   call.ID,
			Type: "function",
			Function: KimiFunctionCall{
				Name:      call.Name,
				Arguments: string(argsJSON),
			},
		})
	}
	return kimiCalls
}
