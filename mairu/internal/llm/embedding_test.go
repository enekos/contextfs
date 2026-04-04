package llm

import "testing"

func TestEnsureEmbeddingDimension(t *testing.T) {
	if !EnsureEmbeddingDimension(make([]float32, 3072), 3072) {
		t.Fatal("expected true for matching dimensions")
	}
	if EnsureEmbeddingDimension(make([]float32, 10), 3072) {
		t.Fatal("expected false for mismatching dimensions")
	}
}
