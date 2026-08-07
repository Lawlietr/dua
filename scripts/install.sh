#!/usr/bin/env bash
# install.sh — install dua to ~/.local/bin
# Usage: curl -fsSL "https://raw.githubusercontent.com/Lawlietr/dua/main/scripts/install.sh" | bash

set -euo pipefail

INSTALL_DIR="$HOME/.local/bin"
ARCH=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
URL="https://github.com/Lawlietr/dua/releases/latest/download/dua-linux-${ARCH}.tar.gz"

echo "Installing dua to ${INSTALL_DIR} ..."
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" | tar -xz -C "$INSTALL_DIR"

echo "Done. dua is installed at ${INSTALL_DIR}"
echo ""
echo "Add ${INSTALL_DIR} to your PATH (e.g. in ~/.bashrc or ~/.zshrc):"
echo '  export PATH="$HOME/.local/bin:$PATH"'
