package ast

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// extractMarkdownHeadings
// ---------------------------------------------------------------------------

func TestExtractMarkdownHeadings_BasicLevels(t *testing.T) {
	src := "# H1\n## H2\n### H3\n#### H4\n"
	syms := extractMarkdownHeadings(src)
	if len(syms) != 4 {
		t.Fatalf("want 4 symbols, got %d: %#v", len(syms), syms)
	}
	cases := []struct{ id, name, kind string }{
		{"h1:H1", "H1", "h1"},
		{"h2:H2", "H2", "h2"},
		{"h3:H3", "H3", "h3"},
		{"h4:H4", "H4", "h4"},
	}
	for i, c := range cases {
		s := syms[i]
		if s.ID != c.id || s.Name != c.name || s.Kind != c.kind {
			t.Errorf("[%d] want {%s,%s,%s}, got {%s,%s,%s}", i, c.id, c.name, c.kind, s.ID, s.Name, s.Kind)
		}
	}
}

func TestExtractMarkdownHeadings_DuplicateNames(t *testing.T) {
	src := "## Setup\n\n## Setup\n\n## Setup\n"
	syms := extractMarkdownHeadings(src)
	if len(syms) != 3 {
		t.Fatalf("want 3 symbols, got %d", len(syms))
	}
	if syms[0].ID != "h2:Setup" {
		t.Errorf("first: want h2:Setup, got %s", syms[0].ID)
	}
	if syms[1].ID != "h2:Setup:2" {
		t.Errorf("second: want h2:Setup:2, got %s", syms[1].ID)
	}
	if syms[2].ID != "h2:Setup:3" {
		t.Errorf("third: want h2:Setup:3, got %s", syms[2].ID)
	}
}

func TestExtractMarkdownHeadings_Empty(t *testing.T) {
	if syms := extractMarkdownHeadings(""); len(syms) != 0 {
		t.Fatalf("expected no symbols, got %d", len(syms))
	}
}

func TestExtractMarkdownHeadings_NoHeadings(t *testing.T) {
	src := "Just prose.\n\nAnother paragraph.\n"
	if syms := extractMarkdownHeadings(src); len(syms) != 0 {
		t.Fatalf("expected no symbols, got %d: %#v", len(syms), syms)
	}
}

func TestExtractMarkdownHeadings_TrailingWhitespace(t *testing.T) {
	src := "#  Spaced Title  \n"
	syms := extractMarkdownHeadings(src)
	if len(syms) != 1 || syms[0].Name != "Spaced Title" {
		t.Fatalf("unexpected: %#v", syms)
	}
}

// ---------------------------------------------------------------------------
// extractMarkdownSummary
// ---------------------------------------------------------------------------

