# mxcli local runner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `mxcli local build` and `mxcli local run` commands that build and run a Mendix PAD package on Windows and Linux without Docker.

**Architecture:** `mxcli-launcher` detects the `local` subcommand, downloads `mxcli-local` from GitHub Releases on demand, and delegates to it via `exec`. `mxcli-local` is an independent Cobra binary: `build` calls the existing `docker.Build()`, `run` execs `bin/start` (Linux) or `bin\start.bat` (Windows) from the pre-built PAD — POC-verified to work with HSQLDB and no external dependencies.

**Tech Stack:** Go 1.26, `github.com/spf13/cobra`, existing `cmd/mxcli/docker` package (`Build`, `hasExtractedPADLayout`, `resolveJDK21`), `net/url` for DB URL parsing, `net/http/httptest` for fixture servers.

---

## File Map

| Path | Action | Responsibility |
|------|--------|---------------|
| `cmd/mxcli/docker/testfixtures/pad.go` | **Create** | `FakePAD` — minimal valid PAD dir for tests |
| `cmd/mxcli/docker/local.go` | **Create** | `StartLocal()`, `resolveStartScript()`, `buildLocalEnv()`, `parseDBURL()`, `ProcessStarter` interface |
| `cmd/mxcli/docker/local_test.go` | **Create** | Unit tests for StartLocal, env injection, script selection |
| `cmd/mxcli-local/main.go` | **Create** | Cobra entry point, `Version` var |
| `cmd/mxcli-local/cmd_build.go` | **Create** | `build` subcommand → `docker.Build()` |
| `cmd/mxcli-local/cmd_run.go` | **Create** | `run` subcommand → `docker.StartLocal()` |
| `cmd/mxcli-launcher/testfixtures/component_payload.go` | **Create** | `ComponentPayload`, `BuildComponentPayload()` — generalize DaemonPayload |
| `cmd/mxcli-launcher/testfixtures/fake_daemon.go` | **Modify** | `BuildDaemonPayload` delegates to `BuildComponentPayload` |
| `cmd/mxcli-launcher/paths.go` | **Modify** | Add `localDir()`, `localBinaryPath()`, `localVersionPath()` |
| `cmd/mxcli-launcher/local.go` | **Create** | `runLocal()`, `ensureLocalBinary()`, `downloadLocal()` |
| `cmd/mxcli-launcher/local_test.go` | **Create** | Tests for routing, download, version check |
| `cmd/mxcli-launcher/main.go` | **Modify** | Add `local` case before daemon routing |
| `Makefile` | **Modify** | `build-local`, `install-local` targets |

---

## Task 1 — FakePAD test fixture

**Files:**
- Create: `cmd/mxcli/docker/testfixtures/pad.go`

This fixture encodes the contract of `hasExtractedPADLayout()`. Every required file/dir it checks must be created here. If the layout check ever changes, this fixture must change too.

- [ ] **Step 1: Create the fixture file**

```go
// cmd/mxcli/docker/testfixtures/pad.go
// SPDX-License-Identifier: Apache-2.0

package testfixtures

import (
	"os"
	"path/filepath"
	"testing"
)

// FakePAD creates a minimal valid PAD directory structure for StartLocal tests.
// It satisfies hasExtractedPADLayout() exactly: bin/start (executable),
// lib/runtime/launcher/runtimelauncher.jar, app/, bin/, etc/, lib/ dirs.
type FakePAD struct {
	Dir string
}

// NewFakePAD creates the PAD structure in a temp dir and registers cleanup.
func NewFakePAD(t *testing.T) *FakePAD {
	t.Helper()
	dir := t.TempDir()
	p := &FakePAD{Dir: dir}

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("FakePAD setup: %v", err)
		}
	}
	mkdirAll := func(rel string) {
		must(os.MkdirAll(filepath.Join(dir, rel), 0755))
	}
	writeFile := func(rel, content string, mode os.FileMode) {
		path := filepath.Join(dir, rel)
		must(os.MkdirAll(filepath.Dir(path), 0755))
		must(os.WriteFile(path, []byte(content), mode))
	}

	// Required dirs
	mkdirAll("app/model/lib/userlib")
	mkdirAll("bin")
	mkdirAll("etc/configurations")
	mkdirAll("etc/constants")
	mkdirAll("lib/runtime/lib/x64")
	mkdirAll("lib/runtime/launcher")

	// bin/start — executable POSIX script
	writeFile("bin/start", `#!/bin/sh
exec java -jar "$ROOT_PATH/lib/runtime/launcher/runtimelauncher.jar" "$ROOT_PATH/app/." "$ROOT_PATH/etc/Default"
`, 0755)

	// bin/start.bat — Windows batch script
	writeFile("bin/start.bat", `@echo off
java -jar "%ROOT_PATH%\lib\runtime\launcher\runtimelauncher.jar" "%ROOT_PATH%\app\." "%ROOT_PATH%\etc\Default"
`, 0644)

	// runtimelauncher.jar placeholder (content irrelevant for path tests)
	writeFile("lib/runtime/launcher/runtimelauncher.jar", "fake-jar", 0644)

	// Minimal HOCON config chain
	writeFile("etc/Default", `
include file("etc/configurations/Default.conf")
include file("etc/variables.conf")
`, 0644)
	writeFile("etc/configurations/Default.conf", `
runtime.params {
  DatabaseType = HSQLDB
  DatabaseName = default
}
admin { port = 8090 }
runtime.http { port = 8080 }
`, 0644)
	writeFile("etc/variables.conf", `
