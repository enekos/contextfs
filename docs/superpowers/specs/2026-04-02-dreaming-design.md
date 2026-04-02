# Dreaming: Background Memory Consolidation for contextfs

## Summary

A background memory consolidation system that periodically runs LLM-powered deduction, induction, and context node consolidation passes over accumulated project data. Inspired by Honcho's dreaming architecture, adapted for coding agents. Triggered automatically by the daemon via write-activity heuristics, or manually via CLI.

## Motivation

contextfs currently handles deduplication at write time (via `llmRouter`), but has no mechanism for maintaining memory quality over time. As memories accumulate:
- Near-duplicates slip through (slightly different phrasing, written at different times)
- Facts become stale but aren't invalidated (e.g., "uses Jest" after migrating to Vitest)
- Higher-order patterns across many memories are never synthesized
- Context node abstracts drift out of sync with their children
- Redundant sibling nodes accumulate without consolidation

Dreaming addresses all of these through periodic, idle-triggered background processing.

## Architecture

### New Module

**`src/dreamer.ts`** — single stateless module exporting:
- `dream(project: string): Promise<void>` — runs all three passes sequentially
- `deductionPass(project: string): Promise<void>` — logical cleanup on memories
- `inductionPass(project: string): Promise<void>` — pattern extraction from memories
- `contextNodeConsolidationPass(project: string): Promise<void>` — tree maintenance

No new services, no new database indexes, no new dependencies. Reuses existing `meilisearchDB`, `embedder`, and LLM functions (same Gemini setup as `llmRouter`/`vibeEngine`).

### Integration Points

**Daemon** (`src/daemon.ts`):
- Tracks memory write count per project since last dream
- Manages per-project idle timers
- Enforces cooldown between dream cycles
- Cancels pending dreams when new writes arrive

**CLI** (`src/cli.ts`):
- New command: `context-cli dream -P <project>`
- Bypasses threshold/cooldown, runs immediately
- Reuses existing project flag pattern

**Storage** (`src/storage/meilisearchDB.ts`):
- No schema changes. Reuses existing CRUD + search operations
- New memory category value: `pattern` (added to the category union type)

## Dreaming Passes

### Pass 1: Deduction (Logical Cleanup)

Operates on memories. Goals: merge duplicates, resolve contradictions, adjust importance via corroboration.

**Algorithm:**

1. Fetch all memories for the project (paginated, ordered by `created_at` desc)
2. Build a "processed" set to avoid re-comparing already-handled memories
3. For each unprocessed memory:
   a. Vector-search for candidates with cosine similarity > 0.85 (exclude self, exclude already-processed)
   b. If candidates found, send the memory + candidates to LLM with prompt:
      - For each candidate pair, decide: `MERGE` (combine into one), `CONTRADICTION` (keep newer, delete older), or `KEEP_BOTH`
   c. Execute decisions:
      - `MERGE`: LLM produces merged content, update the newer memory, delete the older
      - `CONTRADICTION`: Delete the older memory
      - `KEEP_BOTH`: No action
   d. Mark all involved memories as processed
4. Corroboration pass: for memories with 3+ near-duplicates that were all kept (KEEP_BOTH), bump importance by 1 (capped at 10)

**LLM prompt structure:**
```
You are a memory consolidation agent. Given a source memory and candidate similar memories, decide for each pair:
- MERGE: The memories say essentially the same thing. Produce a merged version.
- CONTRADICTION: The memories conflict. The newer one is likely current truth.
- KEEP_BOTH: The memories are related but distinct.

Source memory: {content} (created: {date})
Candidates:
1. {content} (created: {date})
2. {content} (created: {date})

Respond as JSON: { decisions: [{ candidateIndex: number, action: "MERGE"|"CONTRADICTION"|"KEEP_BOTH", mergedContent?: string }] }
```

### Pass 2: Induction (Pattern Extraction)

Operates on memories. Goal: synthesize higher-order patterns from clusters of related memories.

**Algorithm:**

1. Fetch all memories for the project (excluding category `derived_pattern` to avoid meta-patterns)
2. Group memories by category
3. Within each category group, cluster by semantic similarity:
   a. Pick an unassigned memory as seed
   b. Vector-search for memories with cosine > 0.75
   c. Form a cluster from seed + results
   d. Mark all as assigned, repeat until all assigned
4. For clusters with 3+ members:
   a. Send cluster contents to LLM with prompt asking for pattern synthesis
   b. LLM produces: pattern description, confidence (low/medium/high), supporting evidence summary
   c. Before storing: vector-search existing `pattern` memories (cosine > 0.85) to avoid duplicate patterns
   d. If no duplicate: store as new memory with category `derived_pattern`, importance = `min(cluster_size + 3, 10)`
   e. If duplicate exists: update existing pattern if the new synthesis is richer

