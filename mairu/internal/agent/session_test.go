package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSessionName(t *testing.T) {
	t.Run("accepts valid names", func(t *testing.T) {
		valid := []string{"default", "feature-123", "session_abc", "a.b.c"}
		for _, name := range valid {
			if err := ValidateSessionName(name); err != nil {
				t.Fatalf("expected %q to be valid, got error: %v", name, err)
			}
		}
	})

	t.Run("rejects invalid names", func(t *testing.T) {
		invalid := []string{"", "  ", "../escape", "name with spaces", "name/slash", "name\\slash"}
		for _, name := range invalid {
			if err := ValidateSessionName(name); err == nil {
				t.Fatalf("expected %q to be invalid", name)
			}
		}
	})
}

func TestListSessions_MigratesLegacyFilePath(t *testing.T) {
	projectRoot := t.TempDir()
	legacyDir := filepath.Join(projectRoot, ".mairu")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}

	legacyPath := filepath.Join(legacyDir, "sessions")
	if err := os.WriteFile(legacyPath, []byte("legacy-non-json"), 0644); err != nil {
		t.Fatalf("failed to create legacy sessions file: %v", err)
	}

	sessions, err := ListSessions(projectRoot)
	if err != nil {
		t.Fatalf("expected no error listing sessions, got: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected no sessions after migration, got: %v", sessions)
	}

	info, err := os.Stat(legacyPath)
	if err != nil {
		t.Fatalf("expected migrated sessions path to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected migrated sessions path to be a directory")
	}
}

func TestLoadSavedSessionMessages_MigratesLegacyJSONAsDefault(t *testing.T) {
	projectRoot := t.TempDir()
	legacyDir := filepath.Join(projectRoot, ".mairu")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}

	legacyPayload := []byte(`[{"role":"user","content":"hello"}]`)
	legacyPath := filepath.Join(legacyDir, "sessions")
	if err := os.WriteFile(legacyPath, legacyPayload, 0644); err != nil {
		t.Fatalf("failed to create legacy sessions file: %v", err)
	}

	messages, err := LoadSavedSessionMessages(projectRoot, "default")
	if err != nil {
		t.Fatalf("expected no error loading migrated default session, got: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 migrated message, got %d", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "hello" {
		t.Fatalf("unexpected migrated message content: %#v", messages[0])
	}
}
