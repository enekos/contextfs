# contextfs

A dynamic context and memory storage system for coding agents. Provides native hybrid retrieval (kNN + BM25 + function scoring) backed by Elasticsearch with Google Gemini embeddings. Exposes two interfaces: CLI and REST API (dashboard).

## Tech Stack

- **Runtime:** Bun 1+, TypeScript (ES2022, CommonJS)
- **Database:** Elasticsearch 8.17+ (Docker)
- **Search:** Native hybrid — kNN dense vector + BM25 full-text + function scoring (importance, recency decay)
- **Embeddings:** Google Gemini (`gemini-embedding-001`, 3072 dims)
- **Frontend:** Svelte 5 + Vite
- **Testing:** Vitest
- **Linting:** oxlint

## Setup

```bash
docker compose up -d    # start Elasticsearch
bun install
bun --cwd dashboard install
cp .env.example .env    # fill in ELASTIC_URL, GEMINI_API_KEY
bun run setup           # create ES indices (destructive — drops and recreates)
```

## Commands

| Command | Description |
|---|---|
| `docker compose up -d` | Start Elasticsearch container |
| `docker compose down` | Stop Elasticsearch container |
| `bun run link` | Build and install `context-cli` globally via `bun link` |
| `bun run build` | Compile TypeScript → `dist/` |
| `bun run typecheck` | Type-check without emit |
| `bun run lint` | Run oxlint on `src/` |
| `bun run test` | Run Vitest tests once |
| `bun run test:watch` | Vitest in watch mode |
| `bun run clean` | Remove `dist/` |
| `bun run setup` | Init/reset Elasticsearch indices |
| `bun run dashboard:api` | Start REST API on port 8787 |
| `bun run dashboard:dev` | Start Svelte dev server on port 5173 |
| `bun run dashboard:build` | Build Svelte UI |

### Evaluation

```bash
bun run eval:retrieval -- --dataset eval/dataset.json --topK 5 --verbose true
bun run eval:retrieval -- --dataset eval/dataset.json --topK 5 --fail-below-mrr 0.8 --fail-below-recall 0.75
```

## Architecture

### Data Types

- **Memories** — facts with category, owner, importance (1–10)
- **Skills** — capability name + description pairs
- **Context Nodes** — hierarchical tree nodes with abstract/overview/content levels, addressed by URI

### Retrieval Pipeline

Elasticsearch handles all scoring natively in a single query:

1. **kNN** — dense vector cosine similarity on Gemini embeddings
2. **BM25** — full-text search with English stemming/stopwords via custom analyzer
3. **Function scoring** — exponential recency decay + field-value importance boost
4. ES merges kNN and text results, summing scores for documents found by both retrievers

Weights (vector, keyword, recency, importance) map directly to ES `boost` parameters — defined in `scorer.ts`.

### Elasticsearch Indices

| Index | Key Fields |
|---|---|
| `contextfs_skills` | name (text), description (text), embedding (dense_vector), project (keyword) |
| `contextfs_memories` | content (text), category/owner (keyword), importance (integer), embedding (dense_vector) |
| `contextfs_context_nodes` | name/abstract/overview/content (text), uri/parent_uri (keyword), ancestors (keyword[]), embedding (dense_vector) |

All text fields use a custom `content_analyzer` (English stemming + stopword removal + optional synonyms). Key text fields also have an `ngram` sub-field for partial/substring matching.

### Search Features

| Feature | Description | Controlled by |
|---|---|---|
| **kNN vector search** | Dense cosine similarity on Gemini embeddings | `weights.vector` |
| **BM25 full-text** | Stemmed English text matching with IDF weighting | `weights.keyword`, `ES_BM25_K1`, `ES_BM25_B` |
| **Fuzzy matching** | Typo tolerance (Levenshtein distance) | `--fuzziness` / `ES_DEFAULT_FUZZINESS` |
| **Phrase boost** | Bonus for exact phrase ordering | `--phraseBoost` / `ES_PHRASE_BOOST` |
| **Ngram partial match** | Substring matching (e.g., "auth" finds "authentication") | Always active on name/content fields |
| **Synonyms** | Custom synonym expansion (e.g., "k8s" → "kubernetes") | `ES_SYNONYMS` env var |
| **Importance boost** | Field-value factor on importance (1-10) | `weights.importance` |
| **Recency decay** | Exponential decay on created_at | `weights.recency`, `ES_RECENCY_SCALE`, `ES_RECENCY_DECAY` |
| **Min score cutoff** | Hard threshold to drop low-confidence results | `--minScore` |
| **Highlights** | Returns `<mark>`-tagged snippets showing matched terms | `--highlight` |
| **Field boosts** | Per-search field weight overrides | `fieldBoosts` option (API only) |

