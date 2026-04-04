package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSavesCacheAfterProcessing(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "mod.ts")
	_ = os.WriteFile(file, []byte("export function hello(){return 'hi';}"), 0o644)
	mgr := &managerStub{}
	d := New(mgr, "proj", dir, Options{})
	_ = d.ProcessAllFiles(context.Background())
	_ = d.SaveCache()
	cachePath := filepath.Join(dir, CacheFilename)
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("expected cache file: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if int(parsed["version"].(float64)) != 1 {
		t.Fatalf("unexpected cache version: %#v", parsed["version"])
	}
}

func TestSkipsUnchangedFilesAfterCacheLoad(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "mod.ts")
	_ = os.WriteFile(file, []byte("export function hello(){return 'hi';}"), 0o644)
	mgr1 := &managerStub{}
	d1 := New(mgr1, "proj", dir, Options{})
	if err := d1.ProcessAllFiles(context.Background()); err != nil {
		t.Fatalf("initial process failed: %v", err)
	}
	_ = d1.SaveCache()
	if len(mgr1.upserts) != 1 {
		t.Fatalf("expected one initial upsert, got %d", len(mgr1.upserts))
	}

	mgr2 := &managerStub{}
	d2 := New(mgr2, "proj", dir, Options{})
	d2.LoadCache()
	if err := d2.ProcessAllFiles(context.Background()); err != nil {
		t.Fatalf("second process failed: %v", err)
	}
	if len(mgr2.upserts) != 0 {
		t.Fatalf("expected no upsert after cache load, got %d", len(mgr2.upserts))
	}
}

func TestReprocessesOnContentChangeBetweenRuns(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "mod.ts")
	_ = os.WriteFile(file, []byte("export function hello(){return 'hi';}"), 0o644)
	mgr1 := &managerStub{}
	d1 := New(mgr1, "proj", dir, Options{})
	if err := d1.ProcessAllFiles(context.Background()); err != nil {
		t.Fatalf("initial process failed: %v", err)
	}
	_ = d1.SaveCache()
	_ = os.WriteFile(file, []byte("export function hello(){return 'changed';}"), 0o644)

	mgr2 := &managerStub{}
	d2 := New(mgr2, "proj", dir, Options{})
	d2.LoadCache()
	if err := d2.ProcessFile(context.Background(), file); err != nil {
		t.Fatalf("second process failed: %v", err)
	}
	if len(mgr2.upserts) != 1 {
		t.Fatalf("expected one changed upsert, got %d", len(mgr2.upserts))
	}
}
