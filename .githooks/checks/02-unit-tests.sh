#!/bin/sh
# L1: unit tests — fast, no external deps, no Docker.
#
# CPU is capped at 85% so the machine stays responsive:
#   - cpulimit(1) if installed: hard CPU cap via kernel scheduler
#   - otherwise: nice -n 15 (lower scheduling priority)
#
# Time guard: compare elapsed seconds against .test-time-baseline (committed).
# Fail if current > baseline * 1.5. Update baseline on every passing run.
# To reset: echo <new_seconds> > .test-time-baseline

REPO="$(git rev-parse --show-toplevel)"
LOG_FILE="${REPO}/.test-fail.log"
BASELINE_FILE="${REPO}/.test-time-baseline"
rm -f "$LOG_FILE"

# Determine parallelism: 85% of nproc.
NCPU=$(nproc)
P85=$(( NCPU * 85 / 100 ))
[ "$P85" -lt 1 ] && P85=1

# CPU limiter: cpulimit if available, else nice.
if command -v cpulimit >/dev/null 2>&1; then
    LIMIT_PCT=$(( NCPU * 85 ))
    RUNNER="cpulimit -l ${LIMIT_PCT} --"
else
    RUNNER="nice -n 15"
fi

echo "pre-commit: running unit tests (cpu≤85%, p=${P85})..."

export CGO_ENABLED=0
START=$(date +%s)
if ! $RUNNER go test \
        -timeout 180s \
        -p "${P85}" \
        -parallel "${P85}" \
        ./... > "$LOG_FILE" 2>&1; then
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

# Time guard. Clamp baseline to MIN_BASELINE so cached runs (0 s) don't let
# the next cold run through unchecked.
MIN_BASELINE=20
if [ -f "$BASELINE_FILE" ]; then
    BASELINE=$(cat "$BASELINE_FILE")
    # Guard against a zero/tiny baseline from a cached run.
    if [ "$BASELINE" -lt "$MIN_BASELINE" ]; then
        BASELINE="$MIN_BASELINE"
    fi
    THRESHOLD=$((BASELINE + BASELINE / 2))
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

# Only update baseline when elapsed ≥ MIN_BASELINE (cold run).
# Cached runs (< MIN_BASELINE) are not representative; keep the existing value.
if [ "$ELAPSED" -ge "$MIN_BASELINE" ]; then
    echo "$ELAPSED" > "$BASELINE_FILE"
fi
echo "pre-commit: unit tests passed (${ELAPSED}s)."