admin.adminPassword = ${?ADMIN_ADMINPASSWORD}
runtime.adminUser.password = ${?RUNTIME_ADMINUSER_PASSWORD}
runtime.params {
  "DatabaseType" = ${?RUNTIME_PARAMS_DATABASETYPE}
  "DatabaseJdbcUrl" = ${?RUNTIME_PARAMS_DATABASEJDBCURL}
  "DatabaseUserName" = ${?RUNTIME_PARAMS_DATABASEUSERNAME}
  "DatabasePassword" = ${?RUNTIME_PARAMS_DATABASEPASSWORD}
}
`, 0644)

	return p
}

// SetJVMHeap adds a jvm.heap entry to a dedicated jvm.conf include file.
// Call before passing Dir to StartLocal.
func (p *FakePAD) SetJVMHeap(t *testing.T, heap string) *FakePAD {
	t.Helper()
	path := filepath.Join(p.Dir, "etc", "jvm.conf")
	if err := os.WriteFile(path, []byte("jvm.heap = "+heap+"\n"), 0644); err != nil {
		t.Fatalf("FakePAD.SetJVMHeap: %v", err)
	}
	return p
}
```

- [ ] **Step 2: Verify the fixture compiles**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
CGO_ENABLED=0 go build ./cmd/mxcli/docker/testfixtures/...
```

Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/docker/testfixtures/pad.go
git commit -m "test(local): add FakePAD fixture encoding PAD layout contract"
```

---

## Task 2 — ProcessStarter interface + CaptureStarter

**Files:**
- Create: `cmd/mxcli/docker/local.go` (interface + CaptureStarter only, StartLocal skeleton)

- [ ] **Step 1: Write the failing test**

```go
// cmd/mxcli/docker/local_test.go
// SPDX-License-Identifier: Apache-2.0

package docker_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker/testfixtures"
)

// CaptureStarter records the exec.Cmd passed to it and returns nil.
type CaptureStarter struct {
	Cmd *exec.Cmd
}

func (c *CaptureStarter) Run(cmd *exec.Cmd) error {
	c.Cmd = cmd
	return nil
}

func TestStartLocal_ExecsCorrectScript(t *testing.T) {
	pad := testfixtures.NewFakePAD(t)
	cs := &CaptureStarter{}

	err := docker.StartLocal(docker.LocalRunOptions{
		PadDir:  pad.Dir,
		Starter: cs,
	})
	if err != nil {
		t.Fatalf("StartLocal: %v", err)
	}
	if cs.Cmd == nil {
		t.Fatal("expected Cmd to be set")
	}

	// On Linux: bin/start. On Windows: cmd.exe /c bin\start.bat
	if runtime.GOOS == "windows" {
		if cs.Cmd.Path == "" || filepath.Base(cs.Cmd.Path) != "cmd.exe" {
			t.Errorf("Windows: expected cmd.exe, got %s", cs.Cmd.Path)
		}
	} else {
		wantScript := filepath.Join(pad.Dir, "bin", "start")
		if cs.Cmd.Path != wantScript {
			t.Errorf("got path %s, want %s", cs.Cmd.Path, wantScript)
		}
	}
}

func TestStartLocal_MissingPAD_ReturnsError(t *testing.T) {
	cs := &CaptureStarter{}
	err := docker.StartLocal(docker.LocalRunOptions{
		PadDir:  t.TempDir(), // empty dir, no PAD layout
		Starter: cs,
	})
	if err == nil {
		t.Fatal("expected error for missing PAD")
	}
	if cs.Cmd != nil {
		t.Fatal("expected no command to be started")
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
CGO_ENABLED=0 go test ./cmd/mxcli/docker/... -run "TestStartLocal" -v 2>&1 | head -20
```

Expected: `FAIL — docker.StartLocal undefined`.

- [ ] **Step 3: Implement ProcessStarter and skeleton StartLocal**

