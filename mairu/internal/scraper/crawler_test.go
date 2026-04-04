package scraper

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCrawlURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>Hello</html>"))
	}))
	defer srv.Close()
	body, err := CrawlURL(srv.URL)
	if err != nil {
		t.Fatalf("crawl failed: %v", err)
	}
	if !strings.Contains(body, "Hello") {
		t.Fatalf("unexpected body: %s", body)
	}
}
