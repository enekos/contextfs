# mairu

![mairu logo](mairu.png)

Unified monorepo for:

- the **Mairu coding agent** (Go, TUI + web)
- the **central context server** (Go)
- the **unified web UI dashboard** (Svelte)

## Repository Layout

```text
.
├── mairu/
│   ├── cmd/                 # Go entrypoint (mairu binary)
│   ├── internal/            # Go agent + context-server internals
│   ├── scripts/             # Local Meilisearch helper script
│   └── ui/                  # Unified Mairu dashboard UI (Svelte) & Go app frontend
├── docs/                    # Specs and project docs
├── package.json             # Monorepo task runner scripts
└── Makefile                 # Dev shortcuts
```

## Requirements

- Bun 1+ (for dashboard UI)
- Go 1.25+
- Docker (optional if using local Meilisearch fallback)
- Gemini API key (unless `ALLOW_ZERO_EMBEDDINGS=true` for local-only testing)

## Quickstart

The easiest way to set up Mairu locally (without Docker) is using the bootstrap script:

```bash
./bootstrap.sh
```

If you prefer to set up manually:

```bash
cp .env.example .env
bun install --cwd mairu/ui
make setup-no-docker
```

Then run:

```bash
make dashboard        # Context server + unified web dashboard
# or
make mairu-web        # Mairu agent web UI
```

## Core Commands

| Command | Description |
|---|---|
| `bun run build` | Build Go `mairu-agent` binary |
| `bun run test` | Run Go test suite |
| `bun run lint` | Run Go vet |
| `bun run setup` | Initialize/reset Meilisearch indexes (destructive) |
| `bun run dashboard` | Run context server + unified Mairu dashboard UI |
| `bun run mairu:build` | Build Go `mairu-agent` binary |
| `bun run mairu:web` | Launch Mairu web UI |

## Local Meilisearch (No Docker)

```bash
make meili-up
make meili-status
make meili-down
make meili-clean
```

Script path: `mairu/scripts/meili-local.sh`.

## Go CLI Commands (Mairu Agent)

With the latest features, the Go CLI implements the ContextFS API fully via the `mairu-agent` binary:

```bash
bun run mairu:build

# Context Server APIs
./mairu/bin/mairu-agent memory search "auth token" -P my-project -k 5
./mairu/bin/mairu-agent node search "authentication architecture" -P my-project -k 5

# Vibe commands (LLM powered mutations and queries)
./mairu/bin/mairu-agent vibe query "how does auth work?" -P my-project
./mairu/bin/mairu-agent vibe mutation "remember we use gRPC internally" -P my-project

# Advanced Tools (Daemon, Ingest & Scraper)
./mairu/bin/mairu-agent daemon ./src -P my-project       # Watch a directory and parse AST
./mairu/bin/mairu-agent ingest design.md -P my-project   # Ingest free-text notes
./mairu/bin/mairu-agent scrape https://example.com       # Scrape web page and store context

# Full TUI or Web Servers
./mairu/bin/mairu-agent tui
./mairu/bin/mairu-agent web -p 8080
./mairu/bin/mairu-agent context-server -p 8788
```

## Environment

Minimal `.env`:

```env
MEILI_URL=http://localhost:7700
MEILI_API_KEY=mairu-dev-key
GEMINI_API_KEY=your_gemini_api_key
EMBEDDING_MODEL=gemini-embedding-001
EMBEDDING_DIM=3072
ALLOW_ZERO_EMBEDDINGS=false
DASHBOARD_API_PORT=8787
```

## License

ISC (`LICENSE`)