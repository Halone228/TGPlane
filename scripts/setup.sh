#!/usr/bin/env bash
# Full project setup: installs Go tooling, runs migrations, creates config files.
# Run after build-tdlib.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$ROOT_DIR"

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[ok]${NC} $*"; }
warn() { echo -e "${YELLOW}[warn]${NC} $*"; }
die()  { echo -e "${RED}[error]${NC} $*"; exit 1; }

# ---------------------------------------------------------------------------
# Check Go
# ---------------------------------------------------------------------------
echo "==> Checking Go"
if ! command -v go &>/dev/null; then
  die "Go not found. Install from https://go.dev/dl/"
fi
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED="1.22"
if [ "$(printf '%s\n' "$REQUIRED" "$GO_VERSION" | sort -V | head -1)" != "$REQUIRED" ]; then
  die "Go ${REQUIRED}+ required, found ${GO_VERSION}"
fi
ok "Go ${GO_VERSION}"

# ---------------------------------------------------------------------------
# Install Go tools
# ---------------------------------------------------------------------------
echo "==> Installing Go tools"
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
ok "migrate installed"

# ---------------------------------------------------------------------------
# Copy example configs
# ---------------------------------------------------------------------------
echo "==> Config files"
for f in config.yaml config.worker.yaml; do
  if [ ! -f "$f" ]; then
    cp "${f}.example" "$f"
    warn "Created ${f} from example — edit it before starting"
  else
    ok "${f} already exists"
  fi
done

# ---------------------------------------------------------------------------
# Download Go modules
# ---------------------------------------------------------------------------
echo "==> Downloading Go modules"
go mod download
ok "modules ready"

# ---------------------------------------------------------------------------
# Check TDLib
# ---------------------------------------------------------------------------
echo "==> Checking TDLib"
if [ ! -f /usr/local/lib/libtdjson_static.a ]; then
  warn "TDLib not found at /usr/local. Run: bash scripts/build-tdlib.sh"
else
  ok "TDLib found"
fi

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
echo ""
echo -e "${GREEN}Setup complete.${NC} Next steps:"
echo "  1. Edit config.yaml and config.worker.yaml"
echo "  2. Start infrastructure: docker compose -f deployments/docker-compose.yml up -d"
echo "  3. Run migrations:       make migrate-up"
echo "  4. Start main node:      make run-main"
echo "  5. Start worker node:    make run-worker"
