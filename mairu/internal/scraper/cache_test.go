package scraper

import "testing"

func TestCachePutGet(t *testing.T) {
	c := NewCache()
	c.Put(Page{URL: "https://x", Content: "body"})
	p, ok := c.Get("https://x")
	if !ok || p.Content != "body" {
		t.Fatalf("unexpected cache entry: %#v %v", p, ok)
	}
}
