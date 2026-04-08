# Enricher Pipeline Phase 1: GitIntent + ChangeVelocity

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a pluggable enrichment pipeline to the daemon that layers semantic intent (from git history) and churn signals (from commit frequency) onto context nodes before they're written to Meilisearch.

**Architecture:** A new `enricher` package defines an `Enricher` interface and `Pipeline` runner. Two concrete enrichers (`GitIntentEnricher`, `ChangeVelocityEnricher`) shell out to `git log`/`git blame` to extract per-file metadata. The pipeline runs inside the daemon's `ProcessFile` method after AST summarization, writing results into the node's `metadata` map. The context server promotes enrichment keys from metadata to top-level Meilisearch fields so they're searchable.

**Tech Stack:** Go 1.25+, `os/exec` for git commands, existing daemon/config/contextsrv packages.

**Spec:** `docs/superpowers/specs/2026-04-08-enricher-chronicle-prefetch-design.md`

---

## File Structure

| Action | Path | Responsibility |
|---|---|---|
| Create | `mairu/internal/enricher/enricher.go` | Interface, FileContext type, Pipeline runner |
| Create | `mairu/internal/enricher/enricher_test.go` | Pipeline unit tests |
| Create | `mairu/internal/enricher/git_intent.go` | GitIntentEnricher implementation |
| Create | `mairu/internal/enricher/git_intent_test.go` | GitIntentEnricher unit tests |
| Create | `mairu/internal/enricher/change_velocity.go` | ChangeVelocityEnricher implementation |
| Create | `mairu/internal/enricher/change_velocity_test.go` | ChangeVelocityEnricher unit tests |
| Modify | `mairu/internal/daemon/daemon.go` | Add enricher pipeline field + call in ProcessFile |
| Modify | `mairu/internal/daemon/daemon_test.go` | Test enricher integration in daemon |
| Modify | `mairu/internal/config/config.go` | Add EnricherConfig section |
| Modify | `mairu/internal/cmd/daemon.go` | Wire enricher pipeline from config |
| Modify | `mairu/internal/contextsrv/service_context.go` | Promote enrichment metadata to top-level Meili fields |

---

### Task 1: Enricher interface and pipeline

**Files:**
- Create: `mairu/internal/enricher/enricher.go`
- Create: `mairu/internal/enricher/enricher_test.go`

- [ ] **Step 1: Write the failing test for Pipeline.Run**

```go
// mairu/internal/enricher/enricher_test.go
package enricher

import (
	"context"
	"fmt"
	"testing"
)

type stubEnricher struct {
	name    string
	called  bool
	key     string
	value   string
	failErr error
}

func (s *stubEnricher) Name() string { return s.name }

func (s *stubEnricher) Enrich(ctx context.Context, fc *FileContext) error {
	s.called = true
	if s.failErr != nil {
		return s.failErr
	}
	fc.Metadata[s.key] = s.value
	return nil
}

func TestPipelineRunsAllEnrichers(t *testing.T) {
	e1 := &stubEnricher{name: "e1", key: "k1", value: "v1"}
	e2 := &stubEnricher{name: "e2", key: "k2", value: "v2"}
	p := NewPipeline([]Enricher{e1, e2})

	fc := &FileContext{
		FilePath: "/tmp/test.go",
		RelPath:  "test.go",
		WatchDir: "/tmp",
		Metadata: map[string]any{},
	}
	p.Run(context.Background(), fc)

	if !e1.called || !e2.called {
		t.Fatal("expected both enrichers to be called")
	}
	if fc.Metadata["k1"] != "v1" || fc.Metadata["k2"] != "v2" {
		t.Fatalf("metadata not set: %v", fc.Metadata)
	}
}

func TestPipelineContinuesOnError(t *testing.T) {
	e1 := &stubEnricher{name: "fail", key: "k1", value: "v1", failErr: fmt.Errorf("boom")}
	e2 := &stubEnricher{name: "ok", key: "k2", value: "v2"}
	p := NewPipeline([]Enricher{e1, e2})

	fc := &FileContext{Metadata: map[string]any{}}
	p.Run(context.Background(), fc)

	if !e2.called {
		t.Fatal("second enricher should run even if first fails")
	}
	if fc.Metadata["k2"] != "v2" {
		t.Fatal("second enricher should still write metadata")
	}
}

func TestPipelineNoEnrichers(t *testing.T) {
	p := NewPipeline(nil)
	fc := &FileContext{Metadata: map[string]any{}}
	p.Run(context.Background(), fc)
	// No panic, no error
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mairu && go test ./internal/enricher/ -v -run TestPipeline`
Expected: Compilation errors — `enricher` package doesn't exist yet.

