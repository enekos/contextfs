package ast

import "testing"

func TestVueDescriber(t *testing.T) {
	d := VueDescriber{}
	g := d.ExtractFileGraph("a.vue", "<script>function mounted(){}</script>")
	if d.LanguageID() != "vue" {
		t.Fatalf("unexpected language id: %s", d.LanguageID())
	}
	if len(g.Symbols) == 0 {
		t.Fatal("expected symbols")
	}
}
