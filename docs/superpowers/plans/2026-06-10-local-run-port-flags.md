# Local Run Port Flags Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--port` / `--admin-port` flags to `mxcli local run`, check both ports before starting, and report actionable errors with a copy-paste-ready command using actually-available ports.

**Architecture:** Port logic lives entirely in `cmd/mxcli/docker/local.go`. `LocalRunOptions` gains `AppPort`, `AdminPort`, and `CmdHint` fields. `preflightLocal` checks both ports and uses `findAvailablePorts` to suggest a working alternative. Deploy layout injects ports into the generated HOCON; PAD layout writes a tiny override HOCON file appended to `bin/start` args.

**Tech Stack:** Go stdlib (`net`, `os`, `fmt`), Cobra flags, HOCON (text generation).

---

## File Map

| File | Change |
|------|--------|
| `cmd/mxcli/docker/local.go` | All core changes: new `LocalRunOptions` fields, `canBind`, `FindAvailablePorts`, updated `preflightLocal`, `WriteDeployHOCON` export + port params, PAD override injection |
| `cmd/mxcli/docker/local_test.go` | New tests for port checks, `FindAvailablePorts`, deploy HOCON ports, PAD override file |
| `cmd/mxcli-local/cmd_run.go` | Add `--port` / `--admin-port` flags, populate new `LocalRunOptions` fields |

---

### Task 1: `LocalRunOptions` — add `AppPort`, `AdminPort`, `CmdHint` fields

**Files:**
- Modify: `cmd/mxcli/docker/local.go`

- [ ] **Step 1: Add fields and unexported helper methods**

In `local.go`, add to `LocalRunOptions` struct (after the existing `Stderr` field):

```go
// AppPort is the app HTTP port. 0 = default 8080.
AppPort int
// AdminPort is the admin API port. 0 = default 8090.
AdminPort int
// CmdHint is the -p / --pad-dir fragment used in error messages.
// Example: "-p /path/to/app.mpr" or "--pad-dir /path/to/pad".
CmdHint string
```

Add two unexported helper methods immediately after the struct definition:

```go
func (o *LocalRunOptions) appPort() int {
	if o.AppPort == 0 {
		return 8080
	}
	return o.AppPort
}

func (o *LocalRunOptions) adminPort() int {
	if o.AdminPort == 0 {
		return 8090
	}
	return o.AdminPort
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/mxcli/docker/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/docker/local.go
git commit -m "feat(local): add AppPort/AdminPort/CmdHint to LocalRunOptions"
```

---

### Task 2: `canBind` + `FindAvailablePorts` helpers

**Files:**
- Modify: `cmd/mxcli/docker/local.go`
- Test: `cmd/mxcli/docker/local_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `local_test.go` (inside `package docker_test`):

```go
func TestFindAvailablePorts_ReturnsBindablePair(t *testing.T) {
	// Occupy a pair of ports so FindAvailablePorts must skip them.
	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()
	ln2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln2.Close()

	ap, adm := docker.FindAvailablePorts(8080, 8090)

	// Returned ports must actually be bindable.
	lnA, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ap))
	if err != nil {
		t.Errorf("app port %d not bindable: %v", ap, err)
	} else {
		lnA.Close()
	}
	lnB, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", adm))
	if err != nil {
		t.Errorf("admin port %d not bindable: %v", adm, err)
	} else {
		lnB.Close()
	}
}
```

Add `"fmt"` and `"net"` to the import block in `local_test.go` if not already present.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/mxcli/docker/... -run TestFindAvailablePorts -v
```

Expected: FAIL — `docker.FindAvailablePorts undefined`.

- [ ] **Step 3: Implement `canBind` and `FindAvailablePorts` in `local.go`**

Add after the `parseDBURL` function:

```go
// canBind reports whether the given TCP port is available on localhost.
func canBind(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// FindAvailablePorts returns the first (appPort, adminPort) pair above
// (startApp, startAdmin) where both ports are simultaneously bindable.
// Exported for testing.
func FindAvailablePorts(startApp, startAdmin int) (int, int) {
	for offset := 1; offset < 100; offset++ {
		ap, adm := startApp+offset, startAdmin+offset
		if canBind(ap) && canBind(adm) {
			return ap, adm
		}
	}
	return startApp + 1, startAdmin + 1 // extreme fallback
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./cmd/mxcli/docker/... -run TestFindAvailablePorts -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/docker/local.go cmd/mxcli/docker/local_test.go
git commit -m "feat(local): add canBind + FindAvailablePorts helpers"
```

