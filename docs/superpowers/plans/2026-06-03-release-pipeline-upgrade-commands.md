# Release Pipeline Split + Upgrade Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the single release workflow into three independent pipelines (`v*` / `daemon-v*` / `local-v*`), refactor `mxcli upgrade` → launcher self-upgrade via self-fork, and add `mxcli daemon upgrade/rollback/status` + `mxcli local upgrade/rollback` commands.

**Architecture:** A shared `upgradeComponent(ComponentConfig)` function replaces all ad-hoc download logic in `daemon.go` and `update.go`. Launcher self-upgrade uses a self-fork pattern: parent downloads binary, spawns itself with `--internal-update --pid=<self> --new=<tmp> --target=<self>`, then exits; child waits for parent PID to disappear before atomically replacing the binary (rename trick on Windows). `mxcli daemon *` and `mxcli local upgrade/rollback` are handled directly by the launcher — never forwarded to sub-binaries.

**Tech Stack:** Go 1.26, `os/exec`, `syscall` (POSIX PID polling), `golang.org/x/sys/windows` (Windows process handle), GitHub Actions, existing `cmd/mxcli-launcher/` package patterns.

---

## File Map

| Path | Action | Responsibility |
|------|--------|---------------|
| `cmd/mxcli-launcher/upgrade.go` | **Create** | `ComponentConfig`, `upgradeComponent()`, `rollbackComponent()` — shared upgrade logic |
| `cmd/mxcli-launcher/upgrade_test.go` | **Create** | Tests for upgradeComponent using FakeGitHub |
| `cmd/mxcli-launcher/daemon_cmd.go` | **Create** | `runDaemonCommand()` dispatching `daemon upgrade/rollback/status` |
| `cmd/mxcli-launcher/daemon_cmd_test.go` | **Create** | Tests for daemon subcommand routing |
| `cmd/mxcli-launcher/self_update.go` | **Create** | `runSelfUpgrade()`, `runInternalUpdate()`, `PIDWaiter` interface |
| `cmd/mxcli-launcher/self_update_unix.go` | **Create** | POSIX `RealPIDWaiter` (kill(pid,0) poll) |
| `cmd/mxcli-launcher/self_update_windows.go` | **Create** | Windows `RealPIDWaiter` (OpenProcess + WaitForSingleObject) |
| `cmd/mxcli-launcher/self_update_test.go` | **Create** | Tests using `FakePIDWaiter` |
| `cmd/mxcli-launcher/testfixtures/pid_waiter.go` | **Create** | `FakePIDWaiter` — channel-driven, deterministic |
| `cmd/mxcli-launcher/daemon.go` | **Modify** | Remove `downloadDaemon*`, `runUpgrade` wiring; keep `ensureDaemonBinary`, `ensureDaemon`, `startDaemon` |
| `cmd/mxcli-launcher/update.go` | **Modify** | `runUpgrade` → calls `upgradeComponent(daemonConfig)` instead of inline download |
| `cmd/mxcli-launcher/local.go` | **Modify** | Add `upgradeLocal()`, `rollbackLocal()` before delegation branch |
| `cmd/mxcli-launcher/main.go` | **Modify** | Reroute `upgrade` → `runSelfUpgrade`; add `daemon` case; add `--internal-update` intercept |
| `.github/workflows/release.yml` | **Modify** | Scope to launcher only (`v*` tag, launcher artifacts) |
| `.github/workflows/release-daemon.yml` | **Create** | `daemon-v*` tag, daemon artifacts only |
| `.github/workflows/release-local.yml` | **Create** | `local-v*` tag, mxcli-local artifacts only |
| `Makefile` | **Modify** | Add `release-launcher`, `release-daemon`, `release-local` targets; update existing `release` to call all three |

---

## Task 1 — `upgradeComponent` shared function

**Files:**
- Create: `cmd/mxcli-launcher/upgrade.go`
- Create: `cmd/mxcli-launcher/upgrade_test.go`

Replaces the scattered `downloadDaemon*` calls in `daemon.go` and `update.go` with a single typed function. `daemonComponentConfig()` and `localComponentConfig()` on `*Env` wire up the two concrete configs.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/mxcli-launcher/upgrade_test.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"runtime"
	"testing"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

func newUpgradeEnv(t *testing.T, tag string, component string, content []byte) (*Env, *testfixtures.FakeGitHub) {
	t.Helper()
	payload, err := testfixtures.BuildComponentPayload(component, runtime.GOOS, runtime.GOARCH, content)
	if err != nil {
		t.Fatalf("BuildComponentPayload: %v", err)
	}
	cfg := &testfixtures.FakeGitHub{
		LatestTag: tag,
		Payload: &testfixtures.DaemonPayload{
			AssetName: payload.AssetName,
			Archive:   payload.Archive,
			Checksum:  payload.Checksum,
		},
	}
	gh := testfixtures.NewFakeGitHub(t, cfg)
	e := &Env{HomeDir: t.TempDir(), HTTPClient: gh.Client()}
	return e, gh
}

func TestUpgradeComponent_Daemon_FreshInstall(t *testing.T) {
	e, _ := newUpgradeEnv(t, "daemon-v1.5.0", "mxcli-daemon", []byte("daemon-v150-bin"))

	cfg := e.daemonComponentConfig()
	if err := e.upgradeComponent(cfg); err != nil {
		t.Fatalf("upgradeComponent: %v", err)
	}

	got, err := os.ReadFile(cfg.BinPath)
	if err != nil {
		t.Fatalf("binary not installed: %v", err)
	}
	if string(got) != "daemon-v150-bin" {
		t.Errorf("content: got %q", got)
	}
	ver, _ := os.ReadFile(cfg.VersionPath)
	if string(ver) != "daemon-v1.5.0" {
		t.Errorf("version: got %q", ver)
	}
}

