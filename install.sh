#!/usr/bin/env bash
set -euo pipefail

REPO="saheersk/lazymongo"
BINARY="lazymongo"
INSTALL_DIR="/usr/local/bin"

# ── detect OS ────────────────────────────────────────────────────────────────
OS="$(uname -s)"
case "$OS" in
  Linux)   os="linux" ;;
  Darwin)  os="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    echo "Download manually from https://github.com/$REPO/releases/latest"
    exit 1
    ;;
esac

# ── detect arch ──────────────────────────────────────────────────────────────
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64)  arch="amd64" ;;
  arm64  | aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    echo "Download manually from https://github.com/$REPO/releases/latest"
    exit 1
    ;;
esac

# ── resolve latest version ────────────────────────────────────────────────────
echo "Fetching latest version of $BINARY..."
VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": *"\(.*\)".*/\1/')"

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version. Check your internet connection."
  exit 1
fi

echo "Installing $BINARY $VERSION ($os/$arch)..."

# ── download & extract ────────────────────────────────────────────────────────
ARCHIVE="${BINARY}_${VERSION#v}_${os}_${arch}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"
TMP="$(mktemp -d)"

curl -fsSL "$URL" -o "$TMP/$ARCHIVE"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
rm "$TMP/$ARCHIVE"

# ── install ───────────────────────────────────────────────────────────────────
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
else
  echo "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
fi

rm -rf "$TMP"

echo ""
echo "$BINARY $VERSION installed to $INSTALL_DIR/$BINARY"
echo "Run: $BINARY --help"
