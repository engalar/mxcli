# Launcher/Daemon/Local Merge Design

## Goals

1. **Single binary** — `mxcli` is the only user-facing binary. No launcher/daemon split.
2. **Remove launcher** — `cmd/mxcli-launcher/` deleted. Its lifecycle management, socket forwarding, and TTY routing become unnecessary.
3. **Merge local** — `cmd/mxcli-local/` commands (`build`, `run`, `reload`) move into the main binary as subcommands.
4. **Simplify release** — Three CI workflows + three tags → one workflow + one tag.
5. **Preserve `serve`/LSP** — daemon's server mode remains, but now the same binary can enter server mode or CLI mode based on subcommand.

## Non-Goals

- Per-MPR process isolation (dropped — single process, shared state)
- Socket protocol (dropped — direct execution)
- Background version check (dropped — `mxcli upgrade` is explicit)

## Architecture

```
Before:
  mxcli (launcher)          → socket → mxcli-daemon (per-MPR or shared)
  mxcli-local               → subprocess (build/run)

After:
  mxcli (merged)
    ├─ describe/show/exec/lint/…  → direct execution
    ├─ build/run/reload            → merged from local
    ├─ upgrade                     → self-upgrade (from launcher)
    ├─ rollback                    → from launcher
    ├─ serve                       → server mode (LSP, socket listener)
    ├─ daemon                      → removed
    └─ local                       → removed
```

### Modes

| Subcommand | Mode | Behavior |
|---|---|---|
| `serve` | Server | Listens on Unix socket, handles JSON-framed requests |
| `upgrade`/`rollback` | Self-upgrade | Spawns upgrade helper, replaces binary |
| `build`/`run`/`reload` | Direct | From local; `build` may exec mxbuild |
| Everything else | Direct | describe/exec/lint/check/diff/etc. |

### Server Mode (`serve`)

The only case that needs a long-running process. `mxcli serve [--socket path]`:
1. Listens on Unix socket (or stdio for LSP)
2. Accepts JSON-framed requests (same protocol as current launcher→daemon)
3. Forwards each request to the Cobra command tree
4. Returns JSON response with stdout/stderr/exitcode

This is what IDEs and `mxcli -c "..."` call into. But `mxcli -c "..."` without `serve` runs directly — no socket hop.

## File Changes

### Deleted (from launcher)

| File | Lines | Reason |
|---|---|---|
| `cmd/mxcli-launcher/` | 3387 total | Entire package deleted |

### Deleted (from local)

| File | Lines | Reason |
|---|---|---|
| `cmd/mxcli-local/` | ~500 total | Commands merged into main |

### New files in `cmd/mxcli/`

| File | Contents | Source |
|---|---|---|
| `cmd_upgrade.go` | `upgradeCmd`, `rollbackCmd`, self-upgrade logic | Ported from launcher `self_update.go` + `upgrade.go` |
| `cmd_build.go` | `buildCmd` | Ported from local `cmd_build.go` |
| `cmd_run.go` | `runCmd` | Ported from local `cmd_run.go` |
| `cmd_reload.go` | `reloadCmd` | Ported from local `cmd_reload.go` |
| `cmd_upgrade_test.go` | Tests for upgrade/rollback | Ported |

### Modified in `cmd/mxcli/`

| File | Change |
|---|---|
| `main.go` | Add `CommitSHA` (already done), add upgrade/build/run/reload subcommands to root |
| `cmd_serve.go` | Already exists — the serve/LSP handler. No change needed. |

## Self-Upgrade Design

Self-upgrade in merged binary follows the same pattern as the current launcher:

```
mxcli upgrade
  ↓
1. Download new binary to ~/.mxcli/mxcli.new
2. Rename current binary → mxcli.old
3. Rename new binary → mxcli (same path)
4. Spawn: mxcli --internal-update <pid> <new> <target>
5. Child waits for parent to exit, then renames .old → .new fallback
6. Child exits
```

The `--internal-update` protocol is already implemented in the launcher. Port to the merged binary.

## Release CI Changes

### Current

| Workflow | Triggers | Produces |
|---|---|---|
| `release.yml` | `v*` tag | launcher binaries (6 platforms) |
| `release-daemon.yml` | `daemon-v*` tag | daemon binary |
| `release-local.yml` | `local-v*` tag | local binary |

### After

| Workflow | Triggers | Produces |
|---|---|---|
| `release.yml` | `v*` tag | single `mxcli` binary (6 platforms) |

### Tag Changes

| Current | After |
|---|---|
| `v0.28.0` (launcher) | `v0.28.0` (merged binary) |
| `daemon-v0.28.0` | deleted |
| `local-v0.9.0` | deleted |

### install.sh

Current: downloads launcher from GitHub Release, launcher downloads daemon + local.

After: downloads single `mxcli` binary from GitHub Release. No secondary downloads.

## Migration Plan

### Phase 1: Move local commands (safe, no behavior change)

1. Copy `cmd_build.go`, `cmd_run.go`, `cmd_reload.go` from `cmd/mxcli-local/` to `cmd/mxcli/`
2. Rename package `main` → `main` (same package name)
3. Register subcommands on root Cobra command
4. Delete `cmd/mxcli-local/`
5. Update `Makefile` (remove `LOCAL_LDFLAGS`, `LOCAL_PATH`, `release-local` target)

### Phase 2: Move upgrade logic

1. Copy `self_update.go`, `upgrade.go`, `lock_*.go` from launcher to daemon
2. Port to use daemon's path constants (no more `e.daemonBinaryPath()` — just `os.Executable()`)
3. `update.go` (background version check) is NOT ported — upgrade is user-initiated only
4. Register `upgradeCmd` and `rollbackCmd` as subcommands
5. Update `install.sh` to download single binary

### Phase 3: Delete launcher

1. Verify all launcher commands are covered:
   - `upgrade` → merged Phase 2
   - `rollback` → merged Phase 2
   - `version`/`--version` → already in daemon
   - `daemon` → deleted (no separate daemon process)
   - `local` → merged Phase 1 (commands direct)
   - TTY routing → deleted (direct execution)
   - Socket forwarding → deleted (direct execution)
2. Delete `cmd/mxcli-launcher/`
3. Update `Makefile` (remove `LAUNCHER_LDFLAGS`, `release-launcher` target)
4. Update `.github/workflows/release.yml` to build merged binary
5. Delete `.github/workflows/release-daemon.yml`
6. Delete `.github/workflows/release-local.yml`

### Phase 4: Cleanup

1. Delete `internal/launcherproto/` (socket protocol — no longer needed)
2. Delete `forward.go` from daemon (if it exists)
3. Update docs/README
4. Test install.sh end-to-end

## Files That Disappear Entirely

```
cmd/mxcli-launcher/     (3,387 lines)
cmd/mxcli-local/        (~500 lines)
internal/launcherproto/ (~200 lines)
.github/workflows/release-daemon.yml
.github/workflows/release-local.yml
```

**Net LOC reduction: ~4,000 lines deleted, ~800 added (upgrade logic port) = ~3,200 net removal.**
