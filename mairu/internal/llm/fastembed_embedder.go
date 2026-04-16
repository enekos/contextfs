package llm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	fastembed "github.com/anush008/fastembed-go"
)

// FastEmbedder generates embeddings locally using small ONNX models.
type FastEmbedder struct {
	model  fastembed.EmbeddingModel
	dim    int
	engine *fastembed.FlagEmbedding
}

// modelDimensionMap holds the known dimensions for fastembed models.
var modelDimensionMap = map[fastembed.EmbeddingModel]int{
	fastembed.AllMiniLML6V2: 384,
	fastembed.BGEBaseEN:     768,
	fastembed.BGEBaseENV15:  768,
	fastembed.BGESmallEN:    384,
	fastembed.BGESmallENV15: 384,
	fastembed.BGESmallZH:    512,
}

// modelNameMap translates config model names to fastembed enums.
var modelNameMap = map[string]fastembed.EmbeddingModel{
	"fast-all-MiniLM-L6-v2":  fastembed.AllMiniLML6V2,
	"fast-bge-base-en":       fastembed.BGEBaseEN,
	"fast-bge-base-en-v1.5":  fastembed.BGEBaseENV15,
	"fast-bge-small-en":      fastembed.BGESmallEN,
	"fast-bge-small-en-v1.5": fastembed.BGESmallENV15,
	"fast-bge-small-zh-v1.5": fastembed.BGESmallZH,
}

// NewFastEmbedder creates a local embedder using the fastembed library.
func NewFastEmbedder(model string, dim int) (*FastEmbedder, error) {
	fmodel, ok := modelNameMap[model]
	if !ok {
		// Default to a small, fast model if the requested name is unknown.
		fmodel = fastembed.BGESmallENV15
	}

	knownDim, hasDim := modelDimensionMap[fmodel]
	if hasDim && dim == 0 {
		dim = knownDim
	}
	if dim == 0 {
		dim = 384
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	cacheDir := filepath.Join(home, ".cache", "mairu", "fastembed")
	_ = os.MkdirAll(cacheDir, 0755)

	showProgress := false
	engine, err := fastembed.NewFlagEmbedding(&fastembed.InitOptions{
		Model:                fmodel,
		CacheDir:             cacheDir,
		ShowDownloadProgress: &showProgress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init fastembed: %w", err)
	}

	return &FastEmbedder{
		model:  fmodel,
		dim:    dim,
		engine: engine,
	}, nil
}

// GetEmbedding returns a single embedding vector.
func (f *FastEmbedder) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	batch, err := f.GetEmbeddingsBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(batch) == 0 {
		return nil, fmt.Errorf("fastembed returned empty embedding batch")
	}
	return batch[0], nil
}

// GetEmbeddingDimension returns the dimension of the model's embeddings.
func (f *FastEmbedder) GetEmbeddingDimension() int {
	return f.dim
}

// GetEmbeddingsBatch returns embedding vectors for multiple texts.
func (f *FastEmbedder) GetEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	vecs, err := f.engine.Embed(texts, 32)
	if err != nil {
		return nil, fmt.Errorf("fastembed inference failed: %w", err)
	}

	// Ensure output dimension matches expected size.
	for i, v := range vecs {
		if len(v) != f.dim {
			return nil, fmt.Errorf("fastembed returned dimension %d, expected %d (index %d)", len(v), f.dim, i)
		}
	}

	return vecs, nil
}
