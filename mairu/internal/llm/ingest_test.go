package llm

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

type fakeGen struct {
	calls int
	out   []string
	errs  []error
}

func (f *fakeGen) GenerateContent(_ context.Context, _ string, _ string) (string, error) {
	f.calls++
	idx := f.calls - 1
	if idx < len(f.errs) && f.errs[idx] != nil {
		return "", f.errs[idx]
	}
	if idx < len(f.out) {
		return f.out[idx], nil
	}
	return "[]", nil
}

type statusErr struct {
	code int
	msg  string
}

func (s statusErr) Error() string   { return s.msg }
func (s statusErr) StatusCode() int { return s.code }

func TestParseTextRequiresAPIKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	_, err := ParseTextIntoContextNodes(context.Background(), &fakeGen{}, "m", "text", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTextTooLarge(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "x")
	huge := make([]byte, MaxInputChars+1)
	_, err := ParseTextIntoContextNodes(context.Background(), &fakeGen{}, "m", string(huge), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTextSuccess(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "x")
	g := &fakeGen{out: []string{`[{"uri":"contextfs://a","name":"A","abstract":"B","parent_uri":null}]`}}
	nodes, err := ParseTextIntoContextNodes(context.Background(), g, "m", "text", "contextfs://test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Name != "A" {
		t.Fatalf("unexpected nodes: %#v", nodes)
	}
}

func TestParseTextRetries429(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "x")
	oldSleep := sleepFn
	sleepFn = func(time.Duration) {}
	defer func() { sleepFn = oldSleep }()
	g := &fakeGen{
		errs: []error{statusErr{code: 429, msg: "rate"}},
		out:  []string{"", `[{"uri":"contextfs://a","name":"A","abstract":"B","parent_uri":null}]`},
	}
	nodes, err := ParseTextIntoContextNodes(context.Background(), g, "m", "text", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.calls != 2 || len(nodes) != 1 {
		t.Fatalf("expected retry and one node, calls=%d nodes=%d", g.calls, len(nodes))
	}
}

func TestParseTextInvalidNodes(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "x")
	g := &fakeGen{out: []string{`[{"invalid":"x"}]`}}
	_, err := ParseTextIntoContextNodes(context.Background(), g, "m", "text", "")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestShouldRetryFetchFailure(t *testing.T) {
	if !shouldRetry(errors.New("fetch failed on upstream")) {
		t.Fatal("expected fetch failure to retry")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	_ = os.Setenv("GEMINI_API_KEY", "")
	os.Exit(code)
}
