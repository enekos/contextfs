package scraper

import "testing"

func TestScrapeManagerUsesCache(t *testing.T) {
	m := NewManager(NewCache())
	page1, _ := m.ProcessHTML("https://x", "<html><body>one</body></html>")
	page2, _ := m.ProcessHTML("https://x", "<html><body>two</body></html>")
	if page1.Content != page2.Content {
		t.Fatalf("expected cached page content, got %q vs %q", page1.Content, page2.Content)
	}
}
