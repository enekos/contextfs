package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mairu/internal/contextsrv"
)

type mockManager struct {
	upserts []upsertCall
	nodes   map[string]contextsrv.ContextNode
}

type upsertCall struct {
	URI       string
	Name      string
	Abstract  string
	Overview  string
	Content   string
	ParentURI string
	Project   string
	Metadata  map[string]any
	CreatedAt *time.Time
}

func (m *mockManager) UpsertFileContextNode(ctx context.Context, uri, name, abstractText, overviewText, content, parentURI, project string, metadata map[string]any, createdAt *time.Time) error {
	m.upserts = append(m.upserts, upsertCall{
		URI: uri, Name: name, Abstract: abstractText, Overview: overviewText, Content: content,
		ParentURI: parentURI, Project: project, Metadata: metadata, CreatedAt: createdAt,
	})
	if m.nodes == nil {
		m.nodes = make(map[string]contextsrv.ContextNode)
	}
	m.nodes[uri] = contextsrv.ContextNode{
		URI:       uri,
		Name:      name,
		Abstract:  abstractText,
		Overview:  overviewText,
		Content:   content,
		Project:   project,
		CreatedAt: time.Now(),
	}
	return nil
}

func (m *mockManager) GetContextNode(ctx context.Context, uri string) (contextsrv.ContextNode, error) {
	if n, ok := m.nodes[uri]; ok {
		return n, nil
	}
	return contextsrv.ContextNode{}, fmt.Errorf("not found")
}

func TestIngest_BasicCommitAndSnapshot(t *testing.T) {
	repoDir := setupTestRepo(t)
	writeFile(t, repoDir, "main.go", "package main\n\nfunc main() {}\n")
	gitCommit(t, repoDir, "initial commit")

	mgr := &mockManager{}
	ing := NewIngester(mgr)
	opts := IngestOptions{
		Project:           "testproj",
		RepoDir:           repoDir,
		Since:             time.Now().AddDate(0, 0, -1),
		MaxFilesPerCommit: 50,
		MaxContentChars:   16000,
	}

	if err := ing.Ingest(context.Background(), opts); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Should have commit, snapshot, and diff nodes
	var commitCount, snapshotCount, diffCount int
	for _, u := range mgr.upserts {
		if strings.Contains(u.URI, "/git/commit/") {
			commitCount++
		}
		if strings.Contains(u.URI, "/git/snapshot/") {
			snapshotCount++
		}
		if strings.Contains(u.URI, "/git/diff/") {
			diffCount++
		}
	}

	if commitCount != 1 {
		t.Errorf("expected 1 commit node, got %d", commitCount)
	}
	if snapshotCount != 1 {
		t.Errorf("expected 1 snapshot node, got %d", snapshotCount)
	}
	if diffCount != 1 {
		t.Errorf("expected 1 diff node, got %d", diffCount)
	}
}

func TestIngest_SkipsUnsupportedExtensions(t *testing.T) {
	repoDir := setupTestRepo(t)
	writeFile(t, repoDir, "readme.bin", "binary data")
	gitCommit(t, repoDir, "add binary")

	mgr := &mockManager{}
	ing := NewIngester(mgr)
	opts := IngestOptions{
		Project:           "testproj",
		RepoDir:           repoDir,
		Since:             time.Now().AddDate(0, 0, -1),
		MaxFilesPerCommit: 50,
		MaxContentChars:   16000,
	}

	if err := ing.Ingest(context.Background(), opts); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Commit node should exist, but no snapshot/diff for .bin
	var commitCount, snapshotCount int
	for _, u := range mgr.upserts {
		if strings.Contains(u.URI, "/git/commit/") {
			commitCount++
		}
		if strings.Contains(u.URI, "/git/snapshot/") {
			snapshotCount++
		}
	}

	if commitCount != 1 {
		t.Errorf("expected 1 commit node, got %d", commitCount)
	}
	if snapshotCount != 0 {
		t.Errorf("expected 0 snapshot nodes for binary, got %d", snapshotCount)
	}
}

