package llm

import "context"

// Embedder is the minimal interface for generating text embeddings.
type Embedder interface {
	GetEmbedding(ctx context.Context, text string) ([]float32, error)
	GetEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error)
	GetEmbeddingDimension() int
}