```go
// cmd/mxcli/docker/local.go
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ProcessStarter abstracts exec.Cmd execution for testing.
type ProcessStarter interface {
	Run(cmd *exec.Cmd) error
}

// RealStarter executes the command for real (used in production).
type RealStarter struct{}

func (r *RealStarter) Run(cmd *exec.Cmd) error { return cmd.Run() }

// LocalRunOptions configures StartLocal.
type LocalRunOptions struct {
	// PadDir is the PAD output directory (.docker/build/ by default).
	PadDir string
	// DB is an optional postgres:// URL. Empty = use PAD defaults (HSQLDB).
	DB string
	// Stdout for runtime log output (defaults to os.Stdout).
	Stdout io.Writer
	// Stderr for runtime error output (defaults to os.Stderr).
	Stderr io.Writer
	// Starter is the process runner. Nil = RealStarter (exec.Cmd.Run).
	Starter ProcessStarter
}

// StartLocal starts the Mendix runtime from a pre-built PAD directory without Docker.
// It execs bin/start (Linux/macOS) or cmd.exe /c bin\start.bat (Windows),
// blocking until the process exits (Ctrl+C stops it).
func StartLocal(opts LocalRunOptions) error {
	if !hasExtractedPADLayout(opts.PadDir) {
		return fmt.Errorf("no PAD found at %s — run 'mxcli local build -p app.mpr' first", opts.PadDir)
	}

	cmdArgs, err := resolveStartScript(opts.PadDir)
	if err != nil {
		return err
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = opts.PadDir
	cmd.Env = append(os.Environ(), buildLocalEnv(opts.DB)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	starter := opts.Starter
	if starter == nil {
		starter = &RealStarter{}
	}
	return starter.Run(cmd)
}

// resolveStartScript returns [binary, args...] for the platform start script.
func resolveStartScript(padDir string) ([]string, error) {
	switch runtime.GOOS {
	case "windows":
		bat := filepath.Join(padDir, "bin", "start.bat")
		if _, err := os.Stat(bat); err == nil {
			return []string{"cmd.exe", "/c", bat}, nil
		}
		ps1 := filepath.Join(padDir, "bin", "start.ps1")
		if _, err := os.Stat(ps1); err == nil {
			return []string{"powershell.exe", "-ExecutionPolicy", "Bypass", "-File", ps1}, nil
		}
		return nil, fmt.Errorf("no Windows start script (start.bat or start.ps1) found in %s/bin/", padDir)
	default:
		sh := filepath.Join(padDir, "bin", "start")
		if _, err := os.Stat(sh); err != nil {
			return nil, fmt.Errorf("start script not found at %s", sh)
		}
		return []string{sh}, nil
	}
}

// buildLocalEnv returns environment variables required by the Mendix runtime.
// ADMIN_ADMINPASSWORD: M2EE admin API auth (required by runtimelauncher).
// RUNTIME_ADMINUSER_PASSWORD: creates/updates MxAdmin login user on startup.
func buildLocalEnv(dbURL string) []string {
	env := []string{
		"ADMIN_ADMINPASSWORD=Admin123!",
		"RUNTIME_ADMINUSER_PASSWORD=Admin123!",
	}
	if dbURL != "" {
		if dbEnv, err := parseDBURL(dbURL); err == nil {
			env = append(env, dbEnv...)
		}
	}
	return env
}

// parseDBURL converts a postgres:// URL into RUNTIME_PARAMS_* env vars
// consumed by etc/variables.conf inside the PAD.
func parseDBURL(rawURL string) ([]string, error) {
	// Implemented in Task 3.
	return nil, fmt.Errorf("parseDBURL: not implemented")
}
```

- [ ] **Step 4: Run tests — both should pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/docker/... -run "TestStartLocal" -v 2>&1 | tail -10
```

Expected:
```
--- PASS: TestStartLocal_ExecsCorrectScript
--- PASS: TestStartLocal_MissingPAD_ReturnsError
PASS
```

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/docker/local.go cmd/mxcli/docker/local_test.go
git commit -m "feat(local): add StartLocal + ProcessStarter interface"
```

---

## Task 3 — parseDBURL + env injection tests

**Files:**
- Modify: `cmd/mxcli/docker/local.go` (implement `parseDBURL`)
- Modify: `cmd/mxcli/docker/local_test.go` (add tests)

- [ ] **Step 1: Write the failing tests**

Add to `cmd/mxcli/docker/local_test.go`:

```go
func TestParseDBURL_Postgres(t *testing.T) {
	env, err := docker.ParseDBURL("postgres://alice:s3cr3t@db.local:5433/myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]string{
		"RUNTIME_PARAMS_DATABASETYPE":     "PostgreSQL",
		"RUNTIME_PARAMS_DATABASEJDBCURL":  "jdbc:postgresql://db.local:5433/myapp",
		"RUNTIME_PARAMS_DATABASEUSERNAME": "alice",
		"RUNTIME_PARAMS_DATABASEPASSWORD": "s3cr3t",
	}
	got := make(map[string]string)
	for _, kv := range env {
		parts := strings.SplitN(kv, "=", 2)
		got[parts[0]] = parts[1]
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
}

func TestParseDBURL_InvalidScheme(t *testing.T) {
	_, err := docker.ParseDBURL("mysql://user:pass@host/db")
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestStartLocal_InjectsDBEnv(t *testing.T) {
	pad := testfixtures.NewFakePAD(t)
	cs := &CaptureStarter{}

	err := docker.StartLocal(docker.LocalRunOptions{
		PadDir:  pad.Dir,
		DB:      "postgres://bob:pw@localhost:5432/mendix",
		Starter: cs,
	})
	if err != nil {
		t.Fatalf("StartLocal: %v", err)
	}

	envMap := make(map[string]string)
	for _, kv := range cs.Cmd.Env {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if envMap["RUNTIME_PARAMS_DATABASETYPE"] != "PostgreSQL" {
		t.Errorf("DATABASETYPE: got %q, want PostgreSQL", envMap["RUNTIME_PARAMS_DATABASETYPE"])
	}
	if envMap["RUNTIME_PARAMS_DATABASEUSERNAME"] != "bob" {
		t.Errorf("USERNAME: got %q, want bob", envMap["RUNTIME_PARAMS_DATABASEUSERNAME"])
	}
}
```

Add `"strings"` to the test file imports.

Also export `ParseDBURL` (capital P) so tests can call it:

- [ ] **Step 2: Run — verify tests fail**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/docker/... -run "TestParseDBURL|TestStartLocal_InjectsDB" -v 2>&1 | tail -10
```

Expected: `FAIL — ParseDBURL undefined` or `not implemented` error.

- [ ] **Step 3: Implement parseDBURL (and export as ParseDBURL)**

Replace the `parseDBURL` stub and add `ParseDBURL` in `cmd/mxcli/docker/local.go`:

```go
import (
	// add to existing imports:
	"net/url"
	"strings"
)

