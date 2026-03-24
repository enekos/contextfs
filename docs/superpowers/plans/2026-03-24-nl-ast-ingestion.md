# NL AST Ingestion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace compact machine-readable AST notation with human-readable natural language descriptions at statement-level depth, add parallel file processing, and add persistent hash caching for daemon restarts.

**Architecture:** Single-pass AST walker behind a pluggable `LanguageDescriber` interface extracts symbols + edges + NL descriptions. Post-enrichment pass stitches cross-function references. Content fields remapped: abstract=NL summary, overview=compact graph, content=full NL AST.

**Tech Stack:** TypeScript, ts-morph (existing), Vitest (existing)

---

### Task 1: Define LanguageDescriber Interface and Types

**Files:**
- Create: `src/languageDescriber.ts`
- Modify: `src/daemon.ts:37-85` (move shared types out)

- [ ] **Step 1: Write the failing test**

Create test that imports the interface types and the TypeScript describer:

```typescript
// tests/languageDescriber.test.ts
import { describe, expect, it } from "vitest";
import type { LanguageDescriber, FileGraphResult } from "../src/languageDescriber";
import { TypeScriptDescriber } from "../src/typescriptDescriber";

describe("LanguageDescriber interface", () => {
  it("TypeScriptDescriber implements LanguageDescriber", () => {
    const describer: LanguageDescriber = new TypeScriptDescriber();
    expect(describer.languageId).toBe("typescript");
    expect(describer.extensions).toContain(".ts");
    expect(describer.extensions).toContain(".tsx");
    expect(describer.extensions).toContain(".js");
    expect(describer.extensions).toContain(".jsx");
    expect(describer.extensions).toContain(".mjs");
    expect(describer.extensions).toContain(".cjs");
    expect(typeof describer.extractFileGraph).toBe("function");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/languageDescriber.test.ts`
Expected: FAIL — modules don't exist yet

- [ ] **Step 3: Create the interface file**

```typescript
// src/languageDescriber.ts
export type LogicSymbolKind = "cls" | "fn" | "mtd" | "var" | "iface" | "enum" | "type";
export type LogicEdgeKind = "call" | "import" | "read" | "write" | "extends" | "implements";
export type ComplexityBucket = "low" | "medium" | "high";

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
}

export interface LogicEdge {
  kind: LogicEdgeKind;
  from: string;
  to: string;
}

export interface FileGraphResult {
  symbols: LogicSymbol[];
  edges: LogicEdge[];
  imports: string[];
  /** Per-symbol NL descriptions keyed by symbol ID */
  symbolDescriptions: Map<string, string>;
  /** File-level NL summary */
  fileSummary: string;
}

export interface LanguageDescriber {
  readonly languageId: string;
  readonly extensions: ReadonlySet<string>;
  extractFileGraph(filePath: string, sourceText: string): FileGraphResult;
}
```

- [ ] **Step 4: Create a minimal TypeScriptDescriber stub**

```typescript
// src/typescriptDescriber.ts
import type { LanguageDescriber, FileGraphResult } from "./languageDescriber";

export class TypeScriptDescriber implements LanguageDescriber {
  readonly languageId = "typescript";
  readonly extensions: ReadonlySet<string> = new Set([".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"]);

  extractFileGraph(_filePath: string, _sourceText: string): FileGraphResult {
    return {
      symbols: [],
      edges: [],
      imports: [],
      symbolDescriptions: new Map(),
      fileSummary: "",
    };
  }
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `bun run test -- tests/languageDescriber.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add src/languageDescriber.ts src/typescriptDescriber.ts tests/languageDescriber.test.ts
git commit -m "feat: add LanguageDescriber interface and TypeScriptDescriber stub"
```

---

### Task 2: Move AST Extraction Logic into TypeScriptDescriber

**Files:**
- Modify: `src/typescriptDescriber.ts` (add all extraction logic)
- Modify: `src/daemon.ts` (remove extraction methods, import from describer)

This task migrates all existing logic graph extraction from `CodebaseDaemon` into `TypeScriptDescriber.extractFileGraph()`. The ts-morph `Project` instance moves into the describer — the daemon no longer owns a shared `Project`. File discovery switches from `tsProject.addSourceFilesAtPaths` to a recursive `readdirSync` walk filtered by extension. No behavior change yet — same compact graph output, just relocated.

**Breaking change note:** The `contentFormat: "compact" | "full"` option is removed. The "full" mode (raw source as content) is replaced by the new NL layout. This is intentional — the NL content is strictly superior to raw source for search and retrieval. Any code using `contentFormat: "full"` should migrate to the new layout.

- [ ] **Step 1: Write the failing test**

```typescript
// tests/typescriptDescriber.test.ts
import { describe, expect, it } from "vitest";
import { TypeScriptDescriber } from "../src/typescriptDescriber";

