#!/usr/bin/env bash
# install.sh — install dua to ~/.local/bin
# Usage: curl -fsSL "https://raw.githubusercontent.com/Lawlietr/dua/main/scripts/install.sh" | bash

set -euo pipefail

INSTALL_DIR="$HOME/.local/bin"
OS=$(uname -s)
ARCH=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')

case "$OS:$ARCH" in
    Linux:amd64|Linux:arm64) TARBALL="dua-linux-${ARCH}" ;;
    Darwin:arm64)            TARBALL="dua-darwin-arm64" ;;
    *)
        echo "dua installer: unsupported platform ${OS}/${ARCH}" >&2
        echo "Prebuilt binaries: Linux amd64/arm64 and macOS arm64 only." >&2
        exit 1
        ;;
esac

URL="https://github.com/Lawlietr/dua/releases/latest/download/${TARBALL}.tar.gz"

echo "Installing dua (${TARBALL}) to ${INSTALL_DIR} ..."
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" | tar -xz -C "$INSTALL_DIR"

echo "Done. dua is installed at ${INSTALL_DIR}"
echo ""
echo "Add ${INSTALL_DIR} to your PATH (e.g. in ~/.bashrc or ~/.zshrc):"
echo '  export PATH="$HOME/.local/bin:$PATH"'
