#!/bin/sh
# test-section-check.sh — Cross-section mx check validation
#
# Executes helpdesk-app.mdl section-by-section (one `mxcli exec` process per
# -- MARK: section, simulating real-world mxcli exec usage), then runs mx check
# and verifies the error count does not exceed .mx-check-baseline.
#
# This catches regressions where entity/return-type resolution only works
# inside a single executor session but fails when state must come from the MPR.
#
# Usage: make test-section-check
# or:    ./scripts/test-section-check.sh [clean_dir] [baseline_file] [mdl_file]

set -e

CLEAN="${1:-testdata/helpdesk-clean-11.6.6}"
BASELINE="${2:-testdata/helpdesk-golden-11.6.6/.mx-check-baseline}"
MDL="${3:-mdl-examples/use-cases/helpdesk/helpdesk-app.mdl}"
MXCLI="${MXCLI:-./bin/mxcli}"

if [ ! -f "$BASELINE" ]; then
    echo "ERROR: baseline file not found: $BASELINE" >&2
    exit 1
fi
if [ ! -f "$CLEAN/minimal.mpr" ]; then
    echo "ERROR: clean project not found: $CLEAN/minimal.mpr" >&2
    exit 1
fi

# Discover Mendix version.
# v2 MPR (Mendix >= 10.18) stores version in mprcontents/; v1 MPR uses SQLite.
# Try mxcli first (works for both), then fall back to SQLite for v1.
MX_VERSION=$("$MXCLI" -p "$CLEAN/minimal.mpr" -c "show features" 2>/dev/null | grep "Connected to:" | sed 's/.*Mendix //' | tr -d ')')
if [ -z "$MX_VERSION" ]; then
    MX_VERSION=$(sqlite3 "$CLEAN/minimal.mpr" "SELECT _ProductVersion FROM _MetaData" 2>/dev/null | head -1)
fi
if [ -z "$MX_VERSION" ]; then
    echo "ERROR: cannot detect Mendix version from $CLEAN/minimal.mpr" >&2
    exit 1
fi

# Find mx binary.
MX_BIN="$HOME/.mxcli/mxbuild/${MX_VERSION}/modeler/mx"
if [ ! -x "$MX_BIN" ]; then
    minor=$(echo "$MX_VERSION" | cut -d. -f1-2)
    MX_BIN=$(ls "$HOME/.mxcli/mxbuild/${minor}."*/modeler/mx 2>/dev/null | sort -V | tail -1)
fi
if [ -z "$MX_BIN" ] || [ ! -x "$MX_BIN" ]; then
    echo "WARNING: mx ${MX_VERSION} not found — install with:" >&2
    echo "  mxcli setup mxbuild -p $CLEAN/minimal.mpr" >&2
    echo "Skipping mx check (section execution still validated)." >&2
    MX_BIN=""
fi

# Copy clean project to temp dir.
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR" "$SECTION_DIR"' EXIT
cp -r "$CLEAN/." "$TMPDIR/"

# Split MDL into sections at -- MARK: boundaries.
SECTION_DIR=$(mktemp -d)
csplit --quiet --prefix="$SECTION_DIR/section-" --suffix="%02d.mdl" \
    "$MDL" '/^-- MARK:/' '{*}'

echo "Running section-by-section execution ($(ls "$SECTION_DIR"/section-*.mdl | wc -l) sections)..."

FAILED=""
for f in "$SECTION_DIR"/section-*.mdl; do
    mark=$(head -1 "$f" | tr -d '\r')
    if ! "$MXCLI" -p "$TMPDIR/minimal.mpr" exec "$f" >/dev/null 2>&1; then
        echo "  FAIL: $mark" >&2
        FAILED="$FAILED $f"
    fi
done

if [ -n "$FAILED" ]; then
    echo "FAIL: $FAILED" >&2
    exit 1
fi
echo "  All sections executed successfully."

# Run mx check if binary is available.
if [ -n "$MX_BIN" ]; then
    baseline=$(cat "$BASELINE" | tr -d '[:space:]')
    output=$("$MX_BIN" check "$TMPDIR/minimal.mpr" 2>&1)
    errors=$(echo "$output" | grep -c '^\[error\]' || true)

    if [ "$errors" -gt "$baseline" ]; then
        new=$((errors - baseline))
        echo "" >&2
        echo "FAIL: mx check found $new new error(s) ($errors total, baseline $baseline)." >&2
        echo "" >&2
        echo "$output" | grep '^\[error\]' >&2
        exit 1
    fi
    echo "mx check: PASS ($errors error(s), baseline $baseline, version $MX_VERSION)."
fi

echo "PASS: test-section-check"
