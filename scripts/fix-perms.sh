#!/bin/bash
# fix-perms.sh — Reset ownership and permissions for multi-user repo access.
#
# Run with sudo whenever permission issues arise:
#   sudo scripts/fix-perms.sh
#
# This ensures both Linux users in the devshare group can read/write
# the repo without stepping on each other's permissions.

set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GROUP="${GROUP:-devshare}"
OWNER="${OWNER:-eg}"

if [[ $EUID -ne 0 ]]; then
  echo "ERROR: This script must be run as root (sudo)." >&2
  echo "  sudo $0" >&2
  exit 1
fi

echo "=== Fixing permissions for $REPO_DIR ==="
echo "  Owner: $OWNER   Group: $GROUP"
echo ""

# ── Fix ownership ────────────────────────────────────────────────
# All files belong to $OWNER:$GROUP so both users in the group can
# access them (group permissions in the next step).
echo "--- Step 1: Fixing ownership (chown) ---"
chown -R "$OWNER":"$GROUP" "$REPO_DIR"

# ── Fix directory permissions ────────────────────────────────────
# Drwxrws--- (2770) = rwx for owner+group, setgid bit so new files
# inherit the group.  The setgid bit is critical — without it, a
# file created by user B gets user B's primary group instead of
# 'devshare'.
echo "--- Step 2: Fixing directory permissions (chmod 2770) ---"
find "$REPO_DIR" -type d -exec chmod 2770 {} +

# ── Fix file permissions ─────────────────────────────────────────
# -rw-rw---- (660) = rw for owner+group, nothing for others.
# Exclude files that need +x (scripts, binaries).
echo "--- Step 3: Fixing file permissions (chmod 660 +x) ---"
find "$REPO_DIR" -type f -exec chmod 660 {} +

# Restore execute bits on files that need them.
# Match common executable patterns.
echo "--- Step 4: Restoring execute bits ---"
find "$REPO_DIR" -type f \( \
  -name "*.sh" -o \
  -name "*.py" -o \
  -name "*.pl" -o \
  -name "*.rb" -o \
  -path "*/bin/*" -o \
  -path "*/.githooks/*" -o \
  -name "configure" -o \
  -name "Makefile*" \
\) -exec chmod 770 {} + 2>/dev/null || true

# Restore execute on the mxcli binary
chmod 770 "$REPO_DIR/bin/mxcli" 2>/dev/null || true

# ── Special-case: ensure .git permissions work ───────────────────
# Git needs to be able to write to .git/objects, .git/refs, etc.
if [[ -d "$REPO_DIR/.git" ]]; then
  chmod -R 2770 "$REPO_DIR/.git"
fi

# ── Verify ────────────────────────────────────────────────────────
echo ""
echo "=== Verification (spot check) ==="
STAT=$(stat -c "%a %G %U" "$REPO_DIR" 2>/dev/null || stat -f "%Sp %Sg %Su" "$REPO_DIR" 2>/dev/null)
echo "  Repo root: $STAT"

# Quick test: can the group write a temp file?
TEST_FILE="$REPO_DIR/.perm_test_$$"
if touch "$TEST_FILE" 2>/dev/null; then
  rm -f "$TEST_FILE"
  echo "  Result: OK — group can write to repo"
else
  echo "  WARNING: group still cannot write to repo"
fi

echo ""
echo "=== Done ==="
echo "Run as the other user to verify: ls -la $REPO_DIR"
