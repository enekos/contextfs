# mairu

![mairu logo](mairu.png)

Unified monorepo for:

- the **Mairu coding agent** (Go, CLI, TUI + web)
- the **central context server** (Go)
- the **unified web UI dashboard** (Svelte)

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for contributor workflow and quality gates.

## Repository Layout

```text
.
├── mairu/
│   ├── cmd/                 # Go entrypoint (mairu binary)
│   ├── internal/            # Go agent + context-server internals
│   ├── scripts/             # Local Meilisearch helper script
│   └── ui/                  # Unified Mairu dashboard UI (Svelte) & Go app frontend
├── docs/                    # Specs and project docs
├── package.json             # UI-only Bun scripts
└── Makefile                 # Go + monorepo dev workflows
```

## Requirements

- Bun 1+ (for dashboard UI)
- Go 1.25+
- Docker (optional if using local Meilisearch fallback)
- Gemini API key (for chat)
- ONNX Runtime (optional, for fastembed local embeddings)

## Quickstart

The easiest way to set up Mairu locally (without Docker) is using the bootstrap script:

```bash
./bootstrap.sh
```

Then initialize the configuration and set your API key:

```bash
make mairu-build
./mairu/bin/mairu setup
./mairu/bin/mairu init --defaults
```

Start the services:

```bash
make dashboard        # Context server + unified web dashboard
# or
make mairu-web        # Mairu agent web UI
```

## Configuration

Mairu uses a five-tier TOML configuration cascade:
1. Hardcoded defaults
2. User config (`~/.config/mairu/config.toml`)
3. Project config (`.mairu.toml` in your project root or `.git` parent)
4. Environment variables (`MAIRU_` prefix)
5. CLI flags

Manage your config using the CLI:
```bash
./mairu/bin/mairu config list
./mairu/bin/mairu config set api.gemini_api_key "your-key"
./mairu/bin/mairu init            # interactive project setup
./mairu/bin/mairu doctor          # check system health
```

### Sample `config.toml`

Here is an example of what your `~/.config/mairu/config.toml` or `.mairu.toml` might look like:

```toml
[api]
gemini_api_key = "AIzaSyYourKeyHere..."
meili_url = "http://localhost:7700"
meili_api_key = "contextfs-dev-key"

[daemon]
concurrency = 8
max_file_size = "512KB"
debounce = "200ms"
max_content_chars = 16000

[server]
port = 8788
sqlite_dsn = "file:mairu.db?cache=shared&mode=rwc"

[embedding]
provider = "fastembed"
model = "fast-all-MiniLM-L6-v2"
dimensions = 384

[output]
format = "table"
color = true
```

## Core Commands

| Command | Description |
|---|---|
| `make mairu-build` | Build Go `mairu` binary |
| `make test-go` | Run Go test suite |
| `make lint-go` | Run Go lint (`golangci-lint` or fallback `go vet`) |
| `make check-go` | Run Go fmt check + lint + tests |
| `make check-go-ci` | Run CI-grade Go checks (fmt + lint + race) |
| `make install-hooks` | Install local pre-commit hook (`make check-go`) |
| `make setup` | Initialize/reset Meilisearch indexes (destructive) |
| `make dashboard` | Run context server + unified Mairu dashboard UI |
| `make mairu-web` | Launch Mairu web UI |
| `bun run dashboard:dev` | Run UI-only dev server |
| `bun run dashboard:build` | Build UI-only frontend bundle |

### Go Dev Tooling

For stricter linting, install `golangci-lint` once:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

If `golangci-lint` is not installed, the tooling falls back to `go vet`.

Optional pre-commit hook:

```bash
make install-hooks
```

## Go CLI Commands (Mairu Agent)

With the latest features, the Go CLI implements the ContextFS API fully via the `mairu` binary.
Output defaults to `table`, but you can use `-o json` or `-o plain` for scripting.

```bash
make mairu-build
```

### Configuration & Health
```bash
./mairu/bin/mairu config list
./mairu/bin/mairu doctor
./mairu/bin/mairu init            # interactive project setup
./mairu/bin/mairu setup           # setup configuration
```

### Context Server APIs
```bash
./mairu/bin/mairu memory search "auth token" -P my-project -k 5
./mairu/bin/mairu node search "authentication architecture" -P my-project -k 5
./mairu/bin/mairu skill list -P my-project
./mairu/bin/mairu sync            # force sync search_outbox to Meilisearch
```

