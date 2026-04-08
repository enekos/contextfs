package enricher

import (
	"context"
	"fmt"
)

// FileContext holds the data an enricher needs to augment a daemon-processed file.
// Enrichers write their results into the Metadata map.
type FileContext struct {
	FilePath string         // absolute path to the file
	RelPath  string         // relative path from WatchDir
	WatchDir string         // root directory the daemon watches (typically a git repo root)
	Metadata map[string]any // enrichers add keys here; flows to Meilisearch via the manager
}

// Enricher adds a layer of meaning to a file's context node metadata.
type Enricher interface {
	Name() string
	Enrich(ctx context.Context, fc *FileContext) error
}

// Pipeline holds an ordered list of enrichers and runs them sequentially.
// If an enricher fails, it logs a warning and continues with the next one.
type Pipeline struct {
	enrichers []Enricher
}

// NewPipeline creates a pipeline from the given enrichers.
func NewPipeline(enrichers []Enricher) *Pipeline {
	return &Pipeline{enrichers: enrichers}
}

// Run applies all enrichers to the file context in order.
// Errors from individual enrichers are printed as warnings; the pipeline does not abort.
func (p *Pipeline) Run(ctx context.Context, fc *FileContext) {
	for _, e := range p.enrichers {
		if err := e.Enrich(ctx, fc); err != nil {
			fmt.Printf("[enricher:%s] warning: %v\n", e.Name(), err)
		}
	}
}
