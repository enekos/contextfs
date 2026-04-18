package crawler

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"mairu/internal/llm"
)

// Scraper is the single, unified entry point for all web scraping operations.
// It composes an Engine (HTTP, cache, parsing, concurrency) with an LLM
// Provider to power every scraping mode the codebase needs.
type Scraper struct {
	Engine   *Engine
	Provider llm.Provider
	Embedder llm.Embedder // optional, for RAG mode
}

// NewScraper creates a Scraper with the given engine and provider.
func NewScraper(engine *Engine, provider llm.Provider) *Scraper {
	return &Scraper{
		Engine:   engine,
		Provider: provider,
	}
}

// NewScraperWithRAG creates a Scraper that supports RAG-based extraction.
func NewScraperWithRAG(engine *Engine, provider llm.Provider, embedder llm.Embedder) *Scraper {
	return &Scraper{
		Engine:   engine,
		Provider: provider,
		Embedder: embedder,
	}
}

// ============================================================================
// Single-URL Smart Scrape
// ============================================================================

// Smart extracts structured data from a single URL using an LLM.
func (s *Scraper) Smart(ctx context.Context, targetURL, prompt string) (map[string]any, error) {
	graph := NewGraph(
		&FetchNode{Engine: s.Engine},
		&ParseNode{Engine: s.Engine},
		&ExtractNode{Provider: s.Provider},
	)

	state, err := graph.Run(ctx, State{"url": targetURL, "prompt": prompt})
	if err != nil {
		return nil, err
	}
	if data, ok := state["extracted_data"].(map[string]any); ok {
		return data, nil
	}
	return nil, nil
}

// SmartRAG extracts structured data from a single URL using RAG for large docs.
func (s *Scraper) SmartRAG(ctx context.Context, targetURL, prompt string, chunkSize, topK int) (map[string]any, error) {
	graph := NewGraph(
		&FetchNode{Engine: s.Engine},
		&ParseNode{Engine: s.Engine},
		&RAGExtractNode{
			Provider:  s.Provider,
			Embedder:  s.Embedder,
			ChunkSize: chunkSize,
			TopK:      topK,
		},
	)

	state, err := graph.Run(ctx, State{"url": targetURL, "prompt": prompt})
	if err != nil {
		return nil, err
	}
	if data, ok := state["extracted_data"].(map[string]any); ok {
		return data, nil
	}
	return nil, nil
}

// SmartRefined extracts structured data with prompt refinement.
func (s *Scraper) SmartRefined(ctx context.Context, targetURL, prompt string) (map[string]any, error) {
	graph := NewGraph(
		&FetchNode{Engine: s.Engine},
		&ParseNode{Engine: s.Engine},
		&PromptRefinerNode{Provider: s.Provider},
		&ExtractNode{Provider: s.Provider},
	)

	state, err := graph.Run(ctx, State{"url": targetURL, "prompt": prompt})
	if err != nil {
		return nil, err
	}
	if data, ok := state["extracted_data"].(map[string]any); ok {
		return data, nil
	}
	return nil, nil
}

// ============================================================================
// Multi-URL Scrape
// ============================================================================

// Multi extracts structured data from multiple URLs concurrently.
func (s *Scraper) Multi(ctx context.Context, targetURLs []string, prompt string) (map[string]map[string]any, error) {
	results := make(map[string]map[string]any)
	var mu sync.Mutex

	concurrency := s.Engine.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}

	_, errs := RunWorkers(ctx, targetURLs, concurrency, func(ctx context.Context, url string) (struct{}, error) {
		data, err := s.Smart(ctx, url, prompt)
		if err != nil {
			return struct{}{}, err
		}
		if data != nil {
			mu.Lock()
			results[url] = data
			mu.Unlock()
		}
		return struct{}{}, nil
	})

	var firstErr error
	for _, err := range errs {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if firstErr != nil && len(results) == 0 {
		return nil, firstErr
	}
	return results, nil
}

// ============================================================================
// Search Scrape
// ============================================================================

// Search searches DuckDuckGo and extracts data from the top results.
func (s *Scraper) Search(ctx context.Context, query, prompt string, maxResults int) ([]map[string]any, error) {
	if maxResults <= 0 {
		maxResults = 3
	}

	searchState := State{"search_query": query}
	searchNode := &SearchNode{Engine: s.Engine, MaxResults: maxResults}
	searchState, err := searchNode.Execute(ctx, searchState)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	urls, ok := searchState["search_results"].([]string)
	if !ok || len(urls) == 0 {
		return nil, fmt.Errorf("no search results found")
	}

	var results []map[string]any
	for _, u := range urls {
		data, err := s.Smart(ctx, u, prompt)
		if err == nil && len(data) > 0 {
			data["_source_url"] = u
			results = append(results, data)
		}
	}

	return results, nil
}

// ============================================================================
// Omni Scrape (multi + merge)
// ============================================================================

