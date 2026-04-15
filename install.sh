#!/usr/bin/env bash
set -euo pipefail

REPO="hnnsb/kigo"
BINARY="kigo"
INSTALL_DIR="/usr/local/bin"

echo "🚀 Installing $BINARY..."

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "❌ Unsupported OS: $OS"
    exit 1
    ;;
esac

FILE="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/latest/download/${FILE}"

TMP_DIR="$(mktemp -d)"
cd "$TMP_DIR"

echo "⬇️ Downloading $FILE..."
curl -fsSL "$URL" -o "$FILE"

echo "📦 Extracting..."
tar -xzf "$FILE"

if [ ! -f "$BINARY" ]; then
  echo "❌ Binary not found after extraction"
  exit 1
fi

chmod +x "$BINARY"

echo "📥 Installing to $INSTALL_DIR (may require sudo)..."
sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"

echo "✅ Installed successfully!"
echo "👉 Run: $BINARY --help"

rm -rf "$TMP_DIR"