package llm

func EnsureEmbeddingDimension(vec []float32, expected int) bool {
	return len(vec) == expected
}
