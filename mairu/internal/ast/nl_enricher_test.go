package ast

import (
	"strings"
	"testing"
)

func TestEnrichDescriptions(t *testing.T) {
	in := map[string]string{"fn:a": "A"}
	out := EnrichDescriptions(in, []LogicEdge{{From: "fn:a", To: "fn:b", Kind: "call"}})
	if !strings.Contains(out["fn:a"], "fn:b") {
		t.Fatalf("expected enrichment, got: %s", out["fn:a"])
	}
}
