export interface CrawlOptions {
  seedUrl: string;
  maxDepth: number;
  maxPages: number;
  concurrency: number;
  delayMs: number;
  urlPattern?: string;          // regex string — restrict which URLs to follow
  waitUntil: "networkidle" | "domcontentloaded" | "load";
  selector?: string;             // CSS selector to scope content extraction
}

export interface CrawledPage {
  url: string;
  html: string;
  title: string;
  links: string[];
  depth: number;
}

export interface Section {
  heading: string;
  content: string;
  level: 2 | 3;
}

export interface ExtractedContent {
  title: string;
  markdown: string;
  sections: Section[];
  wordCount: number;
}

export interface PageSummary {
  abstract: string;
  overview: string;
  ai_intent: "fact" | "decision" | "how_to" | "todo" | "warning" | null;
  ai_topics: string[];
  ai_quality_score: number;
}

export interface ScrapeOptions extends CrawlOptions {
  project: string;
  splitSections: boolean;
  dryRun: boolean;
  useRouter: boolean;
}

export interface ScrapeResult {
  pagesTotal: number;
  pagesStored: number;
  pagesUpdated: number;
  pagesSkipped: number;
  sectionsStored: number;
  errors: { url: string; error: string }[];
}

export interface CacheEntry {
  contentHash: string;
  scrapedAt: string;
  uri: string;
}

export type ScrapeCache = Record<string, CacheEntry>;
