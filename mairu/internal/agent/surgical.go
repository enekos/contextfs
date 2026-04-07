package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"mairu/internal/contextsrv"
)

// SurgicalRead extracts specific lines from a file, saving massive amounts of tokens.
func (a *Agent) SurgicalRead(loc contextsrv.SymbolLocation) (string, error) {
	fullPath := filepath.Join(a.root, loc.FilePath)

	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", loc.FilePath, err)
	}
	defer file.Close()

	var result string
	scanner := bufio.NewScanner(file)
	var currentLine uint32 = 0

	// Note: Tree-sitter rows are 0-indexed
	for scanner.Scan() {
		if currentLine >= loc.StartRow && currentLine <= loc.EndRow {
			result += fmt.Sprintf("%d: %s\n", currentLine+1, scanner.Text())
		}
		if currentLine > loc.EndRow {
			break
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	header := fmt.Sprintf("--- Surgical Read: %s (%s) from %s ---\n", loc.Name, loc.Kind, loc.FilePath)
	return header + result, nil
}
