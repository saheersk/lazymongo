#!/usr/bin/env bash
set -euo pipefail

REPO="saheersk/lazymongo"
BINARY="lazymongo"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── detect arch ──────────────────────────────────────────────────────────────
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)           ARCH="amd64" ;;
  arm64 | aarch64)  ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    echo "Download manually: https://github.com/$REPO/releases/latest"
    exit 1 ;;
esac

# ── latest version ────────────────────────────────────────────────────────────
echo "Checking latest release..."
VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | sed 's/.*"tag_name": *"v\([^"]*\)".*/\1/')"

if [ -z "$VERSION" ]; then
  echo "Could not fetch latest version. Check your internet connection."
  exit 1
fi

echo "Installing $BINARY v$VERSION (darwin/$ARCH)..."

# ── download & extract ────────────────────────────────────────────────────────
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

URL="https://github.com/$REPO/releases/download/v${VERSION}/lazymongo_${VERSION}_darwin_${ARCH}.tar.gz"
curl -fsSL "$URL" | tar xz -C "$TMP"

# ── install ───────────────────────────────────────────────────────────────────
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
else
  sudo mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo ""
echo "$BINARY v$VERSION installed → $INSTALL_DIR/$BINARY"
echo "Run: $BINARY --help"
