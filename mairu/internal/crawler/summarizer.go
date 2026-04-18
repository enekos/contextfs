package crawler

import (
	"context"
	"strings"

	"mairu/internal/llm"
	"mairu/internal/prompts"
)

const maxInputChars = 8000 * 4
const shortPageThreshold = 5

func truncateMarkdown(markdown string) string {
	if len(markdown) <= maxInputChars {
		return markdown
	}
	return markdown[:maxInputChars] + "\n\n[content truncated]"
}

func buildPrompt(title, markdown, url string) (string, error) {
	return prompts.Render("scraper_page_summarize", struct {
		URL      string
		Title    string
		Markdown string
	}{
		URL:      url,
		Title:    title,
		Markdown: truncateMarkdown(markdown),
	})
}

func fallbackSummary(title, markdown, url string) PageSummary {
	firstLine := title
	lines := strings.Split(markdown, "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			firstLine = l
			break
		}
	}
	abstract := title + " (" + url + "): " + firstLine
	if len(abstract) > 200 {
		abstract = abstract[:200]
	}
	overview := markdown
	if len(overview) > 500 {
		overview = overview[:500]
	}
	return PageSummary{
		Abstract:       abstract,
		Overview:       overview,
		AIIntent:       nil,
		AITopics:       []string{},
		AIQualityScore: 5,
	}
}

// SummarizePage summarizes the given markdown content using the provided LLM.
func SummarizePage(ctx context.Context, provider llm.Provider, title, markdown, url string) PageSummary {
	words := strings.Fields(markdown)
	if len(words) < shortPageThreshold || provider == nil {
		return fallbackSummary(title, markdown, url)
	}

	prompt, err := buildPrompt(title, markdown, url)
	if err != nil {
		return fallbackSummary(title, markdown, url)
	}

	var parsed PageSummary
	if err := provider.GenerateJSON(ctx, "", prompt, nil, &parsed); err != nil {
		return fallbackSummary(title, markdown, url)
	}

	if parsed.Abstract == "" {
		parsed.Abstract = fallbackSummary(title, markdown, url).Abstract
	}
	if parsed.Overview == "" {
		parsed.Overview = fallbackSummary(title, markdown, url).Overview
	}

	return parsed
}
