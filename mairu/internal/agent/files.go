package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ReadFile reads the full content of a file, adding line numbers.
func (a *Agent) ReadFile(filePath string) (string, error) {
	fullPath := filepath.Join(a.db.Root(), filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	lines := strings.Split(string(content), "\n")
	var result string
	for i, line := range lines {
		result += fmt.Sprintf("%d: %s\n", i+1, line)
	}

	return result, nil
}

// WriteFile overwrites a file completely.
func (a *Agent) WriteFile(filePath string, content string) (string, error) {
	fullPath := filepath.Join(a.db.Root(), filePath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}

	var oldContent []byte
	if _, err := os.Stat(fullPath); err == nil {
		oldContent, _ = os.ReadFile(fullPath)
	}

	err := os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		return "", err
	}

	// Compute diff
	tmpFile, err1 := os.CreateTemp("", "mairu-diff-*")
	tmpFile2, err2 := os.CreateTemp("", "mairu-diff-new-*")
	if err1 == nil && err2 == nil {
		tmpFile.Write(oldContent)
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		tmpFile2.Write([]byte(content))
		tmpFile2.Close()
		defer os.Remove(tmpFile2.Name())

		cmd := exec.Command("diff", "-u", tmpFile.Name(), tmpFile2.Name())
		out, _ := cmd.CombinedOutput()
		diffStr := string(out)
		diffStr = strings.Replace(diffStr, tmpFile.Name(), filePath+" (old)", 1)
		diffStr = strings.Replace(diffStr, tmpFile2.Name(), filePath+" (new)", 1)
		return diffStr, nil
	}

	return "", nil
}

// FindFiles uses glob pattern to find files.
func (a *Agent) FindFiles(pattern string) (string, error) {
	fs := os.DirFS(a.db.Root())
	matches, err := doublestar.Glob(fs, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to search pattern %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		return "No files found matching pattern.", nil
	}

	return strings.Join(matches, "\n"), nil
}

// SearchCodebase runs ripgrep (if available) or standard grep.
func (a *Agent) SearchCodebase(query string) (string, error) {
	// Let's try ripgrep first as it's much faster
	cmd := exec.Command("rg", "-n", query)
	cmd.Dir = a.db.Root()

	out, err := cmd.CombinedOutput()
	if err != nil {
		// If ripgrep fails, maybe it's not installed. Fallback to grep.
		cmd = exec.Command("grep", "-rn", query, ".")
		cmd.Dir = a.db.Root()
		out, err = cmd.CombinedOutput()
		if err != nil && len(out) == 0 {
			return "", fmt.Errorf("search failed or no results found")
		}
	}

	res := string(out)
	if len(res) > 5000 {
		res = res[:5000] + "\n...[Output truncated]"
	}

	if res == "" {
		return "No matches found.", nil
	}

	return res, nil
}