// ParseDBURL converts a postgres:// connection URL to RUNTIME_PARAMS_* env vars.
// Only postgresql/postgres schemes are supported.
func ParseDBURL(rawURL string) ([]string, error) {
	return parseDBURL(rawURL)
}

func parseDBURL(rawURL string) ([]string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DB URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "postgres" && scheme != "postgresql" {
		return nil, fmt.Errorf("unsupported DB scheme %q (only postgres:// is supported)", u.Scheme)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	username := u.User.Username()
	password, _ := u.User.Password()

	jdbcURL := fmt.Sprintf("jdbc:postgresql://%s:%s/%s", host, port, dbName)

	return []string{
		"RUNTIME_PARAMS_DATABASETYPE=PostgreSQL",
		"RUNTIME_PARAMS_DATABASEJDBCURL=" + jdbcURL,
		"RUNTIME_PARAMS_DATABASEUSERNAME=" + username,
		"RUNTIME_PARAMS_DATABASEPASSWORD=" + password,
	}, nil
}
```

- [ ] **Step 4: Run tests — all should pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/docker/... -run "TestParseDBURL|TestStartLocal" -v 2>&1 | tail -15
```

Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/docker/local.go cmd/mxcli/docker/local_test.go
git commit -m "feat(local): implement parseDBURL — postgres:// → RUNTIME_PARAMS_* env vars"
```

---

## Task 4 — `mxcli-local` binary

**Files:**
- Create: `cmd/mxcli-local/main.go`
- Create: `cmd/mxcli-local/cmd_build.go`
- Create: `cmd/mxcli-local/cmd_run.go`

- [ ] **Step 1: Create `main.go`**

```go
// cmd/mxcli-local/main.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

func main() {
	root := &cobra.Command{
		Use:          "mxcli-local",
		Short:        "Build and run Mendix apps without Docker",
		Version:      Version,
		SilenceUsage: true,
	}
	root.AddCommand(buildCmd(), runCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Create `cmd_build.go`**

```go
// cmd/mxcli-local/cmd_build.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

func buildCmd() *cobra.Command {
	var (
		projectPath       string
		skipCheck         bool
		skipUpdateWidgets bool
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a PAD package from an MPR file (no Docker required)",
		Example: `  mxcli-local build -p app.mpr
  mxcli-local build -p app.mpr --skip-check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return docker.Build(docker.BuildOptions{
				ProjectPath:       projectPath,
				SkipCheck:         skipCheck,
				SkipUpdateWidgets: skipUpdateWidgets,
				Stdout:            os.Stdout,
			})
		},
	}

	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to .mpr file (required)")
	cmd.Flags().BoolVar(&skipCheck, "skip-check", false, "Skip mx check before building")
	cmd.Flags().BoolVar(&skipUpdateWidgets, "skip-update-widgets", false, "Skip widget update step")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}
```

- [ ] **Step 3: Create `cmd_run.go`**

```go
// cmd/mxcli-local/cmd_run.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var (
		projectPath string
		dbURL       string
		padDir      string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the Mendix runtime from a pre-built PAD (no Docker required)",
		Long: `Run the Mendix runtime using the PAD built by 'mxcli local build'.
Blocks in the foreground — press Ctrl+C to stop.

Default database: HSQLDB (embedded, no external database needed).
Override with --db for PostgreSQL.`,
		Example: `  mxcli-local run -p app.mpr
  mxcli-local run -p app.mpr --db postgres://user:pass@localhost/mendix`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := padDir
			if dir == "" {
				if projectPath == "" {
					return fmt.Errorf("either -p or --pad-dir is required")
				}
				dir = filepath.Join(filepath.Dir(projectPath), ".docker", "build")
			}
			return docker.StartLocal(docker.LocalRunOptions{
				PadDir: dir,
				DB:     dbURL,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			})
		},
	}

	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to .mpr file (derives PAD dir as .docker/build/)")
	cmd.Flags().StringVar(&dbURL, "db", "", "Database URL (postgres://user:pass@host/db). Default: HSQLDB (embedded)")
	cmd.Flags().StringVar(&padDir, "pad-dir", "", "Explicit PAD directory (overrides -p)")
	return cmd
}
```

- [ ] **Step 4: Verify the binary compiles**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
CGO_ENABLED=0 go build -o bin/mxcli-local ./cmd/mxcli-local
```

Expected: `bin/mxcli-local` created, exits 0.

- [ ] **Step 5: Smoke-test the binary**

```bash
./bin/mxcli-local --help
./bin/mxcli-local build --help
./bin/mxcli-local run --help
```

Expected: help text printed for each, exits 0.

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli-local/
git commit -m "feat(local): add mxcli-local binary with build + run subcommands"
```

---

## Task 5 — Makefile targets for mxcli-local

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add LOCAL_NAME variable and build-local target**

Add after the `DAEMON_NAME = mxcli-daemon` line:

```makefile
LOCAL_NAME = mxcli-local
LOCAL_PATH = ./cmd/mxcli-local
LOCAL_LDFLAGS = -ldflags "-X main.Version=$(VERSION) -s -w"
```

Add after the existing `build:` target block:

```makefile
build-local:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LOCAL_LDFLAGS) -o $(BUILD_DIR)/$(LOCAL_NAME) $(LOCAL_PATH)
	@echo "Built $(BUILD_DIR)/$(LOCAL_NAME)"

