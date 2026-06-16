#!/usr/bin/env bash
# scripts/validate-academy-capstone.sh
# End-to-end validation of academy/zh/capstone-helpdesk reference implementation.
#
# Flow:
#   1. Create fresh HelpDeskE2E project via mxcli new
#   2. Batch-exec all 8 capstone MDL files
#   3. Run mx check (Studio Pro-level BSON validation, baseline=0)
#   4. Build PAD package via mxcli-local build
#   5. Start runtime for human validation (blocks until Ctrl+C)
#
# Overrides:
#   MXCLI=path/to/mxcli         use compiled binary instead of go run ./cmd/mxcli
#   MXCLI_LOCAL=path/to/local   use compiled binary instead of go run ./cmd/mxcli-local
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
. "$SCRIPT_DIR/lib/mx-check.sh"

MX_VERSION="11.6.6"
PROJECT_NAME="HelpDeskE2E"
MPR="$REPO_ROOT/$PROJECT_NAME/$PROJECT_NAME.mpr"
CAPSTONE_DIR="$REPO_ROOT/academy/zh/capstone-helpdesk/参考实现"
BASELINE=""
SKIP_CREATE=0

# Parse flags
for arg in "$@"; do
    case "$arg" in
        --skip-create) SKIP_CREATE=1 ;;
        *) echo "unknown flag: $arg" >&2; exit 2 ;;
    esac
done

# ── helpers ────────────────────────────────────────────────────────────────

# daemon-routed commands: new, exec, setup mxbuild
run_mxcli() {
    if [ -n "${MXCLI:-}" ]; then
        "$MXCLI" "$@"
    else
        (cd "$REPO_ROOT" && go run ./cmd/mxcli "$@")
    fi
}

# local runtime commands: build, run — bypasses launcher entirely
run_mxcli_local() {
    if [ -n "${MXCLI_LOCAL:-}" ]; then
        "$MXCLI_LOCAL" "$@"
    else
        (cd "$REPO_ROOT" && go run ./cmd/mxcli-local "$@")
    fi
}

cleanup() {
    [ -n "$BASELINE" ] && rm -f "$BASELINE"
    echo
    echo "Project preserved at: $REPO_ROOT/$PROJECT_NAME"
}
trap cleanup EXIT

# ── step 1: fresh project ──────────────────────────────────────────────────

echo "=== validate-academy-capstone ==="
if [ "$SKIP_CREATE" -eq 1 ]; then
    echo "  skipping project creation (--skip-create)"
    if [ ! -f "$MPR" ]; then
        echo "ERROR: --skip-create set but MPR not found: $MPR" >&2
        exit 1
    fi
else
    echo "  creating project $PROJECT_NAME ($MX_VERSION)..."
    rm -rf "$REPO_ROOT/$PROJECT_NAME"
    # mxcli new step 4 (Linux devcontainer binary) may fail on Windows — tolerate it
    # as long as the MPR was successfully created in step 2.
    (cd "$REPO_ROOT" && run_mxcli new "$PROJECT_NAME" --version "$MX_VERSION") || true
    if [ ! -f "$MPR" ]; then
        echo "ERROR: mxcli new failed — MPR not found: $MPR" >&2
        exit 1
    fi
fi

# ── step 2: batch exec ────────────────────────────────────────────────────

echo "  exec 8 MDL files..."
run_mxcli exec \
    "$CAPSTONE_DIR/01-domain.mdl" \
    "$CAPSTONE_DIR/02-microflows.mdl" \
    "$CAPSTONE_DIR/03-nanoflows.mdl" \
    "$CAPSTONE_DIR/04-pages.mdl" \
    "$CAPSTONE_DIR/05-security.mdl" \
    "$CAPSTONE_DIR/06-kb.mdl" \
    "$CAPSTONE_DIR/07-escalation.mdl" \
    "$CAPSTONE_DIR/99-seed-data.mdl" \
    -p "$MPR"

# ── step 3: mx check ──────────────────────────────────────────────────────

echo "  mx check..."
MX_BIN=$(cd "$REPO_ROOT" && go run ./scripts/mx-path/main.go "$MX_VERSION")
BASELINE=$(mktemp)
echo 0 > "$BASELINE"
mx_check_against_baseline "$MPR" "$BASELINE" "$MX_BIN"

# ── step 4: PAD build ─────────────────────────────────────────────────────

echo "  building..."
run_mxcli_local build -p "$MPR"
echo "  build complete."

# ── step 5: human handoff ─────────────────────────────────────────────────

echo
echo "=== Human validation ==="
echo "  URL:      http://localhost:8080"
echo "  Customer: demo_customer@helpdesk.test / Demo12345678"
echo "  Agent:    demo_agent@helpdesk.test    / Demo12345678"
echo "  Manager:  demo_manager@helpdesk.test  / Demo12345678"
echo
echo "Starting runtime — Ctrl+C to stop."
run_mxcli_local run -p "$MPR" --admin-password Admin1234
