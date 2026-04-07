package llm

import (
	"testing"
)

func TestEmbeddingCache_GetPut(t *testing.T) {
	c := NewEmbeddingCache(3)
	vec := []float32{0.1, 0.2, 0.3}

	if _, ok := c.Get("a"); ok {
		t.Fatal("expected cache miss for unseen key")
	}
	c.Put("a", vec)
	got, ok := c.Get("a")
	if !ok {
		t.Fatal("expected cache hit after Put")
	}
	if len(got) != len(vec) || got[0] != vec[0] {
		t.Fatalf("unexpected value: %v", got)
	}
}

func TestEmbeddingCache_LRUEviction(t *testing.T) {
	c := NewEmbeddingCache(2)
	c.Put("a", []float32{1})
	c.Put("b", []float32{2})
	// Access "a" to make "b" the LRU.
	c.Get("a")
	// Adding "c" should evict "b" (LRU).
	c.Put("c", []float32{3})

	if _, ok := c.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected 'a' to still be cached")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("expected 'c' to be cached")
	}
}

func TestEmbeddingCache_ZeroCapacityAlwaysMisses(t *testing.T) {
	c := NewEmbeddingCache(0)
	c.Put("x", []float32{1, 2, 3})
	if _, ok := c.Get("x"); ok {
		t.Fatal("zero-capacity cache should always miss")
	}
}

func TestEmbeddingCache_UpdateExistingKey(t *testing.T) {
	c := NewEmbeddingCache(5)
	c.Put("k", []float32{1})
	c.Put("k", []float32{9})
	got, ok := c.Get("k")
	if !ok || got[0] != 9 {
		t.Fatalf("expected updated value 9, got %v (ok=%v)", got, ok)
	}
}