install-local: build-local
	@LOCAL_DIR="$$HOME/.mxcli/local"; \
	mkdir -p "$$LOCAL_DIR"; \
	cp "$(BUILD_DIR)/$(LOCAL_NAME)" "$$LOCAL_DIR/$(LOCAL_NAME)"; \
	echo "Installed: $$LOCAL_DIR/$(LOCAL_NAME)"
```

Also add `build-local` and `install-local` to the `.PHONY` line.

- [ ] **Step 2: Verify make targets work**

```bash
make build-local
ls -la bin/mxcli-local
make install-local
ls -la ~/.mxcli/local/mxcli-local
```

Expected: binary present in both locations.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add build-local and install-local Makefile targets"
```

---

## Task 6 — ComponentPayload fixture (launcher test infrastructure)

**Files:**
- Create: `cmd/mxcli-launcher/testfixtures/component_payload.go`
- Modify: `cmd/mxcli-launcher/testfixtures/fake_daemon.go`

- [ ] **Step 1: Create `component_payload.go`**

```go
// cmd/mxcli-launcher/testfixtures/component_payload.go
// SPDX-License-Identifier: Apache-2.0

package testfixtures

import (
	"fmt"
)

// ComponentPayload generalises DaemonPayload to any binary component
// (mxcli-daemon, mxcli-local, mxcli launcher).
type ComponentPayload struct {
	AssetName string
	Archive   []byte
	Checksum  string
}

// BuildComponentPayload builds a fake archive for the given component and platform.
// component is the binary name without extension, e.g. "mxcli-local" or "mxcli-daemon".
// Windows → .exe.zip; Linux/Darwin → .tar.zst.
func BuildComponentPayload(component, goos, goarch string, content []byte) (*ComponentPayload, error) {
	dp, err := BuildDaemonPayloadForPlatformNamed(component, goos, goarch, content)
	if err != nil {
		return nil, err
	}
	return &ComponentPayload{
		AssetName: dp.AssetName,
		Archive:   dp.Archive,
		Checksum:  dp.Checksum,
	}, nil
}

// ToComponentPayload converts a DaemonPayload to ComponentPayload for compatibility.
func (d *DaemonPayload) ToComponentPayload() *ComponentPayload {
	return &ComponentPayload{
		AssetName: d.AssetName,
		Archive:   d.Archive,
		Checksum:  d.Checksum,
	}
}

// BuildLocalPayload is a convenience wrapper for mxcli-local payloads.
func BuildLocalPayload(goos, goarch string, content []byte) (*ComponentPayload, error) {
	return BuildComponentPayload("mxcli-local", goos, goarch, content)
}

// localAssetName returns the release asset name for mxcli-local.
func LocalAssetName(goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("mxcli-local-%s-%s.exe.zip", goos, goarch)
	}
	return fmt.Sprintf("mxcli-local-%s-%s.tar.zst", goos, goarch)
}
```

- [ ] **Step 2: Add `BuildDaemonPayloadForPlatformNamed` to `fake_daemon.go`**

In `cmd/mxcli-launcher/testfixtures/fake_daemon.go`, add after `BuildDaemonPayloadForPlatform`:

```go
// BuildDaemonPayloadForPlatformNamed is like BuildDaemonPayloadForPlatform but
// uses an explicit component name instead of "mxcli-daemon".
func BuildDaemonPayloadForPlatformNamed(component, goos, goarch string, content []byte) (*DaemonPayload, error) {
	if goos == "windows" {
		return buildWindowsPayloadNamed(component, goos, goarch, content)
	}
	return buildUnixPayloadNamed(component, goos, goarch, content)
}
```

Then add the two named helpers (copy `buildWindowsPayload`/`buildUnixPayload`, replace `"mxcli-daemon"` with the `component` parameter):

```go
func buildWindowsPayloadNamed(component, goos, goarch string, content []byte) (*DaemonPayload, error) {
	assetName := fmt.Sprintf("%s-%s-%s.exe.zip", component, goos, goarch)
	binaryName := component + ".exe"

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(binaryName)
	if err != nil {
		return nil, fmt.Errorf("zip create: %w", err)
	}
	if _, err := f.Write(content); err != nil {
		return nil, fmt.Errorf("zip write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("zip close: %w", err)
	}
	archiveBytes := buf.Bytes()
	h := sha256.Sum256(archiveBytes)
	return &DaemonPayload{AssetName: assetName, Archive: archiveBytes, Checksum: hex.EncodeToString(h[:])}, nil
}

func buildUnixPayloadNamed(component, goos, goarch string, content []byte) (*DaemonPayload, error) {
	assetName := fmt.Sprintf("%s-%s-%s.tar.zst", component, goos, goarch)
	binaryName := component

	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return nil, fmt.Errorf("zstd writer: %w", err)
	}
	tw := tar.NewWriter(zw)
	hdr := &tar.Header{Name: binaryName, Typeflag: tar.TypeReg, Size: int64(len(content)), Mode: 0755}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}
	tw.Close()
	zw.Close()
	archiveBytes := buf.Bytes()
	h := sha256.Sum256(archiveBytes)
	return &DaemonPayload{AssetName: assetName, Archive: archiveBytes, Checksum: hex.EncodeToString(h[:])}, nil
}
```

- [ ] **Step 3: Verify everything compiles and existing tests still pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -v 2>&1 | tail -20
```

Expected: all existing tests PASS, no new failures.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli-launcher/testfixtures/
git commit -m "test(local): add ComponentPayload generalising DaemonPayload for multi-component downloads"
```

---

## Task 7 — Launcher paths for mxcli-local

