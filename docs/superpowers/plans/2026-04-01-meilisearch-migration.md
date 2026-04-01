# Meilisearch Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Elasticsearch with Meilisearch as the sole storage/search backend.

**Architecture:** New `MeilisearchDB` class replaces `ElasticDB` with the same public method signatures. Hybrid search uses Meilisearch's native vector+keyword, with app-side re-ranking for recency/importance. All ES-specific code, config, and dependencies are removed.

**Tech Stack:** Meilisearch v1.12+, `meilisearch` JS client, Bun, TypeScript

**Spec:** `docs/superpowers/specs/2026-04-01-meilisearch-migration-design.md`

---

### Task 1: Infrastructure — Docker & Dependencies

**Files:**
- Modify: `docker-compose.yml`
- Modify: `package.json`

- [ ] **Step 1: Replace Docker Compose service**

Replace the entire contents of `docker-compose.yml`:

```yaml
services:
  meilisearch:
    image: getmeili/meilisearch:v1.12
    container_name: contextfs-meili
    ports:
      - "7700:7700"
    environment:
      MEILI_MASTER_KEY: "contextfs-dev-key"
      MEILI_ENV: "development"
    volumes:
      - meili_data:/meili_data

volumes:
  meili_data:
    driver: local
```

- [ ] **Step 2: Swap npm dependencies**

Run:
```bash
bun remove @elastic/elasticsearch
bun add meilisearch
```

Expected: `package.json` no longer has `@elastic/elasticsearch`, has `meilisearch`.

- [ ] **Step 3: Start Meilisearch**

Run:
```bash
docker compose down -v
docker compose up -d
```

Expected: Meilisearch running at `http://localhost:7700`. Verify:
```bash
curl http://localhost:7700/health
```
Expected output: `{"status":"available"}`

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml package.json bun.lock
git commit -m "chore: swap elasticsearch for meilisearch in docker and deps"
```

---

### Task 2: Config & Types

**Files:**
- Modify: `src/core/config.ts`
- Modify: `src/core/types.ts`
- Modify: `.env.example`

- [ ] **Step 1: Update config.ts**

Replace the full file `src/core/config.ts` with:

```typescript
import * as dotenv from "dotenv";
import * as path from "path";
import { parsePositiveInt, parseBoolean, parseNonNegativeInt } from "./configParsing";

dotenv.config({ path: path.resolve(__dirname, "..", ".env") });

const DEFAULT_EMBEDDING_MODEL = "gemini-embedding-001";
const DEFAULT_EMBEDDING_DIMENSION = 3072;

const KNOWN_MODEL_DIMENSIONS: Record<string, number> = {
  "gemini-embedding-001": 3072,
  "text-embedding-004": 768,
};

function getEmbeddingDimension(): number {
  const configuredDimension = parsePositiveInt(process.env.EMBEDDING_DIM);
  const model = process.env.EMBEDDING_MODEL || DEFAULT_EMBEDDING_MODEL;
  const inferredDimension = KNOWN_MODEL_DIMENSIONS[model];
  const dimension = configuredDimension ?? inferredDimension ?? DEFAULT_EMBEDDING_DIMENSION;

  if (configuredDimension && inferredDimension && configuredDimension !== inferredDimension) {
    throw new Error(
      `EMBEDDING_DIM (${configuredDimension}) does not match known dimension for ${model} (${inferredDimension})`
    );
  }

  return dimension;
}

export const config = {
  meili: {
    get url() { return process.env.MEILI_URL || "http://localhost:7700"; },
    get apiKey() { return process.env.MEILI_API_KEY || ""; },
    get synonyms(): string[] {
      const raw = process.env.SYNONYMS || "";
      return raw ? raw.split(";").map((s) => s.trim()).filter(Boolean) : [];
    },
    get recencyScale() { return process.env.RECENCY_SCALE || "30d"; },
    get recencyDecay() { return parseFloat(process.env.RECENCY_DECAY || "0.5"); },
  },

  get geminiApiKey() { return process.env.GEMINI_API_KEY; },

  get llmModel() { return process.env.LLM_MODEL || "gemini-2.0-flash-lite"; },

  get dashboardApiPort() { return parsePositiveInt(process.env.DASHBOARD_API_PORT) || 8787; },

  get candidateMultiplier() { return parsePositiveInt(process.env.CANDIDATE_MULTIPLIER) || 4; },

  embedding: {
    get model() { return process.env.EMBEDDING_MODEL || DEFAULT_EMBEDDING_MODEL; },
    get dimension() { return getEmbeddingDimension(); },
    get allowZeroEmbeddings() { return parseBoolean(process.env.ALLOW_ZERO_EMBEDDINGS, true); },
  },

  budget: {
    get memoryPerProject() { return parseNonNegativeInt(process.env.MEMORY_BUDGET_PER_PROJECT) ?? 500; },
    get skillPerProject() { return parseNonNegativeInt(process.env.SKILL_BUDGET_PER_PROJECT) ?? 100; },
    get nodePerProject() { return parseNonNegativeInt(process.env.NODE_BUDGET_PER_PROJECT) ?? 1000; },
  },
};

export function assertEmbeddingDimension(vector: number[], context: string): void {
  const dim = config.embedding.dimension;
  if (vector.length !== dim) {
    throw new Error(
      `Invalid embedding size for ${context}. Expected ${dim}, got ${vector.length}.`
    );
  }
}
```

- [ ] **Step 2: Update types.ts**

In `src/core/types.ts`, rename `ElasticSearchTuning` to `SearchTuning` and remove `fuzziness`/`phraseBoost`:

Replace the `ElasticSearchTuning` interface (lines 21-37) with:

```typescript
/** Search tuning options available on all search methods */
export interface SearchTuning {
  /** Hard minimum score cutoff — results below this are dropped. Default: none */
  minScore?: number;
  /** Return highlighted snippets showing matched terms. Default: false */
  highlight?: boolean;
  /** Custom field boost overrides, e.g. { "name": 5, "content": 1 } */
  fieldBoosts?: Record<string, number>;
  /** Override recency scale (e.g. "30d") */
  recencyScale?: string;
  /** Override recency decay factor (e.g. 0.5) */
  recencyDecay?: number;
}
```

Replace `MemorySearchOptions extends ElasticSearchTuning` with `MemorySearchOptions extends SearchTuning` (line 95).
Replace `SkillSearchOptions extends ElasticSearchTuning` with `SkillSearchOptions extends SearchTuning` (line 106).
Replace `ContextSearchOptions extends ElasticSearchTuning` with `ContextSearchOptions extends SearchTuning` (line 114).

- [ ] **Step 3: Update .env.example**

Replace the full file `.env.example` with:

```
MEILI_URL=http://localhost:7700
MEILI_API_KEY=contextfs-dev-key

GEMINI_API_KEY=your_gemini_api_key

# Embedding configuration
EMBEDDING_MODEL=gemini-embedding-001
EMBEDDING_DIM=3072

# Set true only for local/offline testing
ALLOW_ZERO_EMBEDDINGS=false

# Optional
DASHBOARD_API_PORT=8787

# ─── Budget Limits (per project, 0 = unlimited) ──────────────────────────
# MEMORY_BUDGET_PER_PROJECT=500
# SKILL_BUDGET_PER_PROJECT=100
# NODE_BUDGET_PER_PROJECT=1000

# ── Search tuning ──────────────────────────────────────────────────────────

# Recency decay function
# RECENCY_SCALE=30d     # Half-life: score halves at this age (e.g., 7d, 30d, 90d)
# RECENCY_DECAY=0.5     # Decay factor at scale distance (0-1)

# Synonyms (semicolon-separated groups)
# SYNONYMS=auth,authentication,authn;db,database;k8s,kubernetes;js,javascript;ts,typescript

# ── Optional adaptive retrieval policy (online RL-style tuning) ─────────────────
# RL_ADAPTIVE_ENABLED=false
# RL_PROJECT_ALLOWLIST=my-project,another-project
# RL_EPSILON=0.15
# RL_WARMUP_SAMPLES=5
# RL_POLICY_STORE_PATH=.contextfs-rl-policies.json
# RL_EVENT_LOG_PATH=.contextfs-rl-events.jsonl
```

- [ ] **Step 4: Run typecheck**

Run: `bun run typecheck`

Expected: Errors from files still importing `ElasticDB` / old config props — that's expected at this stage.

- [ ] **Step 5: Commit**

```bash
git add src/core/config.ts src/core/types.ts .env.example
git commit -m "feat: update config and types for meilisearch"
```

---

### Task 3: MeilisearchDB — Core Structure, Init, and CRUD (Skills)

**Files:**
- Create: `src/storage/meilisearchDB.ts`

- [ ] **Step 1: Create meilisearchDB.ts with index constants, constructor, init, and skills CRUD**

Create `src/storage/meilisearchDB.ts`:

```typescript
import { MeiliSearch, Index } from "meilisearch";
import {
  AgentSkill,
  AgentMemory,
  AgentContextNode,
  MemorySearchOptions,
  MemoryCategory,
  SkillSearchOptions,
  ContextSearchOptions,
} from "../core/types";
import { assertEmbeddingDimension, config } from "../core/config";
import {
  DEFAULT_MEMORY_WEIGHTS,
  DEFAULT_SKILL_WEIGHTS,
  DEFAULT_CONTEXT_WEIGHTS,
  normalizeWeights,
} from "./scorer";

