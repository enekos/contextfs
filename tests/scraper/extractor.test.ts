import { describe, it, expect } from "vitest";
import { extractContent } from "../../mairu/contextfs/src/scraper/extractor";

const simpleHtml = `
<html>
<head><title>Test Page</title></head>
<body>
  <nav>Navigation stuff</nav>
  <main>
    <h1>Hello World</h1>
    <p>This is a paragraph with some content about authentication.</p>
    <h2>Section One</h2>
    <p>Content under section one. This section covers the basics of configuration and setup for your application environment.</p>
    <p>Additional details about section one including examples and use cases for developers.</p>
    <h2>Section Two</h2>
    <p>Content under section two. This section explains advanced features and integration patterns.</p>
    <p>More information about section two including API references and code samples.</p>
  </main>
  <footer>Footer stuff</footer>
</body>
</html>
`;

const codeHtml = `
<html>
<body>
  <article>
    <h1>API Reference</h1>
    <p>Use the following code to fetch data from the API endpoint:</p>
    <pre><code>const x = await fetch('/api/data');</code></pre>
    <p>This will return a JSON response with the requested data fields.</p>
  </article>
</body>
</html>
`;

describe("extractContent", () => {
  it("extracts main content and removes nav/footer", () => {
    const result = extractContent(simpleHtml);
    expect(result.markdown).toContain("Hello World");
    expect(result.markdown).toContain("authentication");
    expect(result.markdown).not.toContain("Navigation stuff");
    expect(result.markdown).not.toContain("Footer stuff");
  });

  it("preserves headings as markdown", () => {
    const result = extractContent(simpleHtml);
    expect(result.markdown).toMatch(/#+\s*Section One/);
    expect(result.markdown).toMatch(/#+\s*Section Two/);
  });

  it("splits sections by h2 headings", () => {
    const result = extractContent(simpleHtml);
    expect(result.sections.length).toBeGreaterThanOrEqual(2);
    expect(result.sections[0].heading).toBe("Section One");
    expect(result.sections[0].level).toBe(2);
    expect(result.sections[0].content).toContain("Content under section one");
  });

  it("preserves code blocks", () => {
    const result = extractContent(codeHtml);
    expect(result.markdown).toContain("fetch");
  });

  it("counts words", () => {
    const result = extractContent(simpleHtml);
    expect(result.wordCount).toBeGreaterThan(5);
  });

  it("uses CSS selector when provided", () => {
    const result = extractContent(simpleHtml, { selector: "main" });
    expect(result.markdown).toContain("Hello World");
    expect(result.markdown).not.toContain("Navigation stuff");
  });

  it("returns empty result for empty HTML", () => {
    const result = extractContent("<html><body></body></html>");
    expect(result.wordCount).toBe(0);
    expect(result.sections).toHaveLength(0);
  });
});