- [ ] **Step 3: Implement enricher package**

```go
// mairu/internal/enricher/enricher.go
package enricher

import (
	"context"
	"fmt"
)

// FileContext holds the data an enricher needs to augment a daemon-processed file.
// Enrichers write their results into the Metadata map.
type FileContext struct {
	FilePath string         // absolute path to the file
	RelPath  string         // relative path from WatchDir
	WatchDir string         // root directory the daemon watches (typically a git repo root)
	Metadata map[string]any // enrichers add keys here; flows to Meilisearch via the manager
}

// Enricher adds a layer of meaning to a file's context node metadata.
type Enricher interface {
	Name() string
	Enrich(ctx context.Context, fc *FileContext) error
}

// Pipeline holds an ordered list of enrichers and runs them sequentially.
// If an enricher fails, it logs a warning and continues with the next one.
type Pipeline struct {
	enrichers []Enricher
}

// NewPipeline creates a pipeline from the given enrichers.
func NewPipeline(enrichers []Enricher) *Pipeline {
	return &Pipeline{enrichers: enrichers}
}

// Run applies all enrichers to the file context in order.
// Errors from individual enrichers are printed as warnings; the pipeline does not abort.
func (p *Pipeline) Run(ctx context.Context, fc *FileContext) {
	for _, e := range p.enrichers {
		if err := e.Enrich(ctx, fc); err != nil {
			fmt.Printf("[enricher:%s] warning: %v\n", e.Name(), err)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mairu && go test ./internal/enricher/ -v -run TestPipeline`
Expected: All 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd mairu && git add internal/enricher/enricher.go internal/enricher/enricher_test.go
git commit -m "feat(enricher): add enricher interface and pipeline runner"
```

---

### Task 2: ChangeVelocityEnricher

Starting with this one because it's simpler (no LLM, no blame parsing) — validates the enricher pattern first.

**Files:**
- Create: `mairu/internal/enricher/change_velocity.go`
- Create: `mairu/internal/enricher/change_velocity_test.go`

- [ ] **Step 1: Write the failing test**

The test creates a real git repo with known commit history, then runs the enricher against it.

```go
// mairu/internal/enricher/change_velocity_test.go
package enricher

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a git repo at dir with a tracked file and N commits.
func initGitRepo(t *testing.T, dir, filename string, commitCount int) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)

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

	for i := 0; i < commitCount; i++ {
		content := []byte("line " + string(rune('A'+i)) + "\n")
		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", filename)
		run("commit", "-m", "commit "+string(rune('A'+i)))
	}
	return filePath
}

func TestChangeVelocityEnricher(t *testing.T) {
	dir := t.TempDir()
	filePath := initGitRepo(t, dir, "main.go", 5)

	e := &ChangeVelocityEnricher{LookbackDays: 180}
	fc := &FileContext{
		FilePath: filePath,
		RelPath:  "main.go",
		WatchDir: dir,
		Metadata: map[string]any{},
	}

	if err := e.Enrich(context.Background(), fc); err != nil {
		t.Fatalf("enrich failed: %v", err)
	}

	score, ok := fc.Metadata["enrichment_churn_score"].(float64)
	if !ok {
		t.Fatalf("expected enrichment_churn_score in metadata, got: %v", fc.Metadata)
	}
	if score <= 0 {
		t.Fatalf("expected positive churn score for 5-commit file, got %f", score)
	}

	label, ok := fc.Metadata["enrichment_churn_label"].(string)
	if !ok || label == "" {
		t.Fatalf("expected enrichment_churn_label, got: %v", fc.Metadata)
	}

	total, ok := fc.Metadata["enrichment_total_commits"].(int)
	if !ok || total != 5 {
		t.Fatalf("expected 5 total commits, got: %v", fc.Metadata["enrichment_total_commits"])
	}
}

