package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProcessAllFilesConcurrently(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 20; i++ {
		file := filepath.Join(dir, fmt.Sprintf("module%d.ts", i))
		_ = os.WriteFile(file, []byte(fmt.Sprintf("export function fn%d(){ return %d; }", i, i)), 0o644)
	}
	mgr := &managerStub{}
	d := New(mgr, "proj", dir, Options{Concurrency: 4})
	if err := d.ProcessAllFiles(context.Background()); err != nil {
		t.Fatalf("process all failed: %v", err)
	}
	if len(mgr.upserts) < 18 {
		t.Fatalf("expected near-complete upserts, got %d", len(mgr.upserts))
	}
}

func TestProcessPendingBatchConcurrently(t *testing.T) {
	dir := t.TempDir()
	files := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		p := filepath.Join(dir, fmt.Sprintf("change%d.ts", i))
		_ = os.WriteFile(p, []byte(fmt.Sprintf("export const v%d=%d", i, i)), 0o644)
		files = append(files, p)
	}
	mgr := &managerStub{}
	d := New(mgr, "proj", dir, Options{Concurrency: 4})
	for _, f := range files {
		d.QueueFile(f)
	}
	if err := d.ProcessPendingFiles(context.Background()); err != nil {
		t.Fatalf("pending failed: %v", err)
	}
	if len(mgr.upserts) != 10 {
		t.Fatalf("expected 10 upserts, got %d", len(mgr.upserts))
	}
}