func TestIngest_IdempotentSnapshotReuse(t *testing.T) {
	repoDir := setupTestRepo(t)
	writeFile(t, repoDir, "main.go", "package main\n\nfunc main() {}\n")
	gitCommit(t, repoDir, "initial commit")

	mgr := &mockManager{}
	ing := NewIngester(mgr)
	opts := IngestOptions{
		Project:           "testproj",
		RepoDir:           repoDir,
		Since:             time.Now().AddDate(0, 0, -1),
		MaxFilesPerCommit: 50,
		MaxContentChars:   16000,
	}

	if err := ing.Ingest(context.Background(), opts); err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}
	firstSnapshotUpserts := countSnapshotUpserts(mgr.upserts)

	if err := ing.Ingest(context.Background(), opts); err != nil {
		t.Fatalf("second ingest failed: %v", err)
	}
	secondSnapshotUpserts := countSnapshotUpserts(mgr.upserts)

	if secondSnapshotUpserts != firstSnapshotUpserts {
		t.Errorf("expected no additional snapshot upserts on re-ingest, got %d vs %d", firstSnapshotUpserts, secondSnapshotUpserts)
	}
}

func TestIngest_CreatedAtMatchesCommitDate(t *testing.T) {
	repoDir := setupTestRepo(t)
	writeFile(t, repoDir, "main.go", "package main\n\nfunc main() {}\n")
	gitCommit(t, repoDir, "initial commit")

	// Get the actual commit date
	out, err := exec.Command("git", "-C", repoDir, "log", "-1", "--format=%aI").Output()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}
	commitDate, err := time.Parse(time.RFC3339, strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse commit date: %v", err)
	}

	mgr := &mockManager{}
	ing := NewIngester(mgr)
	opts := IngestOptions{
		Project:           "testproj",
		RepoDir:           repoDir,
		Since:             time.Now().AddDate(0, 0, -1),
		MaxFilesPerCommit: 50,
		MaxContentChars:   16000,
	}

	if err := ing.Ingest(context.Background(), opts); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	for _, u := range mgr.upserts {
		if u.CreatedAt == nil {
			t.Errorf("expected created_at for %s", u.URI)
			continue
		}
		if u.CreatedAt.IsZero() {
			t.Errorf("created_at is zero for %s", u.URI)
			continue
		}
		// Allow a small tolerance because git commit date and system time might differ slightly
		diff := u.CreatedAt.Sub(commitDate)
		if diff < 0 {
			diff = -diff
		}
		if diff > time.Minute {
			t.Errorf("created_at mismatch for %s: got %v, want near %v", u.URI, u.CreatedAt, commitDate)
		}
	}
}

func TestIngest_DryRunDoesNotPersist(t *testing.T) {
	repoDir := setupTestRepo(t)
	writeFile(t, repoDir, "main.go", "package main\n\nfunc main() {}\n")
	gitCommit(t, repoDir, "initial commit")

	mgr := &mockManager{}
	ing := NewIngester(mgr)
	opts := IngestOptions{
		Project:           "testproj",
		RepoDir:           repoDir,
		Since:             time.Now().AddDate(0, 0, -1),
		DryRun:            true,
		MaxFilesPerCommit: 50,
		MaxContentChars:   16000,
	}

	if err := ing.Ingest(context.Background(), opts); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	if len(mgr.upserts) != 0 {
		t.Errorf("expected 0 upserts in dry-run, got %d", len(mgr.upserts))
	}
}

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", c, err, out)
		}
	}
	return dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func gitCommit(t *testing.T, dir, message string) {
	t.Helper()
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}
}

func countSnapshotUpserts(upserts []upsertCall) int {
	c := 0
	for _, u := range upserts {
		if strings.Contains(u.URI, "/git/snapshot/") {
			c++
		}
	}
	return c
}
