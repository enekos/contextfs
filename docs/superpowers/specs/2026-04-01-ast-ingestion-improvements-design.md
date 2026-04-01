# AST Ingestion Improvements: JSDoc, Byte Offsets, Symbol Hashing

**Date:** 2026-04-01
**Scope:** Three additive improvements to the AST ingestion pipeline. No breaking changes.

## Motivation

Inspired by [jcodemunch-mcp](https://github.com/jgravelle/jcodemunch-mcp), which demonstrates that storing docstrings, byte offsets, and per-symbol content hashes significantly improves code retrieval precision and indexing efficiency. These three improvements are high-value, low-effort additions to the existing pipeline.

## 1. JSDoc/TSDoc Extraction

### Problem

The NL describer generates mechanical descriptions from AST structure ("Assigns X to Y", "If condition, does Z"). Developer-written JSDoc comments are often more semantically useful but are currently ignored.

### Design

**New field on `LogicSymbol`:**
```typescript
docstring?: string;  // first sentence of JSDoc, undefined if none
```

**Extraction logic in `TypeScriptDescriber`:**

For each declaration node, check the previous sibling in the parent's children list. If it is a `comment` node whose text starts with `/**`, extract the doc content:

1. Strip `/**`, `*/`, and leading `*` from each line
2. Take the first sentence (up to first `.` followed by whitespace/newline, or first blank line)
3. Trim to 200 chars max
4. Store on the `LogicSymbol` as `docstring`

For class methods, the comment is a sibling within the class body, same logic applies.

**Usage in `daemon.ts` `buildNLContent()`:**

When rendering a symbol's section, if `docstring` is present, prepend it as an italicized line before the NL description:

```
## Function: validate (exported)
*Validates an email address against RFC 5322 rules.*
Parameters: email
1. Returns `email.includes('@')`
```

**Usage in `buildCompactContent()`:**

Append `doc="first few words..."` to the symbol line in overview, truncated to 60 chars.

**File summary enhancement:**

When generating `fileSummary`, if the file's first comment is a `/**` file-level doc comment (before any import/declaration), use its first sentence as the summary instead of the generated "File containing N exported symbols..." text.

### Edge cases

- No `/**` comment before a declaration: `docstring` remains `undefined`, no behavior change
- `//` or `/* */` comments: ignored, only `/** */` qualifies
- Empty doc comments (`/** */`): treated as no docstring
- Decorators between comment and declaration: walk backwards past decorator nodes to find the comment

## 2. Byte-Offset Storage

### Problem

Symbols only store `line` number. To retrieve exact source code for a symbol, you must re-read and re-parse the file. Byte offsets enable direct extraction.

### Design

**New fields on `LogicSymbol`:**
```typescript
byteStart: number;  // inclusive byte offset into source text
byteEnd: number;    // exclusive byte offset into source text
```

**Source in `TypeScriptDescriber.makeSymbol()`:**

tree-sitter nodes already expose `startIndex` and `endIndex` as byte offsets into the parsed source text. These map directly:

```typescript
byteStart: node.startIndex,
byteEnd: node.endIndex,
```

**Propagation:**

- Stored in `logicGraphMetadata` automatically (it serializes all `LogicSymbol` fields)
- No changes needed in `daemon.ts` serialization — the metadata object already includes the full symbol array
- `buildCompactContent()`: no change needed (offsets are in metadata, not in the compact text view)

### Validation

For variable declarators, the node is the `variable_declarator` not the `lexical_declaration`. This means `byteStart`/`byteEnd` covers the declarator (`foo = ...`) not the keyword (`const foo = ...`). This is intentional — the declarator is the semantically meaningful unit. For export-wrapped declarations, the node is the inner declaration, not the `export_statement` wrapper.

## 3. Symbol-Level Content Hashing

### Problem

When a file changes, the daemon re-processes all symbols even if only one function was edited. With per-symbol content hashes, future optimizations can skip re-embedding unchanged symbols.

### Design

**New field on `LogicSymbol`:**
```typescript
contentHash: string;  // SHA1 hex of the symbol's source text
```

**Source in `TypeScriptDescriber.makeSymbol()`:**

```typescript
contentHash: createHash("sha1").update(node.text).digest("hex"),
```

This uses the same `createHash` from Node's `crypto` module already imported in `daemon.ts`. The import needs to be added to `typescriptDescriber.ts`.

**Propagation:**

- Stored in `logicGraphMetadata` alongside other symbol fields
- The daemon's file-level `nodePayloadHashes` continues to work as before
- Symbol-level hashing is purely additive — it enables future per-symbol differential updates but doesn't change current processing flow

### Future use (not in this PR)

When contextfs moves to per-symbol indexing (each symbol as its own context node), the daemon can compare `contentHash` values between old and new file parses to:
- Skip re-embedding symbols whose content hasn't changed
- Only update the specific symbols that were modified
- Track symbol renames (same hash, different name)

## Files Modified

| File | Changes |
|---|---|
| `src/ast/languageDescriber.ts` | Add `docstring?`, `byteStart`, `byteEnd`, `contentHash` to `LogicSymbol` |
| `src/ast/typescriptDescriber.ts` | Extract JSDoc from preceding comments, populate byte offsets and content hash in `makeSymbol()` |
| `src/daemon.ts` | Use docstrings in `buildNLContent()` and `buildCompactContent()`, use file-level doc as summary |
| `src/ast/tsxDescriber.ts` | Inherits changes from `TypeScriptDescriber` (extends it) |
| `src/ast/vueDescriber.ts` | Inherits script-block changes; no template changes needed |

## Not in Scope

- CLI command for symbol source retrieval (store-only for byte offsets)
- Per-symbol differential re-indexing (future use of content hashes)
- Named import tracking, import graphs, PageRank (separate feature)
- Multi-language support (separate feature)

## Testing

- Unit tests for JSDoc extraction: with/without docstrings, multi-line docs, decorator edge cases
- Unit tests for byte offset correctness: verify offsets match `sourceText.slice(byteStart, byteEnd)` against node.text
- Integration test: process a sample file through the full pipeline, verify metadata contains new fields
- Existing tests must continue to pass (new fields are additive)

## Cache Compatibility

Adding new fields to `LogicSymbol` will change the `payloadHash` for every file on next daemon run, triggering a one-time full re-index. This is acceptable and expected. The `CACHE_VERSION` does not need to bump since the cache format itself hasn't changed — only the derived payload differs.
