package daemon

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"mairu/internal/ast"
)

type sourceSummary struct {
	Abstract   string
	Overview   string
	Content    string
	LogicGraph map[string]any
}

func (d *Daemon) summarizeSourceFile(ctx context.Context, filePath, src string) (sourceSummary, error) {
	var fileDoc string
	var abstract string
	var overview string
	var content string
	var logicGraph map[string]any

	describer := d.getDescriber(filePath)
	if describer != nil {
		fg := describer.ExtractFileGraph(filePath, src)
		abstract = fg.FileSummary
		if fileDoc = extractFileJSDoc(src); fileDoc != "" {
			abstract = fileDoc
		}

		rel, err := mustRel(d.watchDir, filePath)
		if err != nil {
			return sourceSummary{}, err
		}

		summary := ast.SummarizeFile(filepath.ToSlash(rel), describer.LanguageID(), abstract, fg, maxContentChars)
		abstract = summary.Abstract
		overview = summary.Overview
		content = summary.Content
		logicGraph = summary.LogicGraph

		// LLM enrichment pass for markdown files: replaces heuristic abstract and
		// overview with semantically richer descriptions optimized for retrieval.
		if fg.RawContent != "" && d.mdSummarizer != nil {
			if ab, ov, err := d.mdSummarizer.SummarizeMarkdown(ctx, filepath.Base(filePath), src); err == nil {
				abstract = ab
				if ov != "" {
					overview = ov
				}
			} else {
				fmt.Printf("[Daemon] markdown LLM enrichment failed for %s: %v\n", filepath.Base(filePath), err)
			}
		}
	} else {
		// Fallback for languages without describer
		fileDoc = extractFileJSDoc(src)
		symbols := extractSymbols(src)
		edges := extractEdges(src, symbols)
		abstract = fileDoc
		if abstract == "" {
			if len(symbols) == 0 {
				abstract = "This file is empty or contains no declarations."
			} else {
				names := make([]string, 0, len(symbols))
				for _, s := range symbols {
					names = append(names, s.Name)
				}
				sort.Strings(names)
				abstract = "This file defines: " + strings.Join(names, ", ")
			}
		}
		rel, err := mustRel(d.watchDir, filePath)
		if err != nil {
			return sourceSummary{}, err
		}
		lines := []string{
			"File: " + filepath.ToSlash(rel),
			"Language: " + strings.TrimPrefix(strings.ToLower(filepath.Ext(filePath)), "."),
			"LogicGraph: v1",
			fmt.Sprintf("GraphStats: symbols=%d shown=%d edges=%d shown=%d", len(symbols), len(symbols), len(edges), len(edges)),
			"",
			"Symbols:",
		}
		if len(symbols) == 0 {
			lines = append(lines, "- (none)")
		} else {
			for _, s := range symbols {
				doc := ""
				if s.Doc != "" {
					doc = ` doc="` + s.Doc + `"`
				}
				lines = append(lines, fmt.Sprintf("- %s %s%s", s.Kind, s.ID, doc))
			}
		}
		lines = append(lines, "", "Edges:")
		if len(edges) == 0 {
			lines = append(lines, "- (none)")
		} else {
			for _, e := range edges {
				lines = append(lines, fmt.Sprintf("- call %s -> %s", e.From, e.To))
			}
		}
		overview = strings.Join(lines, "\n")
		if len(overview) > maxContentChars {
			overview = overview[:maxContentChars] + "\n...TRUNCATED_BY_MAX_CONTENT_CHARS"
		}
		content = buildNLContent(symbols, edges)
		if len(content) > maxContentChars {
			content = content[:maxContentChars] + "\n\n...TRUNCATED"
		}
		logicGraph = map[string]any{
			"version": 1,
			"symbols": symbols,
			"edges":   edges,
		}
	}

	return sourceSummary{
		Abstract:   abstract,
		Overview:   overview,
		Content:    content,
		LogicGraph: logicGraph,
	}, nil
}
