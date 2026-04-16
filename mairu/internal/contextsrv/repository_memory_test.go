package contextsrv

import (
	"context"
	"testing"
)

func newTestMemoryRepo(t *testing.T) *SQLiteRepository {
	dbPath := t.TempDir() + "/test.db"
	repo, err := NewSQLiteRepository("file:" + dbPath + "?cache=shared&mode=rwc")
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestMemoryCRUD(t *testing.T) {
	repo := newTestMemoryRepo(t)
	ctx := context.Background()

	// Create
	created, err := repo.CreateMemory(ctx, MemoryCreateInput{
		Project:    "proj1",
		Content:    "hello world",
		Category:   "note",
		Owner:      "user",
		Importance: 5,
	})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	if created.Content != "hello world" {
		t.Errorf("unexpected content: %s", created.Content)
	}

	// List
	mems, err := repo.ListMemories(ctx, "proj1", 10)
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	if len(mems) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(mems))
	}

	// Get
	got, err := repo.GetMemory(ctx, created.ID)
	if err != nil {
		t.Fatalf("get memory: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, got.ID)
	}

	// Update
	updated, err := repo.UpdateMemory(ctx, MemoryUpdateInput{
		ID:         created.ID,
		Content:    "updated content",
		Importance: 8,
	})
	if err != nil {
		t.Fatalf("update memory: %v", err)
	}
	if updated.Content != "updated content" {
		t.Errorf("unexpected updated content: %s", updated.Content)
	}

	// Record retrieval
	if err := repo.RecordRetrievals(ctx, []string{created.ID}); err != nil {
		t.Fatalf("record retrieval: %v", err)
	}
	got, _ = repo.GetMemory(ctx, created.ID)
	if got.RetrievalCount != 1 {
		t.Errorf("expected retrieval count 1, got %d", got.RetrievalCount)
	}

	// Increment feedback
	if err := repo.IncrementFeedbackCount(ctx, created.ID); err != nil {
		t.Fatalf("increment feedback: %v", err)
	}
	got, _ = repo.GetMemory(ctx, created.ID)
	if got.FeedbackCount != 1 {
		t.Errorf("expected feedback count 1, got %d", got.FeedbackCount)
	}

	// Delete
	if err := repo.DeleteMemory(ctx, created.ID); err != nil {
		t.Fatalf("delete memory: %v", err)
	}
	_, err = repo.GetMemory(ctx, created.ID)
	if err == nil {
		t.Error("expected error after deleting memory")
	}
}

func TestMemory_ListEmpty(t *testing.T) {
	repo := newTestMemoryRepo(t)
	ctx := context.Background()

	mems, err := repo.ListMemories(ctx, "nonexistent", 10)
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	if len(mems) != 0 {
		t.Errorf("expected 0 memories, got %d", len(mems))
	}
}