**LLM prompt structure:**
```
You are analyzing a cluster of related memories from a coding project. Identify the higher-order pattern they reveal.

Memories:
1. {content}
2. {content}
3. {content}

If these memories reveal a recurring pattern, preference, or behavioral tendency, describe it as a concise, actionable insight. If no meaningful pattern exists, respond with null.

Respond as JSON: { pattern: string | null, confidence: "low"|"medium"|"high", evidence: string }
```

### Pass 3: Context Node Consolidation

Operates on context nodes. Goals: refresh stale abstracts, merge redundant siblings, regroup scattered related nodes.

**3a. Abstract Regeneration:**

1. Fetch all parent nodes (nodes that have children) for the project
2. For each parent, fetch children
3. If any child's `updated_at` > parent's `updated_at`:
   a. Collect children's abstracts
   b. LLM synthesizes a new parent abstract from children
   c. Update parent's abstract field

**3b. Redundant Sibling Detection:**

1. For each parent node, fetch its children
2. Pairwise cosine similarity on children's embeddings
3. For pairs with cosine > 0.9:
   a. Send both nodes to LLM: decide MERGE or KEEP_BOTH
   b. MERGE: LLM produces merged node (combined content, best abstract), update one node, reparent the other's children, delete the other

**3c. Orphan Regrouping:**

1. Fetch all leaf nodes (no children) across the project
2. Cluster by semantic similarity (cosine > 0.8)
3. For clusters of 3+ nodes that span 2+ different parents:
   a. LLM evaluates whether a new grouping parent makes sense
   b. If yes: create a new parent node, reparent the cluster nodes under it
   c. The new parent gets an LLM-generated name and abstract

## Scheduling Heuristics

### Automatic (Daemon)

The daemon maintains per-project dream state:

```typescript
interface DreamState {
  writesSinceLastDream: number;
  lastDreamAt: number | null;       // epoch ms
  idleTimer: ReturnType<typeof setTimeout> | null;
}
```

**Trigger flow:**

1. On every memory write (store/update via daemon or CLI), increment `writesSinceLastDream` for that project
2. Check conditions:
   - `writesSinceLastDream >= DREAM_THRESHOLD` (default: 25)
   - `lastDreamAt` is null OR `Date.now() - lastDreamAt >= DREAM_COOLDOWN` (default: 4 hours)
3. If both met: start/restart idle timer for `DREAM_IDLE_TIMEOUT` (default: 30 minutes)
4. If new write arrives during idle timer: cancel timer, restart from step 2
5. When idle timer fires: execute `dream(project)`, reset `writesSinceLastDream` to 0, update `lastDreamAt`

### Manual (CLI)

```bash
context-cli dream -P my-project
```

Bypasses threshold and cooldown. Runs the three passes immediately. If a dream is already in progress for the project (via daemon), the manual invocation waits or skips.

### Configuration

| Env Variable | Default | Description |
|---|---|---|
| `DREAM_THRESHOLD` | `25` | Minimum new memory writes before scheduling |
| `DREAM_COOLDOWN` | `4h` | Minimum time between automatic dreams |
| `DREAM_IDLE_TIMEOUT` | `30m` | Idle period before executing a scheduled dream |
| `DREAM_ENABLED` | `true` | Master toggle for automatic dreaming |

All values support duration strings parsed the same way as `RECENCY_SCALE`.

## New Memory Category

Add `pattern` to the existing category union:

```
profile | preferences | entities | events | cases | patterns | observation |
reflection | decision | constraint | architecture | derived_pattern
```

Pattern memories are created only by the induction pass and have:
- `category: "derived_pattern"`
- `owner: "system"`
- `importance`: derived from cluster size (min 4, max 10)
- `content`: the synthesized pattern description

## Error Handling

- Each pass is independent. If deduction fails, induction and consolidation still run.
- Individual memory/node operations within a pass use try/catch — one failure doesn't abort the pass.
- LLM call failures use the same retry logic as `llmRouter` (3 attempts, exponential backoff).
- If the entire dream fails, the daemon logs the error and the next automatic dream will be attempted after cooldown.

## Scope Boundaries

**Included:**
- Deduction pass (merge, contradiction, corroboration)
- Induction pass (pattern synthesis)
- Context node consolidation (abstract regen, sibling merge, orphan regrouping)
- Daemon integration with threshold/cooldown/idle scheduling
- CLI manual trigger
- Per-project isolation

**Not included:**
- Dream journal / audit logging
- Cross-project dreaming
- New Meilisearch indexes
- Usage tracking / access counts
- Temporal validity (valid_from/invalidated_at) — future enhancement
- Streaming / progress reporting
- Dashboard UI for dream status
