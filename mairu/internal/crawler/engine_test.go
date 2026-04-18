package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewEngineDefaults(t *testing.T) {
	e := NewEngine(nil)
	if e.Concurrency != 3 {
		t.Errorf("expected default concurrency 3, got %d", e.Concurrency)
	}
	if e.Timeout != 15*time.Second {
		t.Errorf("expected default timeout 15s, got %v", e.Timeout)
	}
	if e.UserAgent != "mairu-crawler/1.0" {
		t.Errorf("expected default user-agent, got %s", e.UserAgent)
	}
}

func TestEngineFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "mairu-crawler/1.0" {
			t.Errorf("unexpected user-agent: %s", r.Header.Get("User-Agent"))
		}
		w.Write([]byte("<html><body>Hello</body></html>"))
	}))
	defer ts.Close()

	e := NewEngine(nil)
	content, err := e.Fetch(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if content != "<html><body>Hello</body></html>" {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestEngineFetchBadStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	e := NewEngine(nil)
	_, err := e.Fetch(context.Background(), ts.URL)
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestEngineParseHTML(t *testing.T) {
	e := NewEngine(nil)
	html := `<html><head><title>Test</title></head><body><h1>Header</h1><p>Paragraph</p></body></html>`
	doc, err := e.Parse(context.Background(), html, "http://localhost")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if doc == "" {
		t.Error("parsed doc should not be empty")
	}
}

func TestEngineParseJSONBypass(t *testing.T) {
	e := NewEngine(nil)
	json := `{"key": "value"}`
	doc, err := e.Parse(context.Background(), json, "http://localhost")
	if err != nil {
		t.Fatalf("parse failed on JSON: %v", err)
	}
	if doc != json {
		t.Errorf("JSON should pass through unchanged, got: %s", doc)
	}
}

func TestEngineCrawl(t *testing.T) {
	visited := make(map[string]bool)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited[r.URL.Path] = true
		switch r.URL.Path {
		case "/":
			w.Write([]byte(`<html><body><a href="/page1">Page 1</a><a href="/page2">Page 2</a></body></html>`))
		case "/page1":
			w.Write([]byte(`<html><body><p>Page 1 content</p></body></html>`))
		case "/page2":
			w.Write([]byte(`<html><body><p>Page 2 content</p></body></html>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	e := NewEngine(nil)
	out, err := e.Crawl(context.Background(), CrawlOptions{
		SeedURL:  ts.URL,
		MaxPages: 10,
		MaxDepth: 2,
	})
	if err != nil {
		t.Fatalf("crawl failed: %v", err)
	}

	var pages []CrawledPage
	for page := range out {
		pages = append(pages, page)
	}

	if len(pages) < 1 {
		t.Fatalf("expected at least 1 page, got %d", len(pages))
	}
	if pages[0].URL != ts.URL {
		t.Errorf("expected first page to be seed URL, got %s", pages[0].URL)
	}
}

func TestRunWorkers(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	results, errs := RunWorkers(context.Background(), items, 2, func(_ context.Context, n int) (int, error) {
		return n * 2, nil
	})

	if len(results) != len(items) {
		t.Fatalf("expected %d results, got %d", len(items), len(results))
	}
	for i, r := range results {
		if r != items[i]*2 {
			t.Errorf("expected %d, got %d", items[i]*2, r)
		}
		if errs[i] != nil {
			t.Errorf("unexpected error at index %d: %v", i, errs[i])
		}
	}
}

func TestShouldFollowURL(t *testing.T) {
	seed := "https://example.com"
	cases := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/page", true},
		{"https://other.com/page", false},
		{"mailto:test@example.com", false},
		{"javascript:void(0)", false},
		{"https://example.com/image.png", false},
		{"", false},
	}
	for _, c := range cases {
		got := shouldFollowURL(c.url, seed, "")
		if got != c.expected {
			t.Errorf("shouldFollowURL(%q) = %v, want %v", c.url, got, c.expected)
		}
	}
}

func TestNormalizeLinks(t *testing.T) {
	base := "https://example.com/path/"
	links := []string{"page1", "/page2", "https://example.com/page3", "page1"}
	result := normalizeLinks(links, base)

	seen := make(map[string]bool)
	for _, r := range result {
		if seen[r] {
			t.Errorf("duplicate link: %s", r)
		}
		seen[r] = true
	}

	if len(result) != 3 {
		t.Errorf("expected 3 unique links, got %d: %v", len(result), result)
	}
}
