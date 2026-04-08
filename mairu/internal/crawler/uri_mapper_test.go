package crawler

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://example.com", "http://example.com"},
		{"http://example.com/", "http://example.com"},
		{"http://example.com/path/", "http://example.com/path"},
		{"http://example.com/path#fragment", "http://example.com/path"},
	}

	for _, tt := range tests {
		got := NormalizeURL(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeURL(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestURLToURI(t *testing.T) {
	tests := []struct {
		input    string
		section  []string
		expected string
	}{
		{"http://example.com", nil, "contextfs://scraped/example-com"},
		{"http://www.example.com", nil, "contextfs://scraped/example-com"},
		{"http://example.com:8080", nil, "contextfs://scraped/example-com-8080"},
		{"http://example.com/path/to/page", nil, "contextfs://scraped/example-com/path/to/page"},
		{"http://example.com/path", []string{"Section 1!"}, "contextfs://scraped/example-com/path/section-1"},
	}

	for _, tt := range tests {
		got := URLToURI(tt.input, tt.section...)
		if got != tt.expected {
			t.Errorf("URLToURI(%q, %v) = %q; want %q", tt.input, tt.section, got, tt.expected)
		}
	}
}

func TestURLToParentURI(t *testing.T) {
	tests := []struct {
		input    string
		expected *string
	}{
		{"http://example.com", nil}, // contextfs://scraped/example-com has 4 parts
		{"http://example.com/path", ptr("contextfs://scraped/example-com")},
		{"http://example.com/path/to", ptr("contextfs://scraped/example-com/path")},
	}

	for _, tt := range tests {
		got := URLToParentURI(tt.input)
		if tt.expected == nil {
			if got != nil {
				t.Errorf("URLToParentURI(%q) = %v; want nil", tt.input, *got)
			}
		} else {
			if got == nil || *got != *tt.expected {
				t.Errorf("URLToParentURI(%q) = %v; want %v", tt.input, got, *tt.expected)
			}
		}
	}
}

func ptr(s string) *string {
	return &s
}