func TestChangeVelocityNoGitRepo(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "orphan.go")
	os.WriteFile(filePath, []byte("package main"), 0o644)

	e := &ChangeVelocityEnricher{LookbackDays: 180}
	fc := &FileContext{
		FilePath: filePath,
		RelPath:  "orphan.go",
		WatchDir: dir,
		Metadata: map[string]any{},
	}

	// Should not error — just skip enrichment gracefully
	if err := e.Enrich(context.Background(), fc); err != nil {
		t.Fatalf("should not error on non-git directory: %v", err)
	}
	if _, ok := fc.Metadata["enrichment_churn_score"]; ok {
		t.Fatal("should not set churn score when git is unavailable")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mairu && go test ./internal/enricher/ -v -run TestChangeVelocity`
Expected: Compilation error — `ChangeVelocityEnricher` not defined.

- [ ] **Step 3: Implement ChangeVelocityEnricher**

```go
// mairu/internal/enricher/change_velocity.go
package enricher

import (
	"context"
	"math"
	"os/exec"
	"strings"
	"time"
)

// ChangeVelocityEnricher computes churn signals for a file based on git commit frequency.
// It writes enrichment_churn_score (0.0–1.0), enrichment_churn_label, and
// enrichment_total_commits into fc.Metadata.
type ChangeVelocityEnricher struct {
	LookbackDays int // how far back to analyze; 0 defaults to 180
}

func (e *ChangeVelocityEnricher) Name() string { return "change_velocity" }

func (e *ChangeVelocityEnricher) Enrich(ctx context.Context, fc *FileContext) error {
	lookback := e.LookbackDays
	if lookback <= 0 {
		lookback = 180
	}

	timestamps, err := gitCommitTimestamps(ctx, fc.WatchDir, fc.RelPath)
	if err != nil || len(timestamps) == 0 {
		return nil // no git history — skip silently
	}

	now := time.Now()
	cutoff := now.AddDate(0, 0, -lookback)

	var recentCount int
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			recentCount++
		}
	}

	// Churn score: normalized commits-per-day in the lookback window, capped at 1.0.
	// A file changing once per day = 1.0; once per month ≈ 0.03.
	daysInWindow := float64(lookback)
	score := math.Min(float64(recentCount)/daysInWindow, 1.0)

	label := "stable"
	if score >= 0.5 {
		label = "volatile"
	} else if score >= 0.1 {
		label = "moderate"
	}

	fc.Metadata["enrichment_churn_score"] = score
	fc.Metadata["enrichment_churn_label"] = label
	fc.Metadata["enrichment_total_commits"] = len(timestamps)
	fc.Metadata["enrichment_recent_commits"] = recentCount

	return nil
}

// gitCommitTimestamps returns author-date timestamps for all commits touching relPath.
func gitCommitTimestamps(ctx context.Context, repoDir, relPath string) ([]time.Time, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--follow", "--format=%aI", "--", relPath)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var timestamps []time.Time
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, line)
		if err != nil {
			continue
		}
		timestamps = append(timestamps, t)
	}
	return timestamps, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mairu && go test ./internal/enricher/ -v -run TestChangeVelocity`
Expected: Both tests PASS.

- [ ] **Step 5: Commit**

```bash
cd mairu && git add internal/enricher/change_velocity.go internal/enricher/change_velocity_test.go
git commit -m "feat(enricher): add ChangeVelocityEnricher with git commit frequency analysis"
```

---

### Task 3: GitIntentEnricher

**Files:**
- Create: `mairu/internal/enricher/git_intent.go`
- Create: `mairu/internal/enricher/git_intent_test.go`

- [ ] **Step 1: Write the failing test**

```go
// mairu/internal/enricher/git_intent_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mairu && go test ./internal/enricher/ -v -run TestGitIntent`
Expected: Compilation error — `GitIntentEnricher` not defined.

