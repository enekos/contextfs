import { createHash } from "crypto";
import * as fs from "fs";
import type { ScrapeCache as ScrapeCacheMap, CacheEntry } from "./types";

export class ScrapeCache {
  private data: ScrapeCacheMap = {};
  private filePath: string;

  constructor(filePath: string) {
    this.filePath = filePath;
    this.load();
  }

  private load(): void {
    if (!fs.existsSync(this.filePath)) return;
    try {
      const raw = fs.readFileSync(this.filePath, "utf-8");
      this.data = JSON.parse(raw);
    } catch {
      this.data = {};
    }
  }

  save(): void {
    fs.writeFileSync(this.filePath, JSON.stringify(this.data, null, 2), "utf-8");
  }

  get(url: string): CacheEntry | undefined {
    return this.data[url];
  }

  set(url: string, entry: CacheEntry): void {
    this.data[url] = entry;
  }

  isUnchanged(url: string, content: string): boolean {
    const entry = this.get(url);
    if (!entry) return false;
    const hash = createHash("sha1").update(content).digest("hex");
    return entry.contentHash === hash;
  }

  contentHash(content: string): string {
    return createHash("sha1").update(content).digest("hex");
  }
}
