# Mairu Go Runtime

This folder contains the Go runtime for Mairu:

- `cmd/` CLI entrypoints
- `internal/agent` coding agent core
- `internal/contextsrv` centralized context server
- `ui/` web frontend embedded into the Go binary

## Build

From repository root:

```bash
bun run mairu:build
```

Or directly:

```bash
go build -C mairu -o bin/mairu-agent ./cmd/mairu
```

## Run

```bash
./mairu/bin/mairu-agent tui
./mairu/bin/mairu-agent web -p 8080
./mairu/bin/mairu-agent context-server -p 8788
```

## Notes

- The TypeScript context engine now lives at `mairu/contextfs/`.
- The unified dashboard UI lives at `mairu/ui/`.
