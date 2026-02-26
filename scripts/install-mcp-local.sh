#!/usr/bin/env bash

set -euo pipefail

NAME="${PHTUI_MCP_NAME:-phtui-local}"
URL="${PHTUI_MCP_URL:-http://localhost:8080/mcp}"
SETUP_CODEX=true
SETUP_CLAUDE=true

usage() {
  cat <<'EOF'
Install local phtui MCP client entries.

Usage:
  scripts/install-mcp-local.sh [options]

Options:
  --name <name>      MCP server name (default: phtui-local)
  --url <url>        MCP URL (default: http://localhost:8080/mcp)
  --codex-only       Configure Codex only
  --claude-only      Configure Claude Code only
  --help             Show this help

Environment overrides:
  PHTUI_MCP_NAME
  PHTUI_MCP_URL
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name)
      NAME="$2"
      shift 2
      ;;
    --url)
      URL="$2"
      shift 2
      ;;
    --codex-only)
      SETUP_CODEX=true
      SETUP_CLAUDE=false
      shift
      ;;
    --claude-only)
      SETUP_CODEX=false
      SETUP_CLAUDE=true
      shift
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

echo "Installing local MCP entry: name=${NAME}, url=${URL}"

if [[ "${SETUP_CODEX}" == "true" ]]; then
  if command -v codex >/dev/null 2>&1; then
    codex mcp remove "${NAME}" >/dev/null 2>&1 || true
    codex mcp add "${NAME}" --url "${URL}"
    echo "[ok] Codex configured: ${NAME}"
  else
    echo "[skip] codex command not found"
  fi
fi

if [[ "${SETUP_CLAUDE}" == "true" ]]; then
  if command -v claude >/dev/null 2>&1; then
    claude mcp remove "${NAME}" >/dev/null 2>&1 || true
    claude mcp add -t http "${NAME}" "${URL}"
    echo "[ok] Claude Code configured: ${NAME}"
  else
    echo "[skip] claude command not found"
  fi
fi

cat <<EOF

Done.

Next steps:
  1) Start server:
     PORT=8080 go run ./cmd/phtui-mcp
  2) Health check:
     curl -i http://localhost:8080/healthz
EOF
