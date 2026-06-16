#!/bin/sh
# scripts/lib/mx-check.sh — shared helper for running mx check with proper error handling.
#
# Usage:
#   . "$(dirname "$0")/../lib/mx-check.sh"   # source from any script
#   mx_check_against_baseline "$MPR" "$BASELINE_FILE" "$MX_BIN"
#
# Exit codes:
#   0 — PASS (error count ≤ baseline)
#   1 — FAIL (error count > baseline)
#   2 — CRASH (mx check exited non-zero but no [error] lines = tool crash)

# mx_check_against_baseline runs `mx check` on MPR and compares against the
# integer stored in BASELINE_FILE. Detects crashes separately from validation failures.
#
# Args:
#   $1 — path to .mpr file
#   $2 — path to .mx-check-baseline file (integer)
#   $3 — path to mx binary
mx_check_against_baseline() {
    local mpr="$1"
    local baseline_file="$2"
    local mx_bin="${3:-}"

    if [ -z "$mx_bin" ] || [ ! -x "$mx_bin" ]; then
        echo "mx_check_against_baseline: mx binary not found or not executable: $mx_bin" >&2
        return 2
    fi
    if [ ! -f "$mpr" ]; then
        echo "mx_check_against_baseline: MPR not found: $mpr" >&2
        return 2
    fi
    if [ ! -f "$baseline_file" ]; then
        echo "mx_check_against_baseline: baseline file missing: $baseline_file" >&2
        echo "  Create with: echo 0 > $baseline_file" >&2
        return 2
    fi

    local baseline
    baseline=$(cat "$baseline_file" | tr -d '[:space:]')

    local output ec
    output=$("$mx_bin" check "$mpr" 2>&1)
    ec=$?

    # Detect crash: tool exited non-zero but no [error] lines → crash, not validation failure.
    if [ $ec -ne 0 ] && ! echo "$output" | grep -q "^\[error\]"; then
        echo ""
        echo "CRASH: mx check crashed (exit $ec) — no [error] lines in output."
        echo "First 10 lines of output:"
        echo "$output" | head -10
        echo ""
        echo "This indicates a NullReferenceException or similar during MPR postprocessing."
        echo "Common cause: widget BSON missing required fields (e.g. ReturnType for"
        echo "expression-type properties). Check recent widget generation changes."
        return 2
    fi

    local errors
    errors=$(echo "$output" | grep -c "^\[error\]" || true)

    if [ "$errors" -gt "$baseline" ]; then
        local new
        new=$((errors - baseline))
        echo ""
        echo "FAIL: mx check found $new new error(s) ($errors total, baseline $baseline)."
        echo ""
        echo "$output" | grep "^\[error\]"
        return 1
    fi

    echo "PASS: mx check ($errors error(s), baseline $baseline)."
    return 0
}

# mx_discover_binary finds the mx binary for a given MPR file.
# Reads the Mendix version from the MPR SQLite metadata and looks in
# ~/.mxcli/mxbuild/<version>/modeler/mx (or nearest patch fallback).
#
# Args:
#   $1 — path to .mpr file
# Stdout:
#   path to mx binary, or empty if not found
mx_discover_binary() {
    local mpr="$1"
    local mx_version
    mx_version=$(sqlite3 "$mpr" "SELECT _ProductVersion FROM _MetaData" 2>/dev/null | head -1)
    if [ -z "$mx_version" ]; then
        return 0  # v2 MPR or sqlite3 unavailable — caller handles
    fi

    local mx_bin="$HOME/.mxcli/mxbuild/${mx_version}/modeler/mx"
    if [ ! -x "$mx_bin" ]; then
        local minor
        minor=$(echo "$mx_version" | cut -d. -f1-2)
        mx_bin=$(ls "$HOME/.mxcli/mxbuild/${minor}."*/modeler/mx 2>/dev/null | sort -V | tail -1)
    fi

    if [ -x "$mx_bin" ]; then
        echo "$mx_bin"
    fi
}