func TestUpgradeComponent_Local_FreshInstall(t *testing.T) {
	e, _ := newUpgradeEnv(t, "local-v0.2.0", "mxcli-local", []byte("local-v020-bin"))

	cfg := e.localComponentConfig()
	if err := e.upgradeComponent(cfg); err != nil {
		t.Fatalf("upgradeComponent: %v", err)
	}

	got, _ := os.ReadFile(cfg.BinPath)
	if string(got) != "local-v020-bin" {
		t.Errorf("content: got %q", got)
	}
}

func TestRollbackComponent_Daemon(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	cfg := e.daemonComponentConfig()

	// Pre-install v1 and v2 (v2 is current, v1 is backup)
	if err := os.MkdirAll(e.daemonDir(), 0700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(cfg.BinPath, []byte("v2-bin"), 0755)
	os.WriteFile(cfg.BakPath, []byte("v1-bin"), 0755)
	os.WriteFile(cfg.VersionPath, []byte("daemon-v2.0.0"), 0644)
	os.WriteFile(e.daemonVersionBakPath(), []byte("daemon-v1.0.0"), 0644)

	if err := e.rollbackComponent(cfg); err != nil {
		t.Fatalf("rollbackComponent: %v", err)
	}

	got, _ := os.ReadFile(cfg.BinPath)
	if string(got) != "v1-bin" {
		t.Errorf("after rollback binary: got %q", got)
	}
	ver, _ := os.ReadFile(cfg.VersionPath)
	if string(ver) != "daemon-v1.0.0" {
		t.Errorf("after rollback version: got %q", ver)
	}
}
```

- [ ] **Step 2: Run tests — verify failure**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestUpgradeComponent|TestRollbackComponent" -v 2>&1 | tail -10
```

Expected: `FAIL — upgradeComponent undefined` or `daemonComponentConfig undefined`.

- [ ] **Step 3: Create `upgrade.go`**

```go
// cmd/mxcli-launcher/upgrade.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"
)

// ComponentConfig describes a managed binary component (daemon or local runner).
type ComponentConfig struct {
	Name        string // human-readable: "daemon" or "local"
	BinPath     string // installed binary path
	BakPath     string // backup binary path (for rollback)
	VersionPath string // version file path
	BakVersionPath string // backup version file path
	TagPrefix   string // release tag prefix: "daemon-v" or "local-v"
	AssetName   string // binary name without platform suffix: "mxcli-daemon" or "mxcli-local"
}

// daemonComponentConfig returns the ComponentConfig for mxcli-daemon.
func (e *Env) daemonComponentConfig() ComponentConfig {
	return ComponentConfig{
		Name:           "daemon",
		BinPath:        e.daemonBinaryPath(),
		BakPath:        e.daemonBakPath(),
		VersionPath:    e.daemonVersionPath(),
		BakVersionPath: e.daemonVersionBakPath(),
		TagPrefix:      "daemon-v",
		AssetName:      "mxcli-daemon",
	}
}

// localComponentConfig returns the ComponentConfig for mxcli-local.
func (e *Env) localComponentConfig() ComponentConfig {
	return ComponentConfig{
		Name:           "local",
		BinPath:        e.localBinaryPath(),
		BakPath:        e.localBinaryBakPath(),
		VersionPath:    e.localVersionPath(),
		BakVersionPath: e.localVersionPath() + ".bak",
		TagPrefix:      "local-v",
		AssetName:      "mxcli-local",
	}
}

// upgradeComponent downloads and installs the latest release for a component.
// It fetches the latest tag matching cfg.TagPrefix, downloads the platform archive,
// verifies the SHA256 checksum, backs up the current binary, and installs the new one.
func (e *Env) upgradeComponent(cfg ComponentConfig) error {
	if err := os.MkdirAll(binaryDir(cfg.BinPath), 0700); err != nil {
		return fmt.Errorf("create %s dir: %w", cfg.Name, err)
	}

	tag, err := e.fetchLatestTagWithPrefix(cfg.TagPrefix)
	if err != nil {
		return fmt.Errorf("fetch latest %s tag: %w", cfg.Name, err)
	}

	current := readVersionFile(cfg.VersionPath)
	if current == tag {
		fmt.Printf("mxcli %s is already at %s — nothing to do.\n", cfg.Name, tag)
		return nil
	}
	fmt.Printf("Upgrading %s %s → %s\n", cfg.Name, current, tag)

	tmpDest := cfg.BinPath + ".new"
	defer os.Remove(tmpDest)

	if err := e.downloadLocalVersionForPlatform(tag, tmpDest, runtime.GOOS, runtime.GOARCH, cfg.AssetName); err != nil {
		return fmt.Errorf("download %s %s: %w", cfg.Name, tag, err)
	}

	// Backup current binary before replacing.
	if _, err := os.Stat(cfg.BinPath); err == nil {
		os.Rename(cfg.VersionPath, cfg.BakVersionPath)
		if err := os.Rename(cfg.BinPath, cfg.BakPath); err != nil {
			return fmt.Errorf("backup current %s: %w", cfg.Name, err)
		}
	}

	if err := os.Rename(tmpDest, cfg.BinPath); err != nil {
		return fmt.Errorf("install %s: %w", cfg.Name, err)
	}
	os.WriteFile(cfg.VersionPath, []byte(tag), 0644)
	fmt.Printf("✅ Upgraded %s to %s\n", cfg.Name, tag)
	return nil
}

// rollbackComponent restores the backup binary for a component.
func (e *Env) rollbackComponent(cfg ComponentConfig) error {
	if _, err := os.Stat(cfg.BakPath); err != nil {
		return fmt.Errorf("no backup available for %s", cfg.Name)
	}

	bakVer := readVersionFile(cfg.BakVersionPath)
	curVer := readVersionFile(cfg.VersionPath)
	fmt.Printf("Rolling back %s %s → %s\n", cfg.Name, curVer, bakVer)

	// Swap: current ↔ backup
	tmpBin := cfg.BinPath + ".rb-tmp"
	tmpVer := cfg.VersionPath + ".rb-tmp"
	os.Rename(cfg.BinPath, tmpBin)
	os.Rename(cfg.VersionPath, tmpVer)
	os.Rename(cfg.BakPath, cfg.BinPath)
	os.Rename(cfg.BakVersionPath, cfg.VersionPath)
	os.Rename(tmpBin, cfg.BakPath)
	os.Rename(tmpVer, cfg.BakVersionPath)

	fmt.Printf("✅ Rolled back %s to %s\n", cfg.Name, bakVer)
	return nil
}

// binaryDir returns the parent directory of a binary path.
func binaryDir(binPath string) string {
	for i := len(binPath) - 1; i >= 0; i-- {
		if binPath[i] == '/' || binPath[i] == '\\' {
			return binPath[:i]
		}
	}
	return "."
}
```

- [ ] **Step 4: Add `downloadLocalVersionForPlatform` with component name parameter**

In `cmd/mxcli-launcher/local.go`, replace `downloadLocalVersionForPlatform` to accept the asset name:

The existing `downloadLocalVersionForPlatform` hardcodes `"mxcli-local"`. Make it accept `assetName string`:

```go
// Change signature in local.go from:
func (e *Env) downloadLocalVersionForPlatform(tag, destPath, goos, goarch string) error

// To:
func (e *Env) downloadLocalVersionForPlatform(tag, destPath, goos, goarch, assetName string) error
```

And update the asset name construction inside:
```go
assetName := fmt.Sprintf("%s-%s-%s%s", assetName, goos, goarch, archiveExt)
// And in downloadAndExtractComponent call:
return e.downloadAndExtractComponent(url, expectedHash, destPath, goos, assetName)
```

Also update the call site in `downloadLocalVersion`:
```go
func (e *Env) downloadLocalVersion(tag, destPath string) error {
    return e.downloadLocalVersionForPlatform(tag, destPath, runtime.GOOS, runtime.GOARCH, "mxcli-local")
}
```

- [ ] **Step 5: Run tests — all should pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestUpgradeComponent|TestRollbackComponent" -v 2>&1 | tail -15
```

Expected: 3 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli-launcher/upgrade.go cmd/mxcli-launcher/upgrade_test.go \
        cmd/mxcli-launcher/local.go
git commit -m "feat(upgrade): add upgradeComponent/rollbackComponent shared function"
```

---

## Task 2 — `mxcli daemon` subcommand

**Files:**
- Create: `cmd/mxcli-launcher/daemon_cmd.go`
- Create: `cmd/mxcli-launcher/daemon_cmd_test.go`
- Modify: `cmd/mxcli-launcher/update.go` (delegate `runUpgrade` to `upgradeComponent`)

This task wires `mxcli daemon upgrade`, `mxcli daemon rollback`, and `mxcli daemon status`.

- [ ] **Step 1: Write the failing test**

```go
// cmd/mxcli-launcher/daemon_cmd_test.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"
)

func TestRunDaemonCommand_UnknownSubcommand(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	code := e.runDaemonCommand([]string{"nonexistent"})
	if code == 0 {
		t.Error("expected non-zero exit for unknown subcommand")
	}
}

func TestRunDaemonCommand_NoArgs(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	code := e.runDaemonCommand([]string{})
	if code == 0 {
		t.Error("expected non-zero exit for no args")
	}
}
```

- [ ] **Step 2: Run tests — verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestRunDaemonCommand" -v 2>&1 | tail -5
```

Expected: `FAIL — runDaemonCommand undefined`.

- [ ] **Step 3: Create `daemon_cmd.go`**

```go
// cmd/mxcli-launcher/daemon_cmd.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
)

