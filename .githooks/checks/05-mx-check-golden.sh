#!/bin/sh
# Guard: run `mx check` on testdata/helpdesk-golden-11.6.6/ when MPR files are staged.
# Blocks if the staged MPR introduces NEW errors beyond the stored baseline.
# Also blocks if mx check CRASHES (NullReferenceException etc.) — crash = exit 2.
#
# Baseline: testdata/helpdesk-golden-11.6.6/.mx-check-baseline (committed integer, updated
# by make update-helpdesk-golden when the baseline intentionally changes).
#
# mx binary: auto-discovered from ~/.mxcli/mxbuild/<version>/modeler/mx using
# the version recorded in testdata/helpdesk-golden-11.6.6/minimal.mpr.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/../lib/mx-check.sh" 2>/dev/null || {
    # lib not found (hook called from a different CWD) — fall back to inline impl
    mx_check_against_baseline() {
        local mpr="$1" baseline_file="$2" mx_bin="$3"
        local baseline output ec errors
        baseline=$(cat "$baseline_file" | tr -d '[:space:]')
        output=$("$mx_bin" check "$mpr" 2>&1); ec=$?
        if [ $ec -ne 0 ] && ! echo "$output" | grep -q "^\[error\]"; then
            echo "CRASH: mx check crashed (exit $ec)" >&2
            echo "$output" | head -5 >&2
            return 2
        fi
        errors=$(echo "$output" | grep -c "^\[error\]" || true)
        if [ "$errors" -gt "$baseline" ]; then
            echo "FAIL: $errors errors (baseline $baseline)" >&2
            echo "$output" | grep "^\[error\]" >&2
            return 1
        fi
        echo "PASS: $errors error(s), baseline=$baseline"
        return 0
    }
}

staged_mpr=$(git diff --cached --name-only | \
  grep -E '^testdata/helpdesk-golden-11.6.6/(minimal\.mpr|mprcontents/)' | head -1)

if [ -z "$staged_mpr" ]; then
    exit 0
fi

MPR="testdata/helpdesk-golden-11.6.6/minimal.mpr"
BASELINE_FILE="testdata/helpdesk-golden-11.6.6/.mx-check-baseline"

if [ ! -f "$BASELINE_FILE" ]; then
    echo "pre-commit: WARNING: $BASELINE_FILE missing — skipping mx check." >&2
    exit 0
fi

# Discover Mendix version from the MPR SQLite metadata (v2 MPR: mprcontents/ layout).
mx_version=$(sqlite3 "$MPR" "SELECT _ProductVersion FROM _MetaData" 2>/dev/null | head -1)
if [ -z "$mx_version" ]; then
    echo "pre-commit: WARNING: cannot detect Mendix version from $MPR — skipping mx check." >&2
    exit 0
fi

# Find mx binary: exact version first, then nearest patch.
mx_bin="$HOME/.mxcli/mxbuild/${mx_version}/modeler/mx"
if [ ! -x "$mx_bin" ]; then
    minor=$(echo "$mx_version" | cut -d. -f1-2)
    mx_bin=$(ls "$HOME/.mxcli/mxbuild/${minor}."*/modeler/mx 2>/dev/null | sort -V | tail -1)
fi
if [ -z "$mx_bin" ] || [ ! -x "$mx_bin" ]; then
    echo "pre-commit: WARNING: mx ${mx_version} not found — skipping mx check." >&2
    echo "  Install with: mxcli setup mxbuild -v ${mx_version}" >&2
    exit 0
fi

echo "pre-commit: running mx check (Mendix ${mx_version})..." >&2

result=$(mx_check_against_baseline "$MPR" "$BASELINE_FILE" "$mx_bin")
rc=$?

case $rc in
    0)
        echo "pre-commit: $result" >&2
        ;;
    1)
        echo "" >&2
        echo "COMMIT BLOCKED: $result" >&2
        echo "" >&2
        echo "  Fix the errors, then re-run: make update-helpdesk-golden" >&2
        echo "  If errors are intentional (baseline change), update: $BASELINE_FILE" >&2
        echo "" >&2
        echo "SOP: .githooks/sop/05-mx-check-golden.md" >&2
        exit 1
        ;;
    2)
        echo "" >&2
        echo "COMMIT BLOCKED: mx check CRASHED — widget BSON is invalid." >&2
        echo "" >&2
        echo "  This usually means a widget property is missing a required BSON field" >&2
        echo "  (e.g. ReturnType for expression-type properties)." >&2
        echo "  Run: make test-section-check" >&2
        echo "  Then check the crash output above for the failing widget." >&2
        echo "" >&2
        echo "SOP: .githooks/sop/05-mx-check-golden.md" >&2
        exit 1
        ;;
esac
