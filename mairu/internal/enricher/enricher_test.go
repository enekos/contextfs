package enricher

import (
	"context"
	"fmt"
	"testing"
)

type stubEnricher struct {
	name    string
	called  bool
	key     string
	value   string
	failErr error
}

func (s *stubEnricher) Name() string { return s.name }

func (s *stubEnricher) Enrich(ctx context.Context, fc *FileContext) error {
	s.called = true
	if s.failErr != nil {
		return s.failErr
	}
	fc.Metadata[s.key] = s.value
	return nil
}

func TestPipelineRunsAllEnrichers(t *testing.T) {
	e1 := &stubEnricher{name: "e1", key: "k1", value: "v1"}
	e2 := &stubEnricher{name: "e2", key: "k2", value: "v2"}
	p := NewPipeline([]Enricher{e1, e2})

	fc := &FileContext{
		FilePath: "/tmp/test.go",
		RelPath:  "test.go",
		WatchDir: "/tmp",
		Metadata: map[string]any{},
	}
	p.Run(context.Background(), fc)

	if !e1.called || !e2.called {
		t.Fatal("expected both enrichers to be called")
	}
	if fc.Metadata["k1"] != "v1" || fc.Metadata["k2"] != "v2" {
		t.Fatalf("metadata not set: %v", fc.Metadata)
	}
}

func TestPipelineContinuesOnError(t *testing.T) {
	e1 := &stubEnricher{name: "fail", key: "k1", value: "v1", failErr: fmt.Errorf("boom")}
	e2 := &stubEnricher{name: "ok", key: "k2", value: "v2"}
	p := NewPipeline([]Enricher{e1, e2})

	fc := &FileContext{Metadata: map[string]any{}}
	p.Run(context.Background(), fc)

	if !e2.called {
		t.Fatal("second enricher should run even if first fails")
	}
	if fc.Metadata["k2"] != "v2" {
		t.Fatal("second enricher should still write metadata")
	}
}

func TestPipelineNoEnrichers(t *testing.T) {
	p := NewPipeline(nil)
	fc := &FileContext{Metadata: map[string]any{}}
	p.Run(context.Background(), fc)
	// No panic, no error
}
