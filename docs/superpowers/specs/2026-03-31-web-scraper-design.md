# Web Scraper — Design Spec

**Date:** 2026-03-31
**Status:** Approved

## Summary

A Playwright-based web crawler that scrapes entire website trees and stores them as hierarchical context nodes in contextfs. Targets both documentation sites and general websites. Uses LLM for smart content summarization (abstract/overview/topics), and leverages existing infrastructure (embeddings, LLM router dedup, context node storage) for indexing.

**CLI-first** — dashboard API endpoint is out of scope for v1.

## Goals

1. Crawl a website starting from a seed URL, following links up to a configurable depth
2. Render pages with Playwright (full JS execution) so SPAs and dynamic doc sites work
3. Extract clean content from pages (strip nav/footer/chrome)
4. Use LLM to generate abstract, overview, intent, and topics for each page
5. Store pages as hierarchical context nodes with URI structure mirroring the site
6. Support re-scraping with content-hash dedup to avoid redundant writes
7. Provide a CLI command: `context-cli scrape <url> -P <project>`

## Non-Goals

- Real-time monitoring / continuous crawling (daemon-style)
- Dashboard UI for scraping
- Screenshot or image extraction
- PDF/file download handling
- Authentication / login flows (v1 — could be added later)

## Architecture

### New Modules

| File | Role |
|---|---|
| `src/scraper/crawler.ts` | Playwright crawl engine — renders pages, discovers links, manages queue |
| `src/scraper/extractor.ts` | HTML → clean text → structured content (readability + turndown + section splitting) |
| `src/scraper/summarizer.ts` | LLM calls to generate abstract/overview/intent/topics per page |
| `src/scraper/scrapeManager.ts` | Orchestrator — wires crawler → extractor → summarizer → contextManager storage |
| `src/scraper/cache.ts` | Persistent URL → content_hash cache (`.contextfs-scrape-cache.json`) |

### Integration Points (Existing Code)

| Module | Usage |
|---|---|
| `contextManager.addContextNode()` | Store each page as a context node (with LLM router dedup) |
| `embedder.ts` | Generate embeddings for `name + abstract` |
| `llmRouter.ts` | Dedup — decide CREATE/UPDATE/SKIP for pages similar to existing nodes |

### Dependencies (New)

| Package | Purpose |
|---|---|
| `playwright` | Headless browser for JS-rendered page crawling |
| `@mozilla/readability` | Extract main article content from HTML (strip nav/chrome) |
| `turndown` | Convert clean HTML → markdown text |
| `linkedom` | Lightweight DOM for readability (no full browser needed for parsing) |

## Detailed Design

### 1. Crawl Engine (`src/scraper/crawler.ts`)

```typescript
interface CrawlOptions {
  seedUrl: string;
  maxDepth: number;        // default: 3
  maxPages: number;        // default: 50
  concurrency: number;     // default: 3
  delayMs: number;         // default: 500
  urlPattern?: string;     // glob or regex to restrict crawl scope
  waitUntil: "networkidle" | "domcontentloaded" | "load"; // default: "networkidle"
  selector?: string;       // optional CSS selector for content extraction
}

interface CrawledPage {
  url: string;
  html: string;
  title: string;
  links: string[];
  depth: number;
}

async function* crawl(options: CrawlOptions): AsyncGenerator<CrawledPage>
```

**Behavior:**
- Launches a single Playwright browser with N concurrent pages (tabs)
- BFS traversal from seed URL — each depth level completes before the next starts
- Link discovery: extract all `<a href>` from rendered DOM, normalize to absolute URLs, filter by:
  - Same domain as seed URL (default) or match `urlPattern`
  - Skip fragment-only links, mailto:, tel:, javascript:, asset URLs (.png, .pdf, .css, .js, etc.)
  - Deduplicate by normalized URL (strip trailing slash, fragment, sort query params)
- Rate limiting: `delayMs` between requests per concurrent page
- Yields pages as they're crawled (streaming, not batch)
- Graceful shutdown: on error, log and continue with remaining queue

### 2. Content Extractor (`src/scraper/extractor.ts`)

```typescript
interface ExtractedContent {
  title: string;
  markdown: string;          // full cleaned text as markdown
  sections: Section[];       // h2/h3-split sections (optional)
  wordCount: number;
}

interface Section {
  heading: string;
  content: string;
  level: number;             // 2 or 3
}

function extractContent(html: string, options?: { selector?: string }): ExtractedContent
```

**Pipeline:**
1. Parse HTML with `linkedom`
2. If `selector` provided, narrow to that DOM subtree
3. Run `@mozilla/readability` → clean article HTML
4. Convert to markdown with `turndown` (preserve headings, code blocks, lists, tables)
5. Split by h2/h3 headings into `Section[]` for optional child node creation
6. Return structured result

### 3. LLM Summarizer (`src/scraper/summarizer.ts`)

```typescript
interface PageSummary {
  abstract: string;          // 1-2 sentences
  overview: string;          // ~500 tokens, key topics and structure
  ai_intent: string;         // fact | decision | how_to | todo | warning
  ai_topics: string[];       // topic tags
  ai_quality_score: number;  // 1-10
}

async function summarizePage(title: string, markdown: string, url: string): Promise<PageSummary>
```