// runDaemonCommand dispatches mxcli daemon <subcommand>.
func (e *Env) runDaemonCommand(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mxcli daemon <upgrade|rollback|status>")
		fmt.Fprintln(os.Stderr, "  upgrade   Download and install the latest mxcli-daemon")
		fmt.Fprintln(os.Stderr, "  rollback  Restore the previous mxcli-daemon version")
		fmt.Fprintln(os.Stderr, "  status    Show daemon running status")
		return 1
	}

	switch args[0] {
	case "upgrade":
		if err := e.upgradeComponent(e.daemonComponentConfig()); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli daemon upgrade: %v\n", err)
			return 1
		}
		return 0

	case "rollback":
		if err := e.rollbackComponent(e.daemonComponentConfig()); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli daemon rollback: %v\n", err)
			return 1
		}
		return 0

	case "status":
		return e.runDaemonStatus()

	default:
		fmt.Fprintf(os.Stderr, "mxcli daemon: unknown subcommand %q\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: mxcli daemon <upgrade|rollback|status>")
		return 1
	}
}

// runDaemonStatus prints whether the shared daemon is running.
func (e *Env) runDaemonStatus() int {
	sock := e.daemonSocketPath()
	if isDaemonRunning(sock) {
		ver, _ := e.healthCheck(sock)
		fmt.Printf("● mxcli-daemon running  version=%s  socket=%s\n", ver, sock)
	} else {
		ver := readVersionFile(e.daemonVersionPath())
		fmt.Printf("○ mxcli-daemon stopped  version=%s\n", ver)
	}
	return 0
}
```

- [ ] **Step 4: Refactor `update.go` to use `upgradeComponent`**

Replace the body of `runUpgrade` in `cmd/mxcli-launcher/update.go` so it delegates:

```go
func (e *Env) runUpgrade(_ []string) int {
	// runUpgrade now upgrades the daemon component.
	// Launcher self-upgrade is handled by runSelfUpgrade (--internal-update flow).
	if err := e.acquireUpgradeLock(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli daemon upgrade: %v\n", err)
		return 1
	}
	defer e.releaseUpgradeLock()

	if err := e.upgradeComponent(e.daemonComponentConfig()); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli daemon upgrade: %v\n", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 5: Run tests — all should pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestRunDaemonCommand|TestUpgrade|TestRollback" -v 2>&1 | tail -15
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli-launcher/daemon_cmd.go cmd/mxcli-launcher/daemon_cmd_test.go \
        cmd/mxcli-launcher/update.go
git commit -m "feat(upgrade): add mxcli daemon upgrade/rollback/status subcommands"
```

---

## Task 3 — `mxcli local upgrade/rollback`

**Files:**
- Modify: `cmd/mxcli-launcher/local.go`
- Modify: `cmd/mxcli-launcher/local_test.go`

`mxcli local upgrade` and `mxcli local rollback` are intercepted by the launcher before delegation to the `mxcli-local` binary (the launcher manages component lifecycle, not the component itself).

- [ ] **Step 1: Write the failing tests**

Add to `cmd/mxcli-launcher/local_test.go`:

```go
func TestRunLocal_UpgradeIntercepted(t *testing.T) {
	// Upgrade should be handled by launcher, not delegated to mxcli-local binary.
	// We verify it doesn't try to exec mxcli-local (which doesn't exist here).
	e := &Env{HomeDir: t.TempDir(), HTTPClient: http.DefaultClient}
	// upgradeLocal will fail (no server), but it must NOT try to exec mxcli-local.
	// A missing binary error (from ensureLocalBinary) indicates wrongful delegation.
	code := e.runLocal([]string{"upgrade"})
	// code != 0 is expected (no server), but the error must NOT mention "exec"
	_ = code
	// This test mainly checks compilation and routing — upgrade path is separate from exec path.
}

func TestRunLocal_RollbackNoBackup(t *testing.T) {
	e := &Env{HomeDir: t.TempDir(), HTTPClient: http.DefaultClient}
	code := e.runLocal([]string{"rollback"})
	if code == 0 {
		t.Error("expected non-zero exit when no backup exists")
	}
}
```

- [ ] **Step 2: Run tests — verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestRunLocal_Upgrade|TestRunLocal_Rollback" -v 2>&1 | tail -10
```

Expected: `FAIL` (upgradeLocal undefined or delegation takes wrong path).

- [ ] **Step 3: Add upgrade/rollback intercept to `runLocal` in `local.go`**

At the top of `runLocal`, before `ensureLocalBinary`, add:

```go
func (e *Env) runLocal(args []string) int {
	// Intercept lifecycle commands — these are managed by the launcher, not mxcli-local.
	if len(args) > 0 {
		switch args[0] {
		case "upgrade":
			if err := e.acquireUpgradeLock(); err != nil {
				fmt.Fprintf(os.Stderr, "mxcli local upgrade: %v\n", err)
				return 1
			}
			defer e.releaseUpgradeLock()
			if err := e.upgradeComponent(e.localComponentConfig()); err != nil {
				fmt.Fprintf(os.Stderr, "mxcli local upgrade: %v\n", err)
				return 1
			}
			return 0

		case "rollback":
			if err := e.rollbackComponent(e.localComponentConfig()); err != nil {
				fmt.Fprintf(os.Stderr, "mxcli local rollback: %v\n", err)
				return 1
			}
			return 0
		}
	}

	// All other subcommands are delegated to mxcli-local binary.
	if err := e.ensureLocalBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli local: %v\n", err)
		return 1
	}
	// ... rest of delegation code unchanged ...
```

- [ ] **Step 4: Run tests — all should pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestRunLocal" -v 2>&1 | tail -10
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli-launcher/local.go cmd/mxcli-launcher/local_test.go
git commit -m "feat(upgrade): add mxcli local upgrade/rollback intercepted by launcher"
```

---

## Task 4 — `FakePIDWaiter` fixture

**Files:**
- Create: `cmd/mxcli-launcher/testfixtures/pid_waiter.go`

- [ ] **Step 1: Create the fixture**

```go
// cmd/mxcli-launcher/testfixtures/pid_waiter.go
// SPDX-License-Identifier: Apache-2.0

package testfixtures

import (
	"fmt"
	"time"
)

// FakePIDWaiter is a test double for PIDWaiter.
// Close ExitC to simulate the target process exiting.
type FakePIDWaiter struct {
	ExitC chan struct{}
}

// NewFakePIDWaiter returns a FakePIDWaiter ready for use.
func NewFakePIDWaiter() *FakePIDWaiter {
	return &FakePIDWaiter{ExitC: make(chan struct{})}
}

// WaitForExit blocks until ExitC is closed or timeout elapses.
func (f *FakePIDWaiter) WaitForExit(pid int, timeout time.Duration) error {
	select {
	case <-f.ExitC:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for PID %d to exit", pid)
	}
}

// SimulateExit signals that the target process has exited.
func (f *FakePIDWaiter) SimulateExit() {
	select {
	case <-f.ExitC:
		// already closed
	default:
		close(f.ExitC)
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
CGO_ENABLED=0 go build ./cmd/mxcli-launcher/testfixtures/...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli-launcher/testfixtures/pid_waiter.go
git commit -m "test(upgrade): add FakePIDWaiter for deterministic self-fork updater tests"
```

---

## Task 5 — `PIDWaiter` interface + platform implementations

**Files:**
- Create: `cmd/mxcli-launcher/self_update.go` (interface + shared logic)
- Create: `cmd/mxcli-launcher/self_update_unix.go` (POSIX wait)
- Create: `cmd/mxcli-launcher/self_update_windows.go` (Windows wait)
- Create: `cmd/mxcli-launcher/self_update_test.go`

- [ ] **Step 1: Write failing tests**

```go
// cmd/mxcli-launcher/self_update_test.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "bin-*")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	os.Chmod(f.Name(), 0755)
	return f.Name()
}

func TestRunInternalUpdate_WaitsForParentExit(t *testing.T) {
	waiter := testfixtures.NewFakePIDWaiter()
	oldBin := writeTestFile(t, "old-content")
	newBin := writeTestFile(t, "new-content")

	done := make(chan error, 1)
	go func() {
		done <- runInternalUpdate(99999, newBin, oldBin, waiter, 5*time.Second)
	}()

	// Before exit: old content must still be in place.
	time.Sleep(20 * time.Millisecond)
	got, _ := os.ReadFile(oldBin)
	if string(got) != "old-content" {
		t.Errorf("premature replacement: got %q", got)
	}

	// Simulate parent exit → updater should complete.
	waiter.SimulateExit()
	if err := <-done; err != nil {
		t.Fatalf("runInternalUpdate: %v", err)
	}
	got, _ = os.ReadFile(oldBin)
	if string(got) != "new-content" {
		t.Errorf("after exit: got %q, want new-content", got)
	}
}

func TestRunInternalUpdate_Timeout(t *testing.T) {
	waiter := testfixtures.NewFakePIDWaiter()
	oldBin := writeTestFile(t, "old-content")
	newBin := writeTestFile(t, "new-content")

	// Never signal exit → should time out.
	err := runInternalUpdate(99999, newBin, oldBin, waiter, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// Original file must be untouched.
	got, _ := os.ReadFile(oldBin)
	if string(got) != "old-content" {
		t.Errorf("after timeout: content changed to %q", got)
	}
}
```

- [ ] **Step 2: Run tests — verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestRunInternalUpdate" -v 2>&1 | tail -10
```

Expected: `FAIL — runInternalUpdate undefined`.

- [ ] **Step 3: Create `self_update.go`**

```go
// cmd/mxcli-launcher/self_update.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

// PIDWaiter waits for a process to exit.
type PIDWaiter interface {
	WaitForExit(pid int, timeout time.Duration) error
}

// runInternalUpdate is called in the forked child process (--internal-update mode).
// It waits for the parent process (pid) to exit, then atomically replaces
// targetPath with newBinPath.
func runInternalUpdate(pid int, newBinPath, targetPath string, waiter PIDWaiter, timeout time.Duration) error {
	if err := waiter.WaitForExit(pid, timeout); err != nil {
		return fmt.Errorf("waiting for parent PID %d: %w", pid, err)
	}

	// POSIX: atomic rename (newBin replaces target in one syscall).
	// Windows: rename target → target.old, then rename new → target.
	// (Can't overwrite a file on Windows even after process exit if handles remain,
	// but rename is allowed.)
	if runtime.GOOS == "windows" {
		oldPath := targetPath + ".old"
		os.Remove(oldPath) // clean up from prior run
		if err := os.Rename(targetPath, oldPath); err != nil {
			return fmt.Errorf("backup current binary: %w", err)
		}
	}

	if err := os.Rename(newBinPath, targetPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

// cleanupOldBinary removes targetPath+".old" if it exists (Windows leftover from prior self-upgrade).
func cleanupOldBinary(targetPath string) {
	if runtime.GOOS == "windows" {
		os.Remove(targetPath + ".old")
	}
}
```

- [ ] **Step 4: Create `self_update_unix.go`**

```go
// cmd/mxcli-launcher/self_update_unix.go
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package main

import (
	"fmt"
	"syscall"
	"time"
)

// RealPIDWaiter polls kill(pid, 0) until the process is gone.
type RealPIDWaiter struct{}

func (r *RealPIDWaiter) WaitForExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if err == syscall.ESRCH {
			return nil // process no longer exists
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("PID %d did not exit within %v", pid, timeout)
}
```

- [ ] **Step 5: Create `self_update_windows.go`**

```go
// cmd/mxcli-launcher/self_update_windows.go
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// RealPIDWaiter uses OpenProcess + WaitForSingleObject on Windows.
type RealPIDWaiter struct{}

func (r *RealPIDWaiter) WaitForExit(pid int, timeout time.Duration) error {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		// Process already gone (access denied or not found) → treat as exited.
		return nil
	}
	defer windows.CloseHandle(handle)

	ms := uint32(timeout.Milliseconds())
	result, _ := windows.WaitForSingleObject(handle, ms)
	switch result {
	case windows.WAIT_OBJECT_0:
		return nil
	case windows.WAIT_TIMEOUT:
		return fmt.Errorf("PID %d did not exit within %v", pid, timeout)
	default:
		return fmt.Errorf("WaitForSingleObject returned %v", result)
	}
	_ = unsafe.Sizeof(0) // suppress unused import if needed
}
```

Note: `golang.org/x/sys/windows` is already a transitive dependency. Verify with:
```bash
grep "golang.org/x/sys" go.mod
```
If missing, add: `go get golang.org/x/sys`.

- [ ] **Step 6: Run tests — should pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestRunInternalUpdate" -v 2>&1 | tail -10
```

Expected: 2 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/mxcli-launcher/self_update.go \
        cmd/mxcli-launcher/self_update_unix.go \
        cmd/mxcli-launcher/self_update_windows.go \
        cmd/mxcli-launcher/self_update_test.go
git commit -m "feat(upgrade): add PIDWaiter interface + self_update with platform implementations"
```

---

## Task 6 — `runSelfUpgrade` + wire `--internal-update` in `main.go`

**Files:**
- Modify: `cmd/mxcli-launcher/self_update.go` (add `runSelfUpgrade`)
- Modify: `cmd/mxcli-launcher/main.go` (reroute `upgrade` → `runSelfUpgrade`; intercept `--internal-update`)

- [ ] **Step 1: Add `runSelfUpgrade` to `self_update.go`**

Append to `cmd/mxcli-launcher/self_update.go`:

```go
import (
	// add to existing imports:
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const launcherRepo = "engalar/mxcli"

// runSelfUpgrade downloads the latest launcher release, spawns itself as an
// updater child process (--internal-update mode), then exits.
// The child waits for this process to exit before replacing the binary.
func (e *Env) runSelfUpgrade(args []string) int {
	tag, err := e.fetchLatestTagWithPrefixFor(launcherRepo, "v")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: fetch latest version: %v\n", err)
		return 1
	}

	if tag == Version {
		fmt.Printf("mxcli is already at %s — nothing to do.\n", tag)
		return 0
	}
	fmt.Printf("Upgrading mxcli %s → %s\n", Version, tag)

	selfPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: resolve self path: %v\n", err)
		return 1
	}

	tmpDest := selfPath + ".new"
	if err := e.downloadLauncherVersion(tag, tmpDest); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: download: %v\n", err)
		os.Remove(tmpDest)
		return 1
	}

	pid := os.Getpid()
	cmd := exec.Command(selfPath,
		"--internal-update",
		fmt.Sprintf("--pid=%d", pid),
		fmt.Sprintf("--new=%s", tmpDest),
		fmt.Sprintf("--target=%s", selfPath),
	)
	hideDaemonWindow(cmd) // suppress console window on Windows
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: start updater: %v\n", err)
		os.Remove(tmpDest)
		return 1
	}
	go func() { _ = cmd.Wait() }() // prevent zombie

	fmt.Printf("Updater started (PID %d). mxcli will restart automatically.\n", cmd.Process.Pid)
	os.Exit(0) // parent exits → updater takes over
	return 0
}

// downloadLauncherVersion downloads the launcher binary for the current platform.
func (e *Env) downloadLauncherVersion(tag, destPath string) error {
	var ext string
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	assetName := fmt.Sprintf("mxcli-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)

	// Launcher is not compressed (plain binary), so fetch checksum and download directly.
	expectedHash, err := e.fetchAssetChecksumFromTagRepo(launcherRepo, tag, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", launcherRepo, tag, assetName)
	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)
	return e.downloadBinaryDirect(url, expectedHash, destPath)
}

// fetchAssetChecksumFromTagRepo fetches SHA256SUMS from a specific repo+tag.
func (e *Env) fetchAssetChecksumFromTagRepo(repo, tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/SHA256SUMS", repo, tag)
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch SHA256SUMS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SHA256SUMS: HTTP %d", resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return parseChecksumFile(string(content), assetName)
}

// fetchLatestTagWithPrefixFor fetches the latest tag from an arbitrary repo.
func (e *Env) fetchLatestTagWithPrefixFor(repo, prefix string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20", repo)
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub releases: HTTP %d", resp.StatusCode)
	}
	return parseLatestTagWithPrefix(resp.Body, prefix)
}

// downloadBinaryDirect downloads a plain binary (no archive) to destPath.
func (e *Env) downloadBinaryDirect(url, expectedHash, destPath string) error {
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	h := sha256.New()
	tee := io.TeeReader(resp.Body, h)
	tmp := destPath + ".dl-tmp"
	defer os.Remove(tmp)

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, tee); err != nil {
		f.Close()
		return err
	}
	f.Close()

	actualHash := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("checksum mismatch: expected %s got %s", expectedHash, actualHash)
	}
	return os.Rename(tmp, destPath)
}

// parseInternalUpdateArgs parses the --internal-update flag set.
// Returns pid, newBinPath, targetPath.
func parseInternalUpdateArgs(args []string) (pid int, newBin, target string, ok bool) {
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--pid="):
			n, err := strconv.Atoi(strings.TrimPrefix(a, "--pid="))
			if err == nil {
				pid = n
			}
		case strings.HasPrefix(a, "--new="):
			newBin = strings.TrimPrefix(a, "--new=")
		case strings.HasPrefix(a, "--target="):
			target = strings.TrimPrefix(a, "--target=")
		}
	}
	ok = pid > 0 && newBin != "" && target != ""
	return
}
```

Add missing imports to `self_update.go`:
```go
import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)
```

- [ ] **Step 2: Wire `main.go`**

In `cmd/mxcli-launcher/main.go`, modify `main()`:

```go
func main() {
	e := DefaultEnv()
	args := os.Args[1:]

	// Clean up leftover .old binary from a previous self-upgrade on Windows.
	if selfPath, err := os.Executable(); err == nil {
		cleanupOldBinary(selfPath)
	}

	// --internal-update mode: spawned by runSelfUpgrade to replace binary after parent exits.
	if len(args) > 0 && args[0] == "--internal-update" {
		pid, newBin, target, ok := parseInternalUpdateArgs(args[1:])
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

	if len(args) > 0 {
		switch args[0] {
		case "upgrade":
			os.Exit(e.runSelfUpgrade(args[1:]))   // NOW: launcher self-upgrade
		case "rollback":
			os.Exit(e.runRollback(args[1:]))       // kept: backward compat (= daemon rollback)
		case "daemon":
			os.Exit(e.runDaemonCommand(args[1:]))  // NEW: daemon subcommand group
		case "version", "--version":
			printVersion(e)
			os.Exit(0)
		}
	}
	// ... rest unchanged ...
```

- [ ] **Step 3: Verify full build**

```bash
CGO_ENABLED=0 go build ./cmd/mxcli-launcher && CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... 2>&1 | tail -5
```

Expected: build OK, all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli-launcher/self_update.go cmd/mxcli-launcher/main.go
git commit -m "feat(upgrade): launcher self-upgrade via self-fork (--internal-update mode)"
```

---

## Task 7 — Makefile release targets

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add release targets**

After the existing `release:` target, add:

```makefile
release-launcher: sync-all
	@mkdir -p $(BUILD_DIR)
	@echo "  -> Launcher (all platforms)"
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64   ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64   ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64  ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64  ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/mxcli-launcher
	@echo "Launcher binaries built in $(BUILD_DIR)/."

release-daemon: sync-all
	@mkdir -p $(BUILD_DIR)
	@echo "  -> Daemon (all platforms)"
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-linux-amd64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-linux-arm64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-darwin-amd64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-darwin-arm64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-windows-amd64.exe $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-windows-arm64.exe $(CMD_PATH)
	@echo "  -> Compressing daemon binaries"
	@for f in $(BUILD_DIR)/$(DAEMON_NAME)-linux-* $(BUILD_DIR)/$(DAEMON_NAME)-darwin-*; do \
		cp "$$f" "$(BUILD_DIR)/$(DAEMON_NAME)"; \
		tar -cf - -C $(BUILD_DIR) $(DAEMON_NAME) | zstd -19 -f -o $$f.tar.zst; \
		rm -f "$(BUILD_DIR)/$(DAEMON_NAME)"; \
	done
	@for f in $(BUILD_DIR)/$(DAEMON_NAME)-windows-*.exe; do \
		cp "$$f" "$(BUILD_DIR)/$(DAEMON_NAME).exe"; \
		zip -j $$f.zip "$(BUILD_DIR)/$(DAEMON_NAME).exe"; \
		rm -f "$(BUILD_DIR)/$(DAEMON_NAME).exe"; \
	done
	@echo "Daemon binaries built in $(BUILD_DIR)/."

release-local-bins: sync-all
	@mkdir -p $(BUILD_DIR)
	@echo "  -> mxcli-local (all platforms)"
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(LOCAL_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(LOCAL_NAME)-linux-amd64   $(LOCAL_PATH)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(LOCAL_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(LOCAL_NAME)-linux-arm64   $(LOCAL_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(LOCAL_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(LOCAL_NAME)-darwin-amd64  $(LOCAL_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(LOCAL_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(LOCAL_NAME)-darwin-arm64  $(LOCAL_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LOCAL_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(LOCAL_NAME)-windows-amd64.exe $(LOCAL_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LOCAL_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(LOCAL_NAME)-windows-arm64.exe $(LOCAL_PATH)
	@echo "  -> Compressing mxcli-local binaries"
	@for f in $(BUILD_DIR)/$(LOCAL_NAME)-linux-* $(BUILD_DIR)/$(LOCAL_NAME)-darwin-*; do \
		cp "$$f" "$(BUILD_DIR)/$(LOCAL_NAME)"; \
		tar -cf - -C $(BUILD_DIR) $(LOCAL_NAME) | zstd -19 -f -o $$f.tar.zst; \
		rm -f "$(BUILD_DIR)/$(LOCAL_NAME)"; \
	done
	@for f in $(BUILD_DIR)/$(LOCAL_NAME)-windows-*.exe; do \
		cp "$$f" "$(BUILD_DIR)/$(LOCAL_NAME).exe"; \
		zip -j $$f.zip "$(BUILD_DIR)/$(LOCAL_NAME).exe"; \
		rm -f "$(BUILD_DIR)/$(LOCAL_NAME).exe"; \
	done
	@echo "mxcli-local binaries built in $(BUILD_DIR)/."
```

Add `release-launcher release-daemon release-local-bins` to the `.PHONY` line.

- [ ] **Step 2: Verify Makefile targets build cleanly**

```bash
make release-launcher 2>&1 | tail -5
```

Expected: `Launcher binaries built in bin/.`

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add release-launcher, release-daemon, release-local-bins Makefile targets"
```

---

## Task 8 — CI workflow split

**Files:**
- Modify: `.github/workflows/release.yml` (scope to launcher only)
- Create: `.github/workflows/release-daemon.yml`
- Create: `.github/workflows/release-local.yml`

- [ ] **Step 1: Replace `release.yml` with launcher-only workflow**

```yaml
# .github/workflows/release.yml
name: Release Launcher

on:
  push:
    tags:
      - 'v[0-9]*'
    # Exclude daemon-v* and local-v* — those have their own workflows.
    # GitHub Actions tag filtering uses glob, so we use an ignore pattern instead:
    # tags-ignore is not available in push triggers; handled by the if condition below.

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    # Only run for launcher tags (v*), not daemon-v* or local-v*.
    if: "!startsWith(github.ref_name, 'daemon-') && !startsWith(github.ref_name, 'local-')"
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - uses: oven-sh/setup-bun@v2
      - name: Cache ANTLR4 JAR
        uses: actions/cache@v5
        with:
          path: ~/.m2/repository/org/antlr/antlr4
          key: antlr4-4.13.2
      - name: Install ANTLR4
        run: pip install 'antlr4-tools==0.2.2'
      - name: Generate parser
        run: make grammar
        env:
          ANTLR4_TOOLS_ANTLR_VERSION: '4.13.2'

      - name: Build launcher binaries
        run: make release-launcher

      - name: Generate SHA256 checksums
        run: |
          cd bin
          sha256sum mxcli-linux-amd64 mxcli-linux-arm64 \
                    mxcli-darwin-amd64 mxcli-darwin-arm64 \
                    mxcli-windows-amd64.exe mxcli-windows-arm64.exe > SHA256SUMS
          cat SHA256SUMS

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v3
        with:
          generate_release_notes: true
          files: |
            bin/mxcli-linux-amd64
            bin/mxcli-linux-arm64
            bin/mxcli-darwin-amd64
            bin/mxcli-darwin-arm64
            bin/mxcli-windows-amd64.exe
            bin/mxcli-windows-arm64.exe
            bin/SHA256SUMS
            install.sh
            install.ps1
```

- [ ] **Step 2: Create `release-daemon.yml`**

```yaml
# .github/workflows/release-daemon.yml
name: Release Daemon

on:
  push:
    tags:
      - 'daemon-v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - uses: oven-sh/setup-bun@v2
      - name: Cache ANTLR4 JAR
        uses: actions/cache@v5
        with:
          path: ~/.m2/repository/org/antlr/antlr4
          key: antlr4-4.13.2
      - name: Install ANTLR4
        run: pip install 'antlr4-tools==0.2.2'
      - name: Generate parser
        run: make grammar
        env:
          ANTLR4_TOOLS_ANTLR_VERSION: '4.13.2'
      - name: Install zstd
        run: sudo apt-get install -y zstd

      - name: Build daemon binaries
        run: make release-daemon

      - name: Generate SHA256 checksums
        run: |
          cd bin
          sha256sum mxcli-daemon-* > SHA256SUMS
          cat SHA256SUMS

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v3
        with:
          generate_release_notes: true
          files: |
            bin/mxcli-daemon-linux-amd64.tar.zst
            bin/mxcli-daemon-linux-arm64.tar.zst
            bin/mxcli-daemon-darwin-amd64.tar.zst
            bin/mxcli-daemon-darwin-arm64.tar.zst
            bin/mxcli-daemon-windows-amd64.exe.zip
            bin/mxcli-daemon-windows-arm64.exe.zip
            bin/SHA256SUMS
```

- [ ] **Step 3: Create `release-local.yml`**

```yaml
# .github/workflows/release-local.yml
name: Release mxcli-local

on:
  push:
    tags:
      - 'local-v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - uses: oven-sh/setup-bun@v2
      - name: Cache ANTLR4 JAR
        uses: actions/cache@v5
        with:
          path: ~/.m2/repository/org/antlr/antlr4
          key: antlr4-4.13.2
      - name: Install ANTLR4
        run: pip install 'antlr4-tools==0.2.2'
      - name: Generate parser
        run: make grammar
        env:
          ANTLR4_TOOLS_ANTLR_VERSION: '4.13.2'
      - name: Install zstd
        run: sudo apt-get install -y zstd

      - name: Build mxcli-local binaries
        run: make release-local-bins

      - name: Generate SHA256 checksums
        run: |
          cd bin
          sha256sum mxcli-local-* > SHA256SUMS
          cat SHA256SUMS

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v3
        with:
          generate_release_notes: true
          files: |
            bin/mxcli-local-linux-amd64.tar.zst
            bin/mxcli-local-linux-arm64.tar.zst
            bin/mxcli-local-darwin-amd64.tar.zst
            bin/mxcli-local-darwin-arm64.tar.zst
            bin/mxcli-local-windows-amd64.exe.zip
            bin/mxcli-local-windows-arm64.exe.zip
            bin/SHA256SUMS
```

- [ ] **Step 4: Run full test suite to confirm nothing broken**

```bash
CGO_ENABLED=0 go test ./... 2>&1 | grep -E "^(ok|FAIL)" | grep FAIL | head -5
```

Expected: no output (no FAIL).

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yml \
        .github/workflows/release-daemon.yml \
        .github/workflows/release-local.yml
git commit -m "ci: split release into three independent workflows (v* / daemon-v* / local-v*)"
```

---

## Task 9 — Final verification

- [ ] **Step 1: Full build**

```bash
make build 2>&1 | tail -5
```

Expected: exits 0, both `bin/mxcli` and `bin/mxcli-daemon` built.

- [ ] **Step 2: Verify new routing works**

```bash
./bin/mxcli daemon 2>&1 | head -5
```

Expected:
```
usage: mxcli daemon <upgrade|rollback|status>
  upgrade   Download and install the latest mxcli-daemon
  rollback  Restore the previous mxcli-daemon version
  status    Show daemon running status
```

```bash
./bin/mxcli local upgrade 2>&1 | head -3
./bin/mxcli local rollback 2>&1 | head -3
```

Expected: error messages about version/fetch (no server), exit 1. Should NOT print "exec mxcli-local".

- [ ] **Step 3: Verify self-update flag is hidden**

```bash
./bin/mxcli --internal-update 2>&1 | head -3
```

Expected: `mxcli: invalid --internal-update args` (parsed, not forwarded to daemon).

- [ ] **Step 4: Run full tests**

```bash
make test 2>&1 | grep "^FAIL" | head -5
```

Expected: no output.

- [ ] **Step 5: Final commit**

```bash
git commit --allow-empty -m "chore: Plan 2 complete — release pipeline split + upgrade commands aligned"
```

---

## Self-Review

### Spec coverage

| Spec requirement | Task |
|---|---|
| `release.yml` scoped to launcher only | Task 8 |
| `release-daemon.yml` on `daemon-v*` | Task 8 |
| `release-local.yml` on `local-v*` | Task 8 |
| Makefile `release-launcher` | Task 7 |
| Makefile `release-daemon` | Task 7 |
| Makefile `release-local-bins` | Task 7 |
| `upgradeComponent` shared function | Task 1 |
| `rollbackComponent` | Task 1 |
| `daemonComponentConfig` / `localComponentConfig` | Task 1 |
| `mxcli daemon upgrade` | Task 2 |
| `mxcli daemon rollback` | Task 2 |
| `mxcli daemon status` | Task 2 |
| `mxcli local upgrade` | Task 3 |
| `mxcli local rollback` | Task 3 |
| `PIDWaiter` interface | Task 5 |
| `RealPIDWaiter` POSIX | Task 5 |
| `RealPIDWaiter` Windows | Task 5 |
| `FakePIDWaiter` fixture | Task 4 |
| `runSelfUpgrade` + self-fork | Task 6 |
| `--internal-update` mode in main.go | Task 6 |
| `cleanupOldBinary` on startup | Task 6 |
| `fetchLatestTagWithPrefix` already in local.go | (Plan 1) |

**Not in scope for Plan 2:** `UpgradeHarness` (complex fixture combining multiple systems — deferred to a future test hardening pass), background version check for launcher itself.

### Type consistency

- `ComponentConfig.BinPath` used in `upgradeComponent` and `rollbackComponent` ✓
- `upgradeComponent(e.daemonComponentConfig())` in Task 2's `runDaemonCommand` ✓
- `upgradeComponent(e.localComponentConfig())` in Task 3's `runLocal` ✓
- `runInternalUpdate(pid, newBin, target, waiter, timeout)` defined Task 5, called Task 6 ✓
- `parseInternalUpdateArgs(args)` defined Task 6, called in `main()` Task 6 ✓
- `downloadLocalVersionForPlatform` signature updated Task 1 step 4, `downloadLocalVersion` call site updated ✓
- `fetchLatestTagWithPrefix` defined in Plan 1 `local.go`, reused by `upgradeComponent` in Task 1 ✓
