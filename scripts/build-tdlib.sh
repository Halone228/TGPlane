#!/usr/bin/env bash
# Builds TDLib and installs it to /usr/local.
# Supports: Arch/CachyOS, Ubuntu/Debian, Fedora/RHEL/CentOS, macOS (Homebrew).
# Run once before building the Go project.
#
# Usage:
#   bash scripts/build-tdlib.sh              — clone TDLib (go-tdlib v0.7.6 commit) and build
#   TDLIB_DIR=/path/to/tdlib bash scripts/build-tdlib.sh  — build from existing dir
#   TDLIB_COMMIT=<hash> bash scripts/build-tdlib.sh       — use specific commit
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

TDLIB_REPO="https://github.com/tdlib/td.git"
# Commit required by go-tdlib v0.7.6 (version 1.8.62, 2024-11-27)
TDLIB_COMMIT="${TDLIB_COMMIT:-22d49d5b87a4d5fc60a194dab02dd1d71529687f}"
TD_DIR="${TDLIB_DIR:-${ROOT_DIR}/.tdlib-src}"
BUILD_DIR="${TD_DIR}/build"

# ---------------------------------------------------------------------------
# Detect OS and install build dependencies
# ---------------------------------------------------------------------------

install_deps() {
  if command -v pacman &>/dev/null; then
    echo "==> Detected Arch / CachyOS (pacman)"
    sudo pacman -S --needed --noconfirm cmake gcc openssl gperf git
    # CachyOS ships zlib-ng-compat instead of zlib
    if ! pacman -Q zlib-ng-compat &>/dev/null && ! pacman -Q zlib &>/dev/null; then
      sudo pacman -S --needed --noconfirm zlib
    fi

  elif command -v apt-get &>/dev/null; then
    echo "==> Detected Debian / Ubuntu (apt)"
    sudo apt-get update -qq
    sudo apt-get install -y --no-install-recommends \
      cmake g++ libssl-dev zlib1g-dev gperf git

  elif command -v dnf &>/dev/null; then
    echo "==> Detected Fedora / RHEL (dnf)"
    sudo dnf install -y cmake gcc-c++ openssl-devel zlib-devel gperf git

  elif command -v yum &>/dev/null; then
    echo "==> Detected CentOS / RHEL legacy (yum)"
    sudo yum install -y cmake gcc-c++ openssl-devel zlib-devel gperf git

  elif command -v brew &>/dev/null; then
    echo "==> Detected macOS (Homebrew)"
    brew install cmake openssl gperf git
    OPENSSL_ROOT="$(brew --prefix openssl)"
    export CMAKE_EXTRA_ARGS="-DOPENSSL_ROOT_DIR=${OPENSSL_ROOT}"

  else
    echo "WARNING: Unknown package manager."
    echo "         Install manually: cmake g++ openssl-dev zlib-dev gperf git"
  fi
}

# ---------------------------------------------------------------------------
# Clone TDLib if needed
# ---------------------------------------------------------------------------

clone_tdlib() {
  if [ -f "${TD_DIR}/CMakeLists.txt" ]; then
    echo "==> TDLib source found at ${TD_DIR} — skipping clone"
    return
  fi

  echo "==> Cloning TDLib (commit ${TDLIB_COMMIT}) into ${TD_DIR}"
  git clone "${TDLIB_REPO}" "${TD_DIR}"
  git -C "${TD_DIR}" checkout "${TDLIB_COMMIT}"
}

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

install_deps
clone_tdlib

echo "==> Building TDLib from ${TD_DIR}"

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

echo ""
echo "==> Done."
echo "    Build the worker with:"
echo "    CGO_LDFLAGS_ALLOW=\"-Wl,--whole-archive.*|-Wl,--no-whole-archive\" go build ./cmd/worker/..."