**Behavior:**
- Single LLM call per page with a structured prompt
- Input: page title + truncated markdown (cap at ~8k tokens to control cost)
- Output: structured JSON with all summary fields
- Uses existing LLM infrastructure (Gemini via the project's LLM setup)
- Skip LLM for very short pages (< 50 words) — generate minimal abstract from title + first sentence

### 4. URI & Hierarchy Strategy

Map website URL structure to contextfs URIs:

```
Seed: https://docs.example.com/api/v2/auth

URI mapping:
  https://docs.example.com/           → contextfs://scraped/docs-example-com/
  https://docs.example.com/api/       → contextfs://scraped/docs-example-com/api
  https://docs.example.com/api/v2/    → contextfs://scraped/docs-example-com/api/v2
  https://docs.example.com/api/v2/auth → contextfs://scraped/docs-example-com/api/v2/auth
```

**Rules:**
- Domain normalized: dots → hyphens, lowercase
- Path segments map 1:1 to URI segments
- `parent_uri` = URI with last segment removed
- If `--split-sections` enabled, h2/h3 sections become children:
  `contextfs://scraped/docs-example-com/api/v2/auth/tokens` (from `## Tokens` heading)
- Intermediate parent nodes auto-created with minimal content if they weren't crawled

### 5. Scrape Manager (`src/scraper/scrapeManager.ts`)

The orchestrator that wires everything together:

```typescript
interface ScrapeOptions extends CrawlOptions {
  project: string;
  splitSections: boolean;    // default: false
  dryRun: boolean;           // default: false
  useRouter: boolean;        // default: true (LLM dedup)
}

interface ScrapeResult {
  pagesTotal: number;
  pagesStored: number;
  pagesUpdated: number;
  pagesSkipped: number;
  sectionsStored: number;
  errors: { url: string; error: string }[];
}

async function scrapeAndIngest(options: ScrapeOptions): Promise<ScrapeResult>
```

**Pipeline per page:**
1. Receive `CrawledPage` from crawler
2. Check scrape cache — skip if content hash unchanged
3. Run extractor → `ExtractedContent`
4. Run LLM summarizer → `PageSummary`
5. Build context node: URI from URL, abstract/overview from summary, content = markdown
6. Call `contextManager.addContextNode()` with `useRouter` for dedup
7. If `splitSections` and page has sections, store each as a child node
8. Update scrape cache
9. Log progress

**Dry-run mode:** Crawl and extract but don't store — print the planned URI tree and page summaries.

### 6. Persistent Cache (`src/scraper/cache.ts`)

```typescript
// .contextfs-scrape-cache.json
{
  "https://docs.example.com/api": {
    "contentHash": "a1b2c3...",
    "scrapedAt": "2026-03-31T10:00:00Z",
    "uri": "contextfs://scraped/docs-example-com/api"
  }
}
```

- Load on start, save on completion
- Content hash = SHA1 of extracted markdown text
- On re-scrape: skip pages with unchanged content hash

### 7. CLI Command

Added to `src/cli.ts`:

```
context-cli scrape <url>

Options:
  -P, --project <project>      Project namespace (required)
  -d, --depth <n>              Max crawl depth (default: 3)
  -m, --max-pages <n>          Max pages to crawl (default: 50)
  -c, --concurrency <n>        Parallel browser pages (default: 3)
  --delay <ms>                 Delay between requests (default: 500)
  --pattern <glob>             URL pattern filter (restrict crawl scope)
  --selector <css>             CSS selector for content extraction
  --split-sections             Split pages into section child nodes by h2/h3
  --wait-until <event>         Page load event (networkidle|domcontentloaded|load)
  --dry-run                    Show crawl plan without storing
  --no-router                  Skip LLM dedup (always create)
```

**Output:**
```
Scraping https://docs.example.com ...
  [1/24] /getting-started .............. stored (new)
  [2/24] /api/authentication ........... stored (new)
  [3/24] /api/endpoints ................ updated (similar to existing)
  [4/24] /changelog .................... skipped (unchanged)
  ...

Done: 24 pages crawled, 20 stored, 2 updated, 2 skipped, 0 errors
```

## LLM Usage

| Stage | LLM Calls | Purpose |
|---|---|---|
| Summarization | 1 per page | Generate abstract, overview, intent, topics, quality score |
| Dedup (router) | 0-1 per page | Only when existing node has cosine similarity ≥ 0.75 |
| Crawl / Extract | 0 | Pure Playwright + DOM parsing, no LLM needed |

**Cost estimate:** ~50 pages × 1 LLM call each = ~50 calls. With truncation to 8k input tokens, this is modest.

## Testing Strategy

- **Unit tests:** Extractor (HTML → markdown, section splitting) with fixture HTML files
- **Unit tests:** URI mapping (URL → contextfs URI, parent derivation)
- **Unit tests:** Cache (load/save/hash comparison)
- **Integration test:** Crawl a local test server (Playwright test fixtures) → verify nodes stored correctly
- **Summarizer:** Mock LLM calls, verify prompt structure and response parsing

## Future Extensions (Out of Scope for v1)

- Dashboard API endpoint for triggering scrapes
- Authentication support (cookie injection, login flows)
- Scheduled re-scraping (cron-style)
- PDF / file download handling
- Image extraction and description
- Sitemap.xml parsing for faster discovery
- Robots.txt respect
