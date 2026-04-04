package agent

import (
	"fmt"
	"github.com/google/generative-ai-go/genai"
)

func (a *Agent) GetRecentContext() string {
	history := a.llm.GetHistory()
	var conversation string

	// Get up to the last 10 messages
	start := len(history) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(history); i++ {
		c := history[i]
		if c.Role == "user" || c.Role == "model" {
			var textContent string
			for _, p := range c.Parts {
				if t, ok := p.(genai.Text); ok {
					textContent += string(t)
				}
			}
			if textContent != "" {
				conversation += fmt.Sprintf("[%s]: %s\n\n", c.Role, textContent)
			}
		}
	}
	return conversation
}
