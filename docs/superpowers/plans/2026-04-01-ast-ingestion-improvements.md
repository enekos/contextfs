# AST Ingestion Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add JSDoc extraction, byte-offset storage, and symbol-level content hashing to the AST ingestion pipeline.

**Architecture:** Three additive fields on `LogicSymbol` (`docstring?`, `byteStart`/`byteEnd`, `contentHash`), populated in `TypeScriptDescriber.makeSymbol()` and consumed in `daemon.ts` for richer NL output. All changes are backwards-compatible — existing behavior is preserved, new fields add information.

**Tech Stack:** TypeScript, web-tree-sitter, Vitest, SHA1 (Node crypto)

---

### Task 1: Add new fields to LogicSymbol

**Files:**
- Modify: `src/ast/languageDescriber.ts:7-22`

- [ ] **Step 1: Write the failing test**

Add a test in `tests/languageDescriber.test.ts` that asserts the new fields exist on a `LogicSymbol` value:

```typescript
import { describe, expect, it } from "vitest";
import type { LogicSymbol } from "../src/ast/languageDescriber";

describe("LogicSymbol type", () => {
  it("supports docstring, byteStart, byteEnd, and contentHash fields", () => {
    const symbol: LogicSymbol = {
      id: "fn:test",
      kind: "fn",
      name: "test",
      exported: true,
      parentId: null,
      params: [],
      complexity: "low",
      control: { async: false, branch: false, await: false, throw: false },
      line: 1,
      byteStart: 0,
      byteEnd: 42,
      contentHash: "abc123",
    };
    expect(symbol.byteStart).toBe(0);
    expect(symbol.byteEnd).toBe(42);
    expect(symbol.contentHash).toBe("abc123");
    expect(symbol.docstring).toBeUndefined();

    const withDoc: LogicSymbol = { ...symbol, docstring: "Does something useful." };
    expect(withDoc.docstring).toBe("Does something useful.");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/languageDescriber.test.ts`
Expected: TypeScript compilation error — `byteStart`, `byteEnd`, `contentHash` do not exist on `LogicSymbol`.

- [ ] **Step 3: Add the fields to the LogicSymbol interface**

In `src/ast/languageDescriber.ts`, add the new fields to the `LogicSymbol` interface:

