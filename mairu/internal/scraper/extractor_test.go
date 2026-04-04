package scraper

import "testing"

func TestExtractPage(t *testing.T) {
	p := ExtractPage("https://x", "<html><head><title>T</title></head><body>Hello <b>world</b></body></html>")
	if p.Title != "T" {
		t.Fatalf("expected title T, got %q", p.Title)
	}
	if p.Content == "" {
		t.Fatal("expected content")
	}
}
