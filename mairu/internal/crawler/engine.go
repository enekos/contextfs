package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	markdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
)

var assetExtensions = regexp.MustCompile(`(?i)\.(png|jpg|jpeg|gif|svg|webp|ico|css|js|ts|woff|woff2|ttf|eot|pdf|zip|tar|gz|mp4|mp3|wav)$`)

// Engine is the mechanical heart of the crawler. It owns HTTP, caching,
// parsing, and concurrency so that every scraping mode shares the same
// infrastructure.
type Engine struct {
	Client      *http.Client
	Cache       *Cache
	Concurrency int
	UserAgent   string
	Timeout     time.Duration
}

// NewEngine creates an Engine with sensible defaults.
func NewEngine(cache *Cache) *Engine {
	return &Engine{
		Client:      &http.Client{Timeout: 15 * time.Second},
		Cache:       cache,
		Concurrency: 3,
		UserAgent:   "mairu-crawler/1.0",
		Timeout:     15 * time.Second,
	}
}

// Fetch performs an HTTP GET, respecting cache when available.
func (e *Engine) Fetch(ctx context.Context, targetURL string) (string, error) {
	if e.Cache != nil {
		if entry, ok := e.Cache.Get(targetURL); ok {
			// We only cache the hash, not the content, so we still need to fetch
			// to verify freshness. For now, just fetch and compare.
			_ = entry
		}
	}

	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: e.Timeout}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("fetch: failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", e.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch: bad status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("fetch: read failed: %w", err)
	}

	content := string(bodyBytes)

	if e.Cache != nil {
		e.Cache.Set(targetURL, CacheEntry{
			ContentHash: e.Cache.ContentHash(content),
			ScrapedAt:   time.Now().Format(time.RFC3339),
			URI:         URLToURI(targetURL),
		})
	}

	return content, nil
}

// Parse converts HTML to clean markdown/text using go-readability.
func (e *Engine) Parse(_ context.Context, htmlContent, pageURL string) (string, error) {
	trimmed := strings.TrimSpace(htmlContent)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") ||
		strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<rss") {
		return trimmed, nil
	}
	if !strings.HasPrefix(trimmed, "<") && strings.Contains(strings.SplitN(trimmed, "\n", 2)[0], ",") {
		return trimmed, nil
	}

	parsedURL, _ := url.Parse(pageURL)
	if parsedURL == nil {
		parsedURL, _ = url.Parse("http://localhost")
	}

	article, err := readability.FromReader(strings.NewReader(htmlContent), parsedURL)
	if err != nil {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
		if err == nil {
			return strings.TrimSpace(doc.Text()), nil
		}
		return "", fmt.Errorf("parse: readability failed: %w", err)
	}

	md, err := markdown.ConvertString(article.Content)
	if err == nil && md != "" {
		return md, nil
	}
	return article.TextContent, nil
}

// Crawl performs a breadth-first crawl starting from seedURL.
func (e *Engine) Crawl(ctx context.Context, opts CrawlOptions) (<-chan CrawledPage, error) {
	out := make(chan CrawledPage, 10)

	go func() {
		defer close(out)

		visited := make(map[string]bool)
		type queueItem struct {
			url   string
			depth int
		}
		queue := []queueItem{{url: opts.SeedURL, depth: 0}}
		pageCount := 0

		concurrency := opts.Concurrency
		if concurrency <= 0 {
			concurrency = e.Concurrency
		}
		if concurrency <= 0 {
			concurrency = 1
		}

		for len(queue) > 0 && pageCount < opts.MaxPages {
			var currentLevel []queueItem
			for _, item := range queue {
				if !visited[item.url] {
					visited[item.url] = true
					currentLevel = append(currentLevel, item)
				}
			}
			queue = nil

			resultsChan := make(chan *CrawledPage, len(currentLevel))
			var wg sync.WaitGroup
			sem := make(chan struct{}, concurrency)

			for _, item := range currentLevel {
				if pageCount >= opts.MaxPages {
					break
				}
				wg.Add(1)
				go func(qItem queueItem) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					page, err := e.fetchPage(ctx, qItem.url)
					if err == nil && page != nil {
						page.Depth = qItem.depth
						resultsChan <- page
					} else {
						resultsChan <- nil
					}
				}(item)
			}

			go func() {
				wg.Wait()
				close(resultsChan)
			}()

			for page := range resultsChan {
				if page == nil || pageCount >= opts.MaxPages {
					continue
				}
				page.Links = filterLinks(page.Links, opts.SeedURL, opts.URLPattern)
				pageCount++
				out <- *page

				if page.Depth < opts.MaxDepth {
					for _, link := range page.Links {
						if !visited[link] && pageCount < opts.MaxPages {
							queue = append(queue, queueItem{url: link, depth: page.Depth + 1})
						}
					}
				}
			}

			if opts.DelayMs > 0 && len(queue) > 0 {
				time.Sleep(time.Duration(opts.DelayMs) * time.Millisecond)
			}
		}
	}()

	return out, nil
}

func (e *Engine) fetchPage(ctx context.Context, targetURL string) (*CrawledPage, error) {
	html, err := e.Fetch(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	title := doc.Find("title").Text()
	var hrefs []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			hrefs = append(hrefs, href)
		}
	})

	return &CrawledPage{
		URL:   targetURL,
		HTML:  html,
		Title: title,
		Links: normalizeLinks(hrefs, targetURL),
	}, nil
}

func shouldFollowURL(testURL, seedOrigin, urlPattern string) bool {
	if testURL == "" || strings.HasPrefix(testURL, "#") || strings.HasPrefix(testURL, "mailto:") || strings.HasPrefix(testURL, "tel:") || strings.HasPrefix(testURL, "javascript:") {
		return false
	}
	parsed, err := url.Parse(testURL)
	if err != nil {
		return false
	}
	seedParsed, err := url.Parse(seedOrigin)
	if err != nil {
		return false
	}
	if parsed.Host != "" && parsed.Host != seedParsed.Host {
		return false
	}
	if assetExtensions.MatchString(parsed.Path) {
		return false
	}
	if urlPattern != "" {
		matched, _ := regexp.MatchString(urlPattern, parsed.Path)
		if !matched {
			return false
		}
	}
	return true
}

func normalizeLinks(hrefs []string, baseURL string) []string {
	seen := make(map[string]bool)
	var results []string
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	for _, href := range hrefs {
		parsed, err := url.Parse(href)
		if err != nil {
			continue
		}
		abs := base.ResolveReference(parsed)
		abs.Fragment = ""
		normalized := abs.String()
		if strings.HasSuffix(normalized, "/") && len(normalized) > 1 {
			normalized = strings.TrimSuffix(normalized, "/")
		}
		if !seen[normalized] {
			seen[normalized] = true
			results = append(results, normalized)
		}
	}
	return results
}

func filterLinks(urls []string, seedOrigin, urlPattern string) []string {
	var results []string
	for _, u := range urls {
		if shouldFollowURL(u, seedOrigin, urlPattern) {
			results = append(results, u)
		}
	}
	return results
}

// RunWorkers executes a function for each item with bounded concurrency.
func RunWorkers[T any, R any](ctx context.Context, items []T, concurrency int, worker func(context.Context, T) (R, error)) ([]R, []error) {
	if concurrency <= 0 {
		concurrency = 3
	}

	results := make([]R, len(items))
	errs := make([]error, len(items))

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(idx int, it T) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := worker(ctx, it)
			results[idx] = res
			errs[idx] = err
		}(i, item)
	}

	wg.Wait()
	return results, errs
}
