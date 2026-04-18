package crawler

import (
	"context"
	"testing"
)

func TestScraperInitialization(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraper(engine, nil)
	if s == nil {
		t.Fatal("NewScraper returned nil")
	}
	if s.Engine != engine {
		t.Error("scraper engine mismatch")
	}

	rag := NewScraperWithRAG(engine, nil, nil)
	if rag == nil {
		t.Fatal("NewScraperWithRAG returned nil")
	}
}

func TestScraperSmartGraph(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraper(engine, nil)

	// With nil provider it should fail at extraction, but graph init is valid
	_, err := s.Smart(context.Background(), "invalid-url", "test prompt")
	if err == nil {
		t.Error("expected error for invalid URL with nil provider")
	}
}

func TestScraperSmartRAGGraph(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraperWithRAG(engine, nil, nil)

	_, err := s.SmartRAG(context.Background(), "invalid-url", "test prompt", 1000, 3)
	if err == nil {
		t.Error("expected error for invalid URL with nil provider")
	}
}

func TestScraperSmartRefinedGraph(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraper(engine, nil)

	_, err := s.SmartRefined(context.Background(), "invalid-url", "test prompt")
	if err == nil {
		t.Error("expected error for invalid URL with nil provider")
	}
}

func TestScraperMultiEmpty(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraper(engine, nil)

	results, err := s.Multi(context.Background(), []string{}, "prompt")
	if err != nil {
		t.Errorf("empty multi run should not fail: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestScraperOmniEmpty(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraper(engine, nil)

	results, err := s.Omni(context.Background(), []string{}, "prompt")
	if err != nil {
		t.Errorf("empty omni run should not fail: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 omni results, got %d", len(results))
	}
}

func TestScraperSearchEmptyQuery(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraper(engine, nil)

	_, err := s.Search(context.Background(), "", "prompt", 0)
	if err == nil {
		t.Error("expected error on empty query")
	}
}

func TestScraperDepthInvalidURL(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraper(engine, nil)

	_, err := s.Depth(context.Background(), "invalid-url", "prompt", 1)
	if err == nil {
		t.Error("expected error for depth search with invalid URL")
	}
}

func TestScraperScriptInvalidURL(t *testing.T) {
	engine := NewEngine(nil)
	s := NewScraper(engine, nil)

	_, err := s.Script(context.Background(), "invalid", "prompt")
	if err == nil {
		t.Error("expected error on invalid URL")
	}
}

func TestScraperFetchMissingEngine(t *testing.T) {
	s := &Scraper{Engine: nil}
	_, err := s.Fetch(context.Background(), "http://example.com")
	if err == nil {
		t.Error("expected error when engine is nil")
	}
}

func TestScraperCrawlMissingEngine(t *testing.T) {
	s := &Scraper{Engine: nil}
	_, err := s.Crawl(context.Background(), CrawlOptions{SeedURL: "http://example.com"})
	if err == nil {
		t.Error("expected error when engine is nil")
	}
}

func TestScraperIngestMissingEngine(t *testing.T) {
	s := &Scraper{Engine: nil}
	_, err := s.Ingest(context.Background(), ScrapeOptions{}, nil)
	if err == nil {
		t.Error("expected error when engine is nil")
	}
}