### Key Modules

| File | Role |
|---|---|
| `src/elasticDB.ts` | DB layer: CRUD, hybrid search (kNN + BM25 + function_score), tree queries |
| `src/contextManager.ts` | High-level API used by CLI |
| `src/embedder.ts` | Gemini embedding calls |
| `src/scorer.ts` | Hybrid weight definitions (mapped to ES boosts) |
| `src/llmRouter.ts` | LLM-powered deduplication (CREATE / UPDATE / SKIP) |
| `src/ingestor.ts` | Free-form text → structured context nodes |
| `src/vibeEngine.ts` | LLM-driven free-text query planning and mutation planning |
| `src/cli.ts` | CLI entry point |
| `src/dashboardApi.ts` | REST API for dashboard |
| `src/evaluate.ts` | Evaluation harness entry point |

### Hierarchical Context (Tree Queries)

Context nodes store a materialized `ancestors` array. Tree operations:
- **Subtree**: single ES query — `term: { ancestors: nodeUri }` finds all descendants
- **Path**: get node's ancestors array, then `terms: { uri: [...ancestors] }` fetches the full chain

### LLM Deduplication

Before writing, `llmRouter` does a vector-only kNN search. If cosine similarity ≥ 0.75, an LLM decides whether to CREATE, UPDATE, or SKIP the new entry.

## Environment Variables

See `.env.example` for the full list. Required:

```
ELASTIC_URL=http://localhost:9200
GEMINI_API_KEY=
EMBEDDING_MODEL=gemini-embedding-001
EMBEDDING_DIM=3072
```

Optional:
```
ELASTIC_USERNAME=            # for secured clusters
ELASTIC_PASSWORD=
ALLOW_ZERO_EMBEDDINGS=false  # set true for local testing without Gemini
DASHBOARD_API_PORT=8787
```

### Elasticsearch Fine-Tuning

These control index-level settings (**require `bun run setup` after changing**):

```
ES_BM25_K1=1.2              # Term frequency saturation (higher = more weight to repeated terms)
ES_BM25_B=0.75              # Document length normalization (0 = none, 1 = full)
ES_SYNONYMS=auth,authentication,authn;db,database;k8s,kubernetes
```

These control query-level defaults (overridable per-search via CLI flags or API params):

```
ES_DEFAULT_FUZZINESS=auto    # Typo tolerance: auto, 0, 1, 2
ES_PHRASE_BOOST=2.0          # Boost for exact phrase matches (0 = disabled)
ES_RECENCY_SCALE=30d         # Recency half-life (e.g., 7d, 30d, 90d)
ES_RECENCY_DECAY=0.5         # Decay factor at scale distance
```

# Agent Integration Instructions

To integrate OpenContextFS into Claude or Opencode using the CLI, refer to this section. You must use the terminal (`bash` tool) to invoke `context-cli`.

**IMPORTANT**: Always use the `-P, --project <project>` flag when managing or searching memories/context so that information is correctly isolated by project.

### 1. Saving context/memory
Whenever you learn something new, solve a complex bug, or want to remember a project convention:
```bash
context-cli memory store "In project X, we use Vitest instead of Jest for unit testing." -c observation -o agent -i 5 -P my-project
```

### 2. Searching memory
If you are starting a new session or need to recall constraints or architecture decisions for the current task:
```bash
context-cli memory search "testing framework" -k 5 -P my-project

# With fine-tuning: fuzzy matching + exact phrase boost + highlights
context-cli memory search "authentcation setup" -k 5 -P my-project --fuzziness auto --phraseBoost 3 --highlight

# Strict mode: only high-confidence results
context-cli memory search "JWT tokens" -k 10 --minScore 5 --fuzziness 0
```

### 3. Managing Context Nodes (Hierarchical Knowledge)
For broader documentation or code architecture, you can store and read nodes:
```bash
context-cli node store "contextfs://my-project/backend/auth" "Auth Module" "Uses JWT with RSA signatures." -P my-project
context-cli node ls "contextfs://my-project/backend" -P my-project
```

### 4. Free-text Query (vibe-query)
When you need to explore the knowledge base with a natural language question:
```bash
context-cli vibe-query "how does the authentication system work?" -P my-project -k 5
```
The LLM plans and executes multi-store searches automatically.

### 5. Free-text Mutation (vibe-mutation)
When you want to add or update entries from a natural language description:
```bash
context-cli vibe-mutation "remember that we switched from REST to gRPC for internal service calls" -P my-project
```
The LLM plans mutations, shows a diff, and waits for interactive approval. Use `-y` to auto-approve.

Agents should proactively search memories when beginning a task and store important discoveries or user preferences as they work.