func TestExtractMarkdownSummary_H1WithParagraph(t *testing.T) {
	src := "# My Project\n\nA tool for managing context.\n\n## Install\n"
	got := extractMarkdownSummary(src)
	want := "My Project: A tool for managing context."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractMarkdownSummary_MultiLineParagraph(t *testing.T) {
	// Multiple lines before the blank line are joined with spaces
	src := "# Title\n\nFirst line.\nSecond line.\n\n## Section\n"
	got := extractMarkdownSummary(src)
	if !strings.Contains(got, "First line.") || !strings.Contains(got, "Second line.") {
		t.Errorf("expected both lines joined, got %q", got)
	}
}

func TestExtractMarkdownSummary_H1Only(t *testing.T) {
	// No paragraph after H1 — summary is just the title
	src := "# My Project\n\n## Installation\n"
	got := extractMarkdownSummary(src)
	if got != "My Project" {
		t.Errorf("got %q, want \"My Project\"", got)
	}
}

func TestExtractMarkdownSummary_NoH1(t *testing.T) {
	src := "## Installation\n\nRun the script.\n"
	got := extractMarkdownSummary(src)
	if got != "Markdown document" {
		t.Errorf("got %q, want \"Markdown document\"", got)
	}
}

func TestExtractMarkdownSummary_EmptyString(t *testing.T) {
	if got := extractMarkdownSummary(""); got != "Markdown document" {
		t.Errorf("got %q", got)
	}
}

func TestExtractMarkdownSummary_YAMLFrontmatter(t *testing.T) {
	src := "---\ntitle: Test\ndate: 2024-01-01\n---\n\n# My Project\n\nSome description.\n"
	got := extractMarkdownSummary(src)
	if !strings.HasPrefix(got, "My Project") {
		t.Errorf("frontmatter not skipped, got %q", got)
	}
	if strings.Contains(got, "title:") || strings.Contains(got, "date:") {
		t.Errorf("frontmatter leaked into summary: %q", got)
	}
}

func TestExtractMarkdownSummary_TruncatesAt200(t *testing.T) {
	longDesc := strings.Repeat("word ", 60) // well over 200 chars
	src := "# Title\n\n" + longDesc + "\n"
	got := extractMarkdownSummary(src)
	if len(got) > 203 { // 200 chars + "..."
		t.Errorf("summary too long: %d chars", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected trailing '...', got %q", got)
	}
}

func TestExtractMarkdownSummary_SkipsImageLinesAfterH1(t *testing.T) {
	src := "# Title\n\n![badge](https://ci.example.com/badge)\n\nReal description here.\n"
	got := extractMarkdownSummary(src)
	if strings.Contains(got, "![badge]") {
		t.Errorf("image line leaked into summary: %q", got)
	}
	// Should fall through to the next text paragraph
	if !strings.Contains(got, "Real description here.") {
		t.Errorf("expected real description in summary, got %q", got)
	}
}

func TestExtractMarkdownSummary_StopsAtNextHeading(t *testing.T) {
	src := "# Title\n\n## Sub\n\nParagraph under sub.\n"
	got := extractMarkdownSummary(src)
	// The paragraph is under a different heading — should NOT appear in the title summary
	if strings.Contains(got, "Paragraph under sub") {
		t.Errorf("text from sub-section leaked into top-level summary: %q", got)
	}
	if got != "Title" {
		t.Errorf("got %q, want \"Title\"", got)
	}
}

// ---------------------------------------------------------------------------
// MarkdownDescriber.ExtractFileGraph
// ---------------------------------------------------------------------------

func TestMarkdownDescriber_LanguageID(t *testing.T) {
	if got := (MarkdownDescriber{}).LanguageID(); got != "markdown" {
		t.Errorf("LanguageID = %q, want \"markdown\"", got)
	}
}

func TestMarkdownDescriber_Extensions(t *testing.T) {
	exts := (MarkdownDescriber{}).Extensions()
	want := map[string]bool{".md": true, ".mdx": true}
	if len(exts) != len(want) {
		t.Fatalf("Extensions = %v, want [.md .mdx]", exts)
	}
	for _, e := range exts {
		if !want[e] {
			t.Errorf("unexpected extension %q", e)
		}
	}
}

func TestMarkdownDescriber_ExtractFileGraph_Headings(t *testing.T) {
	src := "# Title\n\nDescription.\n\n## Section A\n\n## Section B\n"
	g := (MarkdownDescriber{}).ExtractFileGraph("README.md", src)

	if len(g.Symbols) != 3 {
		t.Fatalf("want 3 symbols (H1+2xH2), got %d: %#v", len(g.Symbols), g.Symbols)
	}
	if g.Symbols[0].Kind != "h1" || g.Symbols[1].Kind != "h2" || g.Symbols[2].Kind != "h2" {
		t.Errorf("unexpected symbol kinds: %#v", g.Symbols)
	}
}

func TestMarkdownDescriber_ExtractFileGraph_RawContent(t *testing.T) {
	src := "# Title\n\nSome content.\n"
	g := (MarkdownDescriber{}).ExtractFileGraph("README.md", src)
	if g.RawContent != src {
		t.Errorf("RawContent not set to full source\ngot:  %q\nwant: %q", g.RawContent, src)
	}
}

func TestMarkdownDescriber_ExtractFileGraph_FileSummary(t *testing.T) {
	src := "# My Tool\n\nDoes something useful.\n"
	g := (MarkdownDescriber{}).ExtractFileGraph("README.md", src)
	if g.FileSummary == "" {
		t.Fatal("FileSummary is empty")
	}
	if !strings.Contains(g.FileSummary, "My Tool") {
		t.Errorf("FileSummary missing title: %q", g.FileSummary)
	}
}

func TestMarkdownDescriber_ExtractFileGraph_NoEdgesNoImports(t *testing.T) {
	src := "# Title\n\n## Section\n"
	g := (MarkdownDescriber{}).ExtractFileGraph("README.md", src)
	if len(g.Edges) != 0 {
		t.Errorf("expected no edges, got %d", len(g.Edges))
	}
	if len(g.Imports) != 0 {
		t.Errorf("expected no imports, got %d", len(g.Imports))
	}
}