- [ ] **Step 3: Implement GitIntentEnricher**

```go
// mairu/internal/enricher/git_intent.go
package enricher

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var conventionalPrefixRe = regexp.MustCompile(`^(\w+)(?:\(.+?\))?:\s+(.+)`)

// GitIntentEnricher annotates files with why they exist, extracted from git commit
// messages. It parses conventional commit prefixes and produces a human-readable
// intent summary. No LLM calls — heuristic only (LLM batch summarization is a
// future enhancement).
type GitIntentEnricher struct {
	MaxCommits int // how many recent commits to analyze; 0 defaults to 20
}

func (e *GitIntentEnricher) Name() string { return "git_intent" }

func (e *GitIntentEnricher) Enrich(ctx context.Context, fc *FileContext) error {
	maxCommits := e.MaxCommits
	if maxCommits <= 0 {
		maxCommits = 20
	}

	messages, err := gitCommitMessages(ctx, fc.WatchDir, fc.RelPath, maxCommits)
	if err != nil || len(messages) == 0 {
		return nil
	}

	prefixCounts := map[string]int{}
	var subjects []string

	for _, msg := range messages {
		match := conventionalPrefixRe.FindStringSubmatch(msg)
		if match != nil {
			prefix := strings.ToLower(match[1])
			prefixCounts[prefix]++
			subjects = append(subjects, match[2])
		} else {
			subjects = append(subjects, msg)
		}
	}

	intent := buildIntentSummary(subjects, prefixCounts, len(messages))

	fc.Metadata["enrichment_intent"] = intent
	if len(prefixCounts) > 0 {
		fc.Metadata["enrichment_commit_prefixes"] = prefixCounts
	}

	return nil
}

func buildIntentSummary(subjects []string, prefixes map[string]int, totalCommits int) string {
	var parts []string

	// Summarize commit type distribution
	if len(prefixes) > 0 {
		var typeParts []string
		for prefix, count := range prefixes {
			typeParts = append(typeParts, fmt.Sprintf("%d %s", count, prefix))
		}
		parts = append(parts, fmt.Sprintf("Commit history (%d commits): %s.", totalCommits, strings.Join(typeParts, ", ")))
	} else {
		parts = append(parts, fmt.Sprintf("Commit history: %d commits (no conventional prefixes).", totalCommits))
	}

	// Include the most recent commit subjects (up to 5) for context
	maxSubjects := 5
	if len(subjects) < maxSubjects {
		maxSubjects = len(subjects)
	}
	if maxSubjects > 0 {
		recent := subjects[:maxSubjects]
		parts = append(parts, "Recent changes: "+strings.Join(recent, "; ")+".")
	}

	return strings.Join(parts, " ")
}

// gitCommitMessages returns the subject line of the most recent N commits touching relPath.
func gitCommitMessages(ctx context.Context, repoDir, relPath string, maxCommits int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "log",
		fmt.Sprintf("-%d", maxCommits),
		"--format=%s",
		"--", relPath,
	)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var messages []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			messages = append(messages, line)
		}
	}
	return messages, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mairu && go test ./internal/enricher/ -v -run TestGitIntent`
Expected: All 3 tests PASS.

- [ ] **Step 5: Run all enricher tests**

Run: `cd mairu && go test ./internal/enricher/ -v`
Expected: All tests PASS (pipeline + change velocity + git intent).

- [ ] **Step 6: Commit**

```bash
cd mairu && git add internal/enricher/git_intent.go internal/enricher/git_intent_test.go
git commit -m "feat(enricher): add GitIntentEnricher with conventional commit parsing"
```

---

### Task 4: Config section for enrichers

**Files:**
- Modify: `mairu/internal/config/config.go`

- [ ] **Step 1: Read the current config file**

Run: `cd mairu && head -100 internal/config/config.go`
Confirm current state matches what we expect (Config struct, AllKeys, setDefaults).

