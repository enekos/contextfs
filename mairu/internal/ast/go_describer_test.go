package ast

import "testing"

func TestGoDescriber(t *testing.T) {
	d := GoDescriber{}
	g := d.ExtractFileGraph("a.go", "package a\nfunc Hello(){}\n")
	if len(g.Symbols) != 1 || g.Symbols[0].Name != "Hello" {
		t.Fatalf("unexpected symbols: %#v", g.Symbols)
	}
}
