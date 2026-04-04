package ast

import "testing"

func TestTSXDescriber(t *testing.T) {
	d := TSXDescriber{}
	g := d.ExtractFileGraph("a.tsx", "export function Comp(){ return <div/> }")
	if len(g.Symbols) == 0 {
		t.Fatal("expected symbols")
	}
}
