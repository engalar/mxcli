# Academy Capstone E2E Validation — Design Spec

**Date:** 2026-06-16  
**Status:** Approved

## Problem

The `academy/zh/capstone-helpdesk/参考实现/` contains 8 MDL reference files that are manually maintained. There is no automated way to verify that they remain correct after changes to mxcli's executor, grammar, or backend. A broken reference file is only discovered when a learner follows the course.

## Goal

A local, manually-triggered Bash script that runs a full end-to-end validation of the Capstone reference implementation and hands off to a human for final UI verification.

## Scope

- **In scope:** Capstone reference files (`01-domain.mdl` … `99-seed-data.mdl`)
- **Out of scope:** Per-module reference files (01–12), CI/CD integration, structured report output

## Validation Depth

**Level C — Full validation:**

1. Execute all 8 MDL files against a fresh Mendix project
2. Run `mx check` (Studio Pro–level BSON validation) — zero new errors allowed
3. `go run ./cmd/mxcli-local build` — PAD package must build successfully
4. `go run ./cmd/mxcli-local run` — app starts; human validates UI, login flows, and demo data

## Execution Flow

```
① Fresh project
   rm -rf ./HelpDeskE2E
   run_mxcli new HelpDeskE2E --version 11.6.6
   MPR = ./HelpDeskE2E/HelpDeskE2E.mpr

② Batch exec (single command, all 8 MDL files)
   run_mxcli exec \
     academy/zh/capstone-helpdesk/参考实现/01-domain.mdl \
     academy/zh/capstone-helpdesk/参考实现/02-microflows.mdl \
     academy/zh/capstone-helpdesk/参考实现/03-nanoflows.mdl \
     academy/zh/capstone-helpdesk/参考实现/04-pages.mdl \
     academy/zh/capstone-helpdesk/参考实现/05-security.mdl \
     academy/zh/capstone-helpdesk/参考实现/06-kb.mdl \
     academy/zh/capstone-helpdesk/参考实现/07-escalation.mdl \
     academy/zh/capstone-helpdesk/参考实现/99-seed-data.mdl \
     -p ./HelpDeskE2E/HelpDeskE2E.mpr

③ mx check (via scripts/lib/mx-check.sh)
   baseline = 0  (fresh project, zero errors expected)
   MX_BIN  = resolved via go run ./scripts/mx-path/main.go 11.6.6

④ PAD build
   run_mxcli_local build -p ./HelpDeskE2E/HelpDeskE2E.mpr

⑤ Human handoff
   Print demo account credentials, then:
   run_mxcli_local run -p ./HelpDeskE2E/HelpDeskE2E.mpr --admin-password Admin1234
   (blocks in foreground; human presses Ctrl+C when done)
```

## Files Produced

| Path | Purpose |
|------|---------|
| `scripts/validate-academy-capstone.sh` | Main validation script |
| `scripts/mx-path/main.go` | Go helper: resolves mx binary path via `docker.CachedMxPath()` |
| `.gitignore` entry `/HelpDeskE2E/` | Exclude generated project from version control |
| `Makefile` target `validate-academy-capstone` | Convenience entry point |

## Script Details

### Shebang & options

```bash
#!/usr/bin/env bash
set -euo pipefail
```

Consistent with `scripts/run-mdl-tests.sh` and `scripts/generate-sbom.sh`.

### mxcli invocation — two `go run` helpers

`cmd/mxcli` (launcher/daemon) handles `new`, `exec`, `setup`. `cmd/mxcli-local` is an independent binary for `build` and `run` — no launcher routing needed. Both support an env-var override for cases where a compiled binary is available:

```bash
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
```

### mx binary discovery — `scripts/mx-path/main.go`

A minimal Go program that calls `docker.CachedMxPath(version)` directly — the same internal function used by `mxcli setup show`. No string parsing:

```go
package main

import (
    "fmt"
    "os"
    "github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

func main() {
    if len(os.Args) != 2 {
        fmt.Fprintln(os.Stderr, "usage: go run ./scripts/mx-path/main.go <version>")
        os.Exit(2)
    }
    version := os.Args[1]
    path := docker.CachedMxPath(version)
    if path == "" {
        fmt.Fprintf(os.Stderr, "ERROR: mx %s not cached\n  Run: mxcli setup mxbuild --version %s\n",
            version, version)
        os.Exit(1)
    }
    fmt.Print(path)
}
```

Called from the script as:

```bash
MX_BIN=$(cd "$REPO_ROOT" && go run ./scripts/mx-path/main.go "$MX_VERSION")
```

### mx check integration

Sources `scripts/lib/mx-check.sh` and calls `mx_check_against_baseline` with a temp file containing `0`:

```bash
. "$SCRIPT_DIR/lib/mx-check.sh"
BASELINE=$(mktemp)
echo 0 > "$BASELINE"
trap 'rm -f "$BASELINE"' EXIT
mx_check_against_baseline "$MPR" "$BASELINE" "$MX_BIN"
```

Exit codes from the library: 0 = PASS, 1 = FAIL (new errors), 2 = CRASH.

### Error handling

- `set -euo pipefail` stops on any command failure
- On failure, `HelpDeskE2E/` is **preserved** (not cleaned up) for debugging
- A `trap … EXIT` prints the project path on exit so the developer knows where to look

### Output style

Plain text, no ANSI colors. Consistent with existing scripts (`test-section-check.sh`):

```
=== validate-academy-capstone ===
  creating project HelpDeskE2E (11.6.6)...
  exec 8 MDL files...
  mx check...
PASS: mx check (0 errors, baseline 0).
  building...
  build complete.

=== Human validation ===
  URL:      http://localhost:8080
  Customer: demo_customer@helpdesk.test / Demo12345678
  Agent:    demo_agent@helpdesk.test    / Demo12345678
  Manager:  demo_manager@helpdesk.test  / Demo12345678

Starting runtime — Ctrl+C to stop.
```

### Makefile target

```makefile
## validate-academy-capstone: run full e2e validation of academy capstone reference implementation
validate-academy-capstone:
	@./scripts/validate-academy-capstone.sh
```

No `build` dependency — `go run` recompiles on demand.

## Prerequisites

| Requirement | How to satisfy |
|-------------|----------------|
| Go toolchain in PATH | All invocations use `go run` — no compiled binary needed |
| mx 11.6.6 cached | `go run ./cmd/mxcli setup mxbuild --version 11.6.6` (one-time download) |
| Java 21 in PATH | Required by `go run ./cmd/mxcli-local run` JVM runtime |

## Non-Goals

- No per-step mx check (one check at the end is sufficient; intermediate states may have expected incomplete references)
- No CI/CD wiring (local only)
- No structured report file (terminal output is sufficient)
- No module-level validation (01–12 modules validated indirectly via Capstone)