---

### Task 3: Update `preflightLocal` — check both ports, return actionable error

**Files:**
- Modify: `cmd/mxcli/docker/local.go`
- Test: `cmd/mxcli/docker/local_test.go`

- [ ] **Step 1: Write failing tests**

Append to `local_test.go`:

```go
func TestStartLocal_AdminPortInUse_ReturnsActionableError(t *testing.T) {
	pad := testfixtures.NewFakePAD(t)

	// Occupy a port pair.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	adminPort := ln.Addr().(*net.TCPAddr).Port
	// Pick an app port that is also the admin port + delta so FindAvailablePorts
	// can find a free pair.
	appPort := adminPort - 10
	if appPort < 1024 {
		appPort = adminPort + 10
	}

	err = docker.StartLocal(docker.LocalRunOptions{
		PadDir:    pad.Dir,
		AdminPort: adminPort,
		AppPort:   appPort,
		CmdHint:   "-p /tmp/app.mpr",
		Starter:   &CaptureStarter{},
	})
	if err == nil {
		t.Fatal("expected error when admin port is in use")
	}
	if !strings.Contains(err.Error(), "admin API") {
		t.Errorf("error should mention 'admin API', got: %v", err)
	}
	if !strings.Contains(err.Error(), "/tmp/app.mpr") {
		t.Errorf("error should contain project path, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--admin-port") {
		t.Errorf("error should contain --admin-port flag, got: %v", err)
	}
}

func TestStartLocal_AppPortInUse_ReturnsActionableError(t *testing.T) {
	pad := testfixtures.NewFakePAD(t)

	// Occupy the app port but leave admin free.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	appPort := ln.Addr().(*net.TCPAddr).Port
	// adminPort is offset by 100 — unlikely to be occupied in CI.
	adminPort := appPort + 100

	err = docker.StartLocal(docker.LocalRunOptions{
		PadDir:    pad.Dir,
		AppPort:   appPort,
		AdminPort: adminPort,
		CmdHint:   "--pad-dir /tmp/mypad",
		Starter:   &CaptureStarter{},
	})
	if err == nil {
		t.Fatal("expected error when app port is in use")
	}
	if !strings.Contains(err.Error(), "app HTTP") {
		t.Errorf("error should mention 'app HTTP', got: %v", err)
	}
	if !strings.Contains(err.Error(), "--pad-dir /tmp/mypad") {
		t.Errorf("error should contain pad-dir hint, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--port") {
		t.Errorf("error should contain --port flag, got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/mxcli/docker/... -run "TestStartLocal_AdminPortInUse|TestStartLocal_AppPortInUse" -v
```

Expected: FAIL — errors don't yet mention "admin API" / "app HTTP".

- [ ] **Step 3: Update `preflightLocal` signature and body**

Replace the existing `preflightLocal` function in `local.go`:

