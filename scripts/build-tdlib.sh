#!/usr/bin/env bash
# Builds tdlib from the telegram-database submodule and installs it to /usr/local.
# Run once before building the Go project.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
TD_DIR="$ROOT_DIR/telegram-database"
BUILD_DIR="$TD_DIR/build"

echo "==> Building tdlib from $TD_DIR"

# Dependencies (Arch / CachyOS)
if command -v pacman &>/dev/null; then
  sudo pacman -S --needed --noconfirm cmake gcc openssl gperf
  # zlib: prefer zlib-ng-compat (CachyOS) if present, else zlib
  if ! pacman -Q zlib-ng-compat &>/dev/null && ! pacman -Q zlib &>/dev/null; then
    sudo pacman -S --needed --noconfirm zlib
  fi
fi

mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

cmake \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_INSTALL_PREFIX=/usr/local \
  ..

make -j"$(nproc)"

echo "==> Installing tdlib to /usr/local (requires sudo)"
sudo make install

echo "==> Done. You can now run: go build ./..."
