package agent

import (
	"context"
	"fmt"

	"mairu/internal/llm"
)

type fetchURLTool struct{}

func (t *fetchURLTool) Definition() llm.Tool {
	return llm.Tool{
		Name:        "fetch_url",
		Description: "Fetch the text content of a web page by URL. Useful for reading documentation or external resources.",
		Parameters: &llm.JSONSchema{
			Type: llm.TypeObject,
			Properties: map[string]*llm.JSONSchema{
				"url": {Type: llm.TypeString, Description: "The full URL to fetch (e.g., https://example.com)."},
			},
			Required: []string{"url"},
		},
	}
}

func (t *fetchURLTool) Execute(ctx context.Context, args map[string]any, a *Agent, outChan chan<- AgentEvent) (map[string]any, error) {
	urlToFetch, _ := args["url"].(string)
	outChan <- AgentEvent{Type: "status", Content: fmt.Sprintf("🌐 Fetching URL: %s", urlToFetch)}
	content, err := a.scraper().Fetch(ctx, urlToFetch)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	return map[string]any{"content": content}, nil
}

type scrapeURLTool struct{}

func (t *scrapeURLTool) Definition() llm.Tool {
	return llm.Tool{
		Name:        "scrape_url",
		Description: "Scrape a web page and extract structured information based on a prompt. Use this when you need specific data extracted intelligently from a website.",
		Parameters: &llm.JSONSchema{
			Type: llm.TypeObject,
			Properties: map[string]*llm.JSONSchema{
				"url":    {Type: llm.TypeString, Description: "The full URL to scrape (e.g., https://example.com)."},
				"prompt": {Type: llm.TypeString, Description: "The instructions on what information to extract from the page."},
			},
			Required: []string{"url", "prompt"},
		},
	}
}

func (t *scrapeURLTool) Execute(ctx context.Context, args map[string]any, a *Agent, outChan chan<- AgentEvent) (map[string]any, error) {
	urlToScrape, _ := args["url"].(string)
	prompt, _ := args["prompt"].(string)
	outChan <- AgentEvent{Type: "status", Content: fmt.Sprintf("🕸️ Scraping URL: %s", urlToScrape)}
	data, err := a.scraper().Smart(ctx, urlToScrape, prompt)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	return map[string]any{"data": data}, nil
}

type searchWebTool struct{}

func (t *searchWebTool) Definition() llm.Tool {
	return llm.Tool{
		Name:        "search_web",
		Description: "Search the web for a query and extract structured information from the top results based on a prompt.",
		Parameters: &llm.JSONSchema{
			Type: llm.TypeObject,
			Properties: map[string]*llm.JSONSchema{
				"query":  {Type: llm.TypeString, Description: "The search query to look up on the web."},
				"prompt": {Type: llm.TypeString, Description: "The instructions on what information to extract from the search results."},
			},
			Required: []string{"query", "prompt"},
		},
	}
}

func (t *searchWebTool) Execute(ctx context.Context, args map[string]any, a *Agent, outChan chan<- AgentEvent) (map[string]any, error) {
	query, _ := args["query"].(string)
	prompt, _ := args["prompt"].(string)
	outChan <- AgentEvent{Type: "status", Content: fmt.Sprintf("🔍 Searching Web: %s", query)}
	data, err := a.scraper().Search(ctx, query, prompt, 3)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	return map[string]any{"data": data}, nil
}
