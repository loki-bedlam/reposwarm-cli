#!/bin/sh
set -e

REPO="loki-bedlam/reposwarm-cli"
VERSION="v1.0.0"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

BINARY="reposwarm-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"

echo "Installing reposwarm ${VERSION} (${OS}/${ARCH})..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP/reposwarm"
chmod +x "$TMP/reposwarm"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/reposwarm" "$INSTALL_DIR/reposwarm"
else
  sudo mv "$TMP/reposwarm" "$INSTALL_DIR/reposwarm"
fi

echo "✓ Installed to ${INSTALL_DIR}/reposwarm"
reposwarm --version