const EMBEDDING_DIM = config.embedding.dimension;
const CANDIDATE_MULTIPLIER = config.candidateMultiplier;
const AI_QUALITY_FUNCTION_WEIGHT = 2;

export const SKILLS_INDEX = "contextfs_skills";
export const MEMORIES_INDEX = "contextfs_memories";
export const CONTEXT_INDEX = "contextfs_context_nodes";

/** Parse a duration string like "30d", "7d", "90d" to milliseconds. */
function parseDurationMs(duration: string): number {
  const match = duration.match(/^(\d+)([dhms])$/);
  if (!match) return 30 * 24 * 60 * 60 * 1000; // default 30d
  const value = parseInt(match[1], 10);
  const unit = match[2];
  switch (unit) {
    case "d": return value * 24 * 60 * 60 * 1000;
    case "h": return value * 60 * 60 * 1000;
    case "m": return value * 60 * 1000;
    case "s": return value * 1000;
    default: return 30 * 24 * 60 * 60 * 1000;
  }
}

export class MeilisearchDB {
  private client: MeiliSearch;
  private initialized = false;

  constructor(url: string, apiKey?: string) {
    this.client = new MeiliSearch({ host: url, apiKey: apiKey || undefined });
  }

  private async ensureInitialized() {
    if (this.initialized) return;
    try {
      await this.initIndices();
      this.initialized = true;
    } catch (e: any) {
      if (e?.code === "ECONNREFUSED" || e?.type === "MeiliSearchCommunicationError") {
        console.error("❌ Meilisearch connection failed. Is Docker running? (Run: docker compose up -d)");
        process.exit(1);
      }
      throw e;
    }
  }

  async initIndices() {
    // Create indexes (no-op if they already exist)
    const indexes = [
      { uid: SKILLS_INDEX, primaryKey: "id" },
      { uid: MEMORIES_INDEX, primaryKey: "id" },
      { uid: CONTEXT_INDEX, primaryKey: "uri" },
    ];

    for (const idx of indexes) {
      const task = await this.client.createIndex(idx.uid, { primaryKey: idx.primaryKey });
      await this.client.waitForTask(task.taskUid);
    }

    // Configure skills index
    const skillsIndex = this.client.index(SKILLS_INDEX);
    await this.waitForSettings(skillsIndex, {
      searchableAttributes: ["name", "description"],
      filterableAttributes: ["project", "ai_intent", "ai_topics", "created_at", "updated_at"],
      sortableAttributes: ["updated_at", "created_at"],
    });

    // Configure memories index
    const memoriesIndex = this.client.index(MEMORIES_INDEX);
    await this.waitForSettings(memoriesIndex, {
      searchableAttributes: ["content"],
      filterableAttributes: ["project", "category", "owner", "importance", "ai_intent", "ai_topics", "created_at", "updated_at"],
      sortableAttributes: ["updated_at", "created_at", "importance"],
    });

    // Configure context nodes index
    const contextIndex = this.client.index(CONTEXT_INDEX);
    await this.waitForSettings(contextIndex, {
      searchableAttributes: ["name", "abstract", "overview", "content"],
      filterableAttributes: ["project", "uri", "parent_uri", "ancestors", "is_deleted", "ai_intent", "ai_topics", "created_at", "updated_at"],
      sortableAttributes: ["updated_at", "created_at"],
    });

    // Configure embedders on all indexes
    for (const uid of [SKILLS_INDEX, MEMORIES_INDEX, CONTEXT_INDEX]) {
      const index = this.client.index(uid);
      const task = await index.updateEmbedders({
        default: { source: "userProvided", dimensions: EMBEDDING_DIM },
      } as any);
      await this.client.waitForTask(task.taskUid);
    }

    // Push synonyms
    const synonymGroups = config.meili.synonyms;
    if (synonymGroups.length > 0) {
      const synonymMap: Record<string, string[]> = {};
      for (const group of synonymGroups) {
        const words = group.split(",").map((w) => w.trim()).filter(Boolean);
        for (const word of words) {
          synonymMap[word] = words.filter((w) => w !== word);
        }
      }
      for (const uid of [SKILLS_INDEX, MEMORIES_INDEX, CONTEXT_INDEX]) {
        const task = await this.client.index(uid).updateSynonyms(synonymMap);
        await this.client.waitForTask(task.taskUid);
      }
    }
  }

  private async waitForSettings(index: Index, settings: {
    searchableAttributes?: string[];
    filterableAttributes?: string[];
    sortableAttributes?: string[];
  }) {
    if (settings.searchableAttributes) {
      const task = await index.updateSearchableAttributes(settings.searchableAttributes);
      await this.client.waitForTask(task.taskUid);
    }
    if (settings.filterableAttributes) {
      const task = await index.updateFilterableAttributes(settings.filterableAttributes);
      await this.client.waitForTask(task.taskUid);
    }
    if (settings.sortableAttributes) {
      const task = await index.updateSortableAttributes(settings.sortableAttributes);
      await this.client.waitForTask(task.taskUid);
    }
  }

  async resetIndices() {
    for (const uid of [SKILLS_INDEX, MEMORIES_INDEX, CONTEXT_INDEX]) {
      try {
        const task = await this.client.deleteIndex(uid);
        await this.client.waitForTask(task.taskUid);
      } catch (e: any) {
        if (e?.code !== "index_not_found") throw e;
      }
    }
  }

  /** Cluster/instance stats for the dashboard */
  async getClusterStats() {
    await this.ensureInitialized();
    const stats = await this.client.getStats();

    const indices: Record<string, any> = {};
    for (const uid of [SKILLS_INDEX, MEMORIES_INDEX, CONTEXT_INDEX]) {
      const s = stats.indexes[uid];
      indices[uid] = {
        docs: s?.numberOfDocuments ?? 0,
        deletedDocs: 0,
        sizeBytes: 0,
      };
    }

    return {
      clusterName: "meilisearch",
      status: "green",
      numberOfNodes: 1,
      activeShards: 0,
      relocatingShards: 0,
      unassignedShards: 0,
      indices,
    };
  }

  async countByProject(index: string, project: string): Promise<number> {
    await this.ensureInitialized();
    const idx = this.client.index(index);
    const res = await idx.search("", {
      filter: `project = "${this.escapeFilterValue(project)}"`,
      limit: 0,
    });
    return res.estimatedTotalHits ?? 0;
  }

  async bulkIndex(ops: Array<{ index: string; id: string; body: object }>): Promise<{
    successful: number;
    failed: number;
    errors: Array<{ id: string; error: string }>;
  }> {
    if (ops.length === 0) return { successful: 0, failed: 0, errors: [] };

    // Group by index
    const byIndex = new Map<string, Array<Record<string, any>>>();
    for (const op of ops) {
      const docs = byIndex.get(op.index) ?? [];
      docs.push({ ...op.body });
      byIndex.set(op.index, docs);
    }

    const errors: Array<{ id: string; error: string }> = [];
    let successful = 0;

    for (const [indexUid, docs] of byIndex) {
      const index = this.client.index(indexUid);
      const task = await index.addDocuments(docs);
      const result = await this.client.waitForTask(task.taskUid);
      if (result.status === "succeeded") {
        successful += docs.length;
      } else {
        for (const doc of docs) {
          errors.push({ id: doc.id || doc.uri || "unknown", error: result.error?.message || "Unknown error" });
        }
      }
    }

    return { successful, failed: errors.length, errors };
  }

  // ---------------------------------------------------------------------------
  // Skills
  // ---------------------------------------------------------------------------

  async addSkill(skill: AgentSkill, embedding: number[]) {
    await this.ensureInitialized();
    assertEmbeddingDimension(embedding, "MeilisearchDB.addSkill");
    const ts = new Date().toISOString();
    const index = this.client.index(SKILLS_INDEX);
    const doc = {
      id: skill.id,
      project: skill.project || null,
      name: skill.name,
      description: skill.description,
      ai_intent: skill.ai_intent ?? null,
      ai_topics: skill.ai_topics ?? null,
      ai_quality_score: skill.ai_quality_score ?? null,
      _vectors: { default: embedding },
      metadata: skill.metadata || null,
      created_at: skill.created_at || ts,
      updated_at: skill.updated_at || ts,
    };
    const task = await index.addDocuments([doc]);
    await this.client.waitForTask(task.taskUid);
  }

