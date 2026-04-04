package ast

import "testing"

func TestBaseExtract(t *testing.T) {
	src := "import { x } from './x'\nexport function hello(){ return 1 }\nexport function run(){ return hello() }"
	g := BaseExtract(src)
	if len(g.Symbols) < 2 {
		t.Fatalf("expected symbols, got %#v", g.Symbols)
	}
	if len(g.Imports) != 1 {
		t.Fatalf("expected one import, got %d", len(g.Imports))
	}
}

func TestSortHelpers(t *testing.T) {
	s := SortSymbols([]LogicSymbol{{ID: "b"}, {ID: "a"}})
	if s[0].ID != "a" {
		t.Fatalf("unexpected sort: %#v", s)
	}
	e := SortEdges([]LogicEdge{{From: "b", To: "x"}, {From: "a", To: "x"}})
	if e[0].From != "a" {
		t.Fatalf("unexpected edge sort: %#v", e)
	}
}
