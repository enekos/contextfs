package agent

import (
	"mairu/internal/db"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurgicalRead(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mairu_surgical_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	agent := &Agent{
		db: db.NewTestDB(tempDir),
	}

	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\n"
	filePath := "code.go"
	fullPath := filepath.Join(tempDir, filePath)

	err = os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	loc := db.SymbolLocation{
		FilePath: filePath,
		Name:     "TestFunc",
		Kind:     "function",
		StartRow: 2, // 0-indexed, so "line 3"
		EndRow:   4, // "line 5"
	}

	res, err := agent.SurgicalRead(loc)
	if err != nil {
		t.Fatalf("SurgicalRead error: %v", err)
	}

	expectedContent := "3: line 3\n4: line 4\n5: line 5\n"
	if !strings.Contains(res, expectedContent) {
		t.Errorf("expected result to contain:\n%s\nGot:\n%s", expectedContent, res)
	}

	expectedHeader := "--- Surgical Read: TestFunc (function) from code.go ---"
	if !strings.Contains(res, expectedHeader) {
		t.Errorf("expected result to contain header:\n%s\nGot:\n%s", expectedHeader, res)
	}
}