  async updateSkill(
    id: string,
    updates: {
      name?: string;
      description?: string;
      ai_intent?: AgentSkill["ai_intent"];
      ai_topics?: AgentSkill["ai_topics"];
      ai_quality_score?: AgentSkill["ai_quality_score"];
      metadata?: Record<string, any>;
    },
    embedding?: number[]
  ) {
    await this.ensureInitialized();
    const doc: Record<string, any> = { id, updated_at: new Date().toISOString() };
    if (updates.name !== undefined) doc.name = updates.name;
    if (updates.description !== undefined) doc.description = updates.description;
    if (updates.ai_intent !== undefined) doc.ai_intent = updates.ai_intent;
    if (updates.ai_topics !== undefined) doc.ai_topics = updates.ai_topics;
    if (updates.ai_quality_score !== undefined) doc.ai_quality_score = updates.ai_quality_score;
    if (updates.metadata !== undefined) doc.metadata = updates.metadata;
    if (embedding) {
      assertEmbeddingDimension(embedding, "MeilisearchDB.updateSkill");
      doc._vectors = { default: embedding };
    }
    const index = this.client.index(SKILLS_INDEX);
    const task = await index.updateDocuments([doc]);
    await this.client.waitForTask(task.taskUid);
  }

  async searchSkills(
    queryEmbedding: number[],
    queryText: string,
    options: SkillSearchOptions = {}
  ): Promise<(AgentSkill & { _score: number; _highlight?: Record<string, string[]> })[]> {
    await this.ensureInitialized();
    assertEmbeddingDimension(queryEmbedding, "MeilisearchDB.searchSkills");
    const topK = options.topK ?? 10;
    const ow = options.weights ?? DEFAULT_SKILL_WEIGHTS;
    const w = normalizeWeights({
      vector: ow.vector, keyword: ow.keyword,
      recency: ow.recency ?? 0, importance: 0,
    });

    const filters = this.buildSkillFilters(options);
    const semanticRatio = w.vector / (w.vector + w.keyword);
    const fetchLimit = topK * CANDIDATE_MULTIPLIER;

    const searchParams: any = {
      vector: queryEmbedding,
      hybrid: { semanticRatio, embedder: "default" },
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit: fetchLimit,
      showRankingScore: true,
    };

    if (options.highlight) {
      searchParams.attributesToHighlight = ["name", "description"];
      searchParams.highlightPreTag = "<mark>";
      searchParams.highlightPostTag = "</mark>";
    }

    if (options.minScore) {
      searchParams.rankingScoreThreshold = options.minScore;
    }

    const index = this.client.index(SKILLS_INDEX);
    const res = await index.search(queryText, searchParams);

    return this.rerankAndMap<AgentSkill>(res.hits, w, topK, options);
  }

  async searchSkillsByVector(
    queryEmbedding: number[],
    options: { topK?: number; project?: string } = {}
  ): Promise<(AgentSkill & { _score: number })[]> {
    await this.ensureInitialized();
    assertEmbeddingDimension(queryEmbedding, "MeilisearchDB.searchSkillsByVector");
    const topK = options.topK ?? 10;
    const filters: string[] = [];
    if (options.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);

    const index = this.client.index(SKILLS_INDEX);
    const res = await index.search("", {
      vector: queryEmbedding,
      hybrid: { semanticRatio: 1.0, embedder: "default" },
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit: topK,
      showRankingScore: true,
    } as any);

    return res.hits.map((hit: any) => ({
      ...this.stripVectors(hit),
      _score: hit._rankingScore ?? 0,
    }));
  }

  async listSkills(options?: SkillSearchOptions, limit = 100, offset = 0): Promise<AgentSkill[]> {
    await this.ensureInitialized();
    const filters: string[] = [];
    if (options?.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);

    const index = this.client.index(SKILLS_INDEX);
    const res = await index.getDocuments({
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit,
      offset,
      fields: ["id", "project", "name", "description", "ai_intent", "ai_topics", "ai_quality_score", "metadata", "created_at", "updated_at"],
    });
    return res.results.map((doc: any) => this.stripVectors(doc));
  }

  async getSkill(id: string): Promise<AgentSkill | null> {
    await this.ensureInitialized();
    try {
      const index = this.client.index(SKILLS_INDEX);
      const doc = await index.getDocument(id, {
        fields: ["id", "project", "name", "description", "ai_intent", "ai_topics", "ai_quality_score", "metadata", "created_at", "updated_at"],
      });
      return doc as AgentSkill;
    } catch (e: any) {
      if (e?.code === "document_not_found") return null;
      throw e;
    }
  }

  async deleteSkill(id: string) {
    await this.ensureInitialized();
    try {
      const index = this.client.index(SKILLS_INDEX);
      const task = await index.deleteDocument(id);
      await this.client.waitForTask(task.taskUid);
    } catch (e: any) {
      if (e?.code !== "document_not_found") throw e;
    }
  }

  private buildSkillFilters(options: SkillSearchOptions): string[] {
    const filters: string[] = [];
    if (options.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);
    if (options.maxAgeDays) {
      const cutoff = new Date(Date.now() - options.maxAgeDays * 24 * 60 * 60 * 1000).toISOString();
      filters.push(`created_at >= "${cutoff}"`);
    }
    return filters;
  }

  // ---------------------------------------------------------------------------
  // Helpers (continued in next tasks)
  // ---------------------------------------------------------------------------

  private rerankAndMap<T>(
    hits: any[],
    weights: { vector: number; keyword: number; recency: number; importance: number },
    topK: number,
    options: { recencyScale?: string; recencyDecay?: number; highlight?: boolean } = {}
  ): (T & { _score: number; _highlight?: Record<string, string[]> })[] {
    const now = Date.now();
    const scaleMs = parseDurationMs(options.recencyScale || config.meili.recencyScale);
    const decay = options.recencyDecay ?? config.meili.recencyDecay;

    const scored = hits.map((hit: any) => {
      let score = hit._rankingScore ?? 0;

      // Recency decay
      if (weights.recency > 0 && hit.created_at) {
        const ageMs = now - new Date(hit.created_at).getTime();
        const recencyScore = Math.pow(decay, ageMs / scaleMs);
        score += recencyScore * weights.recency;
      }

      // Importance boost
      if (weights.importance > 0 && hit.importance != null) {
        score += (hit.importance / 10) * weights.importance;
      }

      // AI quality boost
      if (hit.ai_quality_score != null && hit.ai_quality_score > 0) {
        score += (hit.ai_quality_score / 10) * AI_QUALITY_FUNCTION_WEIGHT * 0.1;
      }

      const result: any = {
        ...this.stripVectors(hit),
        _score: score,
      };

      // Map highlights
      if (options.highlight && hit._formatted) {
        const highlight: Record<string, string[]> = {};
        for (const [key, val] of Object.entries(hit._formatted)) {
          if (typeof val === "string" && val.includes("<mark>")) {
            highlight[key] = [val];
          }
        }
        if (Object.keys(highlight).length > 0) {
          result._highlight = highlight;
        }
      }

      return result;
    });

    return scored
      .sort((a: any, b: any) => b._score - a._score)
      .slice(0, topK);
  }

  private stripVectors(doc: any): any {
    const { _vectors, _rankingScore, _formatted, _matchesPosition, ...rest } = doc;
    return rest;
  }

  private escapeFilterValue(value: string): string {
    return value.replace(/"/g, '\\"');
  }
}
```

- [ ] **Step 2: Run typecheck on the new file**

Run: `bun run typecheck 2>&1 | head -30`

Expected: `meilisearchDB.ts` itself should have no type errors. Other files will have errors (they still import `ElasticDB`).

- [ ] **Step 3: Commit**

```bash
git add src/storage/meilisearchDB.ts
git commit -m "feat: add MeilisearchDB with core structure, init, and skills CRUD"
```

---

### Task 4: MeilisearchDB — Memories CRUD & Search

**Files:**
- Modify: `src/storage/meilisearchDB.ts`

- [ ] **Step 1: Add memories methods to MeilisearchDB**

Add the following methods to the `MeilisearchDB` class, after the `deleteSkill` method and before the `buildSkillFilters` method:

```typescript
  // ---------------------------------------------------------------------------
  // Memories
  // ---------------------------------------------------------------------------

  async addMemory(memory: AgentMemory, embedding: number[]) {
    await this.ensureInitialized();
    assertEmbeddingDimension(embedding, "MeilisearchDB.addMemory");
    const ts = new Date().toISOString();
    const index = this.client.index(MEMORIES_INDEX);
    const doc = {
      id: memory.id,
      project: memory.project || null,
      content: memory.content,
      category: memory.category,
      owner: memory.owner,
      importance: memory.importance,
      ai_intent: memory.ai_intent ?? null,
      ai_topics: memory.ai_topics ?? null,
      ai_quality_score: memory.ai_quality_score ?? null,
      _vectors: { default: embedding },
      metadata: memory.metadata || null,
      created_at: memory.created_at || ts,
      updated_at: memory.updated_at || ts,
    };
    const task = await index.addDocuments([doc]);
    await this.client.waitForTask(task.taskUid);
  }

