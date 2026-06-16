#!/usr/bin/env bash
# scripts/validate-academy-capstone.sh
# End-to-end validation of academy/zh/capstone-helpdesk reference implementation.
#
# Steps:
#   new    — rm -rf HelpDeskE2E + mxcli new (fresh project)
#   exec   — mdlrun 8 MDL files
#   check  — mx check (Studio Pro-level BSON validation, baseline=0)
#   build  — mxcli-local build (PAD package)
#   run    — mxcli-local run (blocks until Ctrl+C, human validation)
#
# Usage:
#   ./validate-academy-capstone.sh                 # full run (all 5 steps)
#   ./validate-academy-capstone.sh --from exec     # skip new,  run exec→check→build→run
#   ./validate-academy-capstone.sh --from check    # skip new+exec, run check→build→run
#   ./validate-academy-capstone.sh --from build    # skip new+exec+check, run build→run
#   ./validate-academy-capstone.sh --from run      # just start the runtime
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
FROM_STEP="new"

# Parse flags
while [ $# -gt 0 ]; do
    case "$1" in
        --from)
            shift
            FROM_STEP="${1:-}"
            ;;
        --from=*)
            FROM_STEP="${1#--from=}"
            ;;
        --skip-create)
            FROM_STEP="exec"  # backwards compat
            ;;
        *)
            echo "unknown flag: $1" >&2
            echo "Usage: $0 [--from new|exec|check|build|run]" >&2
            exit 2
            ;;
    esac
    shift
done

case "$FROM_STEP" in
    new|exec|check|build|run) ;;
    *) echo "ERROR: --from must be one of: new exec check build run" >&2; exit 2 ;;
esac

# ── helpers ────────────────────────────────────────────────────────────────

step_enabled() {
    local order="new exec check build run"
    local pos_from pos_step i=0
    for s in $order; do
        [ "$s" = "$FROM_STEP" ] && pos_from=$i
        [ "$s" = "$1" ]        && pos_step=$i
        i=$((i+1))
    done
    [ "${pos_step:-99}" -ge "${pos_from:-0}" ]
}

# daemon-routed commands: new, exec, setup mxbuild
run_mxcli() {
    if [ -n "${MXCLI:-}" ]; then
        "$MXCLI" "$@"
    else
        (cd "$REPO_ROOT" && go run ./cmd/mxcli "$@")
    fi
}

# multi-file MDL execution (cmd/mdlrun supports multiple positional file args)
run_mdlrun() {
    (cd "$REPO_ROOT" && go run ./cmd/mdlrun "$@")
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

echo "=== validate-academy-capstone (from: $FROM_STEP) ==="

# ── step 1: new ────────────────────────────────────────────────────────────

if step_enabled new; then
    echo "  creating project $PROJECT_NAME ($MX_VERSION)..."
    # On Windows, Studio Pro or other tools may hold file handles on a previous
    # project directory. Detect this early and give a helpful message.
    if [ -d "$REPO_ROOT/$PROJECT_NAME" ]; then
        if ! rm -rf "$REPO_ROOT/$PROJECT_NAME" 2>/dev/null; then
            echo "ERROR: cannot delete $REPO_ROOT/$PROJECT_NAME — a process has it locked." >&2
            echo "  Close Studio Pro (or any tool) that has HelpDeskE2E open, then retry." >&2
            echo "  Or skip creation: $0 --from exec" >&2
            exit 1
        fi
    fi
    # mxcli new step 4 (Linux devcontainer binary) may fail on Windows — tolerate it
    # as long as the MPR was successfully created in step 2.
    (cd "$REPO_ROOT" && run_mxcli new "$PROJECT_NAME" --version "$MX_VERSION") || true
    if [ ! -f "$MPR" ]; then
        echo "ERROR: mxcli new failed — MPR not found: $MPR" >&2
        exit 1
    fi
else
    echo "  skipping new (--from $FROM_STEP)"
    if [ ! -f "$MPR" ]; then
        echo "ERROR: MPR not found: $MPR — run without --from to create it first." >&2
        exit 1
    fi
fi

# ── step 2: exec ───────────────────────────────────────────────────────────

if step_enabled exec; then
    echo "  exec 8 MDL files..."
    run_mdlrun -p "$MPR" \
        "$CAPSTONE_DIR/01-domain.mdl" \
        "$CAPSTONE_DIR/02-microflows.mdl" \
        "$CAPSTONE_DIR/03-nanoflows.mdl" \
        "$CAPSTONE_DIR/04-pages.mdl" \
        "$CAPSTONE_DIR/05-security.mdl" \
        "$CAPSTONE_DIR/06-kb.mdl" \
        "$CAPSTONE_DIR/07-escalation.mdl" \
        "$CAPSTONE_DIR/99-seed-data.mdl"
else
    echo "  skipping exec"
fi

# ── step 3: check ──────────────────────────────────────────────────────────

if step_enabled check; then
    echo "  mx check..."
    MX_BIN=$(cd "$REPO_ROOT" && go run ./scripts/mx-path/main.go "$MX_VERSION") || {
        echo "  FAIL: could not resolve mx $MX_VERSION binary" >&2
        echo "  Debug: cd $REPO_ROOT && go run ./scripts/mx-path/main.go $MX_VERSION" >&2
        exit 1
    }
    echo "  mx binary: $MX_BIN"
    BASELINE=$(mktemp)
    echo 0 > "$BASELINE"
    mx_check_against_baseline "$MPR" "$BASELINE" "$MX_BIN" || {
        rc=$?
        echo "  FAIL: mx_check_against_baseline returned $rc" >&2
        exit $rc
    }
else
    echo "  skipping mx check"
fi

# ── step 4: build ──────────────────────────────────────────────────────────

if step_enabled build; then
    echo "  building..."
    run_mxcli_local build -p "$MPR"
    echo "  build complete."
else
    echo "  skipping build"
fi

# ── step 5: run ────────────────────────────────────────────────────────────

if step_enabled run; then
    echo
    echo "=== Human validation ==="
    echo "  URL:      http://localhost:8080"
    echo "  Customer: demo_customer@helpdesk.test / Demo12345678"
    echo "  Agent:    demo_agent@helpdesk.test    / Demo12345678"
    echo "  Manager:  demo_manager@helpdesk.test  / Demo12345678"
    echo
    echo "Starting runtime — Ctrl+C to stop."
    run_mxcli_local run -p "$MPR" --admin-password Admin1234
fi
