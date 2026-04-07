package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMultiEdit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mairu_edit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	agent := &Agent{
		root: tempDir,
	}

	filePath := "test_edit.txt"
	fullPath := filepath.Join(tempDir, filePath)

	initialContent := "line 1\nline 2\nline 3\nline 4\nline 5"
	err = os.WriteFile(fullPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Single Block Edit", func(t *testing.T) {
		edits := []EditBlock{
			{
				StartLine: 2,
				EndLine:   3, // Replaces line 2 and line 3
				Content:   "new line 2\nnew line 3",
			},
		}

		_, err := agent.MultiEdit(filePath, edits)
		if err != nil {
			t.Fatalf("MultiEdit failed: %v", err)
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatal(err)
		}

		expected := "line 1\nnew line 2\nnew line 3\nline 4\nline 5"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}
	})

	t.Run("Multiple Block Edit", func(t *testing.T) {
		// Reset content
		os.WriteFile(fullPath, []byte(initialContent), 0644)

		edits := []EditBlock{
			{
				StartLine: 2,
				EndLine:   2,
				Content:   "line 2 replaced",
			},
			{
				StartLine: 4,
				EndLine:   5, // line 4 and 5
				Content:   "line 4+5 replaced",
			},
		}

		_, err := agent.MultiEdit(filePath, edits)
		if err != nil {
			t.Fatalf("MultiEdit failed: %v", err)
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatal(err)
		}

		expected := "line 1\nline 2 replaced\nline 3\nline 4+5 replaced"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}
	})

	t.Run("Invalid Edit Bounds", func(t *testing.T) {
		edits := []EditBlock{
			{
				StartLine: 10,
				EndLine:   11,
				Content:   "out of bounds",
			},
		}
		_, err := agent.MultiEdit(filePath, edits)
		if err == nil {
			t.Fatal("expected error for out of bounds edit, got nil")
		}
	})
}
