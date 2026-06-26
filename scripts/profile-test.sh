#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# profile-test.sh — Go test performance profiler
#
# Runs go test ONCE with CPU/memory/coverage profiles, parses per-test
# timing, and generates human-readable reports.
#
# Usage:
#   ./scripts/profile-test.sh <package> [test-flags...]
#
# Examples:
#   ./scripts/profile-test.sh ./cmd/mxcli/docker/
#   ./scripts/profile-test.sh ./mdl/executor/ -run TestMxCheck
#   ./scripts/profile-test.sh ./internal/bsoncompare/ -count=3
#
# Output: profile/ directory with CPU, mem, coverage profiles + timing report

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OUT_DIR="$PROJECT_DIR/profile"
PACKAGE="${1:?Usage: $0 <package> [flags...]}"
shift

mkdir -p "$OUT_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
SAFE_NAME=$(echo "$PACKAGE" | tr '/.' '_' | tr -d '@' | sed 's/^_*//')

CPU_PROFILE="$OUT_DIR/${SAFE_NAME}_cpu_${TIMESTAMP}.pprof"
MEM_PROFILE="$OUT_DIR/${SAFE_NAME}_mem_${TIMESTAMP}.pprof"
COVER_FILE="$OUT_DIR/${SAFE_NAME}_cover_${TIMESTAMP}.out"
RAW_OUT="$OUT_DIR/${SAFE_NAME}_raw_${TIMESTAMP}.txt"
SORTED_OUT="$OUT_DIR/${SAFE_NAME}_sorted_${TIMESTAMP}.txt"
REPORT_FILE="$OUT_DIR/${SAFE_NAME}_report_${TIMESTAMP}.md"

echo "=== Go Test Performance Profiler ==="
echo "Package:     $PACKAGE"
echo "Out dir:     $OUT_DIR"
echo "Timestamp:   $TIMESTAMP"
echo ""

# ── Single test run with all profiles ────────────────────────────────────
echo "── Running tests with CPU + mem + coverage profiles ──"
echo ""

START_TS=$(date +%s)

set +e
go test -tags integration -count=1 -timeout 30m -v \
  -cpuprofile "$CPU_PROFILE" \
  -memprofile "$MEM_PROFILE" \
  -coverprofile "$COVER_FILE" -covermode=atomic \
  "$PACKAGE" "$@" 2>&1 | tee "$RAW_OUT"
TEST_EXIT=$?
set -e

END_TS=$(date +%s)
ELAPSED=$((END_TS - START_TS))

# ── Parse per-test timing ────────────────────────────────────────────────
grep -E '^--- (PASS|FAIL):' "$RAW_OUT" \
  | sed -E 's/^--- (PASS|FAIL): //' \
  | sed -E 's/.*\(([0-9.]+)s\)$/\1 \0/' \
  | sort -rn \
  > "$SORTED_OUT" || true

TOTAL_TESTS=$(wc -l < "$SORTED_OUT" 2>/dev/null || echo 0)
SLOWEST=$(head -1 "$SORTED_OUT" | awk '{print $1}' 2>/dev/null || echo "N/A")

echo ""
echo "── Results ──"
echo "  Exit code:   $TEST_EXIT"
echo "  Wall time:   ${ELAPSED}s"
echo "  Tests found: $TOTAL_TESTS"
echo "  Slowest:     ${SLOWEST}s"
echo ""

# Top 5 slowest tests
if [ "$TOTAL_TESTS" -gt 0 ]; then
  echo "  Top 5 slowest:"
  head -5 "$SORTED_OUT" | while read -r line; do
    time_s=$(echo "$line" | awk '{print $1}')
    name=$(echo "$line" | cut -d' ' -f2- | sed 's/^.*(//; s/s)$//')
    original=$(echo "$line" | cut -d' ' -f2-)
    printf "    %6.2fs  %s\n" "$time_s" "$original"
  done
  echo ""
fi

# ── Coverage ─────────────────────────────────────────────────────────────
COVER_PCT="N/A"
if [ -f "$COVER_FILE" ] && [ -s "$COVER_FILE" ]; then
  COVER_PCT=$(go tool cover -func="$COVER_FILE" 2>/dev/null | tail -1 | grep -oE '[0-9.]+%' || echo "N/A")
  echo "  Coverage:    ${COVER_PCT}"
fi
echo "  Profiles:    $OUT_DIR/${SAFE_NAME}_*.pprof"
echo ""

# ── pprof summaries ──────────────────────────────────────────────────────
if [ -f "$CPU_PROFILE" ] && [ -s "$CPU_PROFILE" ]; then
  echo "── Top CPU consumers (pprof -top -nodecount=15) ──"
  go tool pprof -top -nodecount=15 "$CPU_PROFILE" 2>/dev/null | tail -n +2 | head -16
  echo ""
fi

if [ -f "$MEM_PROFILE" ] && [ -s "$MEM_PROFILE" ]; then
  echo "── Top memory consumers (pprof -top -nodecount=15) ──"
  go tool pprof -top -nodecount=15 "$MEM_PROFILE" 2>/dev/null | tail -n +2 | head -16
  echo ""
fi

# ── Write report ─────────────────────────────────────────────────────────
{
  echo "# Performance Profile: $PACKAGE"
  echo ""
  echo "| Metric | Value |"
  echo "|--------|-------|"
  echo "| Package | \`$PACKAGE\` |"
  echo "| Timestamp | $TIMESTAMP |"
  echo "| Wall time | ${ELAPSED}s |"
  echo "| Tests | $TOTAL_TESTS |"
  echo "| Slowest test | ${SLOWEST}s |"
  echo "| Coverage | ${COVER_PCT} |"
  echo "| Exit code | $TEST_EXIT |"
  echo ""
  echo "## Slowest Tests"
  echo ""
  echo '```'
  head -20 "$SORTED_OUT"
  echo '```'
  echo ""
  echo "## Profiles"
  echo ""
  echo "- CPU: \`$CPU_PROFILE\`"
  echo "- Memory: \`$MEM_PROFILE\`"
  echo "- Coverage: \`$COVER_FILE\`"
  echo ""
  echo "## Top CPU (pprof -top -nodecount=20)"
  echo ""
  echo '```'
  if [ -f "$CPU_PROFILE" ] && [ -s "$CPU_PROFILE" ]; then
    go tool pprof -top -nodecount=20 "$CPU_PROFILE" 2>/dev/null
  else
    echo "(no CPU profile)"
  fi
  echo '```'
  echo ""
  echo "## Top Memory (pprof -top -nodecount=20)"
  echo ""
  echo '```'
  if [ -f "$MEM_PROFILE" ] && [ -s "$MEM_PROFILE" ]; then
    go tool pprof -top -nodecount=20 "$MEM_PROFILE" 2>/dev/null
  else
    echo "(no mem profile)"
  fi
  echo '```'
} > "$REPORT_FILE"

echo "── Report written ──"
echo "  Report: $REPORT_FILE"
echo ""
echo "  Explore profiles with:"
echo "    go tool pprof -http=:8080 $CPU_PROFILE"
echo "    go tool pprof -http=:8080 $MEM_PROFILE"
echo ""
echo "=== Done (${ELAPSED}s) ==="
