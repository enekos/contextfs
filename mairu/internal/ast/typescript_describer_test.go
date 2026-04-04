package ast

import "testing"

func TestTypeScriptDescriber(t *testing.T) {
	d := TypeScriptDescriber{}
	g := d.ExtractFileGraph("a.ts", "export function hello(){ return 1 }")
	if d.LanguageID() != "typescript" {
		t.Fatalf("unexpected language id: %s", d.LanguageID())
	}
	if len(g.Symbols) == 0 {
		t.Fatal("expected symbols")
	}
}
