#!/usr/bin/env bash
# Wrapper around golang-migrate for TGPlane.
# Usage:
#   ./scripts/migrate.sh up              — apply all pending migrations
#   ./scripts/migrate.sh down            — roll back last migration
#   ./scripts/migrate.sh down N          — roll back N migrations
#   ./scripts/migrate.sh version         — show current version
#   ./scripts/migrate.sh force N         — force set version (danger)
#   ./scripts/migrate.sh DSN="..." up    — override DSN inline
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
MIGRATIONS_DIR="${ROOT_DIR}/migrations"
CONFIG="${ROOT_DIR}/config.yaml"

# ---------------------------------------------------------------------------
# Resolve DSN
# ---------------------------------------------------------------------------
if [ -z "${DSN:-}" ]; then
  if ! command -v grep &>/dev/null || [ ! -f "$CONFIG" ]; then
    echo "ERROR: config.yaml not found and DSN not set."
    echo "       Copy config.yaml.example to config.yaml and set database.dsn"
    exit 1
  fi
  DSN=$(grep 'dsn:' "$CONFIG" | head -1 | sed 's/.*dsn:[[:space:]]*//' | tr -d '"')
fi

if [ -z "$DSN" ]; then
  echo "ERROR: database.dsn is empty in config.yaml"
  exit 1
fi

# ---------------------------------------------------------------------------
# Check migrate binary
# ---------------------------------------------------------------------------
if ! command -v migrate &>/dev/null; then
  echo "==> Installing golang-migrate..."
  go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi

CMD="${1:-up}"
ARGS="${2:-}"

echo "==> migrate ${CMD} ${ARGS} (${MIGRATIONS_DIR})"
migrate -path "$MIGRATIONS_DIR" -database "$DSN" "$CMD" $ARGS
