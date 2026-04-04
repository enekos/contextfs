package db

import (
	"os"
	"testing"
)

func TestDBGuardsWithoutClient(t *testing.T) {
	testDB := NewTestDB(".")

	if _, err := testDB.FindSymbol("Handler"); err == nil {
		t.Fatal("expected FindSymbol to return error when client is nil")
	}

	if err := testDB.InsertSymbol("id1", "file.go", "Handler", "function", true, 1, 1, 2, 5); err == nil {
		t.Fatal("expected InsertSymbol to return error when client is nil")
	}

	if _, err := testDB.UpsertFile("", "hash"); err == nil {
		t.Fatal("expected UpsertFile to reject empty path")
	}
}

func TestResolveMeiliConfig(t *testing.T) {
	t.Setenv("MEILI_URL", "http://env-meili:7700")
	t.Setenv("MEILI_API_KEY", "env-key")

	host, key := resolveMeiliConfig("", "")
	if host != "http://env-meili:7700" {
		t.Fatalf("expected host from env, got %q", host)
	}
	if key != "env-key" {
		t.Fatalf("expected key from env, got %q", key)
	}

	host, key = resolveMeiliConfig("http://flag-meili:7700", "flag-key")
	if host != "http://flag-meili:7700" {
		t.Fatalf("expected host from provided config, got %q", host)
	}
	if key != "flag-key" {
		t.Fatalf("expected key from provided config, got %q", key)
	}
}

func TestResolveMeiliConfigFallsBackToDefaultHost(t *testing.T) {
	t.Setenv("MEILI_URL", "")
	t.Setenv("MEILI_API_KEY", "")
	_ = os.Unsetenv("MEILI_URL")
	_ = os.Unsetenv("MEILI_API_KEY")

	host, key := resolveMeiliConfig("", "")
	if host != defaultMeiliURL {
		t.Fatalf("expected default host %q, got %q", defaultMeiliURL, host)
	}
	if key != "" {
		t.Fatalf("expected empty key, got %q", key)
	}
}

func TestRootAccessor(t *testing.T) {
	rootPath := "/tmp/test-project"
	db := NewTestDB(rootPath)
	if db.Root() != rootPath {
		t.Fatalf("expected root %q, got %q", rootPath, db.Root())
	}
}
