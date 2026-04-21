#!/usr/bin/env bash
set -euo pipefail

REPO="hnnsb/kigo"
BINARY="kigo"
INSTALL_DIR="/usr/local/bin"

normalize_version() {
  printf '%s' "${1#v}"
}

echo "Installing $BINARY..."

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

LATEST_TAG=""
if LATEST_JSON="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null)"; then
  LATEST_TAG="$(printf '%s\n' "$LATEST_JSON" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')"
  if [ -n "$LATEST_TAG" ]; then
    echo "Latest release version: $LATEST_TAG"
  else
    echo "Warning: Could not parse latest release version from GitHub API response."
  fi
else
  echo "Warning: Could not fetch latest release version from GitHub API."
fi

INSTALLED_PATH=""
if command -v "$BINARY" >/dev/null 2>&1; then
  INSTALLED_PATH="$(command -v "$BINARY")"
elif [ -x "$INSTALL_DIR/$BINARY" ]; then
  INSTALLED_PATH="$INSTALL_DIR/$BINARY"
fi

if [ -n "$INSTALLED_PATH" ]; then
  INSTALLED_OUTPUT="$($INSTALLED_PATH --version 2>/dev/null || true)"
  INSTALLED_TAG="$(printf '%s\n' "$INSTALLED_OUTPUT" | sed -nE 's/.*[Vv]ersion[[:space:]]+([^[:space:]]+).*/\1/p' | head -n1)"

  if [ -n "$INSTALLED_TAG" ]; then
    echo "Installed version: $INSTALLED_TAG"

    if [ -n "$LATEST_TAG" ] && [ "$(normalize_version "$INSTALLED_TAG")" = "$(normalize_version "$LATEST_TAG")" ]; then
      echo "$BINARY $INSTALLED_TAG is already the latest version."
      exit 0
    fi
  else
    echo "Warning: Could not parse installed version from '$INSTALLED_PATH --version' output."
  fi
fi

FILE="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/latest/download/${FILE}"

TMP_DIR="$(mktemp -d)"
cd "$TMP_DIR"

echo "Downloading $FILE..."
curl -fsSL "$URL" -o "$FILE"

echo "Extracting..."
tar -xzf "$FILE"

if [ ! -f "$BINARY" ]; then
  echo "Binary not found after extraction"
  exit 1
fi

chmod +x "$BINARY"

echo "Installing to $INSTALL_DIR (may require sudo)..."
sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"

echo "Installed successfully!"
echo "Run: $BINARY --help"

rm -rf "$TMP_DIR"