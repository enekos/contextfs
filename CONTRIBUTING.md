# Contributing

Thanks for your interest in improving this project.

## Development Setup

1. Install dependencies:

```bash
bun install
bun --cwd dashboard install
```

2. Create a local environment file:

```bash
cp .env.example .env
```

3. Build and typecheck:

```bash
bun run build
bun run typecheck
```

## Useful Commands

- `bun run setup` - reset and initialize database schema (destructive).
- `bun run dashboard:api` - run API for dashboard.
- `bun run dashboard:dev` - run Svelte dashboard.
- `bun run eval:retrieval -- --dataset eval/dataset.json --topK 5 --verbose true` - run retrieval benchmark.

## Contribution Guidelines

- Keep changes focused and incremental.
- Update docs and examples when behavior changes.
- Preserve backward compatibility where practical.
- Do not commit secrets (`.env`, tokens, credentials).

## Pull Requests

Please include:

- What changed and why.
- How you tested the change.
- Any migration notes for users.
