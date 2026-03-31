import { Readability } from "@mozilla/readability";
import { parseHTML } from "linkedom";
import TurndownService from "turndown";
import type { ExtractedContent, Section } from "./types";

const td = new TurndownService({
  headingStyle: "atx",
  codeBlockStyle: "fenced",
  bulletListMarker: "-",
});

export function extractContent(
  html: string,
  options?: { selector?: string }
): ExtractedContent {
  const { document } = parseHTML(html);

  // Narrow to selector if provided
  let root: Element | typeof document = document;
  if (options?.selector) {
    const el = document.querySelector(options.selector);
    if (el) root = el as unknown as typeof document;
  }

  // Try readability on the root element's outerHTML
  const htmlForReadability =
    root === document ? html : (root as unknown as Element).outerHTML;
  const { document: rdDoc } = parseHTML(
    `<html><body>${htmlForReadability}</body></html>`
  );
  const reader = new Readability(rdDoc as unknown as Document);
  const article = reader.parse();

  // Fall back to direct body content if readability fails
  let contentHtml: string;
  let title: string;

  if (article && article.content) {
    contentHtml = article.content;
    title = article.title || "";
  } else {
    // Fallback: use the selected element or body directly
    if (root === document) {
      const body = document.querySelector("body");
      if (!body || !body.textContent?.trim()) {
        return { title: "", markdown: "", sections: [], wordCount: 0 };
      }
      contentHtml = body.innerHTML;
    } else {
      const el = root as unknown as Element;
      if (!el.textContent?.trim()) {
        return { title: "", markdown: "", sections: [], wordCount: 0 };
      }
      contentHtml = el.innerHTML;
    }
    const titleEl = document.querySelector("title");
    title = titleEl?.textContent?.trim() ?? "";
  }

  // Extract the original h1 text so splitSections can skip it if Readability promoted h1→h2
  const originalH1 = document.querySelector("h1")?.textContent?.trim();
  const markdown = td.turndown(contentHtml);
  const sections = splitSections(contentHtml, originalH1);
  const wordCount = markdown.split(/\s+/).filter(Boolean).length;

  return {
    title,
    markdown,
    sections,
    wordCount,
  };
}

function collectElements(node: Element): Element[] {
  const tag = node.tagName?.toLowerCase();
  // If it's a heading or paragraph/pre/ul/ol/table, collect directly
  if (tag && /^(h[1-6]|p|pre|ul|ol|table|blockquote|figure)$/.test(tag)) {
    return [node];
  }
  // Otherwise recurse into children (handles wrapper divs from Readability)
  const result: Element[] = [];
  for (const child of Array.from(node.childNodes ?? [])) {
    result.push(...collectElements(child as Element));
  }
  return result;
}

function splitSections(html: string, skipHeading?: string): Section[] {
  const { document } = parseHTML(`<div>${html}</div>`);
  const sections: Section[] = [];
  let currentHeading: { text: string; level: 2 | 3 } | null = null;
  const currentContent: string[] = [];

  const flush = () => {
    if (currentHeading && currentContent.length > 0) {
      sections.push({
        heading: currentHeading.text,
        level: currentHeading.level,
        content: td.turndown(currentContent.join("\n")),
      });
    }
    currentContent.length = 0;
  };

  const rootDiv = document.querySelector("div");
  const elements = rootDiv ? collectElements(rootDiv) : [];

  for (const el of elements) {
    const tag = el.tagName?.toLowerCase();
    if (tag === "h2" || tag === "h3") {
      flush();
      const headingText = el.textContent?.trim() ?? "";
      // Skip headings that match the article title (Readability promotes h1→h2)
      if (skipHeading && headingText === skipHeading) {
        currentHeading = null;
        continue;
      }
      currentHeading = {
        text: headingText,
        level: tag === "h2" ? 2 : 3,
      };
    } else if (currentHeading) {
      const outerHTML = (el as Element & { outerHTML?: string }).outerHTML;
      currentContent.push(outerHTML ?? el.textContent ?? "");
    }
  }
  flush();
  return sections;
}