  async updateMemory(
    id: string,
    updates: {
      content?: string;
      category?: MemoryCategory;
      importance?: number;
      ai_intent?: AgentMemory["ai_intent"];
      ai_topics?: AgentMemory["ai_topics"];
      ai_quality_score?: AgentMemory["ai_quality_score"];
      metadata?: Record<string, any>;
    },
    embedding?: number[]
  ) {
    await this.ensureInitialized();
    const doc: Record<string, any> = { id, updated_at: new Date().toISOString() };
    if (updates.content !== undefined) doc.content = updates.content;
    if (updates.category !== undefined) doc.category = updates.category;
    if (updates.importance !== undefined) doc.importance = updates.importance;
    if (updates.ai_intent !== undefined) doc.ai_intent = updates.ai_intent;
    if (updates.ai_topics !== undefined) doc.ai_topics = updates.ai_topics;
    if (updates.ai_quality_score !== undefined) doc.ai_quality_score = updates.ai_quality_score;
    if (updates.metadata !== undefined) doc.metadata = updates.metadata;
    if (embedding) {
      assertEmbeddingDimension(embedding, "MeilisearchDB.updateMemory");
      doc._vectors = { default: embedding };
    }
    const index = this.client.index(MEMORIES_INDEX);
    const task = await index.updateDocuments([doc]);
    await this.client.waitForTask(task.taskUid);
  }

  async searchMemories(
    queryEmbedding: number[],
    queryText: string,
    options: MemorySearchOptions = {}
  ): Promise<(AgentMemory & { _score: number; _highlight?: Record<string, string[]> })[]> {
    await this.ensureInitialized();
    assertEmbeddingDimension(queryEmbedding, "MeilisearchDB.searchMemories");
    const topK = options.topK ?? 10;
    const ow = options.weights ?? DEFAULT_MEMORY_WEIGHTS;
    const w = normalizeWeights({
      vector: ow.vector, keyword: ow.keyword,
      recency: ow.recency ?? 0, importance: ow.importance ?? 0,
    });

    const filters = this.buildMemoryFilters(options);
    const semanticRatio = w.vector / (w.vector + w.keyword);
    const fetchLimit = topK * CANDIDATE_MULTIPLIER;

    const searchParams: any = {
      vector: queryEmbedding,
      hybrid: { semanticRatio, embedder: "default" },
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit: fetchLimit,
      showRankingScore: true,
    };

    if (options.highlight) {
      searchParams.attributesToHighlight = ["content"];
      searchParams.highlightPreTag = "<mark>";
      searchParams.highlightPostTag = "</mark>";
    }

    if (options.minScore) {
      searchParams.rankingScoreThreshold = options.minScore;
    }

    const index = this.client.index(MEMORIES_INDEX);
    const res = await index.search(queryText, searchParams);

    return this.rerankAndMap<AgentMemory>(res.hits, w, topK, options);
  }

  async searchMemoriesByVector(
    queryEmbedding: number[],
    options: { topK?: number; project?: string } = {}
  ): Promise<(AgentMemory & { _score: number })[]> {
    await this.ensureInitialized();
    assertEmbeddingDimension(queryEmbedding, "MeilisearchDB.searchMemoriesByVector");
    const topK = options.topK ?? 10;
    const filters: string[] = [];
    if (options.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);

    const index = this.client.index(MEMORIES_INDEX);
    const res = await index.search("", {
      vector: queryEmbedding,
      hybrid: { semanticRatio: 1.0, embedder: "default" },
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit: topK,
      showRankingScore: true,
    } as any);

    return res.hits.map((hit: any) => ({
      ...this.stripVectors(hit),
      _score: hit._rankingScore ?? 0,
    }));
  }

  async listMemories(options?: MemorySearchOptions, limit = 100, offset = 0): Promise<AgentMemory[]> {
    await this.ensureInitialized();
    const filters: string[] = [];
    if (options?.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);

    const index = this.client.index(MEMORIES_INDEX);
    const res = await index.getDocuments({
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit,
      offset,
      fields: ["id", "project", "content", "category", "owner", "importance", "ai_intent", "ai_topics", "ai_quality_score", "metadata", "created_at", "updated_at"],
    });
    return res.results.map((doc: any) => this.stripVectors(doc));
  }

  async getMemory(id: string): Promise<AgentMemory | null> {
    await this.ensureInitialized();
    try {
      const index = this.client.index(MEMORIES_INDEX);
      const doc = await index.getDocument(id, {
        fields: ["id", "project", "content", "category", "owner", "importance", "ai_intent", "ai_topics", "ai_quality_score", "metadata", "created_at", "updated_at"],
      });
      return doc as AgentMemory;
    } catch (e: any) {
      if (e?.code === "document_not_found") return null;
      throw e;
    }
  }

  async deleteMemory(id: string) {
    await this.ensureInitialized();
    try {
      const index = this.client.index(MEMORIES_INDEX);
      const task = await index.deleteDocument(id);
      await this.client.waitForTask(task.taskUid);
    } catch (e: any) {
      if (e?.code !== "document_not_found") throw e;
    }
  }

  private buildMemoryFilters(options: MemorySearchOptions): string[] {
    const filters: string[] = [];
    if (options.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);
    if (options.owner) filters.push(`owner = "${this.escapeFilterValue(options.owner)}"`);
    if (options.category) filters.push(`category = "${this.escapeFilterValue(options.category)}"`);
    if (options.minImportance) filters.push(`importance >= ${options.minImportance}`);
    if (options.maxAgeDays) {
      const cutoff = new Date(Date.now() - options.maxAgeDays * 24 * 60 * 60 * 1000).toISOString();
      filters.push(`created_at >= "${cutoff}"`);
    }
    return filters;
  }
```

- [ ] **Step 2: Run typecheck**

Run: `bun run typecheck 2>&1 | grep meilisearchDB`

Expected: No errors in `meilisearchDB.ts`.

- [ ] **Step 3: Commit**

```bash
git add src/storage/meilisearchDB.ts
git commit -m "feat: add memories CRUD and search to MeilisearchDB"
```

---

### Task 5: MeilisearchDB — Context Nodes CRUD, Search, and Tree Queries

**Files:**
- Modify: `src/storage/meilisearchDB.ts`

- [ ] **Step 1: Add context node methods to MeilisearchDB**

Add the following methods to `MeilisearchDB`, after the `buildMemoryFilters` method and before the `rerankAndMap` helper:

```typescript
  // ---------------------------------------------------------------------------
  // Context Nodes
  // ---------------------------------------------------------------------------

  async addContextNode(node: AgentContextNode, embedding: number[]) {
    await this.ensureInitialized();
    assertEmbeddingDimension(embedding, "MeilisearchDB.addContextNode");
    const ts = new Date().toISOString();
    const ancestors = await this.computeAncestors(node.parent_uri);

    const index = this.client.index(CONTEXT_INDEX);
    const doc = {
      uri: node.uri,
      project: node.project || null,
      parent_uri: node.parent_uri,
      ancestors,
      name: node.name,
      abstract: node.abstract,
      overview: node.overview || null,
      content: node.content || null,
      ai_intent: node.ai_intent ?? null,
      ai_topics: node.ai_topics ?? null,
      ai_quality_score: node.ai_quality_score ?? null,
      _vectors: { default: embedding },
      metadata: node.metadata || null,
      created_at: node.created_at || ts,
      updated_at: node.updated_at || ts,
      is_deleted: false,
      deleted_at: null,
      version_history: [],
    };
    const task = await index.addDocuments([doc]);
    await this.client.waitForTask(task.taskUid);
  }