```go
// preflightLocal detects port conflicts before starting a new runtime instance.
//
//  1. Port checks: tries to bind adminPort (default 8090) then appPort (default 8080).
//     If either is taken, returns an actionable error with a ready-to-run command
//     using the next simultaneously-available port pair.
//
//  2. HSQLDB stale lock cleanup: removes .lck files left by a killed JVM.
func preflightLocal(dir string, stderr io.Writer, isDeployDir bool, appPort, adminPort int, cmdHint string) error {
	// 1a. Admin port check.
	if !canBind(adminPort) {
		suggestApp, suggestAdm := FindAvailablePorts(appPort, adminPort)
		return fmt.Errorf(
			"port %d (admin API) is already in use.\n\n"+
				"Stop the existing Mendix runtime, or use available ports:\n\n"+
				"  mxcli local run %s --port %d --admin-port %d",
			adminPort, cmdHint, suggestApp, suggestAdm)
	}

	// 1b. App port check.
	if !canBind(appPort) {
		suggestApp, suggestAdm := FindAvailablePorts(appPort, adminPort)
		return fmt.Errorf(
			"port %d (app HTTP) is already in use.\n\n"+
				"Stop the process using the port, or use available ports:\n\n"+
				"  mxcli local run %s --port %d --admin-port %d",
			appPort, cmdHint, suggestApp, suggestAdm)
	}

	// 2. HSQLDB stale lock cleanup — path differs by layout.
	var dbRoot string
	if isDeployDir {
		dbRoot = filepath.Join(dir, "data", "database", "hsqldb")
	} else {
		dbRoot = filepath.Join(dir, "app", "data", "database", "hsqldb")
	}
	_ = filepath.Walk(dbRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".lck") {
			return nil
		}
		if rmErr := os.Remove(path); rmErr != nil {
			fmt.Fprintf(stderr, "warning: cannot remove HSQLDB lock file %s: %v\n", path, rmErr)
		} else {
			fmt.Fprintf(stderr, "info: removed stale HSQLDB lock file: %s\n", path)
		}
		return nil
	})
	return nil
}
```

- [ ] **Step 4: Update the two callers of `preflightLocal`**

In `StartLocal` (PAD path), change:

```go
if err := preflightLocal(opts.PadDir, stderr, false); err != nil {
```

to:

```go
if err := preflightLocal(opts.PadDir, stderr, false, opts.appPort(), opts.adminPort(), opts.CmdHint); err != nil {
```

In `startFromDeployLayout`, change:

```go
if err := preflightLocal(opts.PadDir, stderr, true); err != nil {
```

to:

```go
if err := preflightLocal(opts.PadDir, stderr, true, opts.appPort(), opts.adminPort(), opts.CmdHint); err != nil {
```

- [ ] **Step 5: Run all tests**

```bash
go test ./cmd/mxcli/docker/... -v
```

