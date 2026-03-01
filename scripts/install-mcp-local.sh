#!/usr/bin/env bash

set -euo pipefail

NAME="${PHTUI_MCP_NAME:-phtui-local}"
NPX_CMD="${PHTUI_MCP_NPX_CMD:-npx -y @qxinm/phtui-mcp}"
SETUP_CODEX=true
SETUP_CLAUDE=true

usage() {
  cat <<'EOF'
Install local phtui MCP client entries.

Usage:
  scripts/install-mcp-local.sh [options]

Options:
  --name <name>      MCP server name (default: phtui-local)
  --npx-cmd <cmd>    Local stdio command (default: npx -y @qxinm/phtui-mcp)
  --codex-only       Configure Codex only
  --claude-only      Configure Claude Code only
  --help             Show this help

Environment overrides:
  PHTUI_MCP_NAME
  PHTUI_MCP_NPX_CMD
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name)
      NAME="$2"
      shift 2
      ;;
    --npx-cmd)
      NPX_CMD="$2"
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

echo "Installing local MCP entry: name=${NAME}, command=${NPX_CMD}"

if [[ "${SETUP_CODEX}" == "true" ]]; then
  if command -v codex >/dev/null 2>&1; then
    codex mcp remove "${NAME}" >/dev/null 2>&1 || true
    codex mcp add "${NAME}" -- ${NPX_CMD}
    echo "[ok] Codex configured: ${NAME}"
  else
    echo "[skip] codex command not found"
  fi
fi

if [[ "${SETUP_CLAUDE}" == "true" ]]; then
  if command -v claude >/dev/null 2>&1; then
    claude mcp remove "${NAME}" >/dev/null 2>&1 || true
    claude mcp add "${NAME}" -- ${NPX_CMD}
    echo "[ok] Claude Code configured: ${NAME}"
  else
    echo "[skip] claude command not found"
  fi
fi

cat <<EOF

Done.

Next steps:
  1) Verify MCP server starts via npx:
     ${NPX_CMD}
  2) In your client prompt, request phtui tools.
EOF
