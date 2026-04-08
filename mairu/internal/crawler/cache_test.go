package crawler

import (
	"path/filepath"
	"testing"
)

func TestCache(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.json")

	c := NewCache(cachePath)
	if c == nil {
		t.Fatal("expected non-nil cache")
	}

	content := "test content"
	hash := c.ContentHash(content)

	entry := CacheEntry{
		ContentHash: hash,
		ScrapedAt:   "2023-01-01T00:00:00Z",
		URI:         "contextfs://scraped/example",
	}

	c.Set("http://example.com", entry)

	got, ok := c.Get("http://example.com")
	if !ok {
		t.Fatal("expected to find entry")
	}
	if got.ContentHash != hash {
		t.Fatalf("expected hash %s, got %s", hash, got.ContentHash)
	}

	if !c.IsUnchanged("http://example.com", content) {
		t.Fatal("expected content to be unchanged")
	}
	if c.IsUnchanged("http://example.com", "different content") {
		t.Fatal("expected content to be changed")
	}

	err := c.Save()
	if err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Test loading
	c2 := NewCache(cachePath)
	got2, ok2 := c2.Get("http://example.com")
	if !ok2 {
		t.Fatal("expected to load entry from saved cache")
	}
	if got2.ContentHash != hash {
		t.Fatalf("expected hash %s, got %s", hash, got2.ContentHash)
	}
}

func TestCache_NoFile(t *testing.T) {
	c := NewCache("")
	c.Set("test", CacheEntry{ContentHash: "hash"})
	if _, ok := c.Get("test"); !ok {
		t.Fatal("expected to get entry even without file")
	}
	if err := c.Save(); err != nil {
		t.Fatalf("expected nil error when saving with no file, got %v", err)
	}
}