**Files:**
- Modify: `cmd/mxcli-launcher/paths.go`

- [ ] **Step 1: Write the failing test**

```go
// cmd/mxcli-launcher/local_test.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLocalDir(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	want := filepath.Join(e.HomeDir, ".mxcli", "local")
	if got := e.localDir(); got != want {
		t.Errorf("localDir: got %s, want %s", got, want)
	}
}

func TestLocalBinaryPath(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	p := e.localBinaryPath()
	base := filepath.Base(p)
	if runtime.GOOS == "windows" {
		if base != "mxcli-local.exe" {
			t.Errorf("Windows: got %s, want mxcli-local.exe", base)
		}
	} else {
		if base != "mxcli-local" {
			t.Errorf("non-Windows: got %s, want mxcli-local", base)
		}
	}
}
```

- [ ] **Step 2: Run — verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestLocalDir|TestLocalBinaryPath" -v 2>&1 | tail -10
```

Expected: `FAIL — localDir undefined`.

- [ ] **Step 3: Add paths to `paths.go`**

Append to `cmd/mxcli-launcher/paths.go`:

```go
func (e *Env) localDir() string { return filepath.Join(e.HomeDir, ".mxcli", "local") }

func (e *Env) localBinaryPath() string {
	name := "mxcli-local"
	if runtime.GOOS == "windows" {
		name = "mxcli-local.exe"
	}
	return filepath.Join(e.localDir(), name)
}

func (e *Env) localBinaryBakPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(e.localDir(), "mxcli-local.bak.exe")
	}
	return filepath.Join(e.localDir(), "mxcli-local.bak")
}

func (e *Env) localVersionPath() string    { return filepath.Join(e.localDir(), "version") }
func (e *Env) localLastCheckPath() string  { return filepath.Join(e.localDir(), "last-check") }
func (e *Env) localUpdateAvailablePath() string {
	return filepath.Join(e.localDir(), "update-available")
}
```

- [ ] **Step 4: Run tests — should pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestLocalDir|TestLocalBinaryPath" -v 2>&1 | tail -10
```

Expected: 2 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli-launcher/paths.go cmd/mxcli-launcher/local_test.go
git commit -m "feat(local): add local binary path helpers to launcher"
```

---

## Task 8 — ensureLocalBinary + downloadLocal

**Files:**
- Create: `cmd/mxcli-launcher/local.go`
- Modify: `cmd/mxcli-launcher/local_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `cmd/mxcli-launcher/local_test.go`:

```go
import (
	"runtime"
	"testing"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

func newLocalInstallEnv(t *testing.T, tag string, content []byte) (*Env, *testfixtures.FakeGitHub) {
	t.Helper()
	payload, err := testfixtures.BuildLocalPayload(runtime.GOOS, runtime.GOARCH, content)
	if err != nil {
		t.Fatalf("BuildLocalPayload: %v", err)
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

func TestEnsureLocalBinary_FreshInstall(t *testing.T) {
	e, _ := newLocalInstallEnv(t, "local-v0.1.0", []byte("local-binary-content"))

	if err := e.ensureLocalBinary(); err != nil {
		t.Fatalf("ensureLocalBinary: %v", err)
	}

	content, err := os.ReadFile(e.localBinaryPath())
	if err != nil {
		t.Fatalf("binary not installed: %v", err)
	}
	if string(content) != "local-binary-content" {
		t.Errorf("binary content mismatch: got %q", content)
	}
}

func TestEnsureLocalBinary_AlreadyInstalled(t *testing.T) {
	e, gh := newLocalInstallEnv(t, "local-v0.1.0", []byte("binary"))

	// Pre-install
	if err := e.ensureLocalBinary(); err != nil {
		t.Fatalf("first install: %v", err)
	}
	initialRequests := len(gh.RequestLog())

	// Second call should not download again
	if err := e.ensureLocalBinary(); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(gh.RequestLog()) != initialRequests {
		t.Errorf("expected no new requests, got %d new", len(gh.RequestLog())-initialRequests)
	}
}
```

- [ ] **Step 2: Run — verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestEnsureLocalBinary" -v 2>&1 | tail -10
```

Expected: `FAIL — ensureLocalBinary undefined`.

- [ ] **Step 3: Create `local.go`**

```go
// cmd/mxcli-launcher/local.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

const localRepo = "engalar/mxcli"

// runLocal delegates all `mxcli local *` subcommands to mxcli-local.
// It ensures the binary is installed, then execs it with inherited stdio.
func (e *Env) runLocal(args []string) int {
	if err := e.ensureLocalBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli local: %v\n", err)
		return 1
	}
	cmd := exec.Command(e.localBinaryPath(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "mxcli local: %v\n", err)
		return 1
	}
	return 0
}

// ensureLocalBinary ensures ~/.mxcli/local/mxcli-local is present.
// Downloads the latest local-v* release if missing.
func (e *Env) ensureLocalBinary() error {
	if err := os.MkdirAll(e.localDir(), 0700); err != nil {
		return fmt.Errorf("create local dir: %w", err)
	}
	if localBinaryExists(e.localBinaryPath()) {
		return nil
	}
	fmt.Fprintln(os.Stderr, "mxcli: mxcli-local not found, downloading latest version...")
	return e.downloadLocal(e.localBinaryPath())
}

func localBinaryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// downloadLocal fetches the latest local-v* release and installs it.
func (e *Env) downloadLocal(destPath string) error {
	tag, err := e.fetchLatestLocalTag()
	if err != nil {
		return err
	}
	return e.downloadLocalVersion(tag, destPath)
}

