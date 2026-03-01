#!/bin/sh
set -e

REPO="loki-bedlam/reposwarm-cli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "❌ Unsupported OS: $OS"; exit 1 ;;
esac

BINARY="reposwarm-${OS}-${ARCH}"

GH_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"

printf "Installing reposwarm (%s/%s)... " "$OS" "$ARCH"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

if curl -fsSL -L "$GH_URL" -o "$TMP/reposwarm" 2>/dev/null; then
  true
else
  echo "failed"
  echo "❌ Could not download binary. Check https://github.com/${REPO}/releases"
  exit 1
fi

chmod +x "$TMP/reposwarm"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/reposwarm" "$INSTALL_DIR/reposwarm"
else
  sudo mv "$TMP/reposwarm" "$INSTALL_DIR/reposwarm"
fi

VERSION=$(reposwarm --version 2>/dev/null | awk '{print $NF}')

echo "done"
echo ""
echo "  ✅ reposwarm ${VERSION} installed to ${INSTALL_DIR}/reposwarm"
echo ""
echo "  Get started:"
echo "    reposwarm config init              Set up API connection"
echo "    reposwarm status                   Check connection"
echo "    reposwarm repos list               List tracked repositories"
echo "    reposwarm discover                 Auto-discover CodeCommit repos"
echo "    reposwarm results list             Browse investigation results"
echo "    reposwarm investigate <repo>       Start an investigation"
echo ""
echo "  All commands support --json for agent/script consumption."
echo "  Run 'reposwarm help' for the full command list."
echo ""
