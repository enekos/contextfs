package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSummarizeMarkdownDoc_Success(t *testing.T) {
	g := &fakeGen{out: []string{`{"abstract":"A project tool","overview":"Covers install and usage"}`}}
	ab, ov, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", "# Hello\n\nWorld")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ab != "A project tool" {
		t.Errorf("abstract = %q, want \"A project tool\"", ab)
	}
	if ov != "Covers install and usage" {
		t.Errorf("overview = %q, want \"Covers install and usage\"", ov)
	}
	if g.calls != 1 {
		t.Errorf("calls = %d, want 1", g.calls)
	}
}

func TestSummarizeMarkdownDoc_FencedJSON(t *testing.T) {
	resp := "```json\n{\"abstract\":\"Summary\",\"overview\":\"Details\"}\n```"
	g := &fakeGen{out: []string{resp}}
	ab, ov, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", "content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ab != "Summary" {
		t.Errorf("abstract = %q, want \"Summary\"", ab)
	}
	if ov != "Details" {
		t.Errorf("overview = %q, want \"Details\"", ov)
	}
}

func TestSummarizeMarkdownDoc_PlainFences(t *testing.T) {
	resp := "```\n{\"abstract\":\"OK\",\"overview\":\"\"}\n```"
	g := &fakeGen{out: []string{resp}}
	ab, _, err := SummarizeMarkdownDoc(context.Background(), g, "m", "f.md", "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ab != "OK" {
		t.Errorf("abstract = %q, want \"OK\"", ab)
	}
}

func TestSummarizeMarkdownDoc_LLMError(t *testing.T) {
	// Non-retryable error — should fail immediately without retries
	g := &fakeGen{errs: []error{errors.New("fatal")}}
	_, _, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", "content")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if g.calls != 1 {
		t.Errorf("expected exactly 1 call on non-retryable error, got %d", g.calls)
	}
}

func TestSummarizeMarkdownDoc_InvalidJSON(t *testing.T) {
	g := &fakeGen{out: []string{"this is not json"}}
	_, _, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", "content")
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestSummarizeMarkdownDoc_EmptyAbstract(t *testing.T) {
	g := &fakeGen{out: []string{`{"abstract":"","overview":"Some overview"}`}}
	_, _, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", "content")
	if err == nil {
		t.Fatal("expected error when abstract is empty")
	}
}

func TestSummarizeMarkdownDoc_EmptyOverviewIsOK(t *testing.T) {
	// overview can be empty — only abstract is required
	g := &fakeGen{out: []string{`{"abstract":"Good abstract","overview":""}`}}
	ab, ov, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", "content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ab != "Good abstract" {
		t.Errorf("abstract = %q", ab)
	}
	if ov != "" {
		t.Errorf("overview = %q, want empty", ov)
	}
}

func TestSummarizeMarkdownDoc_TruncatesLargeContent(t *testing.T) {
	huge := strings.Repeat("x", MaxInputChars+1000)
	g := &fakeGen{out: []string{`{"abstract":"Truncated","overview":"Fine"}`}}
	ab, _, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", huge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ab != "Truncated" {
		t.Errorf("abstract = %q", ab)
	}
	if g.calls != 1 {
		t.Errorf("calls = %d, want 1", g.calls)
	}
}

func TestSummarizeMarkdownDoc_Retries429(t *testing.T) {
	oldSleep := sleepFn
	sleepFn = func(time.Duration) {}
	defer func() { sleepFn = oldSleep }()

	g := &fakeGen{
		errs: []error{statusErr{code: 429, msg: "rate limited"}},
		out:  []string{"", `{"abstract":"Retried","overview":"OK"}`},
	}
	ab, _, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", "content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ab != "Retried" {
		t.Errorf("abstract = %q, want \"Retried\"", ab)
	}
	if g.calls != 2 {
		t.Errorf("calls = %d, want 2 (1 fail + 1 retry)", g.calls)
	}
}

func TestSummarizeMarkdownDoc_ExhaustsRetries(t *testing.T) {
	oldSleep := sleepFn
	sleepFn = func(time.Duration) {}
	defer func() { sleepFn = oldSleep }()

	retryErr := statusErr{code: 503, msg: "unavailable"}
	g := &fakeGen{
		errs: []error{retryErr, retryErr, retryErr},
	}
	_, _, err := SummarizeMarkdownDoc(context.Background(), g, "m", "README.md", "content")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if g.calls != maxRetries {
		t.Errorf("calls = %d, want %d", g.calls, maxRetries)
	}
}