- [ ] **Step 2: Add EnricherConfig to Config struct**

In `mairu/internal/config/config.go`, add the enricher config types and wire them in:

```go
// Add to Config struct (after Output field):
type Config struct {
	API       APIConfig       `mapstructure:"api"`
	Search    SearchConfig    `mapstructure:"search"`
	Daemon    DaemonConfig    `mapstructure:"daemon"`
	Server    ServerConfig    `mapstructure:"server"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Output    OutputConfig    `mapstructure:"output"`
	Enricher  EnricherConfig  `mapstructure:"enricher"`
}

// Add new types (after OutputConfig):
type EnricherConfig struct {
	GitIntent       GitIntentConfig       `mapstructure:"git_intent"`
	ChangeVelocity  ChangeVelocityConfig  `mapstructure:"change_velocity"`
}

type GitIntentConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	MaxCommits int  `mapstructure:"max_commits"`
}

type ChangeVelocityConfig struct {
	Enabled      bool `mapstructure:"enabled"`
	LookbackDays int  `mapstructure:"lookback_days"`
}
```

- [ ] **Step 3: Add defaults in setDefaults**

```go
// Add to setDefaults function:
v.SetDefault("enricher.git_intent.enabled", true)
v.SetDefault("enricher.git_intent.max_commits", 20)
v.SetDefault("enricher.change_velocity.enabled", true)
v.SetDefault("enricher.change_velocity.lookback_days", 180)
```

- [ ] **Step 4: Add keys to AllKeys**

```go
// Add to AllKeys() slice:
"enricher.git_intent.enabled", "enricher.git_intent.max_commits",
"enricher.change_velocity.enabled", "enricher.change_velocity.lookback_days",
```

- [ ] **Step 5: Run config tests**

Run: `cd mairu && go test ./internal/config/ -v`
Expected: PASS — defaults load correctly.

- [ ] **Step 6: Commit**

```bash
cd mairu && git add internal/config/config.go
git commit -m "feat(config): add enricher config section for git_intent and change_velocity"
```

---

### Task 5: Integrate enricher pipeline into daemon

**Files:**
- Modify: `mairu/internal/daemon/daemon.go`
- Modify: `mairu/internal/daemon/daemon_test.go`

- [ ] **Step 1: Write the failing integration test**

Add to `mairu/internal/daemon/daemon_test.go`:

```go
func TestProcessFileRunsEnricherPipeline(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "feature.ts")
	src := "export function greet(name: string) { return name; }"
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := &managerStub{}
	enricherCalled := false
	stub := &stubDaemonEnricher{
		enrichFn: func(ctx context.Context, fc *enricher.FileContext) error {
			enricherCalled = true
			fc.Metadata["enrichment_test"] = "hello"
			return nil
		},
	}
	pipeline := enricher.NewPipeline([]enricher.Enricher{stub})
	d := New(mgr, "proj", dir, Options{EnricherPipeline: pipeline})

	if err := d.ProcessFile(context.Background(), file); err != nil {
		t.Fatalf("process failed: %v", err)
	}
	if !enricherCalled {
		t.Fatal("enricher pipeline was not called")
	}
	if len(mgr.upserts) != 1 {
		t.Fatalf("expected one upsert, got %d", len(mgr.upserts))
	}
	if mgr.upserts[0].Metadata["enrichment_test"] != "hello" {
		t.Fatalf("enrichment data not in metadata: %v", mgr.upserts[0].Metadata)
	}
}
```

Also add the test stub type at the top of the test file:

```go
type stubDaemonEnricher struct {
	enrichFn func(ctx context.Context, fc *enricher.FileContext) error
}

