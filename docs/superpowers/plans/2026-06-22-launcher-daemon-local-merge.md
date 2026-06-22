# Launcher/Daemon/Local Merge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Merge three binaries (launcher, daemon, local) into one `mxcli` binary, delete launcher/local packages, simplify CI to one workflow.

**Architecture:** Move local's `build`/`run`/`reload` commands and launcher's `upgrade`/`rollback` into `cmd/mxcli/`. Delete `cmd/mxcli-launcher/`, `cmd/mxcli-local/`, `internal/launcherproto/`. CI builds single binary from `v*` tag.

**Tech Stack:** Go, Cobra CLI, GitHub Actions

## Global Constraints

- All existing daemon subcommands must remain unchanged (describe/show/exec/lint/check/diff/etc.)
- Self-upgrade `--internal-update` protocol must be preserved
- `install.sh` must download a single binary, not trigger secondary downloads
- Tag `daemon-v*` and `local-v*` become unused; only `v*` is needed
- Per-MPR process isolation is dropped; serve/LSP mode preserved
- Background version check is dropped (explicit `upgrade` only)

---

### Task 1: Move local commands into daemon

**Files:**
- Copy: `cmd/mxcli-local/cmd_build.go` → `cmd/mxcli/cmd_build.go`
- Copy: `cmd/mxcli-local/cmd_run.go` → `cmd/mxcli/cmd_run.go`
- Copy: `cmd/mxcli-local/cmd_reload.go` → `cmd/mxcli/cmd_reload.go`
- Modify: `cmd/mxcli/main.go` — register build/run/reload subcommands

**Interfaces:**
- Consumes: `docker` package from `cmd/mxcli/docker/` (already imported by daemon)
- Produces: `buildCmd`, `runCmd`, `reloadCmd` functions returning `*cobra.Command`

- [ ] **Step 1:** Copy the three local command files to `cmd/mxcli/`

```bash
cp cmd/mxcli-local/cmd_build.go cmd/mxcli/cmd_build.go
cp cmd/mxcli-local/cmd_run.go cmd/mxcli/cmd_run.go
cp cmd/mxcli-local/cmd_reload.go cmd/mxcli/cmd_reload.go
```

