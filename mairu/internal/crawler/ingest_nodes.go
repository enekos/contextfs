package crawler

import (
	"context"
	"fmt"
	"mairu/internal/contextsrv"
)

// MarkdownExtractNode extracts Markdown from HTML
type MarkdownExtractNode struct {
	Selector string
}

func (n *MarkdownExtractNode) Name() string {
	return "MarkdownExtract"
}

func (n *MarkdownExtractNode) Execute(ctx context.Context, state State) (State, error) {
	html, ok := state["html"].(string)
	if !ok {
		return state, fmt.Errorf("missing html in state")
	}
	urlStr, _ := state["url"].(string)

	content := ExtractContent(html, n.Selector, urlStr)
	if content.Markdown == "" {
		state["skipped"] = true
		return state, nil
	}

	state["title"] = content.Title
	state["markdown"] = content.Markdown
	return state, nil
}

// SummarizeNode generates a summary using Gemini
type SummarizeNode struct {
	APIKey string
}

func (n *SummarizeNode) Name() string {
	return "Summarize"
}

func (n *SummarizeNode) Execute(ctx context.Context, state State) (State, error) {
	if skipped, _ := state["skipped"].(bool); skipped {
		return state, nil
	}

	title, _ := state["title"].(string)
	markdown, _ := state["markdown"].(string)
	urlStr, _ := state["url"].(string)

	summary := SummarizePage(ctx, n.APIKey, title, markdown, urlStr)
	state["abstract"] = summary.Abstract
	state["overview"] = summary.Overview

	return state, nil
}

// StoreContextNode stores the resulting context node
type StoreContextNode struct {
	StoreFn NodeStoreFunc
	Project string
	DryRun  bool
}

func (n *StoreContextNode) Name() string {
	return "StoreContext"
}

func (n *StoreContextNode) Execute(ctx context.Context, state State) (State, error) {
	if skipped, _ := state["skipped"].(bool); skipped || n.DryRun || n.StoreFn == nil {
		return state, nil
	}

	urlStr, _ := state["url"].(string)
	title, _ := state["title"].(string)
	markdown, _ := state["markdown"].(string)
	abstract, _ := state["abstract"].(string)
	overview, _ := state["overview"].(string)

	uri := URLToURI(urlStr)
	parentURI := URLToParentURI(urlStr)

	err := n.StoreFn(ctx, contextsrv.ContextCreateInput{
		URI:       uri,
		Project:   n.Project,
		ParentURI: parentURI,
		Name:      title,
		Abstract:  abstract,
		Overview:  overview,
		Content:   markdown,
	})

	if err != nil {
		return state, err
	}

	state["stored"] = true
	return state, nil
}