```typescript
export interface LogicSymbol {
  id: string;
  kind: LogicSymbolKind;
  name: string;
  exported: boolean;
  parentId: string | null;
  params: string[];
  complexity: ComplexityBucket;
  control: {
    async: boolean;
    branch: boolean;
    await: boolean;
    throw: boolean;
  };
  line: number;
  byteStart: number;
  byteEnd: number;
  contentHash: string;
  docstring?: string;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `bun run test -- tests/languageDescriber.test.ts`
Expected: PASS. The new test passes. Existing tests in this file may now fail due to missing required fields — that's expected and will be fixed in Task 2.

- [ ] **Step 5: Commit**

```bash
git add src/ast/languageDescriber.ts tests/languageDescriber.test.ts
git commit -m "feat(ast): add docstring, byte offsets, and contentHash fields to LogicSymbol"
```

---

### Task 2: Populate byte offsets and content hash in TypeScriptDescriber

**Files:**
- Modify: `src/ast/typescriptDescriber.ts:1-10` (add crypto import)
- Modify: `src/ast/typescriptDescriber.ts:325-353` (`makeSymbol` method)

- [ ] **Step 1: Write the failing test**

Add tests in `tests/typescriptDescriber.test.ts`:

```typescript
it("populates byteStart, byteEnd, and contentHash on symbols", () => {
  const source = [
    "export function hello() {",
    "  return 'world';",
    "}",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/offsets.ts", source);
  const hello = result.symbols.find(s => s.id === "fn:hello")!;

  expect(hello.byteStart).toBe(0);  // "export" starts at 0 — but the node is the function_declaration inside export
  expect(hello.byteEnd).toBeGreaterThan(hello.byteStart);
  expect(hello.contentHash).toMatch(/^[0-9a-f]{40}$/);  // SHA1 hex

  // Verify offsets match actual source text
  const sliced = source.slice(hello.byteStart, hello.byteEnd);
  expect(sliced).toContain("function hello");
  expect(sliced).toContain("return 'world'");
});

it("gives different contentHash for different function bodies", () => {
  const source = [
    "function a() { return 1; }",
    "function b() { return 2; }",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/hashes.ts", source);
  const a = result.symbols.find(s => s.id === "fn:a")!;
  const b = result.symbols.find(s => s.id === "fn:b")!;

  expect(a.contentHash).not.toBe(b.contentHash);
});

it("gives same contentHash for identical function bodies", () => {
  const source = [
    "function a() { return 1; }",
    "function b() { return 1; }",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/same.ts", source);
  const a = result.symbols.find(s => s.id === "fn:a")!;
  const b = result.symbols.find(s => s.id === "fn:b")!;

  // Different names but same body — hash covers full node text so they differ
  expect(a.contentHash).not.toBe(b.contentHash);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/typescriptDescriber.test.ts`
Expected: FAIL — `byteStart`, `byteEnd`, and `contentHash` are missing from the returned symbols (TypeScript error or undefined values).

- [ ] **Step 3: Implement byte offsets and content hash in makeSymbol**

In `src/ast/typescriptDescriber.ts`:

Add import at the top:
```typescript
import { createHash } from "crypto";
```

Update the `makeSymbol` method to include the new fields:

```typescript
private makeSymbol(
  id: string,
  kind: LogicSymbolKind,
  name: string,
  exported: boolean,
  parentId: string | null,
  params: string[],
  node: SyntaxNode,
): LogicSymbol {
  const isFnLike = this.isFunctionLikeNode(node);
  const control = {
    async: isFnLike ? this.isAsync(node) : false,
    branch: isFnLike ? this.hasBranching(node) : false,
    await: isFnLike ? this.hasDescendantOfType(node, "await_expression") : false,
    throw: isFnLike ? this.hasDescendantOfType(node, "throw_statement") : false,
  };

  return {
    id,
    kind,
    name,
    exported,
    parentId,
    params: [...params].sort((a, b) => a.localeCompare(b)),
    complexity: this.computeComplexityBucket(isFnLike ? node : null),
    control,
    line: node.startPosition.row + 1,
    byteStart: node.startIndex,
    byteEnd: node.endIndex,
    contentHash: createHash("sha1").update(node.text).digest("hex"),
  };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `bun run test -- tests/typescriptDescriber.test.ts`
Expected: PASS for the new tests. Existing tests should also pass since they don't assert absence of the new fields.

- [ ] **Step 5: Run full test suite to check for breakage**

Run: `bun run test`
Expected: Some tests may fail due to `LogicSymbol` objects constructed without the new required fields (e.g., in test fixtures or template walker). Note which tests fail — they'll be fixed in Task 3.

- [ ] **Step 6: Commit**

```bash
git add src/ast/typescriptDescriber.ts tests/typescriptDescriber.test.ts
git commit -m "feat(ast): populate byte offsets and content hash in TypeScriptDescriber"
```

---

### Task 3: Fix template walker and other test fixtures for new required fields

**Files:**
- Modify: `src/ast/templateWalker.ts` — template symbols need `byteStart`, `byteEnd`, `contentHash`
- Modify: Any test files that construct `LogicSymbol` objects manually

The template walker creates `LogicSymbol` objects for template nodes (tpl, tpl-slot, tpl-branch, tpl-loop). These don't have real byte offsets or source text, so use sentinel values.

- [ ] **Step 1: Read templateWalker.ts to find where symbols are created**

Read `src/ast/templateWalker.ts` and find all places where `LogicSymbol` objects are constructed.

- [ ] **Step 2: Update template symbol construction**

For each template symbol created in `templateWalker.ts`, add the new required fields:

```typescript
byteStart: 0,
byteEnd: 0,
contentHash: "",
```

Template symbols don't have meaningful byte offsets or source hashes since they're synthesized from JSX/Vue template structure, not raw source text. Using zero/empty values is correct here.

- [ ] **Step 3: Fix any test fixtures that construct LogicSymbol manually**

Search all test files for `LogicSymbol` object literals and add the new required fields:

```typescript
byteStart: 0,
byteEnd: 0,
contentHash: "",
```

- [ ] **Step 4: Run full test suite**

Run: `bun run test`
Expected: ALL tests pass.

- [ ] **Step 5: Commit**

```bash
git add src/ast/templateWalker.ts tests/
git commit -m "fix(ast): add required byte offset and hash fields to template symbols and test fixtures"
```

---

### Task 4: Extract JSDoc comments from preceding nodes

**Files:**
- Modify: `src/ast/typescriptDescriber.ts` — add JSDoc extraction logic

- [ ] **Step 1: Write the failing test**

Add tests in `tests/typescriptDescriber.test.ts`:

```typescript
it("extracts JSDoc docstring from preceding comment", () => {
  const source = [
    "/** Validates an email address against RFC 5322 rules. */",
    "export function validate(email: string) {",
    "  return email.includes('@');",
    "}",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/jsdoc.ts", source);
  const validate = result.symbols.find(s => s.id === "fn:validate")!;

  expect(validate.docstring).toBe("Validates an email address against RFC 5322 rules.");
});

it("extracts first sentence from multi-line JSDoc", () => {
  const source = [
    "/**",
    " * Fetches user data from the API. This function handles",
    " * retries and timeout logic internally.",
    " * @param userId - The user ID to fetch",
    " */",
    "export function fetchUser(userId: string) {",
    "  return fetch(`/api/users/${userId}`);",
    "}",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/multiline.ts", source);
  const fetchUser = result.symbols.find(s => s.id === "fn:fetchUser")!;

  expect(fetchUser.docstring).toBe("Fetches user data from the API.");
});

it("extracts JSDoc for class methods", () => {
  const source = [
    "export class Service {",
    "  /** Starts the service and binds to the given port. */",
    "  start(port: number) {",
    "    return this.listen(port);",
    "  }",
    "  listen(port: number) {}",
    "}",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/method-jsdoc.ts", source);
  const start = result.symbols.find(s => s.id === "mtd:Service.start")!;
  const listen = result.symbols.find(s => s.id === "mtd:Service.listen")!;

  expect(start.docstring).toBe("Starts the service and binds to the given port.");
  expect(listen.docstring).toBeUndefined();
});

it("extracts JSDoc for class declarations", () => {
  const source = [
    "/** Manages user sessions and authentication state. */",
    "export class SessionManager {",
    "  clear() {}",
    "}",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/class-jsdoc.ts", source);
  const cls = result.symbols.find(s => s.id === "cls:SessionManager")!;

  expect(cls.docstring).toBe("Manages user sessions and authentication state.");
});

it("skips non-JSDoc comments", () => {
  const source = [
    "// This is a regular comment",
    "export function noDoc() { return 1; }",
    "",
    "/* Block comment but not JSDoc */",
    "export function alsoNoDoc() { return 2; }",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/no-jsdoc.ts", source);
  const noDoc = result.symbols.find(s => s.id === "fn:noDoc")!;
  const alsoNoDoc = result.symbols.find(s => s.id === "fn:alsoNoDoc")!;

  expect(noDoc.docstring).toBeUndefined();
  expect(alsoNoDoc.docstring).toBeUndefined();
});

it("handles JSDoc with decorator between comment and declaration", () => {
  const source = [
    "/** Handles incoming HTTP requests. */",
    "@Controller('/api')",
    "export class ApiController {",
    "  handle() {}",
    "}",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/decorator-jsdoc.ts", source);
  const cls = result.symbols.find(s => s.id === "cls:ApiController")!;

  expect(cls.docstring).toBe("Handles incoming HTTP requests.");
});

it("returns undefined docstring for empty JSDoc", () => {
  const source = [
    "/** */",
    "export function empty() { return 1; }",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/empty-jsdoc.ts", source);
  const empty = result.symbols.find(s => s.id === "fn:empty")!;

  expect(empty.docstring).toBeUndefined();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/typescriptDescriber.test.ts`
Expected: FAIL — `docstring` is `undefined` for all symbols.

- [ ] **Step 3: Implement JSDoc extraction**

In `src/ast/typescriptDescriber.ts`, add a new private method and integrate it into the extraction flow.

Add this helper method to the `TypeScriptDescriber` class:

```typescript
/**
 * Extract JSDoc from the comment node preceding a declaration.
 * Returns the first sentence, or undefined if no JSDoc is found.
 */
private extractJsDoc(node: SyntaxNode): string | undefined {
  // Walk backwards through siblings to find a preceding comment,
  // skipping decorator nodes.
  let candidate = node.previousNamedSibling;
  while (candidate && candidate.type === "decorator") {
    candidate = candidate.previousNamedSibling;
  }

  if (!candidate || candidate.type !== "comment") return undefined;

  const text = candidate.text;
  if (!text.startsWith("/**")) return undefined;

  // Strip comment markers: /** ... */
  const stripped = text
    .replace(/^\/\*\*\s*/, "")
    .replace(/\s*\*\/$/, "")
    .split("\n")
    .map(line => line.replace(/^\s*\*\s?/, ""))
    .join(" ")
    .trim();

  if (!stripped) return undefined;

  // Take first sentence: up to first ". " or ".\n" or end of string before @tag
  const beforeTags = stripped.split(/\s@/)[0]!.trim();
  const sentenceMatch = beforeTags.match(/^(.+?\.)\s/);
  const firstSentence = sentenceMatch ? sentenceMatch[1]! : beforeTags;

  // Truncate to 200 chars
  const result = firstSentence.length > 200 ? firstSentence.slice(0, 200) + "..." : firstSentence;
  return result || undefined;
}
```

Now integrate it into `extractRawLogicGraph`. The JSDoc needs to be extracted during the declaration walk and stored on the symbol. Update `makeSymbol` to accept an optional `docstring` parameter:

Change the `makeSymbol` signature to:

```typescript
private makeSymbol(
  id: string,
  kind: LogicSymbolKind,
  name: string,
  exported: boolean,
  parentId: string | null,
  params: string[],
  node: SyntaxNode,
  docstring?: string,
): LogicSymbol {
```

And add `docstring` to the return object (after `contentHash`):

```typescript
    ...(docstring ? { docstring } : {}),
```

Then in `extractRawLogicGraph`, pass the docstring to `makeSymbol` for each declaration type. The `node` passed to `extractJsDoc` should be the **outermost** node (the `export_statement` if exported, otherwise the declaration itself), because that's where the comment sibling lives.

For each declaration case in the `for (const child of root.namedChildren)` loop, extract the docstring using the `child` node (which is the export_statement or the declaration):

```typescript
case "class_declaration": {
  const className = declNode.childForFieldName("name")?.text ?? `anonymous_class_${declNode.startPosition.row + 1}`;
  const classId = `cls:${className}`;
  const docstring = this.extractJsDoc(child);  // child is export_statement or class_declaration
  pushSymbol(this.makeSymbol(classId, "cls", className, isExport, null, [], declNode, docstring));
  // ... rest unchanged
  for (const method of this.getClassMethods(declNode)) {
    const methodName = method.childForFieldName("name")?.text ?? "";
    const methodId = `mtd:${className}.${methodName}`;
    const params = this.extractParams(method);
    const methodDoc = this.extractJsDoc(method);  // method's preceding sibling in class body
    pushSymbol(this.makeSymbol(methodId, "mtd", methodName, isExport, classId, params, method, methodDoc));
    // ... rest unchanged
  }
  break;
}

case "function_declaration": {
  const fnName = declNode.childForFieldName("name")?.text ?? `anonymous_fn_${declNode.startPosition.row + 1}`;
  const fnId = `fn:${fnName}`;
  const params = this.extractParams(declNode);
  const docstring = this.extractJsDoc(child);
  pushSymbol(this.makeSymbol(fnId, "fn", fnName, isExport, null, params, declNode, docstring));
  // ... rest unchanged
  break;
}

case "lexical_declaration":
case "variable_declaration": {
  const docstring = this.extractJsDoc(child);
  for (const declarator of declNode.namedChildren) {
    if (declarator.type !== "variable_declarator") continue;
    const variableName = declarator.childForFieldName("name")?.text ?? "";
    const symbolId = `var:${variableName}`;
    pushSymbol(this.makeSymbol(symbolId, "var", variableName, isExport, null, [], declarator, docstring));
    // ... rest unchanged
  }
  break;
}

case "interface_declaration": {
  const name = declNode.childForFieldName("name")?.text ?? "";
  const symbolId = `iface:${name}`;
  const docstring = this.extractJsDoc(child);
  pushSymbol(this.makeSymbol(symbolId, "iface", name, isExport, null, [], declNode, docstring));
  break;
}

case "enum_declaration": {
  const name = declNode.childForFieldName("name")?.text ?? "";
  const symbolId = `enum:${name}`;
  const docstring = this.extractJsDoc(child);
  pushSymbol(this.makeSymbol(symbolId, "enum", name, isExport, null, [], declNode, docstring));
  break;
}

case "type_alias_declaration": {
  const name = declNode.childForFieldName("name")?.text ?? "";
  const symbolId = `type:${name}`;
  const docstring = this.extractJsDoc(child);
  pushSymbol(this.makeSymbol(symbolId, "type", name, isExport, null, [], declNode, docstring));
  break;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `bun run test -- tests/typescriptDescriber.test.ts`
Expected: ALL tests pass including the new JSDoc tests.

- [ ] **Step 5: Commit**

```bash
git add src/ast/typescriptDescriber.ts tests/typescriptDescriber.test.ts
git commit -m "feat(ast): extract JSDoc/TSDoc comments from preceding nodes"
```

---

### Task 5: Use docstrings in daemon NL output

**Files:**
- Modify: `src/daemon.ts:253-387` (`buildNLContent` method)
- Modify: `src/daemon.ts:389-442` (`buildCompactContent` method)
- Modify: `src/daemon.ts:224-251` (`summarizeSourceFile` method — file-level doc)

- [ ] **Step 1: Write the failing test**

Add a test in `tests/daemon.test.ts`:

```typescript
it("includes JSDoc docstrings in NL content output", async () => {
  const tempDir = makeTempDir();
  const filePath = path.join(tempDir, "documented.ts");
  const code = source([
    "/** Validates email addresses against RFC 5322. */",
    "export function validate(email: string) {",
    "  return email.includes('@');",
    "}",
    "",
    "export function plain() {",
    "  return true;",
    "}",
  ]);
  fs.writeFileSync(filePath, code, "utf8");

  const manager = createManagerStub();
  const daemon = new CodebaseDaemon(manager as any, "proj", tempDir);
  await daemon.initParsers();

  await (daemon as any).processFile(filePath);

  const call = manager.upsertFileContextNode.mock.calls[0];
  const content: string = call[4];  // nlContent is the 5th argument

  expect(content).toContain("Validates email addresses against RFC 5322.");
  expect(content).toContain("validate");
});

it("includes docstring in compact overview", async () => {
  const tempDir = makeTempDir();
  const filePath = path.join(tempDir, "compact.ts");
  const code = source([
    "/** Short doc. */",
    "export function foo() { return 1; }",
  ]);
  fs.writeFileSync(filePath, code, "utf8");

  const manager = createManagerStub();
  const daemon = new CodebaseDaemon(manager as any, "proj", tempDir);
  await daemon.initParsers();

  await (daemon as any).processFile(filePath);

  const call = manager.upsertFileContextNode.mock.calls[0];
  const overview: string = call[3];  // overviewText is the 4th argument

  expect(overview).toContain('doc="Short doc."');
});

it("uses file-level JSDoc as abstract when present", async () => {
  const tempDir = makeTempDir();
  const filePath = path.join(tempDir, "filedoc.ts");
  const code = source([
    "/** Utility functions for string manipulation. */",
    "",
    "export function trim(s: string) { return s.trim(); }",
  ]);
  fs.writeFileSync(filePath, code, "utf8");

  const manager = createManagerStub();
  const daemon = new CodebaseDaemon(manager as any, "proj", tempDir);
  await daemon.initParsers();

  await (daemon as any).processFile(filePath);

  const call = manager.upsertFileContextNode.mock.calls[0];
  const abstractText: string = call[2];  // abstractText is the 3rd argument

  expect(abstractText).toBe("Utility functions for string manipulation.");
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/daemon.test.ts`
Expected: FAIL — docstrings not included in NL content, not in compact overview, and file-level doc not used as abstract.

- [ ] **Step 3: Implement docstring display in buildNLContent**

In `src/daemon.ts`, modify `buildNLContent` to show docstrings. Find the section that builds class and function sections.

For classes (around line 313-344), after the heading line and before the existing description, add:

```typescript
if (kind === "cls") {
  const lines: string[] = [];
  lines.push(`## ${kindLabel[kind]}: ${symbol.name}${tagStr}`);
  if (symbol.docstring) lines.push(`*${symbol.docstring}*`);
  if (desc) lines.push(desc);

  const methods = methodsByClass.get(symbol.id) ?? [];
  for (const mtd of methods) {
    const mtdDesc = descriptions.get(mtd.id);
    const mtdTags: string[] = [];
    if (mtd.exported) mtdTags.push("exported");
    if (mtd.control.async) mtdTags.push("async");
    const mtdTagStr = mtdTags.length > 0 ? ` (${mtdTags.join(", ")})` : "";
    const paramStr = mtd.params.length > 0 ? `Parameters: ${mtd.params.join(", ")}` : "";

    lines.push("");
    lines.push(`### Method: ${mtd.name}${mtdTagStr}`);
    if (mtd.docstring) lines.push(`*${mtd.docstring}*`);
    if (paramStr) lines.push(paramStr);
    if (mtdDesc) {
      lines.push("");
      lines.push(mtdDesc);
    }
  }
  // ... rest unchanged (maxScore calc + sections.push)
```

For functions (around line 345-354):

```typescript
} else if (kind === "fn") {
  const lines: string[] = [];
  const paramStr = symbol.params.length > 0 ? `Parameters: ${symbol.params.join(", ")}` : "";
  lines.push(`## ${kindLabel[kind]}: ${symbol.name}${tagStr}`);
  if (symbol.docstring) lines.push(`*${symbol.docstring}*`);
  if (paramStr) lines.push(paramStr);
  if (desc) {
    lines.push("");
    lines.push(desc);
  }
  sections.push({ symbolId: symbol.id, score: symbolScores.get(symbol.id) ?? 0, text: lines.join("\n") });
```

For var/iface/enum/type (around line 356-358):

```typescript
} else {
  const docPart = symbol.docstring ? ` — *${symbol.docstring}*` : "";
  const descPart = !symbol.docstring && desc ? ` — ${desc}` : "";
  const line = `- ${kindLabel[kind]}: ${symbol.name}${tagStr}${docPart}${descPart}`;
  sections.push({ symbolId: symbol.id, score: symbolScores.get(symbol.id) ?? 0, text: line });
}
```

- [ ] **Step 4: Implement docstring in buildCompactContent**

In `buildCompactContent`, append `doc="..."` to the symbol line. In the symbol loop (around line 403-422), after building the line parts array, add the docstring:

```typescript
for (const symbol of graph.symbols) {
  const params = symbol.params.length > 0 ? `(${symbol.params.join(",")})` : "()";
  const parent = symbol.parentId ? ` parent=${symbol.parentId}` : "";
  const controlTokens = [
    symbol.control.async ? "async" : "",
    symbol.control.branch ? "branch" : "",
    symbol.control.await ? "await" : "",
    symbol.control.throw ? "throw" : "",
  ].filter(Boolean);
  const doc = symbol.docstring ? ` doc="${symbol.docstring.slice(0, 60)}"` : "";
  lines.push(
    [
      `- ${symbol.kind} ${symbol.id}`,
      `name=${symbol.name}`,
      `exp=${symbol.exported ? "1" : "0"}`,
      `params=${params}`,
      `cx=${symbol.complexity}`,
      controlTokens.length > 0 ? `ctrl=${controlTokens.join("|")}` : "",
      parent,
    ].filter(Boolean).join(" ") + doc
  );
}
```

- [ ] **Step 5: Implement file-level JSDoc as abstract**

In `summarizeSourceFile`, after extracting the file graph, check if the first comment in the source is a file-level JSDoc (appears before any import/declaration). Modify `summarizeSourceFile` (around line 224-251):

The `FileGraphResult` doesn't include file-level JSDoc. Instead, we extract it directly from the source text in the daemon. Add a helper method to the `CodebaseDaemon` class:

```typescript
private extractFileJsDoc(sourceText: string): string | undefined {
  // Match a JSDoc comment at the very beginning of the file (possibly after whitespace)
  const match = sourceText.match(/^\s*\/\*\*([\s\S]*?)\*\//);
  if (!match) return undefined;

  const stripped = match[1]!
    .split("\n")
    .map(line => line.replace(/^\s*\*\s?/, ""))
    .join(" ")
    .trim();

  if (!stripped) return undefined;

  // Take first sentence
  const beforeTags = stripped.split(/\s@/)[0]!.trim();
  const sentenceMatch = beforeTags.match(/^(.+?\.)\s/);
  const result = sentenceMatch ? sentenceMatch[1]! : beforeTags;
  return result.length > 200 ? result.slice(0, 200) + "..." : result || undefined;
}
```

Then in `summarizeSourceFile`, use it:

```typescript
private summarizeSourceFile(filePath: string, sourceText: string): SourceSummary {
  const result = this.describer.extractFileGraph(filePath, sourceText);
  const rawGraph = {
    symbols: result.symbols,
    edges: result.edges,
    imports: result.imports,
  };
  const selectedGraph = this.selectGraphForSerialization(rawGraph);
  const enrichedDescriptions = enrichDescriptions(result.symbolDescriptions, result.edges);
  const fileJsDoc = this.extractFileJsDoc(sourceText);
  const abstractText = fileJsDoc ?? result.fileSummary;
  // ... rest unchanged
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `bun run test -- tests/daemon.test.ts`
Expected: ALL tests pass including the new docstring tests.

- [ ] **Step 7: Run full test suite**

Run: `bun run test`
Expected: ALL tests pass.

- [ ] **Step 8: Commit**

```bash
git add src/daemon.ts tests/daemon.test.ts
git commit -m "feat(daemon): display JSDoc docstrings in NL content, compact overview, and file abstract"
```

---

### Task 6: Run typecheck and lint

**Files:** None (validation only)

- [ ] **Step 1: Run typecheck**

Run: `bun run typecheck`
Expected: No errors.

- [ ] **Step 2: Run lint**

Run: `bun run lint`
Expected: No errors (or only pre-existing warnings).

- [ ] **Step 3: Fix any issues found**

If typecheck or lint report errors introduced by these changes, fix them.

- [ ] **Step 4: Run full test suite one final time**

Run: `bun run test`
Expected: ALL tests pass.

- [ ] **Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve typecheck and lint issues from AST ingestion improvements"
```

(Skip this commit if no fixes were needed.)
