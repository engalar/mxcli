#!/bin/sh
# pre-push: benchmark regression + coverage drop guard.
#
# Baselines (committed):
#   coverage/bench-baseline.txt    — raw Go benchmark output (-count=5)
#   coverage/coverage-baseline.txt — total coverage percentage
#
# Thresholds (override via env):
#   BENCH_REGRESS_PCT  — max allowed slowdown in median ns/op (default: 25)
#   COVER_DROP_PCT     — max allowed coverage drop in %        (default: 1)
#
# Benchmarks run WITHOUT cpulimit: cpulimit uses SIGSTOP/SIGCONT which
# directly skews time.Since() measurements by 30-40%, causing false
# positives. nice -n 15 lowers scheduling priority without affecting
# the clock, giving stable measurements.
#
# Comparison uses MEDIAN of count=5 runs (not last-overwrites-all) so
# a single slow sample does not trigger a false alarm.
#
# To reset baselines after intentional perf change:
#   make bench-baseline

REPO="$(git rev-parse --show-toplevel)"
NCPU=$(nproc)
P85=$(( NCPU * 85 / 100 ))
[ "$P85" -lt 1 ] && P85=1

export CGO_ENABLED=0
BENCH_REGRESS_PCT="${BENCH_REGRESS_PCT:-25}"
COVER_DROP_PCT="${COVER_DROP_PCT:-1}"

BENCH_BASELINE="${REPO}/coverage/bench-baseline.txt"
BENCH_CURRENT="${REPO}/coverage/bench-current.txt"
COV_BASELINE="${REPO}/coverage/coverage-baseline.txt"
COV_PROFILE="${REPO}/coverage/coverage.out"
mkdir -p "${REPO}/coverage"

FAILED=0

# ── Benchmarks ──────────────────────────────────────────────────────────────
# Run WITHOUT cpulimit — clock-based timing is distorted by SIGSTOP/SIGCONT.

echo "pre-push: running benchmarks (count=5, nice -n 15)..."
nice -n 15 go test -bench=. -benchmem -count=5 \
    -p "${P85}" ./... 2>/dev/null | \
    grep -v "^---" > "$BENCH_CURRENT"

if [ ! -s "$BENCH_CURRENT" ]; then
    echo "WARNING: no benchmark output — skipping bench guard."
else
    if [ -f "$BENCH_BASELINE" ]; then
        echo "pre-push: comparing benchmarks (median of 5) against baseline..."

        if command -v benchstat >/dev/null 2>&1; then
            BENCH_DIFF=$(benchstat "$BENCH_BASELINE" "$BENCH_CURRENT" 2>&1)
            echo "$BENCH_DIFF"
            REGRESSIONS=$(echo "$BENCH_DIFF" | awk -v thr="$BENCH_REGRESS_PCT" '
                /\+[0-9]+\.[0-9]+%/ {
                    match($0, /\+([0-9]+\.[0-9]+)%/, m)
                    if (m[1]+0 > thr+0) print $0
                }')
        else
            # Fallback: compute median per benchmark name in each file,
            # then compare medians. Avoids the last-overwrites-all bug.
            REGRESSIONS=$(awk -v thr="$BENCH_REGRESS_PCT" '
                function median(arr, n,    sorted, i, j, tmp) {
                    # insertion sort
                    for (i=2; i<=n; i++) {
                        tmp = arr[i]; j = i-1
                        while (j>=1 && arr[j]>tmp) { arr[j+1]=arr[j]; j-- }
                        arr[j+1] = tmp
                    }
                    return (n%2==1) ? arr[int(n/2)+1] \
                                    : (arr[n/2]+arr[n/2+1])/2
                }
                # Pass 1: collect all ns/op values per benchmark name (baseline).
                NR==FNR && /^Benchmark/ {
                    name = $1
                    for (i=2; i<=NF; i++) {
                        if ($(i) == "ns/op") {
                            n = ++bcnt[name]
                            bval[name, n] = $(i-1)+0
                            break
                        }
                    }
                    next
                }
                # Pass 2: collect ns/op values per benchmark name (current).
                /^Benchmark/ {
                    name = $1
                    for (i=2; i<=NF; i++) {
                        if ($(i) == "ns/op") {
                            n = ++ccnt[name]
                            cval[name, n] = $(i-1)+0
                            break
                        }
                    }
                }
                END {
                    for (name in bcnt) {
                        if (!(name in ccnt)) continue
                        # Build arrays for sorting.
                        for (i=1; i<=bcnt[name]; i++) ba[i] = bval[name, i]
                        for (i=1; i<=ccnt[name]; i++) ca[i] = cval[name, i]
                        bmed = median(ba, bcnt[name])
                        cmed = median(ca, ccnt[name])
                        if (bmed > 0) {
                            pct = (cmed - bmed) / bmed * 100
                            if (pct > thr+0) {
                                printf "  %s: +%.1f%%  (median %.0f → %.0f ns/op)\n",
                                    name, pct, bmed, cmed
                            }
                        }
                        # Reset arrays for next benchmark.
                        delete ba; delete ca
                    }
                }
            ' "$BENCH_BASELINE" "$BENCH_CURRENT")
        fi

        if [ -n "$REGRESSIONS" ]; then
            echo "" >&2
            echo "PUSH BLOCKED: benchmark regression > ${BENCH_REGRESS_PCT}%:" >&2
            echo "$REGRESSIONS" >&2
            echo "" >&2
            echo "To reset: make bench-baseline" >&2
            FAILED=1
        else
            echo "pre-push: benchmarks ok (no regression > ${BENCH_REGRESS_PCT}%)."
        fi
    else
        echo "pre-push: no benchmark baseline — recording now."
    fi

    if [ "$FAILED" -eq 0 ]; then
        cp "$BENCH_CURRENT" "$BENCH_BASELINE"
    fi
fi

# ── Coverage ─────────────────────────────────────────────────────────────────

echo "pre-push: running coverage..."
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
