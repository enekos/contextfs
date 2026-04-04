package dreamer

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	p := BuildPrompt("ship migration", []string{"module A", "module B"})
	if !strings.Contains(p, "ship migration") || !strings.Contains(p, "module A") {
		t.Fatalf("unexpected prompt: %s", p)
	}
}