// Omni extracts data from multiple URLs and merges it into a single response.
func (s *Scraper) Omni(ctx context.Context, targetURLs []string, prompt string) (map[string]any, error) {
	multiResults, err := s.Multi(ctx, targetURLs, prompt)
	if err != nil {
		return nil, fmt.Errorf("multi-scrape failed: %w", err)
	}
	if len(multiResults) == 0 {
		return nil, nil
	}

	state := State{"prompt": prompt, "results": multiResults}
	mergeNode := &MergeAnswersNode{Provider: s.Provider}
	finalState, err := mergeNode.Execute(ctx, state)
	if err != nil {
		return nil, err
	}

	if data, ok := finalState["merged_data"].(map[string]any); ok {
		return data, nil
	}
	return nil, nil
}

// ============================================================================
// Depth Search Scrape
// ============================================================================

// Depth crawls a seed URL, discovers relevant links, and scrapes them.
func (s *Scraper) Depth(ctx context.Context, seedURL, prompt string, maxDepth int) (map[string]map[string]any, error) {
	if maxDepth < 0 {
		maxDepth = 0
	}

	visited := make(map[string]bool)
	queue := []string{seedURL}
	var relevantURLs []string

	for depth := 0; depth <= maxDepth; depth++ {
		var nextQueue []string
		for _, currentURL := range queue {
			if visited[currentURL] {
				continue
			}
			visited[currentURL] = true
			relevantURLs = append(relevantURLs, currentURL)

			if depth < maxDepth {
				links, err := s.searchLinks(ctx, currentURL, prompt)
				if err == nil {
					base, err := url.Parse(currentURL)
					if err == nil {
						for _, l := range links {
							parsed, err := url.Parse(l)
							if err == nil {
								abs := base.ResolveReference(parsed)
								abs.Fragment = ""
								normalized := abs.String()
								if !visited[normalized] {
									nextQueue = append(nextQueue, normalized)
								}
							}
						}
					}
				}
			}
		}
		queue = nextQueue
		if len(queue) == 0 {
			break
		}
	}

	if len(relevantURLs) == 0 {
		return nil, fmt.Errorf("no relevant URLs found")
	}

	return s.Multi(ctx, relevantURLs, prompt)
}

func (s *Scraper) searchLinks(ctx context.Context, targetURL, prompt string) ([]string, error) {
	graph := NewGraph(
		&FetchNode{Engine: s.Engine},
		&SearchLinkNode{Provider: s.Provider},
	)
	state, err := graph.Run(ctx, State{"url": targetURL, "prompt": prompt})
	if err != nil {
		return nil, err
	}
	if links, ok := state["relevant_links"].([]string); ok {
		return links, nil
	}
	return nil, nil
}

// ============================================================================
// Script Generation
// ============================================================================

// Script generates a custom Go scraper script for a URL.
func (s *Scraper) Script(ctx context.Context, targetURL, prompt string) (string, error) {
	graph := NewGraph(
		&FetchNode{Engine: s.Engine},
		&MinifyHTMLNode{},
		&GenerateScriptNode{Provider: s.Provider},
	)

	state, err := graph.Run(ctx, State{"url": targetURL, "prompt": prompt})
	if err != nil {
		return "", err
	}
	if script, ok := state["generated_script"].(string); ok {
		return script, nil
	}
	return "", nil
}

// ============================================================================
// Raw Fetch
// ============================================================================

// Fetch performs a raw HTTP GET and returns the response body as a string.
func (s *Scraper) Fetch(ctx context.Context, targetURL string) (string, error) {
	if s.Engine == nil {
		return "", fmt.Errorf("scraper: missing Engine")
	}
	return s.Engine.Fetch(ctx, targetURL)
}

// ============================================================================
// BFS Crawl
// ============================================================================

// Crawl performs a breadth-first crawl starting from a seed URL.
func (s *Scraper) Crawl(ctx context.Context, opts CrawlOptions) (<-chan CrawledPage, error) {
	if s.Engine == nil {
		return nil, fmt.Errorf("scraper: missing Engine")
	}
	return s.Engine.Crawl(ctx, opts)
}

// ============================================================================
// Ingest Pipeline
// ============================================================================

// Ingest crawls a seed URL and stores extracted context nodes.
func (s *Scraper) Ingest(ctx context.Context, options ScrapeOptions, storeFn NodeStoreFunc) (*ScrapeResult, error) {
	out, err := s.Crawl(ctx, options.CrawlOptions)
	if err != nil {
		return nil, err
	}

	graph := NewGraph(
		&MarkdownExtractNode{Selector: options.Selector},
		&SummarizeNode{Provider: s.Provider},
		&StoreContextNode{
			StoreFn: storeFn,
			Project: options.Project,
			DryRun:  options.DryRun,
		},
	)

	result := &ScrapeResult{}
	for page := range out {
		result.PagesTotal++

		state := State{"url": page.URL, "html": page.HTML}
		finalState, err := graph.Run(ctx, state)
		if err != nil {
			result.Errors = append(result.Errors, ScrapeError{URL: page.URL, Error: err.Error()})
		} else {
			if skipped, _ := finalState["skipped"].(bool); skipped {
				result.PagesSkipped++
			} else if stored, _ := finalState["stored"].(bool); stored {
				result.PagesStored++
			}
		}
	}

	return result, nil
}
