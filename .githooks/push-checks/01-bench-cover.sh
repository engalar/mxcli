#!/bin/sh
# pre-push: benchmark regression + coverage drop guard.
#
# Baselines (committed, so shared across the team):
#   coverage/bench-baseline.txt   — raw Go benchmark output (-count=3)
#   coverage/coverage-baseline.txt — total coverage percentage
#
# Thresholds (override via env):
#   BENCH_REGRESS_PCT  — max allowed slowdown in ns/op  (default: 20)
#   COVER_DROP_PCT     — max allowed coverage drop in %  (default: 1)
#
# Tools:
#   benchstat (golang.org/x/perf/cmd/benchstat) — used when installed.
#   Otherwise: built-in awk parser for ns/op comparison.
#   go tool cover — always available.
#
# To reset baselines after an intentional change:
#   make bench-baseline   (or see instructions at end of this file)

REPO="$(git rev-parse --show-toplevel)"
NCPU=$(nproc)
P85=$(( NCPU * 85 / 100 ))
[ "$P85" -lt 1 ] && P85=1

# CPU limiter
if command -v cpulimit >/dev/null 2>&1; then
    LIMIT_PCT=$(( NCPU * 85 ))
    RUNNER="cpulimit -l ${LIMIT_PCT} --"
else
    RUNNER="nice -n 15"
fi

export CGO_ENABLED=0
BENCH_REGRESS_PCT="${BENCH_REGRESS_PCT:-20}"
COVER_DROP_PCT="${COVER_DROP_PCT:-1}"

BENCH_BASELINE="${REPO}/coverage/bench-baseline.txt"
BENCH_CURRENT="${REPO}/coverage/bench-current.txt"
COV_BASELINE="${REPO}/coverage/coverage-baseline.txt"
COV_PROFILE="${REPO}/coverage/coverage.out"
mkdir -p "${REPO}/coverage"

FAILED=0

# ── Benchmarks ──────────────────────────────────────────────────────────────

echo "pre-push: running benchmarks (count=3, cpu≤85%)..."
$RUNNER go test -bench=. -benchmem -count=3 \
    -p "${P85}" ./... 2>/dev/null | \
    grep -v "^---" > "$BENCH_CURRENT"

if [ ! -s "$BENCH_CURRENT" ]; then
    echo "WARNING: no benchmark output produced — skipping bench guard."
else
    if [ -f "$BENCH_BASELINE" ]; then
        echo "pre-push: comparing benchmarks against baseline..."

        if command -v benchstat >/dev/null 2>&1; then
            # benchstat is the gold standard.
            BENCH_DIFF=$(benchstat "$BENCH_BASELINE" "$BENCH_CURRENT" 2>&1)
            echo "$BENCH_DIFF"
            # Detect regressions: lines with +XX.XX% where XX > threshold.
            REGRESSIONS=$(echo "$BENCH_DIFF" | awk -v thr="$BENCH_REGRESS_PCT" '
                /\+[0-9]+\.[0-9]+%/ {
                    match($0, /\+([0-9]+\.[0-9]+)%/, m)
                    if (m[1]+0 > thr+0) print $0
                }')
            if [ -n "$REGRESSIONS" ]; then
                echo "" >&2
                echo "PUSH BLOCKED: benchmark regression > ${BENCH_REGRESS_PCT}%:" >&2
                echo "$REGRESSIONS" >&2
                echo "" >&2
                echo "To reset: make bench-baseline" >&2
                FAILED=1
            fi
        else
            # Fallback: parse ns/op with awk and compare.
            REGRESSIONS=$(awk -v thr="$BENCH_REGRESS_PCT" '
                # Pass 1 (baseline): build name→ns map.
                NR==FNR && /^Benchmark/ {
                    name = $1
                    for (i=2; i<=NF; i++) {
                        if ($(i) == "ns/op") { base[name] = $(i-1)+0; break }
                    }
                    next
                }
                # Pass 2 (current): compare.
                /^Benchmark/ {
                    name = $1
                    for (i=2; i<=NF; i++) {
                        if ($(i) == "ns/op") {
                            curr = $(i-1)+0
                            if (name in base && base[name] > 0) {
                                pct = (curr - base[name]) / base[name] * 100
                                if (pct > thr+0) {
                                    printf "  %s: +%.1f%%  (%.1f → %.1f ns/op)\n",
                                        name, pct, base[name], curr
                                }
                            }
                            break
                        }
                    }
                }
            ' "$BENCH_BASELINE" "$BENCH_CURRENT")
            if [ -n "$REGRESSIONS" ]; then
                echo "" >&2
                echo "PUSH BLOCKED: benchmark regression > ${BENCH_REGRESS_PCT}% ns/op:" >&2
                echo "$REGRESSIONS" >&2
                echo "" >&2
                echo "To reset: make bench-baseline" >&2
                FAILED=1
            else
                echo "pre-push: benchmarks ok (no regression > ${BENCH_REGRESS_PCT}%)."
            fi
        fi
    else
        echo "pre-push: no benchmark baseline — recording now."
    fi

    # Update baseline only when no regression was found.
    if [ "$FAILED" -eq 0 ]; then
        cp "$BENCH_CURRENT" "$BENCH_BASELINE"
    fi
fi

# ── Coverage ─────────────────────────────────────────────────────────────────

echo "pre-push: running coverage..."
# Use nice -n 15 (not cpulimit) for coverage: cpulimit only limits the main
# go test process but not the per-package child binaries, which causes
# go test to write only partial coverage data to the profile.
nice -n 15 go test -timeout 300s -p "${P85}" -parallel "${P85}" \
    -coverprofile="$COV_PROFILE" -covermode=atomic \
    ./... >/dev/null 2>&1

CURRENT_COV=$(go tool cover -func="$COV_PROFILE" 2>/dev/null | \
    awk '/^total:/ { gsub(/%/, "", $NF); printf "%.1f", $NF }')

if [ -z "$CURRENT_COV" ]; then
    echo "WARNING: could not determine coverage — skipping coverage guard."
else
    echo "pre-push: coverage ${CURRENT_COV}%"
    if [ -f "$COV_BASELINE" ]; then
        BASELINE_COV=$(cat "$COV_BASELINE")
        DROPPED=$(awk -v curr="$CURRENT_COV" -v base="$BASELINE_COV" \
            -v thr="$COVER_DROP_PCT" \
            'BEGIN { print (base - curr > thr+0) ? "yes" : "no" }')
        if [ "$DROPPED" = "yes" ]; then
            echo "" >&2
            echo "PUSH BLOCKED: coverage dropped ${BASELINE_COV}% → ${CURRENT_COV}% (limit: -${COVER_DROP_PCT}%)." >&2
            echo "" >&2
            echo "To reset: echo ${CURRENT_COV} > coverage/coverage-baseline.txt" >&2
            FAILED=1
        else
            echo "pre-push: coverage ok (baseline: ${BASELINE_COV}%)."
        fi
    else
        echo "pre-push: no coverage baseline — recording now."
    fi

    if [ "$FAILED" -eq 0 ]; then
        echo "$CURRENT_COV" > "$COV_BASELINE"
    fi
fi

[ "$FAILED" -eq 0 ] && echo "pre-push: all guards passed."
exit "$FAILED"