func (e *Env) fetchLatestLocalTag() (string, error) {
	return e.fetchLatestTagWithPrefix("local-v")
}

// fetchLatestTagWithPrefix finds the latest release whose tag starts with prefix.
// Uses /releases?per_page=20 because /releases/latest returns the global latest
// regardless of tag prefix.
func (e *Env) fetchLatestTagWithPrefix(prefix string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20", localRepo)
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

// downloadLocalVersion downloads and extracts mxcli-local for the current platform.
func (e *Env) downloadLocalVersion(tag, destPath string) error {
	return e.downloadLocalVersionForPlatform(tag, destPath, runtime.GOOS, runtime.GOARCH)
}

func (e *Env) downloadLocalVersionForPlatform(tag, destPath, goos, goarch string) error {
	var archiveExt string
	if goos == "windows" {
		archiveExt = ".exe.zip"
	} else {
		archiveExt = ".tar.zst"
	}
	assetName := fmt.Sprintf("mxcli-local-%s-%s%s", goos, goarch, archiveExt)

	expectedHash, err := e.fetchAssetChecksumFromTag(tag, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", localRepo, tag, assetName)
	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)

	return e.downloadAndExtractComponent(url, expectedHash, destPath, goos, "mxcli-local")
}
```

- [ ] **Step 4: Add the helper functions** that `downloadLocalVersion` needs.

Add to `local.go` (these reuse the patterns from `daemon.go`):

```go
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func parseLatestTagWithPrefix(body io.Reader, prefix string) (string, error) {
	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(body).Decode(&releases); err != nil {
		return "", fmt.Errorf("parse releases JSON: %w", err)
	}
	for _, r := range releases {
		if strings.HasPrefix(r.TagName, prefix) {
			return r.TagName, nil
		}
	}
	return "", fmt.Errorf("no release found with tag prefix %q", prefix)
}

// fetchAssetChecksumFromTag fetches SHA256SUMS from a specific release tag.
func (e *Env) fetchAssetChecksumFromTag(tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/SHA256SUMS", localRepo, tag)
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

// downloadAndExtractComponent downloads an archive, verifies its checksum,
// and extracts the named binary to destPath.
func (e *Env) downloadAndExtractComponent(url, expectedHash, destPath, goos, binaryName string) error {
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	archiveTmp := destPath + ".archive-dl"
	defer os.Remove(archiveTmp)

	h := sha256.New()
	tee := io.TeeReader(resp.Body, h)
	af, err := os.OpenFile(archiveTmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	if _, err := io.Copy(af, tee); err != nil {
		af.Close()
		return err
	}
	af.Close()

	actualHash := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("checksum mismatch: expected %s got %s", expectedHash, actualHash)
	}

	if goos == "windows" {
		return extractZip(archiveTmp, destPath, binaryName+".exe")
	}
	ar, err := os.Open(archiveTmp)
	if err != nil {
		return err
	}
	defer ar.Close()
	return extractTarZst(ar, destPath, binaryName)
}
```

- [ ] **Step 5: Update `FakeGitHub` to serve `/releases` list endpoint**

Add to `fake_github.go`'s `handle()` switch, before the default case:

```go
case strings.Contains(path, "/releases") && !strings.Contains(path, "/releases/"):
    // Serve the releases list endpoint used by fetchLatestTagWithPrefix.
    // Returns a JSON array with the single configured release.
    fmt.Fprintf(w, `[{"tag_name":%q}]`, f.LatestTag)
```

- [ ] **Step 6: Run tests — should pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestEnsureLocalBinary|TestLocalDir|TestLocalBinaryPath" -v 2>&1 | tail -15
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/mxcli-launcher/local.go cmd/mxcli-launcher/local_test.go \
        cmd/mxcli-launcher/testfixtures/fake_github.go
git commit -m "feat(local): add ensureLocalBinary + download from local-v* releases"
```

---

## Task 9 — Wire `local` routing in launcher main.go

**Files:**
- Modify: `cmd/mxcli-launcher/main.go`

- [ ] **Step 1: Write the routing test**

Add to `cmd/mxcli-launcher/local_test.go`:

```go
func TestRunLocal_DelegatesArgs(t *testing.T) {
	// Build a fake mxcli-local that writes its args to a temp file and exits 0.
	argFile := filepath.Join(t.TempDir(), "args.txt")
	fakeBin := buildFakeLocalBinary(t, argFile)

	e := &Env{HomeDir: t.TempDir(), HTTPClient: http.DefaultClient}
	// Pre-install the fake binary so ensureLocalBinary doesn't download.
	if err := os.MkdirAll(e.localDir(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(fakeBin, e.localBinaryPath()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(e.localBinaryPath(), 0755); err != nil {
		t.Fatal(err)
	}

	code := e.runLocal([]string{"build", "-p", "app.mpr"})
	if code != 0 {
		t.Fatalf("runLocal exit code: %d", code)
	}

	got, _ := os.ReadFile(argFile)
	if !strings.Contains(string(got), "build") {
		t.Errorf("args file: got %q, want 'build'", got)
	}
}

// buildFakeLocalBinary compiles a tiny Go program that writes os.Args to argFile and exits 0.
func buildFakeLocalBinary(t *testing.T, argFile string) string {
	t.Helper()
	src := fmt.Sprintf(`package main
import (
	"fmt"
	"os"
	"strings"
)
func main() {
	os.WriteFile(%q, []byte(strings.Join(os.Args[1:], " ")), 0644)
	fmt.Println("fake mxcli-local ok")
}
`, argFile)
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(dir, "fake-local")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binPath, srcPath).CombinedOutput(); err != nil {
		t.Fatalf("build fake binary: %v\n%s", err, out)
	}
	return binPath
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}
```

- [ ] **Step 2: Run — verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestRunLocal_DelegatesArgs" -v 2>&1 | tail -10
```

Expected: test fails (runLocal exists but routing in main not wired yet, so `runLocal` itself can pass — this test verifies the delegation logic in `runLocal`, not the main routing).

- [ ] **Step 3: Add `local` case to `main.go`**

In `cmd/mxcli-launcher/main.go`, add before the `ensureDaemonBinary` call:

```go
// Local commands are delegated directly to mxcli-local binary.
// They bypass the daemon — no daemon needed for PAD build/run.
if len(args) > 0 && args[0] == "local" {
    os.Exit(e.runLocal(args[1:]))
}
```

- [ ] **Step 4: Verify full build still works**

```bash
CGO_ENABLED=0 go build ./cmd/mxcli-launcher
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... 2>&1 | tail -10
```

Expected: build succeeds, all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli-launcher/main.go cmd/mxcli-launcher/local_test.go
git commit -m "feat(local): wire mxcli local subcommand routing in launcher"
```

---

## Task 10 — Integration smoke test (end-to-end, local build)

**Files:**
- No new files — uses existing testdata and the installed binaries

This task verifies the full local workflow against a real MPR file using the locally installed `mxcli-local`.

- [ ] **Step 1: Install mxcli-local locally**

```bash
make install-local
~/.mxcli/local/mxcli-local --version
```

Expected: prints version string, exits 0.

- [ ] **Step 2: Build a PAD from testdata without Docker**

```bash
~/.mxcli/local/mxcli-local build \
  -p testdata/helpdesk-clean-11.6.6/app.mpr \
  --skip-check
```

Expected: output ends with `Build complete.`, `.docker/build/` created under `testdata/helpdesk-clean-11.6.6/`.

- [ ] **Step 3: Verify PAD structure**

```bash
ls testdata/helpdesk-clean-11.6.6/.docker/build/bin/
ls testdata/helpdesk-clean-11.6.6/.docker/build/lib/runtime/launcher/
```

Expected:
```
start  start.bat  start.ps1
runtimelauncher.jar
```

- [ ] **Step 4: Verify `mxcli local run` flag help works**

```bash
~/.mxcli/local/mxcli-local run --help
```

Expected: help text with `--db` and `--pad-dir` flags, exits 0.

- [ ] **Step 5: Clean up testdata PAD**

```bash
rm -rf testdata/helpdesk-clean-11.6.6/.docker/
```

- [ ] **Step 6: Run full test suite**

```bash
make test 2>&1 | tail -5
```

Expected: `ok` for all packages, no `FAIL`.

- [ ] **Step 7: Final commit**

```bash
git commit --allow-empty -m "chore: mxcli local runner Plan 1 complete — build+run without Docker"
```

---

## Self-Review

### Spec coverage check

| Spec section | Covered by |
|---|---|
| `mxcli local build` command | Task 4 `cmd_build.go` |
| `mxcli local run` command | Task 4 `cmd_run.go` |
| `StartLocal()` — exec platform script | Task 2 `local.go` |
| Windows: `bin\start.bat` | Task 2 `resolveStartScript()` |
| Linux: `bin/start` | Task 2 `resolveStartScript()` |
| HSQLDB default (no --db required) | Task 2 `buildLocalEnv()`, Task 3 `cmd_run.go` |
| `--db postgres://` injection | Task 3 `parseDBURL()` |
| `ADMIN_ADMINPASSWORD` required field | Task 2 `buildLocalEnv()` |
| `RUNTIME_ADMINUSER_PASSWORD` for MxAdmin | Task 2 `buildLocalEnv()` |
| Launcher routes `mxcli local` → `mxcli-local` | Task 9 |
| Download `mxcli-local` from `local-v*` release | Task 8 |
| Tag-prefix filtering (`local-v` ≠ `daemon-v`) | Task 8 `fetchLatestTagWithPrefix()` |
| FakePAD fixture | Task 1 |
| ProcessStarter + CaptureStarter | Task 2 |
| ComponentPayload generalization | Task 6 |
| Makefile `build-local` + `install-local` | Task 5 |
| Integration smoke test | Task 10 |

**Out of scope for Plan 1 (Plan 2):** release pipeline split, `mxcli daemon upgrade`, `mxcli local upgrade`, self-fork updater, `MultiReleaseFakeGitHub`, `PIDWaiter`.

### Type consistency check

- `LocalRunOptions.Starter` → `ProcessStarter` interface → used in Task 2, Task 3 tests ✓
- `docker.Build()` signature unchanged — Task 4 calls it directly ✓
- `e.localBinaryPath()` defined Task 7, used Task 8 + Task 9 ✓
- `fetchLatestTagWithPrefix()` defined Task 8, called by `fetchLatestLocalTag()` ✓
- `parseChecksumFile()` used in Task 8 — already defined in `daemon.go` ✓
- `extractZip()` + `extractTarZst()` used in Task 8 — already defined in `daemon.go` ✓
- `FakeGitHub.Payload` field type is `*testfixtures.DaemonPayload` — Task 8 test constructs one from `ComponentPayload` fields ✓
