#!/bin/sh
# mxcli install script — idempotent, works on Linux, macOS, and Windows (Git Bash / MSYS2).
# Usage: curl -fsSL https://raw.githubusercontent.com/engalar/mxcli/dev/install.sh | sh
#
# Optional env vars:
#   MXCLI_INSTALL_DIR  — override install directory (default: /usr/local/bin or ~/.local/bin)
set -e

REPO="engalar/mxcli"
INSTALL_DIR="${MXCLI_INSTALL_DIR:-}"

# ── Detect platform ──────────────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  mingw*|msys*|cygwin*) OS="windows" ;;
esac
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
  linux|darwin|windows) ;;
  *)
    echo "❌ Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

EXT=""
[ "$OS" = "windows" ] && EXT=".exe"

# ── Fetch latest release tag ─────────────────────────────────────────────────
# Use the redirect from /releases/latest — avoids GitHub API rate limits
LATEST=$(curl -fsSL -o /dev/null -w '%{url_effective}' \
  "https://github.com/${REPO}/releases/latest" \
  | sed 's|.*/tag/||' | tr -d '[:space:]')
if [ -z "$LATEST" ]; then
  echo "❌ Could not fetch latest release tag from GitHub." >&2
  exit 1
fi

# ── Idempotent version check ─────────────────────────────────────────────────
if command -v mxcli >/dev/null 2>&1; then
  CURRENT=$(mxcli --version 2>/dev/null | sed 's/.*version //; s/ .*//' || echo "")
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
  if [ "$OS" != "windows" ] && [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi
fi

# ── Update shell PATH (idempotent) ────────────────────────────────────────────
if [ "$INSTALL_DIR" != "/usr/local/bin" ]; then
  if [ "$OS" = "windows" ]; then
    # Git Bash / MSYS2: ensure ~/.bashrc exists and contains the PATH entry
    RC="$HOME/.bashrc"
    touch "$RC"
    if ! grep -qF "$INSTALL_DIR" "$RC" 2>/dev/null; then
      printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$RC"
      echo "  Added $INSTALL_DIR to PATH in $RC (Git Bash)"
    fi

    # Windows user PATH via PowerShell — persists for CMD, PowerShell, and new shells
    if command -v powershell.exe >/dev/null 2>&1; then
      WIN_DIR=$(cygpath -w "$INSTALL_DIR" 2>/dev/null || \
        printf '%s' "$INSTALL_DIR" | sed 's|^/\([a-zA-Z]\)/|\1:\\|;s|/|\\|g')
      CURRENT_WIN=$(powershell.exe -NoProfile -Command \
        "[Environment]::GetEnvironmentVariable('PATH','User')" 2>/dev/null | tr -d '\r\n')
      case ";${CURRENT_WIN};" in
        *";${WIN_DIR};"*)
          : # already present
          ;;
        *)
          NEW_WIN="${CURRENT_WIN};${WIN_DIR}"
          powershell.exe -NoProfile -Command \
            "[Environment]::SetEnvironmentVariable('PATH','${NEW_WIN}','User')" 2>/dev/null \
            && echo "  Added $WIN_DIR to Windows user PATH (takes effect in new shells)"
          ;;
      esac
    fi
  else
    # Linux / macOS: update each shell rc that exists
    for RC in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
      [ -f "$RC" ] || continue
      grep -qF "$INSTALL_DIR" "$RC" && continue
      printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$RC"
      echo "  Added $INSTALL_DIR to PATH in $RC"
    done
  fi
fi

# ── Download launcher binary ──────────────────────────────────────────────────
BIN_NAME="mxcli-${OS}-${ARCH}${EXT}"
BIN_URL="https://github.com/${REPO}/releases/download/${LATEST}/${BIN_NAME}"
SUMS_URL="https://github.com/${REPO}/releases/download/${LATEST}/SHA256SUMS"
TMP=$(mktemp /tmp/mxcli.XXXXXX)
TMP_SUMS=$(mktemp /tmp/mxcli-sums.XXXXXX)
trap 'rm -f "$TMP" "$TMP_SUMS"' EXIT

echo "  Downloading mxcli (${OS}/${ARCH}) from GitHub..."
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
mkdir -p "$INSTALL_DIR"
mv "$TMP" "${INSTALL_DIR}/mxcli${EXT}"

echo ""
echo "✅ mxcli $LATEST installed to ${INSTALL_DIR}/mxcli${EXT}"
echo ""
echo "   Run: mxcli version"
echo ""
if [ "$OS" = "windows" ]; then
  echo "   To use in the current Git Bash session:"
  echo "     export PATH=\"$INSTALL_DIR:\$PATH\""
  echo "   New Git Bash sessions and PowerShell/CMD will pick it up automatically."
elif [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
  echo "   NOTE: Restart your shell or run: export PATH=\"$INSTALL_DIR:\$PATH\""
fi

# Install shell completions (best-effort — requires mxcli in PATH)
if command -v mxcli >/dev/null 2>&1; then
  mxcli setup completions >/dev/null 2>&1 || true
fi
