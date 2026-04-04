# Mairu Project Structure

This repository is organized as a single project named **Mairu** with multiple runtime components.

## Components

- `mairu/` - Go runtime and core product surface
  - `cmd/` - CLI entrypoints
  - `internal/agent/` - coding agent engine
  - `internal/contextsrv/` - centralized context server (HTTP API)
  - `ui/` - embedded web UI for the Go runtime
- `mairu/contextfs/` - TypeScript context engine and CLI
  - `src/` - CLI, dashboard API, ingestion daemon, retrieval logic
  - `scripts/` - local Meilisearch lifecycle script
  - `types/` - custom type roots
- `mairu/ui/` - unified Svelte web UI for chat + context dashboard features
- `tests/` - Vitest coverage for TypeScript engine
- `docs/` - project-level docs, specs, and plans

## Typical Flows

### 1) Run Unified Dashboard Stack

```bash
bun run dashboard
```

This starts:
- `mairu-agent context-server` on port `8788` (via `bun run dashboard:api`)
- `mairu/ui` dev server (via `bun run dashboard:dev`)

### 2) Run Mairu Agent (Go)

```bash
bun run mairu:build
./mairu/bin/mairu-agent tui
```

### 3) Run Central Context Server

```bash
./mairu/bin/mairu-agent context-server -p 8788
```

## Data and Runtime Artifacts

Local Meilisearch artifacts are created either at repository root or under `mairu/contextfs/`, depending on script entrypoint:

- `.tools/` / `.data/` / `.logs/`
- `mairu/contextfs/.tools/` / `mairu/contextfs/.data/` / `mairu/contextfs/.logs/`

Both paths are git-ignored.

## Naming Policy

- Project name: **Mairu**
- Go binary: `mairu-agent`
- TypeScript context CLI: `mairu-context` (with backward-compatible alias `context-cli`)