  async updateContextNode(
    uri: string,
    updates: {
      name?: string;
      abstract?: string;
      overview?: string;
      content?: string;
      ai_intent?: AgentContextNode["ai_intent"];
      ai_topics?: AgentContextNode["ai_topics"];
      ai_quality_score?: AgentContextNode["ai_quality_score"];
      metadata?: Record<string, any>;
    },
    embedding?: number[]
  ) {
    await this.ensureInitialized();

    // Read-modify-write for version history
    const existingNode = await this.getContextNodeRaw(uri);

    if (existingNode) {
      const historyEntry = {
        updated_at: existingNode.updated_at || existingNode.created_at,
        name: existingNode.name,
        abstract: existingNode.abstract,
        overview: existingNode.overview || null,
        content: existingNode.content || null,
      };

      let versionHistory: any[] = existingNode.version_history || [];
      versionHistory.push(historyEntry);
      if (versionHistory.length > 10) {
        versionHistory = versionHistory.slice(-10);
      }

      const doc: Record<string, any> = {
        uri,
        updated_at: new Date().toISOString(),
        version_history: versionHistory,
      };
      if (updates.name !== undefined) doc.name = updates.name;
      if (updates.abstract !== undefined) doc.abstract = updates.abstract;
      if (updates.overview !== undefined) doc.overview = updates.overview;
      if (updates.content !== undefined) doc.content = updates.content;
      if (updates.ai_intent !== undefined) doc.ai_intent = updates.ai_intent;
      if (updates.ai_topics !== undefined) doc.ai_topics = updates.ai_topics;
      if (updates.ai_quality_score !== undefined) doc.ai_quality_score = updates.ai_quality_score;
      if (updates.metadata !== undefined) doc.metadata = updates.metadata;
      if (embedding) {
        assertEmbeddingDimension(embedding, "MeilisearchDB.updateContextNode");
        doc._vectors = { default: embedding };
      }

      const index = this.client.index(CONTEXT_INDEX);
      const task = await index.updateDocuments([doc]);
      await this.client.waitForTask(task.taskUid);
    } else {
      // Node doesn't exist yet — partial update
      const doc: Record<string, any> = { uri, updated_at: new Date().toISOString() };
      if (updates.name !== undefined) doc.name = updates.name;
      if (updates.abstract !== undefined) doc.abstract = updates.abstract;
      if (updates.overview !== undefined) doc.overview = updates.overview;
      if (updates.content !== undefined) doc.content = updates.content;
      if (updates.ai_intent !== undefined) doc.ai_intent = updates.ai_intent;
      if (updates.ai_topics !== undefined) doc.ai_topics = updates.ai_topics;
      if (updates.ai_quality_score !== undefined) doc.ai_quality_score = updates.ai_quality_score;
      if (updates.metadata !== undefined) doc.metadata = updates.metadata;
      if (embedding) {
        assertEmbeddingDimension(embedding, "MeilisearchDB.updateContextNode");
        doc._vectors = { default: embedding };
      }
      const index = this.client.index(CONTEXT_INDEX);
      const task = await index.updateDocuments([doc]);
      await this.client.waitForTask(task.taskUid);
    }
  }

  async searchContextNodes(
    queryEmbedding: number[],
    queryText: string,
    options: ContextSearchOptions = {}
  ): Promise<(AgentContextNode & { _score: number; _highlight?: Record<string, string[]> })[]> {
    await this.ensureInitialized();
    assertEmbeddingDimension(queryEmbedding, "MeilisearchDB.searchContextNodes");
    const topK = options.topK ?? 10;
    const ow = options.weights ?? DEFAULT_CONTEXT_WEIGHTS;
    const w = normalizeWeights({
      vector: ow.vector, keyword: ow.keyword,
      recency: ow.recency ?? 0, importance: 0,
    });

    const filters = this.buildContextFilters(options);
    const semanticRatio = w.vector / (w.vector + w.keyword);
    const fetchLimit = topK * CANDIDATE_MULTIPLIER;

    const searchParams: any = {
      vector: queryEmbedding,
      hybrid: { semanticRatio, embedder: "default" },
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit: fetchLimit,
      showRankingScore: true,
    };

    if (options.highlight) {
      searchParams.attributesToHighlight = ["name", "abstract", "overview", "content"];
      searchParams.highlightPreTag = "<mark>";
      searchParams.highlightPostTag = "</mark>";
    }

    if (options.minScore) {
      searchParams.rankingScoreThreshold = options.minScore;
    }

    const index = this.client.index(CONTEXT_INDEX);
    const res = await index.search(queryText, searchParams);

    return this.rerankAndMap<AgentContextNode>(res.hits, w, topK, options);
  }

  async searchContextNodesByVector(
    queryEmbedding: number[],
    options: { topK?: number; project?: string; includeDeleted?: boolean } = {}
  ): Promise<(AgentContextNode & { _score: number })[]> {
    await this.ensureInitialized();
    assertEmbeddingDimension(queryEmbedding, "MeilisearchDB.searchContextNodesByVector");
    const topK = options.topK ?? 10;
    const filters: string[] = [];
    if (options.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);
    if (!options.includeDeleted) filters.push(`is_deleted != true`);

    const index = this.client.index(CONTEXT_INDEX);
    const res = await index.search("", {
      vector: queryEmbedding,
      hybrid: { semanticRatio: 1.0, embedder: "default" },
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit: topK,
      showRankingScore: true,
    } as any);

    return res.hits.map((hit: any) => ({
      ...this.stripVectors(hit),
      _score: hit._rankingScore ?? 0,
    }));
  }

  async listContextNodes(parentUri?: string, options?: ContextSearchOptions, limit = 100, offset = 0): Promise<AgentContextNode[]> {
    await this.ensureInitialized();
    const filters: string[] = [];
    if (options?.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);
    if (parentUri) filters.push(`parent_uri = "${this.escapeFilterValue(parentUri)}"`);
    if (!options?.includeDeleted) filters.push(`is_deleted != true`);

    const index = this.client.index(CONTEXT_INDEX);
    const res = await index.getDocuments({
      filter: filters.length > 0 ? filters.join(" AND ") : undefined,
      limit,
      offset,
    });
    return res.results.map((doc: any) => this.stripVectors(doc));
  }

  async getContextNode(uri: string): Promise<AgentContextNode | null> {
    await this.ensureInitialized();
    try {
      const index = this.client.index(CONTEXT_INDEX);
      const doc = await index.getDocument(uri);
      const { _vectors, ancestors, ...rest } = doc as any;
      return rest as AgentContextNode;
    } catch (e: any) {
      if (e?.code === "document_not_found") return null;
      throw e;
    }
  }

  async deleteContextNode(uri: string) {
    await this.ensureInitialized();
    const ts = new Date().toISOString();

    // Soft delete descendants
    const descendants = await this.getDescendants(uri);
    if (descendants.length > 0) {
      const updates = descendants.map((d: any) => ({
        uri: d.uri,
        is_deleted: true,
        deleted_at: ts,
      }));
      const index = this.client.index(CONTEXT_INDEX);
      const task = await index.updateDocuments(updates);
      await this.client.waitForTask(task.taskUid);
    }

    // Soft delete the node itself
    try {
      const index = this.client.index(CONTEXT_INDEX);
      const task = await index.updateDocuments([{ uri, is_deleted: true, deleted_at: ts }]);
      await this.client.waitForTask(task.taskUid);
    } catch (e: any) {
      if (e?.code !== "document_not_found") throw e;
    }
  }

  async restoreContextNode(uri: string) {
    await this.ensureInitialized();

    // Restore descendants
    const descendants = await this.getDescendants(uri);
    if (descendants.length > 0) {
      const updates = descendants.map((d: any) => ({
        uri: d.uri,
        is_deleted: false,
        deleted_at: null,
      }));
      const index = this.client.index(CONTEXT_INDEX);
      const task = await index.updateDocuments(updates);
      await this.client.waitForTask(task.taskUid);
    }

    // Restore the node itself
    try {
      const index = this.client.index(CONTEXT_INDEX);
      const task = await index.updateDocuments([{ uri, is_deleted: false, deleted_at: null }]);
      await this.client.waitForTask(task.taskUid);
    } catch (e: any) {
      if (e?.code !== "document_not_found") throw e;
    }
  }

  async getContextSubtree(nodeUri: string, includeDeleted = false): Promise<(AgentContextNode & { depth: number })[]> {
    await this.ensureInitialized();
    const filters: string[] = [`ancestors = "${this.escapeFilterValue(nodeUri)}"`];
    if (!includeDeleted) filters.push(`is_deleted != true`);

    const index = this.client.index(CONTEXT_INDEX);

    // Fetch root node
    let rootNode: any = null;
    try {
      rootNode = await index.getDocument(nodeUri);
    } catch (e: any) {
      if (e?.code === "document_not_found") return [];
      throw e;
    }

    if (!includeDeleted && rootNode.is_deleted) return [];

    // Fetch descendants
    const res = await index.search("", {
      filter: filters.join(" AND "),
      limit: 1000,
    } as any);

    const rootDepth = rootNode.ancestors?.length ?? 0;
    const allNodes = [rootNode, ...res.hits];

    return allNodes
      .map((n: any) => {
        const nodeAncestors: string[] = n.ancestors || [];
        const depth = nodeAncestors.length - rootDepth;
        const { _vectors, ancestors, _rankingScore, _formatted, _matchesPosition, ...rest } = n;
        return { ...rest, depth } as AgentContextNode & { depth: number };
      })
      .sort((a: any, b: any) => a.depth - b.depth);
  }