func (s *stubDaemonEnricher) Name() string { return "stub" }
func (s *stubDaemonEnricher) Enrich(ctx context.Context, fc *enricher.FileContext) error {
	return s.enrichFn(ctx, fc)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mairu && go test ./internal/daemon/ -v -run TestProcessFileRunsEnricherPipeline`
Expected: Compilation error — `enricher` import, `EnricherPipeline` field, `stubDaemonEnricher` type not yet defined.

- [ ] **Step 3: Add enricher pipeline to daemon**

In `mairu/internal/daemon/daemon.go`:

1. Add import:
```go
import (
	// ... existing imports ...
	"mairu/internal/enricher"
)
```

2. Add field to `Options`:
```go
type Options struct {
	MaxFileSizeBytes     int64
	ProcessingDebounceMs int
	Concurrency          int
	MarkdownSummarizer   MarkdownSummarizer
	EnricherPipeline     *enricher.Pipeline
}
```

3. Add field to `Daemon`:
```go
type Daemon struct {
	// ... existing fields ...
	enricherPipeline *enricher.Pipeline
}
```

4. Wire in `New()`:
```go
return &Daemon{
	// ... existing fields ...
	enricherPipeline: opts.EnricherPipeline,
}
```

5. Call in `ProcessFile()`, after building `metadata` (line ~264) and before `payloadHash` computation (line ~265):

```go
metadata := map[string]any{
	"type":        "file",
	"path":        abs,
	"source_hash": contentHash,
	"logic_graph": summary.LogicGraph,
}

// Run enricher pipeline if configured
if d.enricherPipeline != nil {
	rel, _ := filepath.Rel(d.watchDir, abs)
	fc := &enricher.FileContext{
		FilePath: abs,
		RelPath:  filepath.ToSlash(rel),
		WatchDir: d.watchDir,
		Metadata: metadata,
	}
	d.enricherPipeline.Run(ctx, fc)
}

payloadHash := hashText(summary.Abstract + "\n" + summary.Overview + "\n" + summary.Content + "\n" + mustJSON(metadata))
```

- [ ] **Step 4: Run the integration test**

Run: `cd mairu && go test ./internal/daemon/ -v -run TestProcessFileRunsEnricherPipeline`
Expected: PASS.

- [ ] **Step 5: Run all daemon tests**

Run: `cd mairu && go test ./internal/daemon/ -v`
Expected: All existing tests still PASS.

- [ ] **Step 6: Commit**

```bash
cd mairu && git add internal/daemon/daemon.go internal/daemon/daemon_test.go
git commit -m "feat(daemon): integrate enricher pipeline into file processing"
```

---

### Task 6: Wire enricher pipeline in daemon CLI command

**Files:**
- Modify: `mairu/internal/cmd/daemon.go`

- [ ] **Step 1: Wire enrichers from config**

In `mairu/internal/cmd/daemon.go`, after the `opts` variable is built and before `daemon.New()`:

```go
// Add import
import (
	"mairu/internal/enricher"
)

// Inside RunE, after opts is built:
appCfg := GetConfig()
var enrichers []enricher.Enricher
if appCfg.Enricher.GitIntent.Enabled {
	enrichers = append(enrichers, &enricher.GitIntentEnricher{
		MaxCommits: appCfg.Enricher.GitIntent.MaxCommits,
	})
}
if appCfg.Enricher.ChangeVelocity.Enabled {
	enrichers = append(enrichers, &enricher.ChangeVelocityEnricher{
		LookbackDays: appCfg.Enricher.ChangeVelocity.LookbackDays,
	})
}
if len(enrichers) > 0 {
	opts.EnricherPipeline = enricher.NewPipeline(enrichers)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd mairu && go build ./cmd/mairu/`
Expected: Compiles without errors.

- [ ] **Step 3: Run all tests**

Run: `cd mairu && go test ./...`
Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
cd mairu && git add internal/cmd/daemon.go
git commit -m "feat(daemon): wire enricher pipeline from config in daemon command"
```

---

### Task 7: Promote enrichment metadata to top-level Meilisearch fields

**Files:**
- Modify: `mairu/internal/contextsrv/service_context.go`

- [ ] **Step 1: Read the current CreateContextNode method**

Run: Read `mairu/internal/contextsrv/service_context.go` lines 72-116 — the Meilisearch write path when `s.repo == nil`.

- [ ] **Step 2: Add enrichment field promotion**

In `service_context.go`, in the `CreateContextNode` method, after building the Meilisearch `payload` map (around line 98), before the `_vectors` assignment, add promotion of enrichment fields from the input metadata:

```go
// Promote enrichment fields from metadata to top-level Meili fields
// so they're searchable/filterable.
if len(input.Metadata) > 0 {
	var meta map[string]any
	if err := json.Unmarshal(input.Metadata, &meta); err == nil {
		if intent, ok := meta["enrichment_intent"].(string); ok && intent != "" {
			payload["intent"] = intent
		}
		if score, ok := meta["enrichment_churn_score"].(float64); ok {
			payload["churn_score"] = score
		}
		if label, ok := meta["enrichment_churn_label"].(string); ok && label != "" {
			payload["churn_label"] = label
		}
	}
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd mairu && go build ./...`
Expected: Compiles without errors.

- [ ] **Step 4: Run all tests**

Run: `cd mairu && go test ./...`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
cd mairu && git add internal/contextsrv/service_context.go
git commit -m "feat(contextsrv): promote enrichment metadata to top-level Meilisearch fields"
```

---

### Task 8: Add enrichment weight dimensions to scorer

**Files:**
- Modify: `mairu/internal/contextsrv/search_rerank.go`

- [ ] **Step 1: Write the failing test**

Add to the existing `mairu/internal/contextsrv/search_rerank_test.go`:

```go
func TestScoreWithMeiliRanking_ChurnBoost(t *testing.T) {
	now := time.Now()
	opts := SearchOptions{RecencyScale: "30d", RecencyDecay: 0.5}
	defaults := defaultContextWeights(nil)

	// Two identical docs, one with churn data
	baseScore := scoreWithMeiliRanking(0.8, now, 0, opts, defaults, nil)
	churnData := map[string]any{"enrichment_churn_score": 0.8}
	churnScore := scoreWithMeiliRanking(0.8, now, 0, opts, defaults, churnData)

	if churnScore <= baseScore {
		t.Fatalf("churn boost should increase score: base=%f churn=%f", baseScore, churnScore)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd mairu && go test ./internal/contextsrv/ -v -run TestScoreWithMeiliRanking_ChurnBoost`
Expected: Compilation error — `scoreWithMeiliRanking` signature doesn't accept enrichment data.

**Note:** When you change the `scoreWithMeiliRanking` signature, existing tests in `search_rerank_test.go` (lines 55 and 65) will break. Update them to pass `nil` as the last argument:

```go
// Line 55:
score := scoreWithMeiliRanking(1.0, time.Now(), 10, opts, defaultMemoryWeights(nil), nil)
// Line 65:
score := scoreWithMeiliRanking(0.8, time.Time{}, 0, opts, defaultSkillWeights(nil), nil)
```

- [ ] **Step 3: Add churn boost to scorer**

In `mairu/internal/contextsrv/search_rerank.go`:

1. Add `churn` field to `hybridWeights`:
```go
type hybridWeights struct {
	vector     float64
	keyword    float64
	recency    float64
	importance float64
	churn      float64
}
```

2. Update `defaultContextWeights` to include a small churn weight:
```go
func defaultContextWeights(overrides *WeightOverrides) hybridWeights {
	w := hybridWeights{vector: 0.60, keyword: 0.30, recency: 0.05, importance: 0, churn: 0.05}
	return applyOverrides(w, overrides)
}
```

Adjust existing context weights slightly so they still sum properly (vector from 0.65 to 0.60).

3. Add `Churn` to `WeightOverrides`:
```go
type WeightOverrides struct {
	Vector     float64
	Keyword    float64
	Recency    float64
	Importance float64
	Churn      float64
}
```

4. Update `applyOverrides`:
```go
if o.Churn > 0 {
	w.churn = o.Churn
}
```

5. Update `effectiveWeights` normalization:
```go
total := w.vector + w.keyword + w.recency + w.importance + w.churn
// ...
return hybridWeights{
	vector:     w.vector / total,
	keyword:    w.keyword / total,
	recency:    w.recency / total,
	importance: w.importance / total,
	churn:      w.churn / total,
}
```

6. Update `scoreWithMeiliRanking` to accept and use enrichment data:
```go
func scoreWithMeiliRanking(rankingScore float64, createdAt time.Time, importance int, opts SearchOptions, defaults hybridWeights, enrichmentData map[string]any) float64 {
	weights := effectiveWeights(opts, defaults)

	vectorKeywordFraction := weights.vector + weights.keyword
	score := rankingScore * vectorKeywordFraction

	recencyScore := scoreRecency(createdAt, opts.RecencyScale, opts.RecencyDecay)
	importanceScore := 0.0
	if importance > 0 {
		importanceScore = float64(importance) / 10.0
	}

	churnScore := 0.0
	if enrichmentData != nil {
		if cs, ok := enrichmentData["enrichment_churn_score"].(float64); ok {
			churnScore = cs
		}
	}

	score += recencyScore * weights.recency
	score += importanceScore * weights.importance
	score += churnScore * weights.churn

	return score
}
```

7. Update the single caller of `scoreWithMeiliRanking` in `meili.go` (line 215) to pass enrichment data from the doc. Replace:

```go
score := scoreWithMeiliRanking(rankingScore, createdAt, importance, opts, defaults)
```

with:

```go
// Extract enrichment data for scoring (churn_score is promoted to top-level by the service layer)
var enrichmentData map[string]any
if cs, ok := doc["churn_score"].(float64); ok {
	enrichmentData = map[string]any{"enrichment_churn_score": cs}
}
score := scoreWithMeiliRanking(rankingScore, createdAt, importance, opts, defaults, enrichmentData)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mairu && go test ./internal/contextsrv/ -v`
Expected: All tests PASS (existing + new churn test).

- [ ] **Step 5: Commit**

```bash
cd mairu && git add internal/contextsrv/search_rerank.go internal/contextsrv/meili.go
git commit -m "feat(scorer): add churn boost dimension to hybrid search ranking"
```

---

### Task 9: Add enricher config to search weights

**Files:**
- Modify: `mairu/internal/config/config.go`

- [ ] **Step 1: Add churn weight to WeightsConfig**

```go
type WeightsConfig struct {
	Vector     float64 `mapstructure:"vector"`
	Keyword    float64 `mapstructure:"keyword"`
	Recency    float64 `mapstructure:"recency"`
	Importance float64 `mapstructure:"importance"`
	Churn      float64 `mapstructure:"churn"`
}
```

- [ ] **Step 2: Add defaults**

```go
v.SetDefault("search.context.churn", 0.05)
v.SetDefault("search.memories.churn", 0.0)
v.SetDefault("search.skills.churn", 0.0)
```

- [ ] **Step 3: Add to AllKeys**

```go
"search.memories.churn", "search.skills.churn", "search.context.churn",
```

- [ ] **Step 4: Wire WeightsConfig.Churn into WeightOverrides**

Find where `WeightOverrides` is built from config (likely in the context-server or search initialization code) and add:
```go
Churn: cfg.Search.Context.Churn,
```

- [ ] **Step 5: Run tests**

Run: `cd mairu && go test ./...`
Expected: All PASS.

- [ ] **Step 6: Commit**

```bash
cd mairu && git add internal/config/config.go
git commit -m "feat(config): add churn weight to search config"
```

---

### Task 10: End-to-end manual verification

- [ ] **Step 1: Build the binary**

Run: `cd mairu && make build`
Expected: Binary compiled to `bin/mairu`.

- [ ] **Step 2: Verify config defaults**

Run: `cd mairu && bin/mairu config list 2>/dev/null | grep enricher`
Expected: Shows `enricher.git_intent.enabled = true`, `enricher.change_velocity.enabled = true`, etc.

- [ ] **Step 3: Run all tests one final time**

Run: `cd mairu && make test`
Expected: All tests PASS.

- [ ] **Step 4: Commit any remaining changes**

```bash
cd mairu && git status
# If any uncommitted changes, stage and commit
```
