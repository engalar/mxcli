#!/bin/sh
# mxcli install script — idempotent, works on Linux and macOS.
# Usage: curl -fsSL https://raw.githubusercontent.com/engalar/mxcli/dev/install.sh | sh
#
# Optional env vars:
#   MXCLI_INSTALL_DIR  — override install directory (default: /usr/local/bin or ~/.local/bin)
set -e

REPO="engalar/mxcli"
INSTALL_DIR="${MXCLI_INSTALL_DIR:-}"

# ── Detect platform ──────────────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "❌ Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "❌ Unsupported OS: $OS (use install.ps1 on Windows)" >&2
    exit 1
    ;;
esac

# ── Fetch latest release tag ─────────────────────────────────────────────────
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$LATEST" ]; then
  echo "❌ Could not fetch latest release tag from GitHub." >&2
  exit 1
fi

# ── Idempotent version check ─────────────────────────────────────────────────
if command -v mxcli >/dev/null 2>&1; then
  CURRENT=$(mxcli version 2>/dev/null | head -1 | awk '{print $3}' || echo "")
  if [ "$CURRENT" = "$LATEST" ]; then
    echo "✅ mxcli $CURRENT is already up to date."
    exit 0
  fi
  echo "Updating mxcli $CURRENT → $LATEST"
else
  echo "Installing mxcli $LATEST"
fi

# ── Determine install directory ───────────────────────────────────────────────
if [ -z "$INSTALL_DIR" ]; then
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
    # Idempotent PATH entry: add to each shell rc that exists, skip if already present
    for RC in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
      [ -f "$RC" ] || continue
      grep -qF "$INSTALL_DIR" "$RC" && continue
      printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$RC"
      echo "  Added $INSTALL_DIR to PATH in $RC"
    done
  fi
fi

# ── Download launcher binary ──────────────────────────────────────────────────
BIN_NAME="mxcli-${OS}-${ARCH}"
BIN_URL="https://github.com/${REPO}/releases/download/${LATEST}/${BIN_NAME}"
SUMS_URL="https://github.com/${REPO}/releases/download/${LATEST}/SHA256SUMS"
TMP=$(mktemp /tmp/mxcli.XXXXXX)
TMP_SUMS=$(mktemp /tmp/mxcli-sums.XXXXXX)
trap 'rm -f "$TMP" "$TMP_SUMS"' EXIT

echo "  Downloading launcher (${OS}/${ARCH}) from GitHub..."
curl -fsSL --progress-bar "$BIN_URL" -o "$TMP"

echo "  Verifying checksum..."
curl -fsSL "$SUMS_URL" -o "$TMP_SUMS"
EXPECTED=$(grep " ${BIN_NAME}$" "$TMP_SUMS" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
  echo "❌ No checksum entry for ${BIN_NAME} in SHA256SUMS" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
else
  echo "⚠️  sha256sum/shasum not found — skipping checksum verification" >&2
  ACTUAL="$EXPECTED"
fi
if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "❌ Checksum mismatch for ${BIN_NAME}" >&2
  echo "   expected: $EXPECTED" >&2
  echo "   got:      $ACTUAL" >&2
  exit 1
fi

chmod +x "$TMP"

# Atomic install (never leaves a partial binary)
mv "$TMP" "${INSTALL_DIR}/mxcli"

echo ""
echo "✅ mxcli $LATEST installed to ${INSTALL_DIR}/mxcli"
echo "   The daemon (~20 MB) will be downloaded automatically on first use."
echo ""
echo "   Run: mxcli version"
echo ""
if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
  echo "   NOTE: Restart your shell or run: export PATH=\"$INSTALL_DIR:\$PATH\""
fi
