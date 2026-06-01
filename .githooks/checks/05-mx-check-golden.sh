#!/bin/sh
# Guard: run `mx check` on testdata/helpdesk-golden-11.6.6/ when MPR files are staged.
# Blocks if the staged MPR introduces NEW errors beyond the stored baseline.
#
# Baseline: testdata/helpdesk-golden-11.6.6/.mx-check-baseline (committed integer, updated
# by make update-helpdesk-golden when the baseline intentionally changes).
#
# mx binary: auto-discovered from ~/.mxcli/mxbuild/<version>/modeler/mx using
# the version recorded in testdata/helpdesk-golden-11.6.6/minimal.mpr.

staged_mpr=$(git diff --cached --name-only | \
  grep -E '^testdata/helpdesk-golden-11.6.6/(minimal\.mpr|mprcontents/)' | head -1)

if [ -z "$staged_mpr" ]; then
    exit 0
fi

MPR="testdata/helpdesk-golden-11.6.6/minimal.mpr"
BASELINE_FILE="testdata/helpdesk-golden-11.6.6/.mx-check-baseline"

# Read baseline error count.
if [ ! -f "$BASELINE_FILE" ]; then
    echo "pre-commit: WARNING: $BASELINE_FILE missing — skipping mx check." >&2
    exit 0
fi
baseline=$(cat "$BASELINE_FILE" | tr -d '[:space:]')

# Discover Mendix version from the MPR SQLite metadata (v2 MPR: mprcontents/ layout).
mx_version=$(sqlite3 "$MPR" "SELECT _ProductVersion FROM _MetaData" 2>/dev/null | head -1)
if [ -z "$mx_version" ]; then
    echo "pre-commit: WARNING: cannot detect Mendix version from $MPR — skipping mx check." >&2
    exit 0
fi

# Find mx binary: exact version first, then nearest patch.
mx_bin="$HOME/.mxcli/mxbuild/${mx_version}/modeler/mx"
if [ ! -x "$mx_bin" ]; then
    # Fall back to any installed patch of the same minor.
    minor=$(echo "$mx_version" | cut -d. -f1-2)
    mx_bin=$(ls "$HOME/.mxcli/mxbuild/${minor}."*/modeler/mx 2>/dev/null | sort -V | tail -1)
fi
if [ -z "$mx_bin" ] || [ ! -x "$mx_bin" ]; then
    echo "pre-commit: WARNING: mx ${mx_version} not found — skipping mx check." >&2
    echo "  Install with: mxcli setup mxbuild -v ${mx_version}" >&2
    exit 0
fi

echo "pre-commit: running mx check (Mendix ${mx_version})..." >&2

output=$("$mx_bin" check "$MPR" 2>&1)
error_count=$(echo "$output" | grep -c '^\[error\]' || true)

if [ "$error_count" -gt "$baseline" ]; then
    new=$((error_count - baseline))
    echo "" >&2
    echo "COMMIT BLOCKED: mx check found ${new} new error(s) (${error_count} total, baseline ${baseline})." >&2
    echo "" >&2
    echo "$output" | grep '^\[error\]' >&2
    echo "" >&2
    echo "  Fix the errors, then re-run: make update-helpdesk-golden" >&2
    echo "  If errors are intentional (baseline change), update: $BASELINE_FILE" >&2
    echo "" >&2
    echo "SOP: .githooks/sop/05-mx-check-golden.md" >&2
    echo "CONTEXT: ERROR_COUNT=${error_count} BASELINE=${baseline} MX_VERSION=${mx_version}" >&2
    exit 1
fi

echo "pre-commit: mx check passed (${error_count} error(s), baseline ${baseline})." >&2