  async getContextPath(nodeUri: string, includeDeleted = false): Promise<(AgentContextNode & { depth: number })[]> {
    await this.ensureInitialized();
    const node = await this.getContextNodeWithAncestors(nodeUri);
    if (!node) return [];

    const ancestorUris: string[] = (node as any).ancestors || [];
    if (ancestorUris.length === 0) {
      if (!includeDeleted && node.is_deleted) return [];
      const { ancestors: _, _vectors: _v, ...rest } = node as any;
      return [{ ...rest, depth: 0 }];
    }

    // Fetch all ancestor nodes by URI
    const index = this.client.index(CONTEXT_INDEX);
    const allUris = [...ancestorUris, nodeUri];

    const filters: string[] = [
      allUris.map((u) => `uri = "${this.escapeFilterValue(u)}"`).join(" OR "),
    ];
    if (!includeDeleted) filters.push(`is_deleted != true`);

    const res = await index.search("", {
      filter: filters.join(" AND "),
      limit: allUris.length,
    } as any);

    const allNodes = res.hits as any[];
    return allNodes
      .map((n: any) => {
        const nodeAncestors: string[] = n.ancestors || [];
        const depth = nodeAncestors.length;
        const { ancestors: _, _vectors, _rankingScore, _formatted, _matchesPosition, ...rest } = n;
        return { ...rest, depth } as AgentContextNode & { depth: number };
      })
      .sort((a: any, b: any) => a.depth - b.depth);
  }

  private async getDescendants(uri: string): Promise<any[]> {
    const index = this.client.index(CONTEXT_INDEX);
    const res = await index.search("", {
      filter: `ancestors = "${this.escapeFilterValue(uri)}"`,
      limit: 1000,
    } as any);
    return res.hits;
  }

  private async computeAncestors(parentUri: string | null): Promise<string[]> {
    if (!parentUri) return [];
    const parent = await this.getContextNodeWithAncestors(parentUri);
    if (!parent) return [parentUri];
    const parentAncestors: string[] = (parent as any).ancestors || [];
    return [...parentAncestors, parentUri];
  }

  private async getContextNodeWithAncestors(uri: string): Promise<(AgentContextNode & { ancestors?: string[] }) | null> {
    try {
      const index = this.client.index(CONTEXT_INDEX);
      const doc = await index.getDocument(uri);
      const { _vectors, ...rest } = doc as any;
      return rest as AgentContextNode & { ancestors?: string[] };
    } catch (e: any) {
      if (e?.code === "document_not_found") return null;
      throw e;
    }
  }

  /** Internal: get context node with all fields including ancestors */
  private async getContextNodeRaw(uri: string): Promise<any | null> {
    try {
      const index = this.client.index(CONTEXT_INDEX);
      return await index.getDocument(uri);
    } catch (e: any) {
      if (e?.code === "document_not_found") return null;
      throw e;
    }
  }

  private buildContextFilters(options: ContextSearchOptions): string[] {
    const filters: string[] = [];
    if (options.project) filters.push(`project = "${this.escapeFilterValue(options.project)}"`);
    if (options.parentUri) filters.push(`parent_uri = "${this.escapeFilterValue(options.parentUri)}"`);
    if (options.maxAgeDays) {
      const cutoff = new Date(Date.now() - options.maxAgeDays * 24 * 60 * 60 * 1000).toISOString();
      filters.push(`created_at >= "${cutoff}"`);
    }
    if (!options.includeDeleted) filters.push(`is_deleted != true`);
    return filters;
  }
```

- [ ] **Step 2: Run typecheck**

Run: `bun run typecheck 2>&1 | grep meilisearchDB`

Expected: No errors in `meilisearchDB.ts`.

- [ ] **Step 3: Commit**

```bash
git add src/storage/meilisearchDB.ts
git commit -m "feat: add context nodes CRUD, search, and tree queries to MeilisearchDB"
```

---

### Task 6: Rewire Consumers — ContextManager, BatchWriter, Client, Setup

**Files:**
- Modify: `src/storage/contextManager.ts:1-2,29-30`
- Modify: `src/storage/batchWriter.ts:1-2,42-48`
- Modify: `src/storage/client.ts`
- Modify: `src/scripts/setup.ts`
- Modify: `src/index.ts`

- [ ] **Step 1: Update contextManager.ts imports**

In `src/storage/contextManager.ts`:

Replace line 2:
```typescript
import { ElasticDB, MEMORIES_INDEX, SKILLS_INDEX, CONTEXT_INDEX } from "./elasticDB";
```
with:
```typescript
import { MeilisearchDB, MEMORIES_INDEX, SKILLS_INDEX, CONTEXT_INDEX } from "./meilisearchDB";
```

Replace lines 27-30 (the class field + constructor):
```typescript
export class ContextManager {
  private db: ElasticDB;

  constructor(node: string, auth?: { username: string; password: string }) {
    this.db = new ElasticDB(node, auth);
  }
```
with:
```typescript
export class ContextManager {
  private db: MeilisearchDB;

  constructor(url: string, apiKey?: string) {
    this.db = new MeilisearchDB(url, apiKey);
  }
```

- [ ] **Step 2: Update batchWriter.ts imports**

In `src/storage/batchWriter.ts`:

Replace line 2:
```typescript
import { ElasticDB, MEMORIES_INDEX, SKILLS_INDEX, CONTEXT_INDEX } from "./elasticDB";
```
with:
```typescript
import { MeilisearchDB, MEMORIES_INDEX, SKILLS_INDEX, CONTEXT_INDEX } from "./meilisearchDB";
```

Replace `ElasticDB` with `MeilisearchDB` in the class (lines 42-49):
```typescript
export class BatchWriter {
  private db: MeilisearchDB;
  private queue: BatchOp[] = [];
  private readonly batchSize: number;
  private flushTimer: ReturnType<typeof setInterval> | null = null;
  private readonly flushIntervalMs: number;

