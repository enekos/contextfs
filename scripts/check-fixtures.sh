#!/usr/bin/env bash
# check-fixtures.sh — Verify every input fixture has a matching approved file.
#
# Checks:
#   - testdata/*/*.input.* → *.approved.json
#   - testdata/nl/*.input.ts → *.approved.md
#
# Run directly or from the pre-commit hook.

set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
AST_DIR="${ROOT_DIR}/mairu/internal/ast"

missing=()

# Every language input needs an approved JSON.
for input in "${AST_DIR}"/testdata/*/*.input.*; do
  [[ -f "$input" ]] || continue
  # Skip nl/ directory — those are handled separately below.
  [[ "$input" == */testdata/nl/* ]] && continue
  base="${input%.input.*}"
  approved="${base}.approved.json"
  if [[ ! -f "$approved" ]]; then
    missing+=("$approved")
  fi
done

# NL inputs additionally need an approved markdown.
for input in "${AST_DIR}"/testdata/nl/*.input.ts; do
  [[ -f "$input" ]] || continue
  approved="${input/.input.ts/.approved.md}"
  if [[ ! -f "$approved" ]]; then
    missing+=("$approved")
  fi
done

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "[check-fixtures] Missing approved fixture files:"
  printf '  %s\n' "${missing[@]}"
  echo ""
  echo "  Regenerate with:"
  echo "    UPDATE_APPROVED=1 go test -C mairu ./internal/ast/..."
  exit 1
fi

echo "[check-fixtures] All fixture files present."
