#!/usr/bin/env bash
# scripts/validate-academy-capstone.sh
# End-to-end validation of academy/zh/capstone-helpdesk reference implementation.
#
# Steps:
#   new    — rm -rf HelpDeskE2E + mxcli new (fresh project)
#   exec   — pre-build widget + mdlrun MDL files (01-10 + 99)
#   check  — mx check (Studio Pro-level BSON validation, baseline=0)
#   ext    — copy theme SCSS; confirm widget installed
#   build  — mxcli-local build (PAD package)
#   run    — mxcli-local run (blocks until Ctrl+C, human validation)
#
# Usage:
#   ./validate-academy-capstone.sh                # full run (all 6 steps)
#   ./validate-academy-capstone.sh --from exec    # skip new, run exec→check→build→run
#   ./validate-academy-capstone.sh --from check   # skip new+exec, run check→build→run
#   ./validate-academy-capstone.sh --from build   # skip new+exec+check, run build→run
#   ./validate-academy-capstone.sh --from run     # just start the runtime
#
# Modules:
#   01-07  Core modules (domain, microflows, nanoflows, pages, security, KB, escalation)
#   08     Java Action — password hashing
#   09     JS Action   — clipboard, notifications, relative time
#   10     Widget      — TicketStatusBadge (built from source automatically)
#   11     Theme       — brand SCSS appended to main.scss
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
EXT_THEME_SRC="$REPO_ROOT/academy/zh/11-扩展-主题定制/theme/helpdesk-theme.scss"
WIDGET_SOURCE_DIR="$REPO_ROOT/academy/zh/10-扩展-Widget开发/widget-source"
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

# stop_runtime kills any process holding port 8090 (Mendix admin API).
# M2EE /stop requires authentication we don't have here, so we find the
# owning PID via netstat and force-kill it instead.
stop_runtime() {
    # grep 无匹配返回 1，用 || true 防止 pipefail 退出
    _port_pid() {
        netstat -ano 2>/dev/null | grep ":8090 " | grep "LISTENING" | awk '{print $NF}' | head -1 || true
    }
    local pid
    pid=$(_port_pid)
    if [ -z "$pid" ]; then
        pid=$(powershell -NoProfile -Command \
            "(Get-NetTCPConnection -LocalPort 8090 -ErrorAction SilentlyContinue).OwningProcess" \
            2>/dev/null | tr -d '[:space:]') || true
    fi
    if [ -n "$pid" ] && [ "$pid" -gt 0 ] 2>/dev/null; then
        echo "  stopping runtime (PID $pid on :8090)..."
        powershell -NoProfile -Command "Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue" 2>/dev/null || \
            taskkill //F //PID "$pid" >/dev/null 2>&1 || true
        local i=0
        while [ $i -lt 10 ]; do
            sleep 0.5
            pid=$(_port_pid)
            [ -z "$pid" ] && return 0
            i=$((i+1))
        done
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
    # Stop any running Mendix runtime on the default admin port before deletion.
    # On Windows, file handles held by the Java process prevent rm -rf.
    if [ -d "$REPO_ROOT/$PROJECT_NAME" ]; then
        if ! rm -rf "$REPO_ROOT/$PROJECT_NAME" 2>/dev/null; then
            stop_runtime
            if ! rm -rf "$REPO_ROOT/$PROJECT_NAME" 2>/dev/null; then
                echo "ERROR: cannot delete $REPO_ROOT/$PROJECT_NAME — a process has it locked." >&2
                echo "  Close Studio Pro or any app with HelpDeskE2E open, then retry." >&2
                echo "  Or skip creation: $0 --from exec" >&2
                exit 1
            fi
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
    # Build+install widget before exec (needed by 10-widget.mdl)
    if [ -d "$WIDGET_SOURCE_DIR" ]; then
        echo "  pre-exec: building + installing widget from $WIDGET_SOURCE_DIR..."
        run_mxcli widget build --dir "$WIDGET_SOURCE_DIR" --install -p "$MPR" || \
            echo "  WARNING: widget build failed — will skip 10-widget.mdl" >&2
    fi

    mdl_files=(
        "$CAPSTONE_DIR/01-domain.mdl"
        "$CAPSTONE_DIR/02-microflows.mdl"
        "$CAPSTONE_DIR/03-nanoflows.mdl"
        "$CAPSTONE_DIR/04-pages.mdl"
        "$CAPSTONE_DIR/05-security.mdl"
        "$CAPSTONE_DIR/06-kb.mdl"
        "$CAPSTONE_DIR/07-escalation.mdl"
        "$CAPSTONE_DIR/08-java-actions.mdl"
        "$CAPSTONE_DIR/09-js-actions.mdl"
        "$CAPSTONE_DIR/99-seed-data.mdl"
    )

    # 10-widget.mdl only included when widget MPK was successfully built
    if [ -f "$REPO_ROOT/$PROJECT_NAME/widgets/TicketStatusBadge.mpk" ]; then
        mdl_files+=("$CAPSTONE_DIR/10-widget.mdl")
    fi

    echo "  exec ${#mdl_files[@]} MDL files..."
    run_mdlrun -p "$MPR" "${mdl_files[@]}"
else
    echo "  skipping exec"
fi

# ── step 3: check ──────────────────────────────────────────────────────────

if step_enabled check; then
    echo "  mx check..."
    MX_BIN=$(cd "$REPO_ROOT" && timeout 30 go run ./scripts/mx-path/main.go "$MX_VERSION" 2>/dev/null || true)
    if [ -z "$MX_BIN" ] || [ ! -x "$MX_BIN" ]; then
        echo "  SKIP: mx binary not available (install mxbuild or set MXCLI_MX_BUILD_PATH)" >&2
    else
        BASELINE=$(mktemp)
        echo 0 > "$BASELINE"
        mx_check_against_baseline "$MPR" "$BASELINE" "$MX_BIN" || {
            echo "  WARNING: mx check found errors (see above). Continuing..." >&2
        }
    fi
else
    echo "  skipping mx check"
fi

# ── step 4: extensions (theme + optional widget) ──────────────────────────

if step_enabled check; then
    # Always append theme CSS to main.scss (module 11)
    THEME_DEST="$REPO_ROOT/$PROJECT_NAME/theme/web/main.scss"
    if [ -f "$EXT_THEME_SRC" ] && [ -f "$THEME_DEST" ]; then
        echo "" >> "$THEME_DEST"
        echo "// -- helpdesk-theme (module 11) --" >> "$THEME_DEST"
        cat "$EXT_THEME_SRC" >> "$THEME_DEST"
        echo "  theme: helpdesk-theme.scss appended to $THEME_DEST"
    fi

    if [ -f "$REPO_ROOT/$PROJECT_NAME/widgets/TicketStatusBadge.mpk" ]; then
        echo "  widget: TicketStatusBadge.mpk installed"
    fi
fi

# ── step 5: build ──────────────────────────────────────────────────────────

if step_enabled build; then
    echo "  building..."
    run_mxcli_local build -p "$MPR" --skip-check
    echo "  build complete."
else
    echo "  skipping build"
fi

# ── step 6: run ────────────────────────────────────────────────────────────

if step_enabled run; then
    stop_runtime
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
