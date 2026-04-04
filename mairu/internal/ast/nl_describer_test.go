package ast

import (
	"strings"
	"testing"
)

func TestDescribeSymbols(t *testing.T) {
	out := DescribeSymbols(
		[]LogicSymbol{{ID: "fn:validate", Name: "validate", Kind: "fn"}},
		[]LogicEdge{{From: "fn:validate", To: "fn:trim", Kind: "call"}},
	)
	if !strings.Contains(out, "validate") || !strings.Contains(out, "Returns") {
		t.Fatalf("unexpected output: %s", out)
	}
}
