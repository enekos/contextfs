package enricher

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitIntentEnricher(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "auth.go")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")

	// Create conventional commits
	os.WriteFile(filePath, []byte("func validate() {}"), 0o644)
	run("add", "auth.go")
	run("commit", "-m", "feat: add token validation")

	os.WriteFile(filePath, []byte("func validate() { check() }"), 0o644)
	run("add", "auth.go")
	run("commit", "-m", "fix: handle expired tokens")

	os.WriteFile(filePath, []byte("func validate() { check(); log() }"), 0o644)
	run("add", "auth.go")
	run("commit", "-m", "perf: optimize token lookup")

	e := &GitIntentEnricher{MaxCommits: 20}
	fc := &FileContext{
		FilePath: filePath,
		RelPath:  "auth.go",
		WatchDir: dir,
		Metadata: map[string]any{},
	}

	if err := e.Enrich(context.Background(), fc); err != nil {
		t.Fatalf("enrich failed: %v", err)
	}

	intent, ok := fc.Metadata["enrichment_intent"].(string)
	if !ok || intent == "" {
		t.Fatalf("expected enrichment_intent, got: %v", fc.Metadata)
	}

	// Should mention the commit message themes
	if !strings.Contains(intent, "feat") && !strings.Contains(intent, "fix") && !strings.Contains(intent, "perf") {
		t.Fatalf("intent should reference commit prefixes, got: %s", intent)
	}

	prefixes, ok := fc.Metadata["enrichment_commit_prefixes"].(map[string]int)
	if !ok {
		t.Fatalf("expected enrichment_commit_prefixes map, got: %v", fc.Metadata["enrichment_commit_prefixes"])
	}
	if prefixes["feat"] != 1 || prefixes["fix"] != 1 || prefixes["perf"] != 1 {
		t.Fatalf("unexpected prefix counts: %v", prefixes)
	}
}

func TestGitIntentNoHistory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "new.go")
	os.WriteFile(filePath, []byte("package main"), 0o644)

	e := &GitIntentEnricher{MaxCommits: 20}
	fc := &FileContext{
		FilePath: filePath,
		RelPath:  "new.go",
		WatchDir: dir,
		Metadata: map[string]any{},
	}

	if err := e.Enrich(context.Background(), fc); err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if _, ok := fc.Metadata["enrichment_intent"]; ok {
		t.Fatal("should not set intent when git is unavailable")
	}
}

func TestGitIntentNonConventionalCommits(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "utils.go")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")

	os.WriteFile(filePath, []byte("func helper() {}"), 0o644)
	run("add", "utils.go")
	run("commit", "-m", "Added helper function")

	os.WriteFile(filePath, []byte("func helper() { return }"), 0o644)
	run("add", "utils.go")
	run("commit", "-m", "Updated helper to return early")

	e := &GitIntentEnricher{MaxCommits: 20}
	fc := &FileContext{
		FilePath: filePath,
		RelPath:  "utils.go",
		WatchDir: dir,
		Metadata: map[string]any{},
	}

	if err := e.Enrich(context.Background(), fc); err != nil {
		t.Fatalf("enrich failed: %v", err)
	}

	// Should still produce an intent from the raw messages
	intent, ok := fc.Metadata["enrichment_intent"].(string)
	if !ok || intent == "" {
		t.Fatalf("expected intent even without conventional commits, got: %v", fc.Metadata)
	}
}