  constructor(db: MeilisearchDB, options: BatchWriterOptions = {}) {
    this.db = db;
```

- [ ] **Step 3: Update client.ts**

Replace the full contents of `src/storage/client.ts`:

```typescript
import { ContextManager } from "./contextManager";
import { config } from "../core/config";

export function createContextManager(): ContextManager {
  const url = config.meili.url;

  if (!url) {
    throw new Error("Please set MEILI_URL in your .env file or environment.");
  }

  return new ContextManager(url, config.meili.apiKey || undefined);
}
```

- [ ] **Step 4: Update setup.ts**

Replace the full contents of `src/scripts/setup.ts`:

```typescript
import { MeilisearchDB } from "../storage/meilisearchDB";
import { config } from "../core/config";

const url = config.meili.url;

if (!url) {
  console.error("Please set MEILI_URL in your .env file");
  process.exit(1);
}

const db = new MeilisearchDB(url, config.meili.apiKey || undefined);

async function main() {
  console.log("Resetting and initializing Meilisearch indices for Agent Context...");
  try {
    await db.resetIndices();
    await db.initIndices();
    console.log("Successfully reset and initialized indices!");
  } catch (err) {
    console.error("Failed to reset/initialize indices:", err);
  }
}

main();
```

- [ ] **Step 5: Update index.ts**

Replace `src/index.ts`:

```typescript
export * from "./core/types";
export * from "./storage/meilisearchDB";
export * from "./storage/contextManager";
export * from "./storage/client";
export * from "./core/config";
export * from "./storage/scorer";
```

- [ ] **Step 6: Run typecheck**

Run: `bun run typecheck 2>&1 | head -40`

Expected: Errors only from eval files, test files, and tools that still reference `ElasticDB`. The core `src/storage/` and `src/scripts/` should be clean.

- [ ] **Step 7: Commit**

```bash
git add src/storage/contextManager.ts src/storage/batchWriter.ts src/storage/client.ts src/scripts/setup.ts src/index.ts
git commit -m "feat: rewire core consumers from ElasticDB to MeilisearchDB"
```

---

### Task 7: Rewire Eval, Tools, and Dashboard

**Files:**
- Modify: `src/eval/evaluate.ts:5,180-183,246-249`
- Modify: `src/eval/evalSeeder.ts:6,48,124`
- Modify: `tools/clear-tables.ts`
- Modify: `tools/drop-tables.ts`
- Modify: `src/dashboardApi.ts` (no changes needed — uses `createContextManager`)

- [ ] **Step 1: Update evaluate.ts**

In `src/eval/evaluate.ts`:

Replace line 5:
```typescript
import { ElasticDB } from "../storage/elasticDB";
```
with:
```typescript
import { MeilisearchDB } from "../storage/meilisearchDB";
```

Replace lines 180-183 (seed block):
```typescript
    const db = new ElasticDB(
      config.elasticUrl,
      config.elasticUsername ? { username: config.elasticUsername!, password: config.elasticPassword! } : undefined
    );
```
with:
```typescript
    const db = new MeilisearchDB(config.meili.url, config.meili.apiKey || undefined);
```

Replace lines 246-249 (cleanup block):
```typescript
    const db = new ElasticDB(
      config.elasticUrl,
      config.elasticUsername ? { username: config.elasticUsername!, password: config.elasticPassword! } : undefined
    );
```
with:
```typescript
    const db = new MeilisearchDB(config.meili.url, config.meili.apiKey || undefined);
```

- [ ] **Step 2: Update evalSeeder.ts**

In `src/eval/evalSeeder.ts`:

Replace line 6:
```typescript
import { ElasticDB } from "../storage/elasticDB";
```
with:
```typescript
import { MeilisearchDB } from "../storage/meilisearchDB";
```

Replace `ElasticDB` with `MeilisearchDB` in function signatures:

Line 48 — change `db: ElasticDB` to `db: MeilisearchDB`:
```typescript
export async function seedFixtures(db: MeilisearchDB, fixtures: FixtureSpec, log?: (msg: string) => void): Promise<void> {
```

Line 124 — change `db: ElasticDB` to `db: MeilisearchDB`:
```typescript
export async function cleanupFixtures(db: MeilisearchDB, fixtures: FixtureSpec, log?: (msg: string) => void): Promise<void> {
```

- [ ] **Step 3: Update clear-tables.ts**

Replace the full contents of `tools/clear-tables.ts`:

```typescript
import { MeiliSearch } from "meilisearch";
import { SKILLS_INDEX, MEMORIES_INDEX, CONTEXT_INDEX } from "../src/storage/meilisearchDB";
import * as dotenv from "dotenv";
dotenv.config();

const client = new MeiliSearch({
  host: process.env.MEILI_URL || "http://localhost:7700",
  apiKey: process.env.MEILI_API_KEY || "",
});

async function run() {
  for (const uid of [CONTEXT_INDEX, MEMORIES_INDEX, SKILLS_INDEX]) {
    try {
      const task = await client.index(uid).deleteAllDocuments();
      await client.waitForTask(task.taskUid);
    } catch (e: any) {
      if (e?.code !== "index_not_found") console.error(`Error clearing ${uid}:`, e);
    }
  }
  console.log("Indices cleared!");
}
run();
```

- [ ] **Step 4: Update drop-tables.ts**

Replace the full contents of `tools/drop-tables.ts`:

```typescript
import { MeilisearchDB } from "../src/storage/meilisearchDB";
import * as dotenv from "dotenv";
dotenv.config();

const db = new MeilisearchDB(
  process.env.MEILI_URL || "http://localhost:7700",
  process.env.MEILI_API_KEY || ""
);

async function run() {
  await db.resetIndices();
  console.log("Indices dropped!");
}
run();
```

- [ ] **Step 5: Run typecheck**

Run: `bun run typecheck 2>&1 | head -20`

Expected: Errors only from test files that still mock `@elastic/elasticsearch`.

- [ ] **Step 6: Commit**

```bash
git add src/eval/evaluate.ts src/eval/evalSeeder.ts tools/clear-tables.ts tools/drop-tables.ts
git commit -m "feat: rewire eval, tools from ElasticDB to MeilisearchDB"
```

---

### Task 8: Update Tests

**Files:**
- Modify: `tests/batchWriter.test.ts`
- Modify: `tests/budget.test.ts`

- [ ] **Step 1: Rewrite batchWriter.test.ts**

Replace the full contents of `tests/batchWriter.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("../src/storage/embedder", () => ({
  Embedder: {
    getEmbeddings: vi.fn().mockImplementation((texts: string[]) =>
      Promise.resolve(texts.map(() => Array(3072).fill(0)))
    ),
  },
}));

const mockBulkIndex = vi.fn().mockResolvedValue({ successful: 2, failed: 0, errors: [] });

vi.mock("meilisearch", () => ({
  MeiliSearch: vi.fn().mockImplementation(() => ({
    getStats: vi.fn().mockResolvedValue({ indexes: {} }),
    createIndex: vi.fn().mockResolvedValue({ taskUid: 0 }),
    waitForTask: vi.fn().mockResolvedValue({ status: "succeeded" }),
    deleteIndex: vi.fn().mockResolvedValue({ taskUid: 0 }),
    index: vi.fn().mockReturnValue({
      search: vi.fn().mockResolvedValue({ hits: [], estimatedTotalHits: 0 }),
      getDocument: vi.fn().mockRejectedValue({ code: "document_not_found" }),
      getDocuments: vi.fn().mockResolvedValue({ results: [] }),
      addDocuments: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateDocuments: vi.fn().mockResolvedValue({ taskUid: 0 }),
      deleteDocument: vi.fn().mockResolvedValue({ taskUid: 0 }),
      deleteAllDocuments: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateSearchableAttributes: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateFilterableAttributes: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateSortableAttributes: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateEmbedders: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateSynonyms: vi.fn().mockResolvedValue({ taskUid: 0 }),
    }),
  })),
}));

vi.mock("../src/core/config", async (importOriginal) => {
  const original = await importOriginal<typeof import("../src/core/config")>();
  return {
    ...original,
    config: {
      ...original.config,
      meili: { url: "http://localhost:7700", apiKey: "", synonyms: [], recencyScale: "30d", recencyDecay: 0.5 },
      embedding: { model: "test", dimension: 3072, allowZeroEmbeddings: true },
      geminiApiKey: "test",
      budget: { memoryPerProject: 0, skillPerProject: 0, nodePerProject: 0 },
    },
    assertEmbeddingDimension: vi.fn(),
  };
});

import { BatchWriter } from "../src/storage/batchWriter";
import { MeilisearchDB } from "../src/storage/meilisearchDB";
import { Embedder } from "../src/storage/embedder";

describe("BatchWriter", () => {
  let db: MeilisearchDB;
  let writer: BatchWriter;

  beforeEach(() => {
    vi.clearAllMocks();
    db = new MeilisearchDB("http://localhost:7700");
    db.bulkIndex = mockBulkIndex;
    writer = new BatchWriter(db, { batchSize: 3, flushIntervalMs: 50000 });
  });

  it("enqueue does not immediately write", () => {
    writer.enqueue({
      type: "memory",
      data: {
        id: "mem_1", project: "p", content: "test",
        category: "observation", owner: "agent", importance: 5,
        metadata: {}, ai_intent: null, ai_topics: null, ai_quality_score: null,
        created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
      },
    });
    expect(mockBulkIndex).not.toHaveBeenCalled();
  });

  it("flush writes all queued ops", async () => {
    writer.enqueue({
      type: "memory",
      data: {
        id: "mem_1", project: "p", content: "hello",
        category: "observation", owner: "agent", importance: 5,
        metadata: {}, ai_intent: null, ai_topics: null, ai_quality_score: null,
        created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
      },
    });
    writer.enqueue({
      type: "skill",
      data: {
        id: "skill_1", project: "p", name: "code", description: "write code",
        metadata: {}, ai_intent: null, ai_topics: null, ai_quality_score: null,
        created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
      },
    });

    const result = await writer.flush();
    expect(mockBulkIndex).toHaveBeenCalledTimes(1);
    expect(result.successful).toBe(2);
  });
});
```

- [ ] **Step 2: Rewrite budget.test.ts**

Replace the full contents of `tests/budget.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockSearch = vi.fn();

vi.mock("meilisearch", () => ({
  MeiliSearch: vi.fn().mockImplementation(() => ({
    getStats: vi.fn().mockResolvedValue({ indexes: {} }),
    createIndex: vi.fn().mockResolvedValue({ taskUid: 0 }),
    waitForTask: vi.fn().mockResolvedValue({ status: "succeeded" }),
    deleteIndex: vi.fn().mockResolvedValue({ taskUid: 0 }),
    index: vi.fn().mockReturnValue({
      search: mockSearch,
      getDocument: vi.fn().mockRejectedValue({ code: "document_not_found" }),
      getDocuments: vi.fn().mockResolvedValue({ results: [] }),
      addDocuments: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateDocuments: vi.fn().mockResolvedValue({ taskUid: 0 }),
      deleteDocument: vi.fn().mockResolvedValue({ taskUid: 0 }),
      deleteAllDocuments: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateSearchableAttributes: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateFilterableAttributes: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateSortableAttributes: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateEmbedders: vi.fn().mockResolvedValue({ taskUid: 0 }),
      updateSynonyms: vi.fn().mockResolvedValue({ taskUid: 0 }),
    }),
  })),
}));

vi.mock("../src/storage/embedder", () => ({
  Embedder: {
    getEmbedding: vi.fn().mockResolvedValue(Array(3072).fill(0)),
    getEmbeddings: vi.fn().mockResolvedValue([Array(3072).fill(0)]),
  },
}));

vi.mock("../src/llm/llmRouter", () => ({
  decideMemoryAction: vi.fn().mockResolvedValue({ action: "create" }),
  decideContextAction: vi.fn().mockResolvedValue({ action: "create" }),
}));

vi.mock("../src/core/config", async (importOriginal) => {
  const original = await importOriginal<typeof import("../src/core/config")>();
  return {
    ...original,
    config: {
      ...original.config,
      meili: { url: "http://localhost:7700", apiKey: "", synonyms: [], recencyScale: "30d", recencyDecay: 0.5 },
      embedding: { model: "test", dimension: 3072, allowZeroEmbeddings: true },
      geminiApiKey: "test",
      budget: {
        memoryPerProject: 2,
        skillPerProject: 1,
        nodePerProject: 2,
      },
    },
    assertEmbeddingDimension: vi.fn(),
  };
});

import { MeilisearchDB, MEMORIES_INDEX } from "../src/storage/meilisearchDB";
import { ContextManager } from "../src/storage/contextManager";
import type { BudgetExceeded } from "../src/core/types";

describe("countByProject", () => {
  let db: MeilisearchDB;

  beforeEach(() => {
    vi.clearAllMocks();
    db = new MeilisearchDB("http://localhost:7700");
  });

  it("returns count for a project", async () => {
    mockSearch.mockResolvedValue({ hits: [], estimatedTotalHits: 42 });
    const result = await db.countByProject(MEMORIES_INDEX, "my-project");
    expect(result).toBe(42);
  });

  it("returns 0 when no documents match", async () => {
    mockSearch.mockResolvedValue({ hits: [], estimatedTotalHits: 0 });
    const result = await db.countByProject(MEMORIES_INDEX, "empty-project");
    expect(result).toBe(0);
  });
});

describe("Budget enforcement via ContextManager", () => {
  let cm: ContextManager;

  beforeEach(() => {
    vi.clearAllMocks();
    cm = new ContextManager("http://localhost:7700");
  });

  it("blocks memory creation when budget is full", async () => {
    mockSearch.mockResolvedValue({ hits: [], estimatedTotalHits: 2 });
    const result = await cm.addMemory("new fact", "observation", "agent", 5, "test-project", {}, false);
    expect((result as BudgetExceeded).budgetExceeded).toBe(true);
    expect((result as BudgetExceeded).message).toContain("Budget full");
  });

  it("allows memory creation when under budget", async () => {
    mockSearch.mockResolvedValue({ hits: [], estimatedTotalHits: 1 });
    const result = await cm.addMemory("new fact", "observation", "agent", 5, "test-project", {}, false);
    expect((result as BudgetExceeded).budgetExceeded).toBeUndefined();
  });
});
```

- [ ] **Step 3: Run tests**

Run: `bun run test`

Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add tests/batchWriter.test.ts tests/budget.test.ts
git commit -m "test: update tests for meilisearch backend"
```

---

### Task 9: Delete ElasticDB, Clean Up CLI and Dashboard API

**Files:**
- Delete: `src/storage/elasticDB.ts`
- Modify: `src/cli.ts`
- Modify: `src/dashboardApi.ts`

- [ ] **Step 1: Remove fuzziness/phraseBoost from CLI**

In `src/cli.ts`:

1. Delete the `parseFuzziness` helper function (line 16 area).
2. Remove all `.option("--fuzziness <f>", ...)` lines (lines ~97, ~185, ~274, ~714).
3. Remove all `.option("--phraseBoost <n>", ...)` lines (lines ~98, ~186, ~275, ~715).
4. Remove all `fuzziness: parseFuzziness(opts.fuzziness),` lines from action handlers.
5. Remove all `phraseBoost: opts.phraseBoost !== undefined ? parseFloat(opts.phraseBoost) : undefined,` lines from action handlers.
6. Update the `memory search` description from `"Search memories with hybrid kNN + BM25 scoring"` to `"Search memories with hybrid vector + keyword scoring"`.

- [ ] **Step 2: Remove fuzziness/phraseBoost from dashboard API**

In `src/dashboardApi.ts`, remove the fuzziness and phraseBoost parsing (lines ~85-90):

Remove these lines:
```typescript
      // ES tuning params from query string
      const fz = parsed.searchParams.get("fuzziness");
      if (fz) tuning.fuzziness = fz === "auto" ? "auto" : Number(fz);
      const pb = parsed.searchParams.get("phraseBoost");
      if (pb) tuning.phraseBoost = Number(pb);
```

Update the comment to `// Search tuning params from query string`.

- [ ] **Step 3: Delete elasticDB.ts**

```bash
rm src/storage/elasticDB.ts
```

- [ ] **Step 3: Run typecheck**

Run: `bun run typecheck`

Expected: Clean — no errors.

- [ ] **Step 4: Run tests**

Run: `bun run test`

Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add -u
git commit -m "feat: remove ElasticDB, clean up CLI/API fuzziness/phraseBoost refs"
```

---

### Task 10: Update CLAUDE.md and Package Metadata

**Files:**
- Modify: `CLAUDE.md`
- Modify: `package.json` (metadata only — keywords, description)

- [ ] **Step 1: Update CLAUDE.md**

Apply the following changes throughout `CLAUDE.md`:

1. Replace all references to "Elasticsearch" with "Meilisearch" in descriptions
2. Replace `docker.elastic.co` references with Meilisearch Docker
3. Update the "Tech Stack" section: `**Database:** Meilisearch 1.12+ (Docker)`
4. Update `**Search:**` to: `Native hybrid — vector cosine similarity + full-text + app-side re-ranking (importance, recency decay)`
5. Replace `src/storage/elasticDB.ts` references with `src/storage/meilisearchDB.ts`
6. Update the "Elasticsearch Indices" section header to "Meilisearch Indexes"
7. Remove references to `ES_BM25_K1`, `ES_BM25_B`, `ES_DEFAULT_FUZZINESS`, `ES_PHRASE_BOOST`
8. Rename `ES_SYNONYMS` → `SYNONYMS`, `ES_RECENCY_SCALE` → `RECENCY_SCALE`, `ES_RECENCY_DECAY` → `RECENCY_DECAY`
9. Replace `ELASTIC_URL` with `MEILI_URL`, `ELASTIC_USERNAME`/`ELASTIC_PASSWORD` with `MEILI_API_KEY`
10. Update commands table: `docker compose up -d` description to "Start Meilisearch container"
11. Remove the "Elasticsearch Fine-Tuning" env section, replace with "Search Fine-Tuning" referencing only `RECENCY_SCALE`, `RECENCY_DECAY`, `SYNONYMS`
12. Update `ElasticSearchTuning` references to `SearchTuning`
13. Update the key modules table: `src/elasticDB.ts` role to reference Meilisearch
14. Update the search features table to remove `ES_` prefixed controls

- [ ] **Step 2: Update package.json metadata**

In `package.json`, update:
- `"name"` to `"agents-context-meili"` (or keep current if preferred)
- `"description"` to reference Meilisearch instead of Elasticsearch
- `"keywords"` — replace `"elasticsearch"` with `"meilisearch"`, keep the rest

- [ ] **Step 3: Run full validation**

```bash
bun run typecheck && bun run test && bun run lint
```

Expected: All pass.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md package.json
git commit -m "docs: update CLAUDE.md and package metadata for meilisearch"
```

---

### Task 11: Smoke Test — End-to-End Verification

**Files:** None (verification only)

- [ ] **Step 1: Ensure Meilisearch is running**

Run:
```bash
docker compose up -d
curl http://localhost:7700/health
```
Expected: `{"status":"available"}`

- [ ] **Step 2: Run setup**

Run: `bun run setup`

Expected: "Successfully reset and initialized indices!"

- [ ] **Step 3: Test CLI operations**

Run:
```bash
# Store a memory
bun src/cli.ts memory store "Meilisearch migration is complete" -c observation -o agent -i 5 -P test-meili

# Search for it
bun src/cli.ts memory search "meilisearch migration" -k 3 -P test-meili

# Store a context node
bun src/cli.ts node store "contextfs://test-meili/backend" "Backend" "The backend module" -P test-meili

# Search for it
bun src/cli.ts node search "backend module" -k 3 -P test-meili
```

Expected: Each command succeeds, search returns the stored items.

- [ ] **Step 4: Test dashboard API**

Run: `bun run dashboard:api &`

Then:
```bash
curl http://localhost:8787/api/health
curl http://localhost:8787/api/stats
```

Expected: Health returns `{"ok":true}`, stats returns index counts.

Kill the background process after verification.

- [ ] **Step 5: Run eval (if dataset exists)**

Run: `bun run eval:retrieval -- --dataset eval/dataset.json --topK 5 --verbose true 2>&1 | head -20`

Expected: Eval runs without import errors. Results may differ from ES baseline — that's expected.
