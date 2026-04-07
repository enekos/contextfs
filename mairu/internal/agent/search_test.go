package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFallbackSearch(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello world\nthis is a test\nfindme123"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("another file\nwith some other text\nregex: foo_bar_baz"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.TXT"), []byte("uppercase TEXT"), 0644)

	// Create ignore file
	os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("ignored_dir/\n"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "ignored_dir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "ignored_dir", "file4.txt"), []byte("findme123 inside ignored"), 0644)

	// Binary file
	binData := make([]byte, 512)
	binData[0] = 0 // null byte
	copy(binData[1:], []byte("findme123 in binary"))
	os.WriteFile(filepath.Join(tmpDir, "binary.bin"), binData, 0644)

	a := &Agent{
		root: tmpDir,
	}

	t.Run("literal search", func(t *testing.T) {
		res, err := a.fallbackSearch("findme123")
		assert.NoError(t, err)
		assert.Contains(t, res, "file1.txt:3:findme123")
		assert.NotContains(t, res, "file4.txt")  // should be ignored
		assert.NotContains(t, res, "binary.bin") // should be skipped
	})

	t.Run("regex search", func(t *testing.T) {
		res, err := a.fallbackSearch("foo_.*_baz")
		assert.NoError(t, err)
		assert.Contains(t, res, "file2.txt:3:regex: foo_bar_baz")
	})

	t.Run("case insensitive", func(t *testing.T) {
		res, err := a.fallbackSearch("uppercase text")
		assert.NoError(t, err)
		assert.Contains(t, res, "file3.TXT:1:uppercase TEXT")
	})

	t.Run("no match", func(t *testing.T) {
		_, err := a.fallbackSearch("nonexistent_string_12345")
		assert.Error(t, err)
	})
}
