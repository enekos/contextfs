package ast

import "testing"

func TestPythonDescriber(t *testing.T) {
	d := PythonDescriber{}
	g := d.ExtractFileGraph("a.py", "def hello():\n  return 1")
	if len(g.Symbols) != 1 || g.Symbols[0].Name != "hello" {
		t.Fatalf("unexpected symbols: %#v", g.Symbols)
	}
}
