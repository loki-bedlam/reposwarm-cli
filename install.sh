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
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

BINARY="reposwarm-${OS}-${ARCH}"

# Primary: CloudFront (latest build from CodePipeline)
CDN_URL="https://db22kd0yixg8j.cloudfront.net/assets/reposwarm-cli/latest/${BINARY}"
# Fallback: GitHub releases
GH_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"

echo "Installing reposwarm (${OS}/${ARCH})..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

if curl -fsSL "$CDN_URL" -o "$TMP/reposwarm" 2>/dev/null; then
  echo "Downloaded from CDN"
elif curl -fsSL "$GH_URL" -o "$TMP/reposwarm" 2>/dev/null; then
  echo "Downloaded from GitHub"
else
  echo "Error: failed to download binary. Check https://github.com/${REPO}/releases"
  exit 1
fi

chmod +x "$TMP/reposwarm"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/reposwarm" "$INSTALL_DIR/reposwarm"
else
  sudo mv "$TMP/reposwarm" "$INSTALL_DIR/reposwarm"
fi

echo "✓ Installed to ${INSTALL_DIR}/reposwarm"
reposwarm --version