- [ ] **Step 2:** Change package from `main` to `main` in all three files (it's already `main` — no change needed). Update import paths to reference `docker` package as `github.com/mendixlabs/mxcli/cmd/mxcli/docker` (already correct).

- [ ] **Step 3:** Register commands in `cmd/mxcli/main.go`:

Find the block where `execCmd`, `showCmd`, etc. are registered (around line 378). Add:

```go
rootCmd.AddCommand(buildCmd())
rootCmd.AddCommand(runCmd())
rootCmd.AddCommand(reloadCmd())
```

- [ ] **Step 4:** Build to verify:

```bash
go build ./cmd/mxcli/
```

- [ ] **Step 5:** Test local commands work:

```bash
./mxcli build --help
./mxcli run --help
./mxcli reload --help
```

- [ ] **Step 6:** Commit:

```bash
git add cmd/mxcli/cmd_build.go cmd/mxcli/cmd_run.go cmd/mxcli/cmd_reload.go cmd/mxcli/main.go
git commit -m "feat: merge local build/run/reload commands into main binary"
```

### Task 2: Move upgrade/rollback into daemon

**Files:**
- Copy: `cmd/mxcli-launcher/self_update.go` → `cmd/mxcli/cmd_upgrade.go` (port)
- Copy: `cmd/mxcli-launcher/self_update_unix.go` → `cmd/mxcli/self_update_unix.go`
- Copy: `cmd/mxcli-launcher/self_update_windows.go` → `cmd/mxcli/self_update_windows.go`
- Copy: `cmd/mxcli-launcher/upgrade.go` → `cmd/mxcli/upgrade.go`
- Copy: `cmd/mxcli-launcher/lock_unix.go` → `cmd/mxcli/lock_unix.go`
- Copy: `cmd/mxcli-launcher/lock_windows.go` → `cmd/mxcli/lock_windows.go`
- Copy: `cmd/mxcli-launcher/paths.go` → (selectively, only localDir/localBinaryPath)
- Create: `cmd/mxcli/cmd_upgrade.go` — `upgradeCmd`, `rollbackCmd` Cobra commands
- Modify: `cmd/mxcli/main.go` — register upgrade/rollback

**Interfaces:**
- Consumes: `PIDWaiter` interface from launcher
- Produces: `upgradeCmd`, `rollbackCmd` as `*cobra.Command`

- [ ] **Step 1:** Copy platform-specific files:

```bash
cp cmd/mxcli-launcher/self_update_unix.go cmd/mxcli/self_update_unix.go
cp cmd/mxcli-launcher/self_update_windows.go cmd/mxcli/self_update_windows.go
cp cmd/mxcli-launcher/upgrade.go cmd/mxcli/upgrade.go
cp cmd/mxcli-launcher/lock_unix.go cmd/mxcli/lock_unix.go
cp cmd/mxcli-launcher/lock_windows.go cmd/mxcli/lock_windows.go
```

- [ ] **Step 2:** Create `cmd/mxcli/cmd_upgrade.go` with Cobra commands + all supporting functions from launcher (port the following from `cmd/mxcli-launcher/self_update.go` and `cmd/mxcli-launcher/upgrade.go`):

Functions to port (exact signatures from launcher):
- `PIDWaiter` interface — `WaitForExit(pid int, timeout time.Duration) error`
- `RealPIDWaiter` struct — implements `PIDWaiter` via `os.FindProcess` + `os.Process.Wait`
- `runInternalUpdate(pid int, newBinPath, targetPath string, waiter PIDWaiter, timeout time.Duration) error` — waits for parent to exit, atomically replaces binary
- `parseInternalUpdateArgs(args []string) (pid int, newBin, target string, ok bool)` — parses `--internal-update <pid> <new> <target>`
- `cleanupOldBinary(selfPath string)` — cleans up `.old` files from prior updates
- `downloadBinary(tag, destPath string) error` — downloads tagged release from GitHub, writes to destPath. Uses `github.com/engalar/mxcli` repo, resolves asset name from `runtime.GOOS`/`GOARCH`.
- `runSelfUpgrade(args []string) error` — Cobra RunE body
- `runRollback(args []string) error` — Cobra RunE body
- `upgradeCmd` / `rollbackCmd` — `*cobra.Command` vars

The code is a straight port. Key changes from launcher version:
- Replace `e.daemonBinaryPath()` with `os.Executable()` (self-path)
- Replace `e.downloadDaemonVersion(tag, destPath)` with new `downloadBinary()` that downloads the single `mxcli-{os}-{arch}` asset
- Replace launcher's `Env` receiver with free functions

- [ ] **Step 3:** Register in `cmd/mxcli/main.go`:

```go
rootCmd.AddCommand(upgradeCmd)
rootCmd.AddCommand(rollbackCmd)
```

- [ ] **Step 4:** Read `cmd/mxcli-launcher/self_update.go` and `cmd/mxcli-launcher/upgrade.go` for exact function signatures. Port `RealPIDWaiter`, `runInternalUpdate`, `parseInternalUpdateArgs`, `downloadBinary`, `cleanupOldBinary`.

- [ ] **Step 5:** Build and verify:

```bash
go build ./cmd/mxcli/
./mxcli upgrade --help
./mxcli rollback --help
```

- [ ] **Step 6:** Commit:

```bash
git add cmd/mxcli/cmd_upgrade.go cmd/mxcli/upgrade.go cmd/mxcli/lock_*.go cmd/mxcli/self_update_*.go cmd/mxcli/main.go
git commit -m "feat: merge upgrade/rollback commands into main binary"
```

### Task 3: Delete launcher package

**Files:**
- Delete: `cmd/mxcli-launcher/` (entire directory)
- Modify: `Makefile` (remove `LAUNCHER_LDFLAGS`, `release-launcher` target)
- Modify: `cmd/mxcli/main.go` — handle `--internal-update` at startup

- [ ] **Step 1:** Add `--internal-update` handler at the top of `cmd/mxcli/main.go` `main()` function, before `rootCmd.Execute()`:

```go
func main() {
	// --internal-update mode: spawned by upgrade/rollback to replace the binary
	// after the parent process exits. Must be handled before Cobra parsing.
	if len(os.Args) > 1 && os.Args[1] == "--internal-update" {
		pid, newBin, target, ok := parseInternalUpdateArgs(os.Args[2:])
		if !ok {
			fmt.Fprintln(os.Stderr, "mxcli: invalid --internal-update args")
			os.Exit(1)
		}
		if err := runInternalUpdate(pid, newBin, target, &RealPIDWaiter{}, 30*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli: update failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// ... existing main() body ...
```

The key placement: before `cpuProfile`, `PersistentPreRunE`, and `rootCmd.Execute()` — same as the launcher's pattern. The existing CPU profiling flag handling (`MXCLI_CPU_PROFILE`) can stay after the `--internal-update` check since `--internal-update` never uses profiling.

- [ ] **Step 2:** Delete launcher directory:

```bash
rm -rf cmd/mxcli-launcher/
```

- [ ] **Step 3:** Update `Makefile` — remove lines referencing `LAUNCHER_LDFLAGS` and `release-launcher` target. Update `release:` target to build `./cmd/mxcli` directly.

- [ ] **Step 4:** Build and verify:

```bash
go build ./cmd/mxcli/
./mxcli --help  # should show all subcommands
```

- [ ] **Step 5:** Run existing tests:

```bash
go test ./cmd/mxcli/ -count=1 -timeout 120s
```

- [ ] **Step 6:** Commit:

```bash
git add -A
git commit -m "refactor: delete cmd/mxcli-launcher, update Makefile"
```

### Task 4: Delete local package

**Files:**
- Delete: `cmd/mxcli-local/` (entire directory)
- Modify: `Makefile` (remove `LOCAL_LDFLAGS`, `LOCAL_PATH`, `release-local` target)

- [ ] **Step 1:** Delete local directory:

```bash
rm -rf cmd/mxcli-local/
```

- [ ] **Step 2:** Update `Makefile` — remove `LOCAL_LDFLAGS`, `LOCAL_NAME`, `LOCAL_PATH`, `release-local` target, `local-install` target.

- [ ] **Step 3:** Build and verify:

```bash
make build
./bin/mxcli-build --help  # verify build/run/reload still work
```

- [ ] **Step 4:** Commit:

```bash
git add -A
git commit -m "refactor: delete cmd/mxcli-local, update Makefile"
```

### Task 5: Delete launcherproto and socket forwarding

**Files:**
- Delete: `internal/launcherproto/` (entire directory)
- Search and remove references to `launcherproto` in remaining code.

- [ ] **Step 1:** Delete launcherproto:

```bash
rm -rf internal/launcherproto/
```

- [ ] **Step 2:** Search for remaining imports of `launcherproto`:

```bash
grep -rn "launcherproto" cmd/ --include '*.go' || echo "Clean"
```

If found, remove those imports (they should be in deleted launcher files only).

- [ ] **Step 3:** Build:

```bash
go build ./...
```

- [ ] **Step 4:** Commit:

```bash
git add -A
git commit -m "refactor: delete internal/launcherproto, socket forwarding no longer needed"
```

### Task 6: Update CI workflows

**Files:**
- Modify: `.github/workflows/release.yml` — build `./cmd/mxcli` instead of launcher
- Delete: `.github/workflows/release-daemon.yml`
- Delete: `.github/workflows/release-local.yml`
- Modify: `install.sh` — download single binary, no secondary downloads

- [ ] **Step 1:** Read current `release.yml` to understand build matrix.

Currently it builds launcher with `make release-launcher`. Change to build `./cmd/mxcli` directly:

```yaml
- name: Build mxcli binaries
  run: |
    mkdir -p bin
    CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "-X main.Version=${{ github.ref_name }} -X main.BuildTime=$(date -u +'%Y-%m-%dT%H:%M:%SZ') -X main.CommitSHA=$(git rev-parse HEAD) -s -w" -trimpath -o bin/mxcli-linux-amd64   ./cmd/mxcli
    CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "..." -trimpath -o bin/mxcli-linux-arm64   ./cmd/mxcli
    # ... same for darwin/windows ...
```

- [ ] **Step 2:** Delete `release-daemon.yml` and `release-local.yml`:

```bash
rm .github/workflows/release-daemon.yml .github/workflows/release-local.yml
```

- [ ] **Step 3:** Update `install.sh`:

Current: downloads launcher, launcher then downloads daemon + local.

After: downloads single `mxcli` binary from GitHub Release. The binary name changes from `mxcli-linux-amd64` (launcher naming) to `mxcli-linux-amd64` (same — but now it's the full binary). No secondary download or version check needed.

Key changes in install.sh:
- Remove `ensureDaemonBinary` equivalent (no daemon to download)
- Remove version file writing for daemon and local
- The downloaded binary is the complete mxcli

- [ ] **Step 4:** Commit:

```bash
git add -A
git commit -m "ci: update release workflow for single binary, delete daemon/local workflows"
```

### Task 7: Update tag references and documentation

**Files:**
- Modify: `README.md` — update install instructions
- Modify: `docs/` — any references to launcher/daemon/local split

- [ ] **Step 1:** Search for outdated references:

```bash
grep -rn "mxcli-launcher\|mxcli-daemon\|mxcli-local\|daemon-v\|local-v" docs/ --include '*.md' | grep -v superpowers | grep -v archive
```

- [ ] **Step 2:** Update references to reflect single binary.

- [ ] **Step 3:** Update tag instructions in skills:

```bash
# .claude/skills/release-tags/SKILL.md — remove daemon-v and local-v references
```

- [ ] **Step 4:** Commit:

```bash
git add -A
git commit -m "docs: update for merged binary, remove launcher/daemon/local split references"
```

### Task 8: End-to-end validation

- [ ] **Step 1:** Delete old tags and create new single tag:

```bash
git tag -d daemon-v0.28.0 local-v0.9.0 2>/dev/null
git tag v0.28.0 -f
```

- [ ] **Step 2:** Run full test suite:

```bash
go test ./cmd/mxcli/ -count=1 -timeout 120s
go test ./mdl/... -count=1 -timeout 120s
```

- [ ] **Step 3:** Manual smoke test:

```bash
./mxcli --help
./mxcli --version
./mxcli build --help
./mxcli run --help
./mxcli reload --help
./mxcli upgrade --help
./mxcli rollback --help
```

- [ ] **Step 4:** Verify no dangling imports or broken refs:

```bash
go build ./...
```
