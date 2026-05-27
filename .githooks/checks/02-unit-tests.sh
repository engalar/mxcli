#!/bin/sh
# L1: unit tests — fast, no external deps, no Docker.
#
# Time guard: each successful run is appended to .test-time-log and the
# elapsed time is compared against .test-time-baseline. Both files are
# committed so the history is visible and shared.
#
# .test-time-baseline  — last recorded elapsed time (seconds), updated on
#                        every pass. Fail if current > baseline * 1.5.
# .test-time-log       — append-only history: one line per run, format:
#                          ISO8601  elapsed_s  git_sha  branch
#
# To reset the baseline after intentional perf change:
#   echo <new_seconds> > .test-time-baseline
#   git add .test-time-baseline .test-time-log && git commit -m "perf: reset test baseline"

REPO="$(git rev-parse --show-toplevel)"
LOG_FILE="${REPO}/.test-fail.log"
BASELINE_FILE="${REPO}/.test-time-baseline"
TIMELOG_FILE="${REPO}/.test-time-log"
rm -f "$LOG_FILE"

echo "pre-commit: running unit tests..."

START=$(date +%s)
if ! CGO_ENABLED=0 go test -timeout 180s -p "$(nproc)" -parallel "$(nproc)" ./... > "$LOG_FILE" 2>&1; then
    echo "" >&2
    echo "COMMIT BLOCKED: unit tests failed." >&2
    echo "" >&2
    cat "$LOG_FILE" >&2
    echo "SOP: .githooks/sop/02-unit-tests.md" >&2
    echo "CONTEXT: LOG_FILE=${LOG_FILE}" >&2
    exit 1
fi
END=$(date +%s)
ELAPSED=$((END - START))
rm -f "$LOG_FILE"

# Append to time log.
SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
printf "%s\t%ss\t%s\t%s\n" "$TS" "$ELAPSED" "$SHA" "$BRANCH" >> "$TIMELOG_FILE"

# Time guard: compare against last successful run.
if [ -f "$BASELINE_FILE" ]; then
    BASELINE=$(cat "$BASELINE_FILE")
    THRESHOLD=$((BASELINE + BASELINE / 2))   # 150% of baseline
    if [ "$ELAPSED" -gt "$THRESHOLD" ]; then
        echo "" >&2
        echo "COMMIT BLOCKED: test time regression." >&2
        echo "  Current:  ${ELAPSED}s" >&2
        echo "  Baseline: ${BASELINE}s  (limit: ${THRESHOLD}s = baseline × 1.5)" >&2
        echo "" >&2
        tail -10 "$TIMELOG_FILE" | awk -F'\t' '{printf "  %s\t%s\n", $1, $2}' >&2
        echo "" >&2
        echo "To reset: echo ${ELAPSED} > .test-time-baseline" >&2
        exit 1
    fi
fi

# Update baseline to the current run so the guard tracks recent history.
echo "$ELAPSED" > "$BASELINE_FILE"

echo "pre-commit: unit tests passed (${ELAPSED}s)."
