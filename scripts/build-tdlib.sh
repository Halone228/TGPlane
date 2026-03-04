#!/usr/bin/env bash
# Builds TDLib from the telegram-database submodule and installs it to /usr/local.
# Supports: Arch/CachyOS, Ubuntu/Debian, Fedora/RHEL/CentOS, macOS (Homebrew).
# Run once before building the Go project.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
TD_DIR="$ROOT_DIR/telegram-database"
BUILD_DIR="$TD_DIR/build"

# ---------------------------------------------------------------------------
# Detect OS and install dependencies
# ---------------------------------------------------------------------------

install_deps() {
  if command -v pacman &>/dev/null; then
    echo "==> Detected Arch / CachyOS (pacman)"
    sudo pacman -S --needed --noconfirm cmake gcc openssl gperf
    # CachyOS ships zlib-ng-compat instead of zlib
    if ! pacman -Q zlib-ng-compat &>/dev/null && ! pacman -Q zlib &>/dev/null; then
      sudo pacman -S --needed --noconfirm zlib
    fi

  elif command -v apt-get &>/dev/null; then
    echo "==> Detected Debian / Ubuntu (apt)"
    sudo apt-get update -qq
    sudo apt-get install -y --no-install-recommends \
      cmake g++ libssl-dev zlib1g-dev gperf

  elif command -v dnf &>/dev/null; then
    echo "==> Detected Fedora / RHEL (dnf)"
    sudo dnf install -y cmake gcc-c++ openssl-devel zlib-devel gperf

  elif command -v yum &>/dev/null; then
    echo "==> Detected CentOS / RHEL legacy (yum)"
    sudo yum install -y cmake gcc-c++ openssl-devel zlib-devel gperf

  elif command -v brew &>/dev/null; then
    echo "==> Detected macOS (Homebrew)"
    brew install cmake openssl gperf
    # openssl is keg-only; export path for cmake
    OPENSSL_ROOT="$(brew --prefix openssl)"
    export CMAKE_EXTRA_ARGS="-DOPENSSL_ROOT_DIR=${OPENSSL_ROOT}"

  else
    echo "WARNING: Unknown package manager. Install manually: cmake g++ openssl-dev zlib-dev gperf"
  fi
}

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

echo "==> Building TDLib from ${TD_DIR}"

if [ ! -f "${TD_DIR}/CMakeLists.txt" ]; then
  echo "ERROR: telegram-database submodule is not initialised."
  echo "       Run: git submodule update --init --recursive"
  exit 1
fi

install_deps

mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

cmake \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_INSTALL_PREFIX=/usr/local \
  ${CMAKE_EXTRA_ARGS:-} \
  ..

make -j"$(nproc 2>/dev/null || sysctl -n hw.logicalcpu)"

echo "==> Installing TDLib to /usr/local"
sudo make install

echo "==> Done. Build the worker with:"
echo "    CGO_LDFLAGS_ALLOW=\"-Wl,--whole-archive.*|-Wl,--no-whole-archive\" go build ./cmd/worker/..."
