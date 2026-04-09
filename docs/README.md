# Mairu Documentation

Welcome to the Mairu documentation. This repository contains detailed guides and reference materials for using, extending, and developing with Mairu.

## Core Concepts

- **ContextFS**: A hierarchical, graph-based context storage system.
- **Memories**: Fact-based storage for project knowledge, rules, and preferences.
- **Skills**: Capability definitions for agents.
- **Context Nodes**: URI-addressed nodes containing abstract, overview, and content fields (AST-based).
- **Browser Extension**: A Chrome extension (Rust/WASM) that syncs real-time web browsing context into Mairu.
- **Vibe Engine**: An LLM-powered engine for high-level queries and state mutations.
- **Daemon**: A background process for automatic codebase ingestion and AST extraction.

## Guides

- [Project Structure](PROJECT_STRUCTURE.md): Detailed breakdown of the repository.
- [Development Workflow](DEVELOPMENT.md): How to contribute and run tests.
- [Configuration Guide](CONFIGURATION.md): Deep dive into the TOML configuration system.

## Superpowers (Internal Docs)

Internal documentation for advanced features and architectural decisions.

- [Stuck Detector](superpowers/specs/2026-04-08-stuck-detector-design.md): Design and implementation of the automatic stuck detector.
- [Stuck Detector Plan](superpowers/plans/2026-04-08-stuck-detector.md): Execution plan for the stuck detector feature.

## License

ISC
