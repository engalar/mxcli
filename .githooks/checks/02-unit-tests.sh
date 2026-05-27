#!/bin/sh
# L1: unit tests — fast, no external deps, no Docker.
#
# Time guard: elapsed seconds are compared against .test-time-baseline.
# The baseline file is committed so the reference survives machine changes.
#
# .test-time-baseline  — last recorded elapsed time (seconds). Updated on
#                        every passing run. Fail if current > baseline * 1.5.
#
# To reset after an intentional perf change:
#   echo <new_seconds> > .test-time-baseline
#   git add .test-time-baseline && git commit -m "perf: reset test baseline"

REPO="$(git rev-parse --show-toplevel)"
LOG_FILE="${REPO}/.test-fail.log"
BASELINE_FILE="${REPO}/.test-time-baseline"
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
        echo "To reset: echo ${ELAPSED} > .test-time-baseline" >&2
        exit 1
    fi
fi

# Update baseline to current run time.
echo "$ELAPSED" > "$BASELINE_FILE"
echo "pre-commit: unit tests passed (${ELAPSED}s)."
