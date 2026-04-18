package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"mairu/internal/contextsrv"
	"mairu/internal/crawler"
	"mairu/internal/llm"

	"github.com/spf13/cobra"
)

func NewScrapeCmd() *cobra.Command {
	var (
		mode        string
		project     string
		prompt      string
		maxDepth    int
		maxPages    int
		maxResults  int
		concurrency int
		selector    string
		dryRun      bool
		output      string
	)

	cmd := &cobra.Command{
		Use:   "scrape [urls...]",
		Short: "Web scraping and content extraction tools",
		Long: `Unified web scraper. Use --mode to choose the scraping strategy:

  web    - BFS crawl + ingest (requires URL)
  smart  - Single-page LLM extraction (requires URL, --prompt)
  search - Search web + extract (requires query, --prompt)
  multi  - Concurrent multi-URL extraction (requires URLs, --prompt)
  omni   - Multi-URL merged extraction (requires URLs, --prompt)
  depth  - Depth-based link discovery + scrape (requires URL, --prompt)
  script - Generate Go scraper script (requires URL, --prompt)
`,
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			providerCfg := GetLLMProviderConfig()
			var provider llm.Provider
			var err error

			if mode != "web" {
				if providerCfg.APIKey == "" {
					return fmt.Errorf("LLM API key required for mode=%s", mode)
				}
				provider, err = llm.NewProvider(providerCfg)
				if err != nil {
					return fmt.Errorf("failed to init LLM: %w", err)
				}
			}

			engine := crawler.NewEngine(nil)
			engine.Concurrency = concurrency

			scraper := crawler.NewScraper(engine, provider)

			switch mode {
			case "web":
				if len(args) < 1 {
					return fmt.Errorf("mode=web requires a URL argument")
				}
				urlStr := args[0]
				fmt.Printf("Crawling %s...\n", urlStr)

				opts := crawler.ScrapeOptions{
					Project: project,
					DryRun:  dryRun,
					CrawlOptions: crawler.CrawlOptions{
						SeedURL:     urlStr,
						MaxPages:    maxPages,
						MaxDepth:    maxDepth,
						Concurrency: concurrency,
						Selector:    selector,
					},
				}

				storeFn := func(ctx context.Context, input contextsrv.ContextCreateInput) error {
					fmt.Printf("Storing node %s...\n", input.URI)
					var parent string
					if input.ParentURI != nil {
						parent = *input.ParentURI
					}
					return RunNodeStore(input.Project, input.URI, input.Name, input.Abstract, parent, input.Overview, input.Content)
				}

				res, err := scraper.Ingest(ctx, opts, storeFn)
				if err != nil {
					return err
				}
				fmt.Printf("Scraping complete. Total pages: %d, Stored: %d, Skipped: %d\n", res.PagesTotal, res.PagesStored, res.PagesSkipped)
				if len(res.Errors) > 0 {
					fmt.Printf("Errors encountered: %d\n", len(res.Errors))
				}

			case "smart":
				if len(args) < 1 {
					return fmt.Errorf("mode=smart requires a URL argument")
				}
				if prompt == "" {
					return fmt.Errorf("--prompt is required for mode=smart")
				}
				targetURL := args[0]
				fmt.Printf("Running smart scrape on %s...\n", targetURL)

				data, err := scraper.Smart(ctx, targetURL, prompt)
				if err != nil {
					return fmt.Errorf("scrape failed: %w", err)
				}
				if data == nil {
					fmt.Println("No data extracted.")
					return nil
				}
				printJSON(data)
				storeScrapeResult(project, targetURL, "smart", prompt, data)

			case "search":
				if len(args) < 1 {
					return fmt.Errorf("mode=search requires a query argument")
				}
				if prompt == "" {
					return fmt.Errorf("--prompt is required for mode=search")
				}
				query := args[0]
				fmt.Printf("Running search scrape for query '%s'...\n", query)

				results, err := scraper.Search(ctx, query, prompt, maxResults)
				if err != nil {
					return fmt.Errorf("search scrape failed: %w", err)
				}
				if len(results) == 0 {
					fmt.Println("No data extracted from any search result.")
					return nil
				}
				printJSON(results)
				uri := fmt.Sprintf("contextfs://search/%s", url.QueryEscape(query))
				storeScrapeResult(project, uri, "search", query, results)

			case "multi":
				if len(args) < 1 {
					return fmt.Errorf("mode=multi requires at least one URL argument")
				}
				if prompt == "" {
					return fmt.Errorf("--prompt is required for mode=multi")
				}
				fmt.Printf("Running multi-scrape on %d URLs...\n", len(args))

				data, err := scraper.Multi(ctx, args, prompt)
				if err != nil {
					return fmt.Errorf("multi-scrape failed: %w", err)
				}
				if len(data) == 0 {
					fmt.Println("No data extracted.")
					return nil
				}
				printJSON(data)
				for urlStr, result := range data {
					uri := fmt.Sprintf("contextfs://scrape/%s", cleanURL(urlStr))
					resBytes, _ := json.Marshal(result)
					RunNodeStore(project, uri, "Extracted Data", "Data extracted via multi-scrape: "+prompt, "", "", string(resBytes))
				}

			case "omni":
				if len(args) < 1 {
					return fmt.Errorf("mode=omni requires at least one URL argument")
				}
				if prompt == "" {
					return fmt.Errorf("--prompt is required for mode=omni")
				}
				fmt.Printf("Running omni-scrape on %d URLs...\n", len(args))

				data, err := scraper.Omni(ctx, args, prompt)
				if err != nil {
					return fmt.Errorf("omni-scrape failed: %w", err)
				}
				if data == nil {
					fmt.Println("No data extracted.")
					return nil
				}
				printJSON(data)
				uri := fmt.Sprintf("contextfs://omni-scrape/%s", cleanURL(args[0]))
				if len(args) > 1 {
					uri += "-and-others"
				}
				storeScrapeResult(project, uri, "Merged Omni Data", "Data merged via omni-scrape: "+prompt, data)

			case "depth":
				if len(args) < 1 {
					return fmt.Errorf("mode=depth requires a URL argument")
				}
				if prompt == "" {
					return fmt.Errorf("--prompt is required for mode=depth")
				}
				seedURL := args[0]
				fmt.Printf("Running depth-scrape (depth: %d) on %s...\n", maxDepth, seedURL)

				data, err := scraper.Depth(ctx, seedURL, prompt, maxDepth)
				if err != nil {
					return fmt.Errorf("depth-scrape failed: %w", err)
				}
				if len(data) == 0 {
					fmt.Println("No data extracted.")
					return nil
				}
				printJSON(data)
				for urlStr, result := range data {
					uri := fmt.Sprintf("contextfs://scrape/%s", cleanURL(urlStr))
					resBytes, _ := json.Marshal(result)
					RunNodeStore(project, uri, "Extracted Data", "Data extracted via depth-scrape: "+prompt, "", "", string(resBytes))
				}

			case "script":
				if len(args) < 1 {
					return fmt.Errorf("mode=script requires a URL argument")
				}
				if prompt == "" {
					return fmt.Errorf("--prompt is required for mode=script")
				}
				targetURL := args[0]
				fmt.Printf("Generating scraper script for %s...\n", targetURL)

				scriptContent, err := scraper.Script(ctx, targetURL, prompt)
				if err != nil {
					return fmt.Errorf("script generation failed: %w", err)
				}
				if scriptContent == "" {
					fmt.Println("No script generated.")
					return nil
				}
				if output != "" {
					if err := os.WriteFile(output, []byte(scriptContent), 0644); err != nil {
						return fmt.Errorf("failed to write script: %w", err)
					}
					fmt.Printf("Script saved to %s\n", output)
				} else {
					fmt.Printf("\n--- Generated Go Script ---\n\n%s\n\n", scriptContent)
				}

			default:
				return fmt.Errorf("unknown mode: %s", mode)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "web", "Scraping mode: web, smart, search, multi, omni, depth, script")
	cmd.Flags().StringVarP(&project, "project", "P", "default", "Project namespace")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt to instruct LLM what to extract")
	cmd.Flags().IntVar(&maxDepth, "max-depth", 3, "Max depth to crawl (web/depth modes)")
	cmd.Flags().IntVar(&maxPages, "max-pages", 50, "Max pages to crawl (web mode)")
	cmd.Flags().IntVar(&maxResults, "max-results", 3, "Max search results to process (search mode)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 3, "Concurrent requests")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector to target")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run (web mode)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file for generated script (script mode)")

	return cmd
}

func printJSON(v any) {
	jsonBytes, _ := json.MarshalIndent(v, "", "  ")
	fmt.Printf("\n%s\n\n", string(jsonBytes))
}

func cleanURL(raw string) string {
	return strings.ReplaceAll(strings.ReplaceAll(raw, "https://", ""), "http://", "")
}

func storeScrapeResult(project, uri, name, desc string, data any) {
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	content := string(jsonBytes)
	fmt.Printf("Storing extracted data at %s in project '%s'...\n", uri, project)
	RunNodeStore(project, uri, name, desc, "", "", content)
}
