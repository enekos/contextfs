package scraper

import (
	"strings"
	"testing"
)

func TestMapURLToContextURI(t *testing.T) {
	got := MapURLToContextURI("proj", "https://docs.example.com/path/to/page")
	if !strings.Contains(got, "contextfs://proj/web/docs.example.com/path-to-page") {
		t.Fatalf("unexpected uri: %s", got)
	}
}