### Vibe Commands (LLM Powered)
```bash
./mairu/bin/mairu vibe query "how does auth work?" -P my-project
./mairu/bin/mairu vibe mutation "remember we use gRPC internally" -P my-project
```

### AI-Optimized Dev Tools
```bash
./mairu/bin/mairu scan "func main" ./src -e .go -n 10              # semantic search with token budget
./mairu/bin/mairu outline main.go --tree --full                      # file skeleton / AST outline
./mairu/bin/mairu peek main.go -l 10-20 -s MyStruct                 # peek file lines or symbols
./mairu/bin/mairu map ./src -d 3 --sort size                        # token-aware directory tree
./mairu/bin/mairu code search "auth middleware" -P my-project       # semantic code search via daemon
./mairu/bin/mairu impact contextfs://node/my-project/auth.go -P my-project   # blast radius analysis
./mairu/bin/mairu info ./src -e .go --top 10                        # repository stats
./mairu/bin/mairu sys                                               # system health snapshot
./mairu/bin/mairu env .env --reveal                                 # safe environment reader
./mairu/bin/mairu proc ports                                        # list active ports & processes
./mairu/bin/mairu proc top                                          # highest CPU/memory processes
./mairu/bin/mairu dev kill-port 3000                                # kill process on port
./mairu/bin/mairu distill go test ./...                             # run command, capture errors
./mairu/bin/mairu splice main.go -t oldFunc -r new.go               # AST-aware symbol replacement
```

### Daemon, Ingest, Scraper & History
```bash
./mairu/bin/mairu daemon ./src -P my-project                                        # scan directory and extract AST to context nodes
./mairu/bin/mairu ingest design.md --base-uri "contextfs://design" -P my-project -y # parse markdown via LLM and persist
./mairu/bin/mairu scrape web https://example.com -P my-project                       # scrape one URL into context
./mairu/bin/mairu scrape depth https://example.com -d 2 -P my-project                # crawl and summarize web content
./mairu/bin/mairu scrape search "latest go release"                                 # search web and extract structured data
./mairu/bin/mairu history search "test fail"                                        # semantically search bash command history
```

### Git & Docker Helpers
```bash
./mairu/bin/mairu git summary                                       # token-dense git status summary
./mairu/bin/mairu git ingest -P my-project                          # ingest git history into context nodes
./mairu/bin/mairu docker ps                                         # token-friendly container list
./mairu/bin/mairu docker logs <container>                           # token-budgeted container logs
./mairu/bin/mairu docker stats                                      # container resource snapshot
```

### Integrations
```bash
./mairu/bin/mairu github sync-issues -P my-project                  # sync GitHub issues/PRs into context
./mairu/bin/mairu linear sync-issues -P my-project                  # sync Linear issues into context
```

### Analysis & Graph
```bash
./mairu/bin/mairu analyze diff                                      # analyze git diff blast radius
./mairu/bin/mairu analyze graph -P my-project                       # analyze AST graph for skills/flows
./mairu/bin/mairu eval:retrieval                                    # run retrieval evaluation suite
```

### Server Modes & Interfaces
```bash
./mairu/bin/mairu tui                                               # interactive terminal UI
./mairu/bin/mairu web -p 8080                                       # Mairu web interface
./mairu/bin/mairu context-server -p 8788                            # centralized context server
./mairu/bin/mairu utcp -p 8081                                      # UTCP server over HTTP
./mairu/bin/mairu mcp                                               # start MCP server on stdio
./mairu/bin/mairu telegram                                          # start Telegram bot interface
```

### Minion Mode (Unattended Automation)
```bash
./mairu/bin/mairu minion "fix lint errors" --max-retries 2          # one-shot unattended execution
./mairu/bin/mairu minion --github-issue 42 --council                # resolve a GitHub issue autonomously
```

## Environment Variables (Legacy Support)

Mairu supports older environment variables, but `.mairu.toml` or `~/.config/mairu/config.toml` is preferred.
See `mairu config list` for the complete list of settings.

```env
MEILI_URL=http://localhost:7700
MEILI_API_KEY=contextfs-dev-key
GEMINI_API_KEY=your_gemini_api_key
EMBEDDING_PROVIDER=fastembed
EMBEDDING_MODEL=fast-all-MiniLM-L6-v2
EMBEDDING_DIM=384
```

## License

ISC (`LICENSE`)