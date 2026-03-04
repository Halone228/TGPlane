#!/usr/bin/env bash
# Installs Go via the system package manager or official tarball.
# Supports: Arch/CachyOS, Ubuntu/Debian, Fedora/RHEL, macOS (Homebrew).
set -euo pipefail

GO_VERSION="1.25.0"   # minimum required
INSTALL_DIR="/usr/local"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[ok]${NC} $*"; }
warn() { echo -e "${YELLOW}[warn]${NC} $*"; }
die()  { echo -e "${RED}[error]${NC} $*"; exit 1; }

# Already installed?
if command -v go &>/dev/null; then
  CURRENT=$(go version | awk '{print $3}' | sed 's/go//')
  ok "Go ${CURRENT} already installed at $(command -v go)"
  exit 0
fi

echo "==> Installing Go ${GO_VERSION}"

if command -v pacman &>/dev/null; then
  sudo pacman -S --needed --noconfirm go

elif command -v apt-get &>/dev/null; then
  # Official tarball — apt ships old versions
  ARCH=$(uname -m)
  case "$ARCH" in
    x86_64)  GOARCH="amd64" ;;
    aarch64) GOARCH="arm64" ;;
    armv6l)  GOARCH="armv6l" ;;
    *)       die "Unsupported architecture: $ARCH" ;;
  esac
  TARBALL="go${GO_VERSION}.linux-${GOARCH}.tar.gz"
  URL="https://go.dev/dl/${TARBALL}"
  echo "    Downloading ${URL}"
  curl -fsSL "$URL" -o "/tmp/${TARBALL}"
  sudo rm -rf "${INSTALL_DIR}/go"
  sudo tar -C "$INSTALL_DIR" -xzf "/tmp/${TARBALL}"
  rm "/tmp/${TARBALL}"
  # Add to PATH if not already there
  PROFILE="${HOME}/.profile"
  if ! grep -q 'go/bin' "$PROFILE" 2>/dev/null; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> "$PROFILE"
    warn "Added /usr/local/go/bin to PATH in ${PROFILE}. Run: source ${PROFILE}"
  fi

elif command -v dnf &>/dev/null; then
  sudo dnf install -y golang

elif command -v yum &>/dev/null; then
  sudo yum install -y golang

elif command -v brew &>/dev/null; then
  brew install go

else
  die "No supported package manager found. Download Go from https://go.dev/dl/"
fi

ok "Go installed: $(go version 2>/dev/null || echo '(restart shell to update PATH)')"
