package scraper

import "testing"

func TestSummarizePage(t *testing.T) {
	s := SummarizePage(Page{URL: "https://x", Content: "abcdefghijklmnopqrstuvwxyz"}, 5)
	if s.Abstract != "abcde" {
		t.Fatalf("unexpected summary: %q", s.Abstract)
	}
}
