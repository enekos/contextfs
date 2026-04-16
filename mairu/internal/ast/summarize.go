package ast

import (
	"fmt"
	"strings"
)

// FileSummary holds the generated natural-language representations of a source file.
type FileSummary struct {
	Abstract   string
	Overview   string
	Content    string
	LogicGraph map[string]any
}

// SummarizeFile turns an AST FileGraph into human-readable context-node fields.
// relPath is the file path to display (e.g., relative to the project root).
// languageID is the describer's language identifier (e.g., "typescript", "go").
// abstract is a pre-computed short summary (usually fg.FileSummary or a file-level JSDoc).
// maxChars limits the length of Overview and Content.
func SummarizeFile(relPath, languageID, abstract string, fg FileGraph, maxChars int) FileSummary {
	lines := []string{
		"File: " + relPath,
		"Language: " + languageID,
		"LogicGraph: v1",
		fmt.Sprintf("GraphStats: symbols=%d edges=%d", len(fg.Symbols), len(fg.Edges)),
		"",
		"Symbols:",
	}
	if len(fg.Symbols) == 0 {
		lines = append(lines, "- (none)")
	} else {
		for _, s := range fg.Symbols {
			doc := ""
			if s.Doc != "" {
				doc = ` doc="` + s.Doc + `"`
			}
			lines = append(lines, fmt.Sprintf("- %s %s%s", s.Kind, s.ID, doc))
		}
	}
	lines = append(lines, "", "Edges:")
	if len(fg.Edges) == 0 {
		lines = append(lines, "- (none)")
	} else {
		for _, e := range fg.Edges {
			lines = append(lines, fmt.Sprintf("- %s %s -> %s", e.Kind, e.From, e.To))
		}
	}

	overview := strings.Join(lines, "\n")
	if len(overview) > maxChars {
		overview = overview[:maxChars] + "\n...TRUNCATED_BY_MAX_CONTENT_CHARS"
	}

	content := fg.RawContent
	if content == "" {
		content = DescribeSymbols(fg.Symbols, fg.Edges, fg.SymbolDescriptions)
	}
	if len(content) > maxChars {
		content = content[:maxChars] + "\n\n...TRUNCATED"
	}

	logicGraph := map[string]any{
		"version": 1,
		"symbols": fg.Symbols,
		"edges":   fg.Edges,
		"imports": fg.Imports,
	}

	return FileSummary{
		Abstract:   abstract,
		Overview:   overview,
		Content:    content,
		LogicGraph: logicGraph,
	}
}
