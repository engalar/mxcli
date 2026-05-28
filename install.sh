#!/bin/sh
# mxcli install script — idempotent, works on Linux and macOS.
# Usage: curl -fsSL https://raw.githubusercontent.com/engalar/mxcli/main/install.sh | sh
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
BIN_URL="https://github.com/${REPO}/releases/download/${LATEST}/mxcli-${OS}-${ARCH}"
TMP=$(mktemp /tmp/mxcli.XXXXXX)
trap 'rm -f "$TMP"' EXIT

echo "  Downloading launcher (${OS}/${ARCH}) from GitHub..."
curl -fsSL --progress-bar "$BIN_URL" -o "$TMP"
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