describe("TypeScriptDescriber", () => {
  const describer = new TypeScriptDescriber();

  it("extracts symbols and edges from TypeScript source", () => {
    const source = [
      "import { slugify } from './slug';",
      "const INTERNAL_SEED = 42;",
      "",
      "function normalize(name: string) {",
      "  return slugify(`${name}-${INTERNAL_SEED}`);",
      "}",
      "",
      "export function greet(name: string) {",
      "  return normalize(name);",
      "}",
      "",
      "export class UserService {",
      "  public run(input: string) {",
      "    this.bump();",
      "    return greet(input);",
      "  }",
      "  private bump() {",
      "    return normalize('x');",
      "  }",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/feature.ts", source);

    // Symbols
    const symbolIds = result.symbols.map(s => s.id);
    expect(symbolIds).toContain("fn:greet");
    expect(symbolIds).toContain("fn:normalize");
    expect(symbolIds).toContain("cls:UserService");
    expect(symbolIds).toContain("mtd:UserService.run");
    expect(symbolIds).toContain("mtd:UserService.bump");
    expect(symbolIds).toContain("var:INTERNAL_SEED");

    // Edges
    const edgeKeys = result.edges.map(e => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("call:fn:greet->fn:normalize");
    expect(edgeKeys).toContain("call:mtd:UserService.run->mtd:UserService.bump");
    expect(edgeKeys).toContain("call:mtd:UserService.run->fn:greet");
    expect(edgeKeys).toContain("import:file->module:./slug");

    // Imports
    expect(result.imports).toContain("./slug");
  });

  it("extracts symbols from empty file", () => {
    const result = describer.extractFileGraph("/tmp/test/empty.ts", "/* empty */");
    expect(result.symbols).toHaveLength(0);
    expect(result.edges).toHaveLength(0);
    // In Task 2, fileSummary is a basic fallback; Task 4 adds richer summaries
    expect(typeof result.fileSummary).toBe("string");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/typescriptDescriber.test.ts`
Expected: FAIL — stub returns empty arrays

- [ ] **Step 3: Migrate extraction logic into TypeScriptDescriber**

Move all of these methods from `CodebaseDaemon` into `TypeScriptDescriber`:
- `extractRawLogicGraph(sourceFile)` → internal method
- `makeSymbol(...)` → internal method
- `asFunctionLikeNode(node)` → internal method
- `isAsyncFunctionLike(node)` → internal method
- `hasBranching(node)` → internal method
- `computeComplexityBucket(node)` → internal method
- `resolveCallTarget(...)` → internal method
- `pickBestCallableSymbolId(...)` → internal method
- `isDeclarationIdentifier(identifier)` → internal method
- `isWriteIdentifier(identifier)` → internal method
- `sortSymbols(symbols)` → internal method
- `compareSymbols(a, b)` → internal method
- `sortEdges(edges)` → internal method

The `extractFileGraph` method:
1. Creates a ts-morph `Project`, adds the source text as an in-memory source file
2. Calls `extractRawLogicGraph`
3. Returns the raw graph with empty `symbolDescriptions` and a basic `fileSummary` (to be filled in Task 4)

Important: use `project.createSourceFile(filePath, sourceText)` with an in-memory project (no fsHost needed) so each call is isolated and thread-safe for parallel processing.

- [ ] **Step 4: Run test to verify it passes**

Run: `bun run test -- tests/typescriptDescriber.test.ts`
Expected: PASS

- [ ] **Step 5: Update daemon.ts to use TypeScriptDescriber**

Remove the extracted methods from `CodebaseDaemon`. Import `TypeScriptDescriber` and use it in `summarizeSourceFile`:
- **Remove `this.tsProject` field entirely** — no longer needed at daemon level
- Remove `clearSourceFileFromProject` and `refreshFromFileSystem` calls from `processFile`
- Add `private readonly describer: TypeScriptDescriber`
- In `summarizeSourceFile`, accept raw source text, call `describer.extractFileGraph(filePath, sourceText)`, then feed result into `selectGraphForSerialization` and serializers
- **Replace file discovery**: replace `this.tsProject.addSourceFilesAtPaths` + `this.tsProject.getSourceFiles()` in `processAllFiles` with a recursive `readdirSync` walk:

```typescript
private discoverSourceFiles(dir: string): string[] {
  const results: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (!IGNORED_PATH_SEGMENTS.has(entry.name) && !entry.name.startsWith(".")) {
        results.push(...this.discoverSourceFiles(fullPath));
      }
    } else if (SUPPORTED_EXTENSIONS.has(path.extname(entry.name).toLowerCase())) {
      results.push(fullPath);
    }
  }
  return results;
}
```

- Remove `getSourceGlobs()` method
- Remove the `ContentFormat` type and `contentFormat` option from `DaemonOptions` — the new NL layout replaces both compact and full modes

Also remove the type definitions that moved to `languageDescriber.ts` (`LogicSymbolKind`, `LogicEdgeKind`, `ComplexityBucket`, `LogicSymbol`, `LogicEdge`, `RawLogicGraph`, `CallableSymbolRef`) — import them instead.

Keep in `daemon.ts`: `SelectedLogicGraph`, `SourceSummary`, serialization methods (`buildCompactContent`, `buildAbstractText`, `buildOverviewText`, `selectGraphForSerialization`, `symbolScore`), file processing, caching, and watcher logic.

- [ ] **Step 6: Run ALL existing daemon tests to verify no regression**

Run: `bun run test -- tests/daemon.test.ts`
Expected: ALL PASS — behavior is identical, just code moved

- [ ] **Step 7: Commit**

```bash
git add src/typescriptDescriber.ts src/languageDescriber.ts src/daemon.ts tests/typescriptDescriber.test.ts
git commit -m "refactor: extract AST logic into TypeScriptDescriber behind LanguageDescriber interface"
```

---

### Task 3: NL Statement Describer Engine

**Files:**
- Create: `src/nlDescriber.ts`
- Create: `tests/nlDescriber.test.ts`

This is the core NL generation engine. It takes a ts-morph AST node (function/method body) and produces a numbered list of English sentences describing what each statement does.

- [ ] **Step 1: Write failing tests for basic NL patterns**

```typescript
// tests/nlDescriber.test.ts
import { describe, expect, it } from "vitest";
import { describeStatements } from "../src/nlDescriber";
import { Project } from "ts-morph";

function describeFunction(source: string, fnName: string): string {
  const project = new Project({ compilerOptions: { allowJs: true } });
  const sf = project.createSourceFile("/tmp/test.ts", source);
  const fn = sf.getFunction(fnName);
  if (!fn) throw new Error(`Function ${fnName} not found`);
  return describeStatements(fn);
}

describe("nlDescriber", () => {
  it("describes a simple return statement", () => {
    const nl = describeFunction(
      `function greet(name: string) { return "Hello " + name; }`,
      "greet"
    );
    expect(nl).toContain("Returns");
    expect(nl).toContain("name");
  });

  it("describes if/else branches", () => {
    const nl = describeFunction(
      `function check(x: number) {
        if (x > 0) {
          return "positive";
        } else {
          return "non-positive";
        }
      }`,
      "check"
    );
    expect(nl).toContain("x");
    expect(nl).toMatch(/[Ii]f/);
    expect(nl).toMatch(/[Oo]therwise/);
  });

  it("describes for-of loops", () => {
    const nl = describeFunction(
      `function process(items: string[]) {
        for (const item of items) {
          console.log(item);
        }
      }`,
      "process"
    );
    expect(nl).toMatch(/[Ii]terates|[Ll]oops|[Ff]or each/);
    expect(nl).toContain("item");
    expect(nl).toContain("items");
  });

  it("describes try/catch blocks", () => {
    const nl = describeFunction(
      `function safeParse(json: string) {
        try {
          return JSON.parse(json);
        } catch (e) {
          return null;
        }
      }`,
      "safeParse"
    );
    expect(nl).toMatch(/[Aa]ttempts|[Tt]ries/);
    expect(nl).toMatch(/[Ee]rror|fails/);
  });

  it("describes throw statements", () => {
    const nl = describeFunction(
      `function validate(x: any) {
        if (!x) {
          throw new Error("Required");
        }
      }`,
      "validate"
    );
    expect(nl).toMatch(/[Tt]hrows/);
    expect(nl).toContain("Error");
  });

  it("describes variable assignments with function calls", () => {
    const nl = describeFunction(
      `function transform(input: string) {
        const trimmed = input.trim();
        const lower = trimmed.toLowerCase();
        return lower;
      }`,
      "transform"
    );
    expect(nl).toContain("trimmed");
    expect(nl).toContain("trim");
    expect(nl).toContain("lower");
  });

  it("describes await expressions", () => {
    const nl = describeFunction(
      `async function fetchData(url: string) {
        const response = await fetch(url);
        const data = await response.json();
        return data;
      }`,
      "fetchData"
    );
    expect(nl).toMatch(/[Aa]waits/);
    expect(nl).toContain("fetch");
  });

  it("describes switch statements", () => {
    const nl = describeFunction(
      `function classify(status: number) {
        switch (status) {
          case 200: return "ok";
          case 404: return "not found";
          default: return "unknown";
        }
      }`,
      "classify"
    );
    expect(nl).toMatch(/[Ss]witch|[Bb]ased on/);
    expect(nl).toContain("status");
  });

  it("describes while loops", () => {
    const nl = describeFunction(
      `function countdown(n: number) {
        while (n > 0) {
          n--;
        }
      }`,
      "countdown"
    );
    expect(nl).toMatch(/[Ww]hile|[Ll]oops/);
  });

  it("translates common conditions to natural English", () => {
    const nl = describeFunction(
      `function check(x: any, items: string[]) {
        if (x === null) { return "a"; }
        if (!x) { return "b"; }
        if (typeof x === "string") { return "c"; }
        if (items.length > 0) { return "d"; }
        if (x instanceof Error) { return "e"; }
      }`,
      "check"
    );
    expect(nl).toMatch(/`x` is null/);
    expect(nl).toMatch(/`x` is falsy/);
    expect(nl).toMatch(/`x` is a string/);
    expect(nl).toMatch(/`items` is non-empty|`items\.length` is greater than 0/);
    expect(nl).toMatch(/`x` is an instance of `Error`/);
  });

  it("describes nested if inside loop", () => {
    const nl = describeFunction(
      `function filterPositive(nums: number[]) {
        const result: number[] = [];
        for (const n of nums) {
          if (n > 0) {
            result.push(n);
          }
        }
        return result;
      }`,
      "filterPositive"
    );
    expect(nl).toMatch(/[Ii]terates|[Ff]or each/);
    expect(nl).toMatch(/[Ii]f/);
    expect(nl).toContain("result");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/nlDescriber.test.ts`
Expected: FAIL — module doesn't exist

- [ ] **Step 3: Implement nlDescriber.ts**

```typescript
// src/nlDescriber.ts
import { Node, SyntaxKind, type FunctionDeclaration, type MethodDeclaration } from "ts-morph";
```

Implement `describeStatements(node: FunctionDeclaration | MethodDeclaration): string` that:

1. Gets the function body's direct child statements
2. For each statement, pattern-matches the AST node kind and generates English:
   - **VariableStatement**: "Assigns {initializer description} to `{name}`" or "Declares `{name}` as {initializer}"
   - **ReturnStatement**: "Returns {expression description}"
   - **IfStatement**: "If {condition in English}, {then body}. Otherwise, {else body}"
   - **ForOfStatement**: "Iterates over each `{binding}` in `{expression}`, {loop body}"
   - **ForInStatement**: "Iterates over each key `{binding}` in `{expression}`, {loop body}"
   - **ForStatement**: "Loops with {initializer}; while {condition}; {incrementor}: {loop body}"
   - **WhileStatement**: "Loops while {condition}: {loop body}"
   - **DoStatement**: "Loops (do-while {condition}): {loop body}"
   - **TryStatement**: "Attempts to {try body}. If an error occurs ({catch param}), {catch body}. {finally if present}"
   - **ThrowStatement**: "Throws {expression}" — detect `new X(msg)` pattern for "Throws a {X} with message {msg}"
   - **SwitchStatement**: "Based on `{discriminant}`: {case val}: {body}, ... {default}: {body}"
   - **ExpressionStatement**: Describe the expression — call expressions get "Calls `{fn}` with {args}", await gets "Awaits {expr}", assignment gets "Sets `{target}` to {value}"
3. Numbers each statement: "1. ...", "2. ...", etc.
4. Recursion: nested statements in if/for/while/try bodies are described inline with indentation, not numbered separately at the top level

Also implement helper: `describeCondition(node: Node): string` that translates common patterns:
- `x === null` or `x === undefined` or `x == null` → "`x` is null/undefined"
- `!x` → "`x` is falsy"
- `x > 0`, `x < 0`, `x >= n`, etc. → "`x` is greater than 0" etc.
- `typeof x === "string"` → "`x` is a string"
- `x.length > 0` → "`x` is non-empty"
- `x instanceof Y` → "`x` is an instance of `Y`"
- Fallback: use the raw expression text with backtick wrapping

Also implement helper: `describeExpression(node: Node): string` that handles:
- Call expressions: "`fn(args)`" or "calling `obj.method` with {args}"
- Property access: "`obj.prop`"
- Await expressions: "awaiting {inner}"
- Binary expressions: "{left} {op in English} {right}"
- New expressions: "a new `{ClassName}`"
- Template literals: "a template string with {interpolated values}"
- Fallback: raw expression text in backticks

- [ ] **Step 4: Run tests to verify they pass**

Run: `bun run test -- tests/nlDescriber.test.ts`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add src/nlDescriber.ts tests/nlDescriber.test.ts
git commit -m "feat: add NL statement describer engine with AST pattern matching"
```

---

### Task 4: Generate Per-Symbol NL Descriptions in TypeScriptDescriber

**Files:**
- Modify: `src/typescriptDescriber.ts` (wire nlDescriber into extractFileGraph)
- Modify: `tests/typescriptDescriber.test.ts` (add NL description assertions)

- [ ] **Step 1: Write the failing test**

Add to `tests/typescriptDescriber.test.ts`:

```typescript
it("generates NL descriptions for each function/method symbol", () => {
  const source = [
    "export function greet(name: string) {",
    "  const trimmed = name.trim();",
    "  if (!trimmed) {",
    "    throw new Error('Name required');",
    "  }",
    "  return `Hello ${trimmed}`;",
    "}",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/greet.ts", source);

  const greetDesc = result.symbolDescriptions.get("fn:greet");
  expect(greetDesc).toBeDefined();
  expect(greetDesc).toContain("trimmed");
  expect(greetDesc).toMatch(/[Tt]hrows/);
  expect(greetDesc).toMatch(/[Rr]eturns/);
});

it("generates NL descriptions for class methods", () => {
  const source = [
    "export class Calculator {",
    "  add(a: number, b: number) {",
    "    return a + b;",
    "  }",
    "  divide(a: number, b: number) {",
    "    if (b === 0) {",
    "      throw new Error('Division by zero');",
    "    }",
    "    return a / b;",
    "  }",
    "}",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/calc.ts", source);

  expect(result.symbolDescriptions.get("mtd:Calculator.add")).toBeDefined();
  const divideDesc = result.symbolDescriptions.get("mtd:Calculator.divide");
  expect(divideDesc).toBeDefined();
  expect(divideDesc).toMatch(/[Dd]ivision|zero/);
});

it("generates a file summary", () => {
  const source = [
    "export function greet(name: string) { return 'Hello ' + name; }",
    "export function farewell(name: string) { return 'Bye ' + name; }",
  ].join("\n");

  const result = describer.extractFileGraph("/tmp/test/greetings.ts", source);
  expect(result.fileSummary).toBeTruthy();
  expect(result.fileSummary).toMatch(/greet|farewell/i);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/typescriptDescriber.test.ts`
Expected: FAIL — symbolDescriptions is empty Map

- [ ] **Step 3: Wire nlDescriber into TypeScriptDescriber**

In `TypeScriptDescriber.extractFileGraph`:
1. After extracting the raw graph, iterate over function/method symbols
2. For each, find the corresponding ts-morph node and call `describeStatements(node)`
3. Store in the `symbolDescriptions` Map keyed by symbol ID
4. For classes, generate a summary: "Class `{name}` {extends X} with methods: {method list}"
5. For the file summary: "File containing {N} exported symbols: {names}. {brief purpose based on symbol names and types}"

- [ ] **Step 4: Run tests to verify they pass**

Run: `bun run test -- tests/typescriptDescriber.test.ts`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add src/typescriptDescriber.ts tests/typescriptDescriber.test.ts
git commit -m "feat: generate per-symbol NL descriptions in TypeScriptDescriber"
```

---

### Task 5: Post-Enrichment Pass for Cross-References

**Files:**
- Create: `src/nlEnricher.ts`
- Create: `tests/nlEnricher.test.ts`

- [ ] **Step 1: Write the failing test**

```typescript
// tests/nlEnricher.test.ts
import { describe, expect, it } from "vitest";
import { enrichDescriptions } from "../src/nlEnricher";
import type { LogicEdge } from "../src/languageDescriber";

describe("enrichDescriptions", () => {
  it("enriches call references with callee context", () => {
    const descriptions = new Map<string, string>([
      ["fn:process", "1. Calls `validate` with `input`.\n2. Returns the validated result."],
      ["fn:validate", "1. If `input` is falsy, throws an Error with message \"Required\".\n2. Returns `input` trimmed."],
    ]);
    const edges: LogicEdge[] = [
      { kind: "call", from: "fn:process", to: "fn:validate" },
    ];

    const enriched = enrichDescriptions(descriptions, edges);

    const processDesc = enriched.get("fn:process")!;
    expect(processDesc).toContain("validate");
    // Should contain enrichment from validate's description
    expect(processDesc).toMatch(/falsy|Required|trimmed/);
  });

  it("does not recurse beyond depth 1", () => {
    const descriptions = new Map<string, string>([
      ["fn:a", "1. Calls `b`."],
      ["fn:b", "1. Calls `c`."],
      ["fn:c", "1. Returns 42."],
    ]);
    const edges: LogicEdge[] = [
      { kind: "call", from: "fn:a", to: "fn:b" },
      { kind: "call", from: "fn:b", to: "fn:c" },
    ];

    const enriched = enrichDescriptions(descriptions, edges);
    // fn:a should mention b's behavior but NOT c's
    const aDesc = enriched.get("fn:a")!;
    expect(aDesc).toMatch(/[Cc]alls.*b/);
  });

  it("handles symbols with no call edges unchanged", () => {
    const descriptions = new Map<string, string>([
      ["fn:simple", "1. Returns 42."],
    ]);
    const enriched = enrichDescriptions(descriptions, []);
    expect(enriched.get("fn:simple")).toBe("1. Returns 42.");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/nlEnricher.test.ts`
Expected: FAIL — module doesn't exist

- [ ] **Step 3: Implement nlEnricher.ts**

```typescript
// src/nlEnricher.ts
import type { LogicEdge } from "./languageDescriber";

export function enrichDescriptions(
  descriptions: Map<string, string>,
  edges: LogicEdge[]
): Map<string, string> { ... }
```

Logic:
1. Build a map of call edges: `callerSymbolId → Set<calleeSymbolId>`
2. For each description, find call edges originating from that symbol
3. For each callee that has a description, extract the first sentence as a summary
4. Append the callee summary as a parenthetical after mentions of the callee name in the caller's description: e.g., "Calls `validate` (which checks if input is falsy and throws if required)"
5. Only enrich depth 1 — don't recursively expand callee descriptions

- [ ] **Step 4: Run tests to verify they pass**

Run: `bun run test -- tests/nlEnricher.test.ts`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add src/nlEnricher.ts tests/nlEnricher.test.ts
git commit -m "feat: add post-enrichment pass for cross-function NL references"
```

---

### Task 6: Build Full NL Content and Remap Content Fields in Daemon

**Files:**
- Modify: `src/daemon.ts` (remap abstract/overview/content, build NL content from descriptions)
- Modify: `tests/daemon.test.ts` (update assertions for new field layout)

This is where the daemon's output changes. The new layout:
- `abstract` = NL file summary (from `FileGraphResult.fileSummary`)
- `overview` = compact graph notation (current `buildCompactContent` output)
- `content` = full NL AST (assembled from enriched `symbolDescriptions`)

- [ ] **Step 1: Write the failing test**

Add new test to `tests/daemon.test.ts`:

```typescript
it("produces NL content with abstract summary, compact overview, and NL body", async () => {
  const tempDir = makeTempDir();
  const filePath = path.join(tempDir, "service.ts");
  const code = source([
    "export function validate(input: string) {",
    "  if (!input) {",
    "    throw new Error('Required');",
    "  }",
    "  return input.trim();",
    "}",
    "",
    "export function process(data: string) {",
    "  const clean = validate(data);",
    "  return clean.toUpperCase();",
    "}",
  ]);
  fs.writeFileSync(filePath, code, "utf8");

  const manager = createManagerStub();
  const daemon = new CodebaseDaemon(manager as any, "proj", tempDir);
  await (daemon as any).processFile(filePath);

  expect(manager.upsertFileContextNode).toHaveBeenCalledTimes(1);
  const [, , abstractText, overviewText, content] = manager.upsertFileContextNode.mock.calls[0];

  // abstract = NL file summary
  expect(abstractText).toBeTruthy();
  expect(abstractText).toMatch(/validate|process/i);

  // overview = compact graph notation
  expect(overviewText).toContain("Symbols:");
  expect(overviewText).toContain("Edges:");
  expect(overviewText).toContain("fn fn:validate");

  // content = NL descriptions
  expect(content).toMatch(/validate/i);
  expect(content).toMatch(/[Tt]hrows|[Rr]eturns/);
  expect(content).toMatch(/process/i);
  expect(content).toMatch(/[Cc]alls.*validate/);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/daemon.test.ts`
Expected: FAIL — abstract is still empty, overview is empty, content is compact notation

- [ ] **Step 3: Update daemon.ts to remap content fields**

Modify `summarizeSourceFile` to:
1. Call `this.describer.extractFileGraph(filePath, sourceText)` (sourceText passed from processFile)
2. Run `selectGraphForSerialization` on the graph result
3. Run `enrichDescriptions` on the symbolDescriptions + edges
4. Build:
   - `abstractText` = `result.fileSummary`
   - `overviewText` = `buildCompactContent(filePath, selectedGraph)` (the old content)
   - `content` = `buildNLContent(enrichedDescriptions, selectedGraph)` — new method

Implement `buildNLContent(descriptions: Map<string, string>, graph: SelectedLogicGraph): string`:
1. Group symbols by kind (classes first, then functions, then vars, etc.)
2. For each symbol with a description, output a section:
   ```
   ## Function: greet (exported, async)
   Parameters: name (string)

   1. Trims the name by calling trim().
   2. Returns the trimmed name.
   ```
3. For classes, nest methods under the class heading
4. Symbols without descriptions (vars, interfaces, enums, types) get a one-line mention
5. Enforce `MAX_CONTENT_CHARS` truncation on the NL content — **drop entire symbol sections** from the end (least important symbols first, using the existing `symbolScore` ranking) rather than slicing mid-sentence. Append a `...TRUNCATED: {N} symbols omitted` footer when truncation occurs

Update `processFile` to pass `rawContent` (already read) into `summarizeSourceFile`.

Remove the `compactMode` branching — the new layout always populates all three fields.

Update metadata to always include `logic_graph` (it was previously omitted in compact mode).

- [ ] **Step 4: Update existing daemon tests**

Update all assertions in existing tests to match the new field mapping. Here is the updated first test as a reference for the pattern:

```typescript
it("stores NL content with abstract summary, compact overview, and NL body", async () => {
  // ... same setup as before ...
  const [uri, name, abstractText, overviewText, content, parentUri, project, metadata] = call;

  expect(uri).toBe("contextfs://proj/src/domain/feature.ts");
  expect(name).toBe("feature.ts");
  expect(parentUri).toBe("contextfs://proj/src/domain");
  expect(project).toBe("proj");

  // abstract = NL file summary (no longer empty)
  expect(abstractText).toBeTruthy();
  expect(abstractText).toMatch(/greet|UserService|normalize/i);

  // overview = compact graph notation (was previously in content)
  expect(overviewText).toContain("File: src/domain/feature.ts");
  expect(overviewText).toContain("Language: ts");
  expect(overviewText).toContain("LogicGraph: v1");
  expect(overviewText).toContain("Symbols:");
  expect(overviewText).toContain("- fn fn:greet");
  expect(overviewText).toContain("- fn fn:normalize");
  expect(overviewText).toContain("- mtd mtd:UserService.run");
  expect(overviewText).toContain("Edges:");
  expect(overviewText).toContain("- call fn:greet -> fn:normalize");
  expect(overviewText).toContain("- call mtd:UserService.run -> mtd:UserService.bump");
  expect(overviewText).toContain("- import file -> module:./slug");

  // content = NL descriptions (no longer compact notation)
  expect(content).toMatch(/greet/i);
  expect(content).toMatch(/normalize/i);
  expect(content).toMatch(/UserService/i);
  expect(content).not.toContain("- fn fn:greet"); // compact notation is in overview, not content

  // metadata now always includes logic_graph
  expect(metadata.type).toBe("file");
  expect(metadata.path).toBe(filePath);
  expect(metadata.logic_graph).toBeDefined();
});
```

Apply the same pattern to remaining tests:
- Empty file test: `abstractText` → basic fallback string (not empty), `overviewText` → contains compact notation with `(none)`, `content` → minimal NL or empty
- Payload hash test: hash is now computed over NL content, so a code change that doesn't affect the logic graph OR the NL description is what should be skipped. Adjust the "semantically identical" code change accordingly.
- Truncation test: `overviewText` now has the compact graph with truncation markers. `content` has NL with section-level truncation.

- [ ] **Step 5: Run ALL tests**

Run: `bun run test`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add src/daemon.ts tests/daemon.test.ts
git commit -m "feat: remap content fields — abstract=NL summary, overview=compact graph, content=full NL AST"
```

---

### Task 7: Parallel File Processing

**Files:**
- Modify: `src/daemon.ts` (add concurrency pool to processAllFiles and processPendingFiles)
- Modify: `src/daemon.ts:18-22` (add `concurrency` to DaemonOptions)
- Create: `tests/daemonParallel.test.ts`

- [ ] **Step 1: Write the failing test**

```typescript
// tests/daemonParallel.test.ts
import { describe, expect, it, vi } from "vitest";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import { CodebaseDaemon } from "../src/daemon";

function createManagerStub() {
  return {
    upsertFileContextNode: vi.fn().mockResolvedValue(undefined),
    deleteContextNode: vi.fn().mockResolvedValue(undefined),
  };
}

function makeTempDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), "contextfs-parallel-test-"));
}

describe("CodebaseDaemon parallel processing", () => {
  it("processes multiple files concurrently during initial scan", async () => {
    const tempDir = makeTempDir();
    const fileCount = 20;
    for (let i = 0; i < fileCount; i++) {
      fs.writeFileSync(
        path.join(tempDir, `module${i}.ts`),
        `export function fn${i}() { return ${i}; }`,
        "utf8"
      );
    }

    const manager = createManagerStub();
    // Use concurrency 4 to test parallel batching
    const daemon = new CodebaseDaemon(manager as any, "proj", tempDir, {
      concurrency: 4,
    });

    await (daemon as any).processAllFiles();

    expect(manager.upsertFileContextNode).toHaveBeenCalledTimes(fileCount);
  });

  it("processes pending file batch concurrently", async () => {
    const tempDir = makeTempDir();
    const files: string[] = [];
    for (let i = 0; i < 10; i++) {
      const p = path.join(tempDir, `change${i}.ts`);
      fs.writeFileSync(p, `export const v${i} = ${i};`, "utf8");
      files.push(p);
    }

    const manager = createManagerStub();
    const daemon = new CodebaseDaemon(manager as any, "proj", tempDir, {
      concurrency: 4,
    });

    // Queue all files then process
    for (const f of files) {
      (daemon as any).pendingFiles.add(path.resolve(f));
    }
    await (daemon as any).processPendingFiles();

    expect(manager.upsertFileContextNode).toHaveBeenCalledTimes(10);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/daemonParallel.test.ts`
Expected: FAIL — `concurrency` option not recognized (or test may pass if default behavior works, but timing assertions or the option type will fail)

- [ ] **Step 3: Add concurrency pool to daemon**

Add to `DaemonOptions`:
```typescript
concurrency?: number;
```

Default: `const DEFAULT_CONCURRENCY = 8;`

Implement a simple concurrency limiter in `daemon.ts`:

```typescript
private async runWithConcurrency<T>(
  items: T[],
  concurrency: number,
  fn: (item: T) => Promise<void>
): Promise<void> {
  let index = 0;
  const workers = Array.from({ length: Math.min(concurrency, items.length) }, async () => {
    while (index < items.length) {
      const currentIndex = index++;
      await fn(items[currentIndex]);
    }
  });
  await Promise.all(workers);
}
```

Update `processAllFiles`:
```typescript
private async processAllFiles() {
  const files = this.tsProject.getSourceFiles()
    .filter(f => this.shouldProcessFile(f.getFilePath()))
    .map(f => f.getFilePath());
  await this.runWithConcurrency(files, this.concurrency, (f) => this.processFile(f));
}
```

Note: since `processFile` now uses the describer (which creates an isolated in-memory ts-morph project per call), files are safe to process concurrently. The only shared mutable state is the hash maps, but JS is single-threaded so `Map` access between `await` boundaries is safe — `index++` completes synchronously before any `await` yields to the event loop.

Update `processPendingFiles` similarly — collect all pending into an array, clear the set, then process with concurrency.

Note: `tsProject` was already removed in Task 2. File discovery uses the `discoverSourceFiles` recursive walk added in Task 2.

- [ ] **Step 4: Run tests to verify they pass**

Run: `bun run test -- tests/daemonParallel.test.ts`
Expected: ALL PASS

- [ ] **Step 5: Run all tests for regression**

Run: `bun run test`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add src/daemon.ts tests/daemonParallel.test.ts
git commit -m "feat: add parallel file processing with configurable concurrency pool"
```

---

### Task 8: Persistent Hash Cache

**Files:**
- Modify: `src/daemon.ts` (add cache load/save, atomic write)
- Create: `tests/daemonCache.test.ts`

- [ ] **Step 1: Write the failing test**

```typescript
// tests/daemonCache.test.ts
import { afterEach, describe, expect, it, vi } from "vitest";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import { CodebaseDaemon } from "../src/daemon";

function createManagerStub() {
  return {
    upsertFileContextNode: vi.fn().mockResolvedValue(undefined),
    deleteContextNode: vi.fn().mockResolvedValue(undefined),
  };
}

function makeTempDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), "contextfs-cache-test-"));
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Persistent hash cache", () => {
  it("saves cache file after processing", async () => {
    const tempDir = makeTempDir();
    fs.writeFileSync(
      path.join(tempDir, "mod.ts"),
      "export function hello() { return 'hi'; }",
      "utf8"
    );

    const manager = createManagerStub();
    const daemon = new CodebaseDaemon(manager as any, "proj", tempDir);

    await (daemon as any).processAllFiles();
    (daemon as any).saveCache();

    const cachePath = path.join(tempDir, ".contextfs-cache.json");
    expect(fs.existsSync(cachePath)).toBe(true);

    const cache = JSON.parse(fs.readFileSync(cachePath, "utf8"));
    expect(cache.version).toBe(1);
    expect(Object.keys(cache.files).length).toBe(1);
  });

  it("skips unchanged files on restart using loaded cache", async () => {
    const tempDir = makeTempDir();
    const filePath = path.join(tempDir, "mod.ts");
    fs.writeFileSync(filePath, "export function hello() { return 'hi'; }", "utf8");

    // First run: processes and saves cache
    const manager1 = createManagerStub();
    const daemon1 = new CodebaseDaemon(manager1 as any, "proj", tempDir);
    await (daemon1 as any).processAllFiles();
    (daemon1 as any).saveCache();
    expect(manager1.upsertFileContextNode).toHaveBeenCalledTimes(1);

    // Second run: loads cache, skips unchanged file
    const manager2 = createManagerStub();
    const daemon2 = new CodebaseDaemon(manager2 as any, "proj", tempDir);
    (daemon2 as any).loadCache();
    await (daemon2 as any).processAllFiles();

    expect(manager2.upsertFileContextNode).not.toHaveBeenCalled();
  });

  it("reprocesses files when content changes between runs", async () => {
    const tempDir = makeTempDir();
    const filePath = path.join(tempDir, "mod.ts");
    fs.writeFileSync(filePath, "export function hello() { return 'hi'; }", "utf8");

    // First run
    const manager1 = createManagerStub();
    const daemon1 = new CodebaseDaemon(manager1 as any, "proj", tempDir);
    await (daemon1 as any).processAllFiles();
    (daemon1 as any).saveCache();

    // Change the file
    fs.writeFileSync(filePath, "export function hello() { return 'changed'; }", "utf8");

    // Second run: should reprocess
    const manager2 = createManagerStub();
    const daemon2 = new CodebaseDaemon(manager2 as any, "proj", tempDir);
    (daemon2 as any).loadCache();
    await (daemon2 as any).processAllFiles();

    expect(manager2.upsertFileContextNode).toHaveBeenCalledTimes(1);
  });

  it("ignores cache with mismatched version", async () => {
    const tempDir = makeTempDir();
    const filePath = path.join(tempDir, "mod.ts");
    fs.writeFileSync(filePath, "export function hello() { return 'hi'; }", "utf8");

    // Write a cache with wrong version
    const cachePath = path.join(tempDir, ".contextfs-cache.json");
    fs.writeFileSync(cachePath, JSON.stringify({ version: 999, files: {} }), "utf8");

    const manager = createManagerStub();
    const daemon = new CodebaseDaemon(manager as any, "proj", tempDir);
    (daemon as any).loadCache();
    await (daemon as any).processAllFiles();

    // Should process the file since cache version is invalid
    expect(manager.upsertFileContextNode).toHaveBeenCalledTimes(1);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bun run test -- tests/daemonCache.test.ts`
Expected: FAIL — loadCache/saveCache don't exist

- [ ] **Step 3: Implement persistent cache in daemon.ts**

Add constants:
```typescript
const CACHE_FILENAME = ".contextfs-cache.json";
const CACHE_VERSION = 1;
```

Add methods to `CodebaseDaemon`:

```typescript
public loadCache(): void {
  const cachePath = path.join(this.watchDir, CACHE_FILENAME);
  try {
    const raw = fs.readFileSync(cachePath, "utf8");
    const data = JSON.parse(raw);
    if (data.version !== CACHE_VERSION) {
      console.log("[Daemon] Cache version mismatch, ignoring");
      return;
    }
    for (const [absPath, entry] of Object.entries(data.files as Record<string, any>)) {
      this.fileFingerprints.set(absPath, entry.fingerprint);
      this.fileContentHashes.set(absPath, entry.contentHash);
      this.nodePayloadHashes.set(absPath, entry.payloadHash);
    }
    console.log(`[Daemon] Loaded cache with ${Object.keys(data.files).length} entries`);
  } catch {
    // No cache or corrupt — start fresh
  }
}

public saveCache(): void {
  const cachePath = path.join(this.watchDir, CACHE_FILENAME);
  const tmpPath = cachePath + ".tmp";
  const files: Record<string, any> = {};
  for (const [absPath, fingerprint] of this.fileFingerprints) {
    files[absPath] = {
      fingerprint,
      contentHash: this.fileContentHashes.get(absPath) ?? "",
      payloadHash: this.nodePayloadHashes.get(absPath) ?? "",
    };
  }
  const data = JSON.stringify({ version: CACHE_VERSION, files }, null, 2);
  fs.writeFileSync(tmpPath, data, "utf8");
  fs.renameSync(tmpPath, cachePath);
}
```

Call `loadCache()` at the start of `start()` (before initial scan).
Call `saveCache()` at the end of `processAllFiles()` and after each `processPendingFiles()` batch.

- [ ] **Step 4: Run tests to verify they pass**

Run: `bun run test -- tests/daemonCache.test.ts`
Expected: ALL PASS

- [ ] **Step 5: Add `.contextfs-cache.json` to `.gitignore`**

Append to `.gitignore`:
```
# Daemon persistent cache
.contextfs-cache.json
.contextfs-cache.json.tmp
```

- [ ] **Step 6: Run all tests for regression**

Run: `bun run test`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add src/daemon.ts tests/daemonCache.test.ts .gitignore
git commit -m "feat: add persistent hash cache for daemon restart skip of unchanged files"
```

---

### Task 9: Integration Test and Final Verification

**Files:**
- Modify: `tests/daemon.test.ts` (add end-to-end NL quality test)

- [ ] **Step 1: Write an integration test with a realistic code sample**

Add to `tests/daemon.test.ts`:

```typescript
it("produces rich NL content for a realistic async service file", async () => {
  const tempDir = makeTempDir();
  const filePath = path.join(tempDir, "userService.ts");
  const code = source([
    "import { db } from './database';",
    "import { hash } from './crypto';",
    "",
    "interface User {",
    "  id: string;",
    "  name: string;",
    "  email: string;",
    "}",
    "",
    "const MAX_RETRIES = 3;",
    "",
    "export async function createUser(name: string, email: string): Promise<User> {",
    "  const existing = await db.findByEmail(email);",
    "  if (existing) {",
    "    throw new Error('Email already registered');",
    "  }",
    "  const id = hash(email + Date.now());",
    "  const user = { id, name, email };",
    "  await db.insert(user);",
    "  return user;",
    "}",
    "",
    "export async function getUser(id: string): Promise<User | null> {",
    "  let retries = 0;",
    "  while (retries < MAX_RETRIES) {",
    "    try {",
    "      const user = await db.findById(id);",
    "      return user;",
    "    } catch (e) {",
    "      retries++;",
    "      if (retries >= MAX_RETRIES) {",
    "        throw e;",
    "      }",
    "    }",
    "  }",
    "  return null;",
    "}",
  ]);
  fs.writeFileSync(filePath, code, "utf8");

  const manager = createManagerStub();
  const daemon = new CodebaseDaemon(manager as any, "proj", tempDir);
  await (daemon as any).processFile(filePath);

  const [, , abstractText, overviewText, content] = manager.upsertFileContextNode.mock.calls[0];

  // Abstract: meaningful NL summary
  expect(abstractText.length).toBeGreaterThan(20);
  expect(abstractText).toMatch(/createUser|getUser|user/i);

  // Overview: has compact graph
  expect(overviewText).toContain("fn fn:createUser");
  expect(overviewText).toContain("fn fn:getUser");
  expect(overviewText).toContain("Symbols:");

  // Content: NL descriptions with statement-level detail
  expect(content).toMatch(/createUser/);
  expect(content).toMatch(/[Aa]waits|await/);
  expect(content).toMatch(/[Ee]mail.*registered|already/);
  expect(content).toMatch(/[Tt]hrows/);
  expect(content).toMatch(/getUser/);
  expect(content).toMatch(/[Rr]etr(y|ies)|[Ww]hile|[Ll]oop/);
  expect(content).toMatch(/MAX_RETRIES|retries/);
});
```

- [ ] **Step 2: Run test to verify it passes**

Run: `bun run test -- tests/daemon.test.ts`
Expected: ALL PASS

- [ ] **Step 3: Run typecheck and lint**

Run: `bun run typecheck && bun run lint`
Expected: No errors

- [ ] **Step 4: Run full test suite**

Run: `bun run test`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add tests/daemon.test.ts
git commit -m "test: add integration test for NL AST output quality"
```

---

## File Structure Summary

| File | Role |
|---|---|
| `src/languageDescriber.ts` | Interface + shared types (LogicSymbol, LogicEdge, FileGraphResult, LanguageDescriber) |
| `src/typescriptDescriber.ts` | TypeScript/JS implementation of LanguageDescriber using ts-morph |
| `src/nlDescriber.ts` | AST-to-English statement describer engine |
| `src/nlEnricher.ts` | Post-enrichment pass for cross-function references |
| `src/daemon.ts` | File watcher, parallel processing, caching, content assembly (slimmed down) |
| `tests/languageDescriber.test.ts` | Interface contract tests |
| `tests/typescriptDescriber.test.ts` | TypeScript extraction + NL generation tests |
| `tests/nlDescriber.test.ts` | Statement-level NL pattern tests |
| `tests/nlEnricher.test.ts` | Cross-reference enrichment tests |
| `tests/daemonParallel.test.ts` | Parallel processing tests |
| `tests/daemonCache.test.ts` | Persistent cache tests |
| `tests/daemon.test.ts` | Updated integration tests |
