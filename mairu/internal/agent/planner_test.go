package agent

import (
	"testing"

	"mairu/internal/llm"
)

func TestIsComplexPrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		complex bool
	}{
		{"simple greeting", "hi", false},
		{"simple question", "what is 2 + 2", false},
		{"long but simple", "can you explain what this function does", false},
		{"multi-step explicit", "first read the file and then refactor it", true},
		{"length threshold", "a " + string(make([]byte, 100)), true},
		{"implementation keyword", "implement a new auth middleware", true},
		{"search keyword", "search for all usages of this function", true},
		{"and then keyword", "do this and then do that", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fix the length threshold test
			prompt := tt.prompt
			if tt.name == "length threshold" {
				prompt = "word "
				for i := 0; i < 52; i++ {
					prompt += "word "
				}
			}
			got := isComplexPrompt(prompt)
			if got != tt.complex {
				t.Fatalf("isComplexPrompt(%q) = %v, want %v", prompt, got, tt.complex)
			}
		})
	}
}

func TestFilterTools(t *testing.T) {
	all := []llm.Tool{
		{Name: "bash"},
		{Name: "read_file"},
		{Name: "write_file"},
		{Name: "search_codebase"},
	}

	filtered := filterTools(all, []string{"bash", "search_codebase"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(filtered))
	}
	if filtered[0].Name != "bash" || filtered[1].Name != "search_codebase" {
		t.Fatalf("unexpected filtered tools: %v", filtered)
	}

	// Unknown names are ignored
	filtered2 := filterTools(all, []string{"bash", "nonexistent"})
	if len(filtered2) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(filtered2))
	}
}

func TestExtractJSONBlock(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"plain json", "plain json"},
		{"```json\n{\"a\":1}\n```", "{\"a\":1}"},
		{"```\n{\"a\":1}\n```", "{\"a\":1}"},
		{"prefix ```json\n{\"a\":1}\n``` suffix", "{\"a\":1}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractJSONBlock(tt.input)
			if got != tt.want {
				t.Fatalf("extractJSONBlock(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
