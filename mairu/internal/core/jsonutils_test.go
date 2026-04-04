package core

import "testing"

func TestExtractJSONObject(t *testing.T) {
	got := ExtractJSONObject("prefix ```json {\"ok\":true} ``` suffix")
	if got == nil || got["ok"] != true {
		t.Fatalf("expected object with ok=true, got %#v", got)
	}
}

func TestExtractJSONArray(t *testing.T) {
	got := ExtractJSONArray("```json\n[{\"a\":1},{\"a\":2}]\n```")
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
}