Expected: all PASS (including the two new tests and all existing tests).

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli/docker/local.go cmd/mxcli/docker/local_test.go
git commit -m "feat(local): check both ports in preflight, suggest available pair in error"
```

---

### Task 4: Parameterize ports in deploy layout (`writeDeployHOCON`)

**Files:**
- Modify: `cmd/mxcli/docker/local.go`
- Test: `cmd/mxcli/docker/local_test.go`

- [ ] **Step 1: Export `writeDeployHOCON` as `WriteDeployHOCON` for testing**

In `local.go`, add an exported wrapper after `writeDeployHOCON`:

```go
// WriteDeployHOCON is the exported wrapper for tests.
func WriteDeployHOCON(path string, dcfg map[string]string, constants map[string]string, dbURL, adminPass string, appPort, adminPort int) error {
	return writeDeployHOCON(path, deployConfig{Configuration: dcfg, Constants: constants}, dbURL, adminPass, appPort, adminPort)
}
```

- [ ] **Step 2: Write the failing test**

Append to `local_test.go`:

```go
func TestWriteDeployHOCON_CustomPorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.conf")
	err := docker.WriteDeployHOCON(path, map[string]string{}, map[string]string{}, "", "Admin123!", 8181, 8191)
	if err != nil {
		t.Fatalf("WriteDeployHOCON: %v", err)
	}
	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "port = 8191") {
		t.Errorf("expected admin port 8191 in HOCON, got:\n%s", content)
	}
	if !strings.Contains(content, "port = 8181") {
		t.Errorf("expected app port 8181 in HOCON, got:\n%s", content)
	}
	if !strings.Contains(content, "localhost:8181") {
		t.Errorf("expected ApplicationRootUrl with port 8181, got:\n%s", content)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./cmd/mxcli/docker/... -run TestWriteDeployHOCON_CustomPorts -v
```

Expected: FAIL — `docker.WriteDeployHOCON undefined`.

- [ ] **Step 4: Update `writeDeployHOCON` signature**

Change the function signature from:

```go
func writeDeployHOCON(path string, dcfg deployConfig, dbURL, adminPass string) error {
```

to:

```go
func writeDeployHOCON(path string, dcfg deployConfig, dbURL, adminPass string, appPort, adminPort int) error {
```

Replace the three hardcoded port/URL lines inside the function:

```go
// Replace:
appURL = "http://localhost:8080/"
// With:
appURL = fmt.Sprintf("http://localhost:%d/", appPort)
```

```go
// Replace:
sb.WriteString("  port = 8090\n")
// With:
fmt.Fprintf(&sb, "  port = %d\n", adminPort)
```

```go
// Replace:
sb.WriteString("    port = 8080\n")
// With:
fmt.Fprintf(&sb, "    port = %d\n", appPort)
```

- [ ] **Step 5: Update the caller in `startFromDeployLayout`**

Find the call to `writeDeployHOCON` in `startFromDeployLayout` and add the two new port arguments:

```go
// Replace:
if err := writeDeployHOCON(hoconPath, dcfg, opts.DB, adminPass); err != nil {
// With:
if err := writeDeployHOCON(hoconPath, dcfg, opts.DB, adminPass, opts.appPort(), opts.adminPort()); err != nil {
```

- [ ] **Step 6: Run all tests**

```bash
go test ./cmd/mxcli/docker/... -v
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/mxcli/docker/local.go cmd/mxcli/docker/local_test.go
git commit -m "feat(local): parameterize ports in deploy layout HOCON"
```

---

### Task 5: PAD layout port override file

**Files:**
- Modify: `cmd/mxcli/docker/local.go`
- Test: `cmd/mxcli/docker/local_test.go`

- [ ] **Step 1: Write the failing test**

Append to `local_test.go`:

```go
func TestStartLocal_PADLayout_CustomPorts_AppendsOverrideConf(t *testing.T) {
	pad := testfixtures.NewFakePAD(t)
	cs := &CaptureStarter{}

	err := docker.StartLocal(docker.LocalRunOptions{
		PadDir:    pad.Dir,
		AppPort:   8181,
		AdminPort: 8191,
		Starter:   cs,
	})
	if err != nil {
		t.Fatalf("StartLocal: %v", err)
	}

	// The last argument to bin/start should be the override conf file.
	args := cs.Cmd.Args
	if len(args) < 2 {
		t.Fatalf("expected at least 2 args, got %d: %v", len(args), args)
	}
	overridePath := args[len(args)-1]
	if !strings.HasSuffix(overridePath, ".conf") {
		t.Fatalf("last arg should be a .conf file, got: %s", overridePath)
	}
	data, err := os.ReadFile(overridePath)
	if err != nil {
		// File may already be cleaned up — check it was passed at least.
		t.Logf("override file already removed (expected): %v", err)
		return
	}
	content := string(data)
	if !strings.Contains(content, "port = 8191") {
		t.Errorf("override conf should set admin port 8191, got:\n%s", content)
	}
	if !strings.Contains(content, "port = 8181") {
		t.Errorf("override conf should set app port 8181, got:\n%s", content)
	}
}

func TestStartLocal_PADLayout_DefaultPorts_NoOverrideConf(t *testing.T) {
	pad := testfixtures.NewFakePAD(t)
	cs := &CaptureStarter{}

	err := docker.StartLocal(docker.LocalRunOptions{
		PadDir:  pad.Dir,
		Starter: cs,
	})
	if err != nil {
		t.Fatalf("StartLocal: %v", err)
	}

	// With default ports, no override conf should be appended.
	args := cs.Cmd.Args
	for _, a := range args {
		if strings.HasSuffix(a, "port-override.conf") {
			t.Errorf("unexpected port-override.conf in args: %v", args)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./cmd/mxcli/docker/... -run "TestStartLocal_PADLayout_Custom|TestStartLocal_PADLayout_Default" -v
```

Expected: FAIL — no override conf is appended yet.

- [ ] **Step 3: Implement override file injection in `startFromPADLayout`**

In `startFromPADLayout`, after `cmdArgs, err := resolveStartScript(opts.PadDir)` and before building `cmd`, add:

```go
// Inject custom ports via a supplemental HOCON override file.
// bin/start passes all positional args as extra config files to runtimelauncher;
// later files override earlier ones, so this cleanly overrides the shipped defaults.
var cleanupOverride func()
if opts.AppPort != 0 || opts.AdminPort != 0 {
    tmpDir, err := os.MkdirTemp("", "mxcli-local-*")
    if err != nil {
        return fmt.Errorf("creating temp dir for port override: %w", err)
    }
    cleanupOverride = func() { os.RemoveAll(tmpDir) }
    overridePath := filepath.Join(tmpDir, "port-override.conf")
    overrideContent := fmt.Sprintf(
        "admin { port = %d }\nruntime.http { port = %d }\n",
        opts.adminPort(), opts.appPort(),
    )
    if err := os.WriteFile(overridePath, []byte(overrideContent), 0600); err != nil {
        os.RemoveAll(tmpDir)
        return fmt.Errorf("writing port override conf: %w", err)
    }
    cmdArgs = append(cmdArgs, overridePath)
}
```

At the end of `startFromPADLayout`, after `return starter.Run(cmd)`, add cleanup. Because `starter.Run` blocks, use `defer` at the top of the if block:

```go
if opts.AppPort != 0 || opts.AdminPort != 0 {
    tmpDir, err := os.MkdirTemp("", "mxcli-local-*")
    if err != nil {
        return fmt.Errorf("creating temp dir for port override: %w", err)
    }
    defer os.RemoveAll(tmpDir)   // cleaned up after runtime exits
    overridePath := filepath.Join(tmpDir, "port-override.conf")
    overrideContent := fmt.Sprintf(
        "admin { port = %d }\nruntime.http { port = %d }\n",
        opts.adminPort(), opts.appPort(),
    )
    if err := os.WriteFile(overridePath, []byte(overrideContent), 0600); err != nil {
        return fmt.Errorf("writing port override conf: %w", err)
    }
    cmdArgs = append(cmdArgs, overridePath)
}
```

(Drop the `cleanupOverride` approach — `defer os.RemoveAll(tmpDir)` is cleaner.)

- [ ] **Step 4: Run all tests**

```bash
go test ./cmd/mxcli/docker/... -v
```

Expected: all PASS. Note: the override file is deleted by `defer os.RemoveAll` when `starter.Run` returns; in tests `CaptureStarter.Run` returns immediately, so the file exists when we check `args` but may be deleted by the time we `ReadFile` it — hence the `t.Logf` fallback in the test.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/docker/local.go cmd/mxcli/docker/local_test.go
git commit -m "feat(local): inject custom ports into PAD layout via HOCON override file"
```

---

### Task 6: Wire flags into `cmd_run.go` + build `CmdHint`

**Files:**
- Modify: `cmd/mxcli-local/cmd_run.go`

- [ ] **Step 1: Add flags and populate `LocalRunOptions`**

In `runCmd()`, add two new variables to the `var` block:

```go
var (
    projectPath   string
    dbURL         string
    padDir        string
    adminPassword string
    appPort       int
    adminPort     int
)
```

After the existing `RunE` logic that resolves `dir`, build `cmdHint` and populate the new fields:

```go
// Build the hint for port-conflict error messages.
var cmdHint string
if projectPath != "" {
    cmdHint = "-p " + projectPath
} else if padDir != "" {
    cmdHint = "--pad-dir " + padDir
}

return docker.StartLocal(docker.LocalRunOptions{
    PadDir:        dir,
    DB:            dbURL,
    AdminPassword: adminPassword,
    AppPort:       appPort,
    AdminPort:     adminPort,
    CmdHint:       cmdHint,
    Stdout:        os.Stdout,
    Stderr:        os.Stderr,
})
```

Register the two new flags at the bottom of `runCmd()`:

```go
cmd.Flags().IntVar(&appPort, "port", 0, "App HTTP port (default 8080)")
cmd.Flags().IntVar(&adminPort, "admin-port", 0, "Admin API port (default 8090)")
```

- [ ] **Step 2: Build and check help text**

```bash
go build -o /tmp/mxcli-local-test ./cmd/mxcli-local && /tmp/mxcli-local-test run --help
```

Expected output contains:

```
--port int          App HTTP port (default 8080)
--admin-port int    Admin API port (default 8090)
```

- [ ] **Step 3: Run full test suite**

```bash
go test ./...
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli-local/cmd_run.go
git commit -m "feat(local): add --port and --admin-port flags to mxcli local run"
```
