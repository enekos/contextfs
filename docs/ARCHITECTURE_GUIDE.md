# Mairu Architecture & Development Guide

This document provides a high-level overview of the Mairu system architecture and guidelines for developers contributing to the codebase.

## System Overview

Mairu is a context-aware coding agent ecosystem. It bridges the gap between raw codebase/knowledge data and actionable agent intelligence through:
1. **ContextFS**: A hierarchical storage system for memories, skills, and code context.
2. **Hybrid Search**: A retrieval pipeline combining vector embeddings and keyword search, enhanced by app-side re-ranking.
3. **Daemon**: A background process that automatically scans and translates code into human-readable Natural Language descriptions.
4. **Vibe Engine**: An LLM-powered orchestration layer for high-level tasks like query planning, memory mutation, and deduplication.
5. **Browser Extension**: A Chrome MV3 extension with a WASM core that synchronizes web browsing sessions with Mairu's agent context.

## Browser Extension Architecture

The `browser-extension` module bridges `mairu chat` to live web content, enabling the agent to search your browser history and read current pages.

1. **DOM Extraction (WASM/Rust)**: 
   Content scripts send DOM snapshots to the service worker. A compiled Rust WASM core (`browser-extension-core` & `browser-extension-wasm`) executes DOM-based semantic heuristics (using `scraper` rather than regex) to filter noise (nav/footers/ads), extract markdown tables, blockquotes, code blocks, and chunk text.
2. **Session State & Deduplication**:
   The WASM module maintains a rolling session queue (e.g., 50 pages). It hashes pages via `SimHash` (comparing via Hamming distance) to reject functionally duplicate snapshots of the same SPA states.
3. **TF-IDF Search**:
   The WASM module maintains a lightweight `TfIdfIndex` over the session memory, allowing the agent to issue queries like "what was that rust config flag?" against recent browsing.
4. **Native Messaging Bridge**:
   The `browser-extension-host` (Rust binary) serves as a persistent standard I/O bridge linking the Chrome Extension to the Go runtime via local HTTP (`127.0.0.1:7081`). When the Go agent executes the `browser_context` tool, it requests data from this bridge.

## Data Model

Mairu uses three primary data structures:
- **Memories**: User-defined facts categorized by owner, importance, and project.
- **Skills**: Named capabilities with corresponding descriptions.
- **Context Nodes**: Hierarchical, tree-structured nodes (URIs) containing abstracted, overviewed, and detailed natural language content of files or documentation.

## Development Workflow

### Adding New Language Support
The system uses a pluggable `LanguageDescriber` interface. To support a new language:
1. Implement the `LanguageDescriber` interface (`mairu/internal/ast/language_describer.go`).
2. Add your implementation to the `ParserPool` (`mairu/internal/ast/parser_pool.go`).
3. Create test cases in `mairu/internal/ast/testdata/<lang>` and update tests in `mairu/internal/ast/<lang>_describer_test.go`.

### Working with the Daemon
The daemon processes files in parallel. It uses a persistent hash cache (`.contextfs-cache.json`) to skip unchanged files.
- The daemon performs AST extraction and converts code to English statements.
- The `nl_enricher` performs a post-processing pass to cross-reference function calls.

### LLM Interactions
All LLM-powered logic resides in `mairu/internal/llm/`. When adding new AI-driven features:
1. Add prompt templates to `mairu/internal/prompts/`.
2. Ensure you utilize the `Router` for deduplication if creating new context.

## Testing Guidelines

- **Go Unit Tests**: Place tests alongside your code (e.g., `mairu/internal/agent/agent_test.go`).
- **Integration Tests**: Tests requiring Meilisearch/LLMs must be marked and handle setup properly.
- **Evaluation**: Use the `llmeval` package to test LLM-driven features (e.g., retrieval precision, vibe query quality).
