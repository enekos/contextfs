package cmd

import (
	"strings"
	"testing"
)

func TestResolveSocketPath_HonoursEnvVar(t *testing.T) {
	t.Setenv("MAIRU_INGEST_SOCK", "/tmp/custom.sock")

	got, err := resolveSocketPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/tmp/custom.sock" {
		t.Errorf("expected /tmp/custom.sock, got %q", got)
	}
}

func TestResolveSocketPath_DefaultsToHome(t *testing.T) {
	t.Setenv("MAIRU_INGEST_SOCK", "")
	t.Setenv("HOME", t.TempDir())

	got, err := resolveSocketPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const suffix = "/.mairu/ingest.sock"
	if !strings.HasSuffix(got, suffix) {
		t.Errorf("expected path to end with %q, got %q", suffix, got)
	}
}
