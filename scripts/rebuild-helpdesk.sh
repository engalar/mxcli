#!/bin/sh
# rebuild-helpdesk.sh — Rebuild the helpdesk + FT application from MDL
#
# Usage:
#   ./scripts/rebuild-helpdesk.sh                        # → testdata/helpdesk-golden-11.6.6/
#   ./scripts/rebuild-helpdesk.sh --output /tmp/myapp    # → /tmp/myapp/
#   ./scripts/rebuild-helpdesk.sh --keep                  # → <tmpdir>, printed at end
#   ./scripts/rebuild-helpdesk.sh clean_dir mdl_file mxcli_bin
#
# --output DIR  : copy result to DIR (persistent)
# --keep        : print tmpdir path instead of cleaning up

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Defaults
CLEAN="$BASE_DIR/testdata/helpdesk-clean-11.6.6"
MDL="$BASE_DIR/cmd/mxcli/examples/helpdesk-app.mdl"
MXCLI="$BASE_DIR/bin/mxcli"
OUTPUT=""
KEEP=false

# Parse flags + positional args
while [ $# -gt 0 ]; do
    case "$1" in
        --output)
            OUTPUT="$2"; shift 2 ;;
        --output=*)
            OUTPUT="${1#--output=}"; shift ;;
        --keep)
            KEEP=true; shift ;;
        -*)
            echo "Unknown flag: $1" >&2; exit 1 ;;
        *)
            # positional: clean_dir (1), mdl_file (2), mxcli_bin (3)
            if [ -z "$CLEAN_SET" ]; then
                CLEAN="$1"; CLEAN_SET=1
            elif [ -z "$MDL_SET" ]; then
                MDL="$1"; MDL_SET=1
            else
                MXCLI="$1"
            fi
            shift ;;
    esac
done

# Validate inputs
if [ ! -f "$CLEAN/minimal.mpr" ]; then
    echo "ERROR: clean project not found: $CLEAN/minimal.mpr" >&2
    exit 1
fi
if [ ! -f "$MDL" ]; then
    echo "ERROR: MDL file not found: $MDL" >&2
    exit 1
fi
if [ ! -x "$MXCLI" ]; then
    echo "ERROR: mxcli binary not executable: $MXCLI" >&2
    exit 1
fi

TMPDIR=$(mktemp -d)
if [ "$KEEP" = false ] && [ -z "$OUTPUT" ]; then
    trap 'rm -rf "$TMPDIR"' EXIT
fi

echo "==> Phase 0: Copying clean project..."
cp -r "$CLEAN/." "$TMPDIR/"
MPR="$TMPDIR/minimal.mpr"

echo "==> Phase 1: Core + Account Management entities/microflows..."
sed -n '1,1681p' "$MDL" > "$TMPDIR/p1.mdl"
"$MXCLI" exec -p "$MPR" "$TMPDIR/p1.mdl" >"$TMPDIR/p1.log" 2>&1 && echo "  OK" || {
    echo "FAIL" >&2; tail -5 "$TMPDIR/p1.log" >&2; exit 1; }

echo "==> Phase 2: Account Management pages..."
sed -n '1682,2007p' "$MDL" > "$TMPDIR/p2.mdl"
"$MXCLI" exec -p "$MPR" "$TMPDIR/p2.mdl" >"$TMPDIR/p2.log" 2>&1 && echo "  OK" || {
    echo "FAIL" >&2; tail -5 "$TMPDIR/p2.log" >&2; exit 1; }

echo "==> Phase 3: Security, navigation, folder moves..."
sed -n '2008,2292p' "$MDL" > "$TMPDIR/p3.mdl"
"$MXCLI" exec -p "$MPR" "$TMPDIR/p3.mdl" >"$TMPDIR/p3.log" 2>&1 && echo "  OK" || {
    echo "FAIL" >&2; tail -5 "$TMPDIR/p3.log" >&2; exit 1; }

echo "==> Phase 4: Register languages..."
echo "alter settings language add 'zh_CN';" > "$TMPDIR/p4a.mdl"
echo "  zh_CN..."
"$MXCLI" exec -p "$MPR" "$TMPDIR/p4a.mdl" >"$TMPDIR/p4a.log" 2>&1 && echo "    OK" || {
    echo "FAIL" >&2; exit 1; }
echo "alter settings language add 'nl_NL' (checkCompleteness: true);" > "$TMPDIR/p4b.mdl"
echo "  nl_NL..."
"$MXCLI" exec -p "$MPR" "$TMPDIR/p4b.mdl" >"$TMPDIR/p4b.log" 2>&1 && echo "    OK" || {
    echo "FAIL" >&2; exit 1; }
echo "alter settings language add 'fr_FR';" > "$TMPDIR/p4c.mdl"
echo "  fr_FR..."
"$MXCLI" exec -p "$MPR" "$TMPDIR/p4c.mdl" >"$TMPDIR/p4c.log" 2>&1 && echo "    OK" || {
    echo "FAIL" >&2; exit 1; }

echo "==> Phase 5: Translations, FT module, ALTER PAGE..."
sed -n '2320,$p' "$MDL" > "$TMPDIR/p5.mdl"
"$MXCLI" exec -p "$MPR" "$TMPDIR/p5.mdl" >"$TMPDIR/p5.log" 2>&1 && echo "  OK" || {
    echo "FAIL" >&2; tail -5 "$TMPDIR/p5.log" >&2; exit 1; }

# ── Summary ────────────────────────────────────
echo ""
echo "=== Build complete ==="
echo "  Project: $MPR"

echo ""
echo "=== Modules ==="
"$MXCLI" -p "$MPR" -c "show modules" 2>/dev/null \
    | grep -E "^Module |^---| KB | HD | FT |^$"

echo ""
echo "=== Languages ==="
"$MXCLI" -p "$MPR" -c "show languages" 2>/dev/null \
    | grep -E "^| Code " -A 10 | head -8

echo ""
echo "=== Demo users ==="
"$MXCLI" -p "$MPR" -c "show demo users" 2>/dev/null \
    | grep "demo_"

# Optional mx check
MX_BIN=$(ls "$HOME"/.mxcli/mxbuild/*/modeler/mx 2>/dev/null | sort -V | tail -1)
if [ -n "$MX_BIN" ] && [ -x "$MX_BIN" ]; then
    echo ""
    echo "=== mx check ==="
    OUT=$("$MX_BIN" check "$MPR" 2>&1) || true
    echo "$OUT" | grep -E "^\[error\]|^\[warn\]" | head -5
    echo "$OUT" | grep "errors found\|warnings found" | tail -1
fi

# Copy to output dir if requested
if [ -n "$OUTPUT" ]; then
    mkdir -p "$OUTPUT"
    cp -r "$TMPDIR/." "$OUTPUT/"
    echo ""
    echo "Result copied to: $OUTPUT/"
fi

echo ""
echo "Done."
