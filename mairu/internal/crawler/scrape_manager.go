package crawler

import (
	"context"

	"mairu/internal/contextsrv"
)

type NodeStoreFunc func(ctx context.Context, input contextsrv.ContextCreateInput) error

func ScrapeAndIngest(ctx context.Context, options ScrapeOptions, storeFn NodeStoreFunc, geminiAPIKey string) (*ScrapeResult, error) {
	out := make(chan CrawledPage, 10)
	go Crawl(options.CrawlOptions, out)

	graph := NewGraph(
		&MarkdownExtractNode{Selector: options.Selector},
		&SummarizeNode{APIKey: geminiAPIKey},
		&StoreContextNode{
			StoreFn: storeFn,
			Project: options.Project,
			DryRun:  options.DryRun,
		},
	)

	result := &ScrapeResult{}
	for page := range out {
		result.PagesTotal++

		state := State{
			"url":  page.URL,
			"html": page.HTML,
		}

		finalState, err := graph.Run(ctx, state)
		if err != nil {
			result.Errors = append(result.Errors, ScrapeError{URL: page.URL, Error: err.Error()})
		} else {
			if skipped, _ := finalState["skipped"].(bool); skipped {
				result.PagesSkipped++
			} else if stored, _ := finalState["stored"].(bool); stored {
				result.PagesStored++
			}
		}
	}

	return result, nil
}
