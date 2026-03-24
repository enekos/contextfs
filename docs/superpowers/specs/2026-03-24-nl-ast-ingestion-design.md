# Natural Language AST Ingestion — Design Spec

## Goal

Transform the daemon's AST ingestion from compact machine-readable notation into human-readable natural language that explains code logic at statement-level depth, while making the system performant enough for large codebases through parallel processing and persistent caching.

## Architecture

Single-pass AST walker extracts symbols, edges, and raw NL descriptions per statement in one traversal. A lightweight post-enrichment pass stitches cross-function references into the NL output (e.g., resolving "calls X" to "calls the validation function that checks token expiry"). The NL generator is defined behind a pluggable `LanguageDescriber` interface so future languages can be added without touching the daemon core.

## Content Field Layout (changed)

| Field | Before | After |
|---|---|---|
| `abstract` | Empty in compact mode | Concise NL summary of the file's purpose (~1-3 sentences) |
| `overview` | Empty in compact mode | Compact graph notation (what was previously in `content`) |
| `content` | Compact graph notation | Full NL AST with statement-level descriptions |

## NL Description Depth

Full depth — covers:
- Function/method purpose from name, params, return type
- Statement-by-statement logic flow within bodies
- Conditional branches and their conditions in plain English
- Error handling paths (try/catch, throw)
- Loop descriptions (what is iterated, termination conditions)
- Variable transformations and assignments
- Cross-function call descriptions enriched with callee context

Example output for a function:

```
## Function: validateToken (exported, async)
Parameters: token (string), options (ValidationOptions)
Returns: Promise<UserPayload>

Validates an authentication token and extracts the user payload.

1. Checks if the token is empty or undefined — if so, throws an AuthError with message "Token required".
2. Calls `decodeJwt` (a local function that parses the Base64 JWT payload and verifies the signature) passing the token.
3. If decoding fails (catch block), throws an AuthError with message "Invalid token format".
4. Reads the module-level constant `TOKEN_TTL_MS` and computes the token age by subtracting the decoded `issuedAt` from the current timestamp.
5. If the token age exceeds `TOKEN_TTL_MS`, throws an AuthError with message "Token expired".
6. If `options.requireAdmin` is true, checks the decoded payload's `role` field — if not "admin", throws a ForbiddenError.
7. Returns the decoded payload as `UserPayload`.
```

## Pluggable Interface

```typescript
interface LanguageDescriber {
  /** Unique language key, e.g. "typescript", "python" */
  readonly languageId: string;

  /** File extensions this describer handles */
  readonly extensions: ReadonlySet<string>;

  /** Extract logic graph + raw NL descriptions in a single AST pass */
  extractFileGraph(filePath: string, sourceText: string): FileGraphResult;
}

interface FileGraphResult {
  symbols: LogicSymbol[];
  edges: LogicEdge[];
  imports: string[];
  /** Per-symbol NL descriptions keyed by symbol ID */
  symbolDescriptions: Map<string, string>;
  /** File-level NL summary */
  fileSummary: string;
}
```

The daemon selects a describer by file extension. The TypeScript describer uses ts-morph internally. The daemon itself only calls the interface methods.

## NL Generation Strategy (AST Heuristics)

No LLM calls. Pure AST pattern matching:

| AST Pattern | NL Template |
|---|---|
| `if (cond) { ... }` | "If {cond in English}, {body description}" |
| `if (cond) { ... } else { ... }` | "If {cond}, {then}. Otherwise, {else}" |
| `for (const x of items)` | "Iterates over each {x} in {items}" |
| `while (cond)` | "Loops while {cond}" |
| `try { ... } catch (e) { ... }` | "Attempts to {try body}. If an error occurs, {catch body}" |
| `throw new X(msg)` | "Throws a {X} with message {msg}" |
| `return expr` | "Returns {expr description}" |
| `const x = fn(args)` | "Assigns the result of calling {fn} with {args} to {x}" |
| `await expr` | "Awaits {expr description}" |
| `switch (expr) { cases }` | "Switches on {expr}: case {val1}: {body1}, case {val2}: {body2}, ..." |

Conditions are translated: `x === null` → "x is null", `!x` → "x is falsy", `x.length > 0` → "x is non-empty", `typeof x === 'string'` → "x is a string", etc.

## Post-Enrichment Pass

After extraction, the daemon has both the graph (who calls whom) and the per-symbol descriptions. The enrichment pass:

1. For each `call` edge, replaces generic "calls X" with "calls X (which {first sentence of X's description})"
2. Caps enrichment depth at 1 level (no recursive expansion)
3. This is string interpolation over already-extracted data — very fast

## Performance: Parallel Processing

Replace sequential `for...of await` with a concurrency-limited pool:

```typescript
async processFileBatch(files: string[], concurrency: number = 8): Promise<void>
```

- AST extraction is CPU-bound but per-file independent
- ES upserts are I/O-bound and benefit from overlapping
- Default concurrency: 8 (configurable via `DaemonOptions`)

Applied to both initial scan (`processAllFiles`) and batch change processing.

## Performance: Persistent Hash Cache

Persist fingerprint/content/payload hashes to disk so daemon restarts skip unchanged files:

- Cache file: `<watchDir>/.contextfs-cache.json`
- Format: `{ version: 1, files: { [absPath]: { fingerprint, contentHash, payloadHash } } }`
- Loaded on daemon start, saved after each batch
- Atomic write (write to `.contextfs-cache.tmp.json`, then rename)
- Cache invalidation: if cache version doesn't match, full re-scan

## Constraints

- No new dependencies (ts-morph already provides everything needed for TS/JS)
- No LLM/API calls for NL generation
- TS/JS only for now, but interface designed for future languages
- Existing test patterns preserved (manager stub, temp dirs)
- Max content size limit (16KB) still enforced — NL is more verbose, so truncation strategy matters
- `overview` inherits current compact format limits unchanged
