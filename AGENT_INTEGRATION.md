# OpenContextFS Agent Integration Instructions

*Copy and paste the following instructions into your project's `CLAUDE.md`, `.cursorrules`, `AGENTS.md`, or equivalent agent system prompt file to enable persistent memory across agent sessions.*

---

## 🧠 Persistent Memory & Context Management (OpenContextFS)

This project uses **OpenContextFS** (`context-cli`) to persist memory, architectural decisions, and project context across different agent sessions. 

As an AI agent, you have access to the `context-cli` tool via your terminal/bash execution capabilities. You should proactively use this tool to build and query a long-term understanding of the project.

**IMPORTANT:** Always use the `-P, --project <project_name>` flag to ensure context is correctly scoped to this specific project. Replace `<project_name>` with the actual name of this repository/project.

### 1. Saving Memories (Proactive Learning)
Whenever you learn a project convention, make an architectural decision, solve a complex bug, or receive a user preference, you MUST store it as a memory so future sessions can recall it.

```bash
# Basic usage
context-cli memory store "Your observation or fact here" -c <category> -o agent -i <1-10> -P <project_name>

# Example: Saving a testing convention
context-cli memory store "In this project, we use Vitest instead of Jest for unit testing. All tests go in the /tests directory." -c convention -o agent -i 8 -P <project_name>
```
*Note: `importance` (`-i`) is a scale from 1 (trivial) to 10 (critical).*

### 2. Recalling Memories (Context Gathering)
When starting a new task, debugging a recurring issue, or before making architectural choices, you MUST search the memory for existing constraints or decisions.

```bash
# Basic usage
context-cli memory search "your search query" -k <number_of_results> -P <project_name>

# Example: Checking database conventions
context-cli memory search "database schema migrations" -k 5 -P <project_name>
```

### 3. Managing Hierarchical Context Nodes
For broader documentation, module architecture, or high-level summaries, use the hierarchical node system. Nodes are addressed by URIs (e.g., `contextfs://<project_name>/module/submodule`).

```bash
# Storing a node
context-cli node store "contextfs://<project_name>/backend/auth" "Auth Module Architecture" "Uses JWT with RSA signatures and Redis for token blacklisting." -P <project_name>

# Listing/Reading nodes
context-cli node ls "contextfs://<project_name>/backend" -P <project_name>
```

### Core Directives for Agents
1. **Initialize on start:** When beginning a new, large task, do a quick `memory search` for keywords related to the task.
2. **Commit on success:** When you successfully complete a complex task, summarize the "gotchas" or structural decisions and `memory store` them.
3. **Never guess:** If project conventions aren't clear in the codebase, check `context-cli` before asking the user.
