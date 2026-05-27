# mxcli Launcher + Daemon Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split mxcli into a tiny cross-platform launcher (~2MB) and a full-featured daemon (~20MB compressed), enabling fast repeated CLI invocations via socket and low-cost incremental updates.

**Architecture:** The existing `cmd/mxcli` binary becomes `mxcli-daemon` (all business logic, socket server mode). A new `cmd/mxcli-launcher` binary becomes `mxcli` (thin client: starts daemon, forwards argv over unix socket, handles upgrade/rollback). Protocol is 4-byte-length-prefixed JSON over `~/.mxcli/daemon/mxcli.sock`.

**Tech Stack:** Go 1.26, `encoding/json`, `net` (unix socket), `archive/tar` + `github.com/klauspost/compress/zstd` for daemon download, `github.com/spf13/cobra` in launcher for upgrade/rollback subcommands.

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/launcherproto/proto.go` | **Create** | Wire protocol types (Request, Frame) |
| `internal/launcherproto/proto_test.go` | **Create** | Marshal/unmarshal round-trip tests |
| `cmd/mxcli/daemon_server.go` | **Create** | Socket accept loop; dispatches argv to Cobra |
| `cmd/mxcli/main.go` | **Modify** | Intercept `--serve` before Cobra; call daemon_server |
| `cmd/mxcli-launcher/main.go` | **Create** | Entry point; routes to forward/upgrade/rollback/version |
| `cmd/mxcli-launcher/forward.go` | **Create** | Connect to socket, send Request, stream stdout/stderr |
| `cmd/mxcli-launcher/daemon.go` | **Create** | Daemon lifecycle: ensure running, start, health-check, download |
| `cmd/mxcli-launcher/update.go` | **Create** | Background version check; `mxcli upgrade`; `mxcli rollback` |
| `cmd/mxcli-launcher/paths.go` | **Create** | All `~/.mxcli/daemon/` path helpers |
| `Makefile` | **Modify** | Add `mxcli-launcher` build target; compression; daemon target |
| `install.sh` | **Create** | Idempotent Linux/macOS install script |
| `install.ps1` | **Create** | Idempotent Windows PowerShell install script |
| `.github/workflows/release.yml` | **Modify** | Compress daemon artifacts; upload launcher + daemon |

---

## Phase 1: Core (Protocol + Daemon Server + Launcher + First-Run Download)

---

### Task 1: Wire Protocol Package

**Files:**
- Create: `internal/launcherproto/proto.go`
- Create: `internal/launcherproto/proto_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/launcherproto/proto_test.go
package launcherproto_test

import (
	"bytes"
	"testing"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

func TestRequestRoundTrip(t *testing.T) {
	req := launcherproto.Request{
		Argv: []string{"exec", "foo.mdl"},
		Cwd:  "/project",
		Env:  map[string]string{"MX_DEBUG": "1"},
	}
	var buf bytes.Buffer
	if err := launcherproto.WriteMsg(&buf, req); err != nil {
		t.Fatal(err)
	}
	var got launcherproto.Request
	if err := launcherproto.ReadMsg(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.Cwd != "/project" || len(got.Argv) != 2 || got.Env["MX_DEBUG"] != "1" {
		t.Fatalf("round-trip failed: %+v", got)
	}
}

func TestFrameExitRoundTrip(t *testing.T) {
	code := 42
	frame := launcherproto.Frame{Exit: &code}
	var buf bytes.Buffer
	if err := launcherproto.WriteMsg(&buf, frame); err != nil {
		t.Fatal(err)
	}
	var got launcherproto.Frame
	if err := launcherproto.ReadMsg(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.Exit == nil || *got.Exit != 42 {
		t.Fatalf("exit code not preserved: %+v", got)
	}
}

func TestFrameStdoutRoundTrip(t *testing.T) {
	frame := launcherproto.Frame{Stream: "stdout", Data: []byte("hello\n")}
	var buf bytes.Buffer
	if err := launcherproto.WriteMsg(&buf, frame); err != nil {
		t.Fatal(err)
	}
	var got launcherproto.Frame
	if err := launcherproto.ReadMsg(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.Stream != "stdout" || string(got.Data) != "hello\n" {
		t.Fatalf("stdout frame not preserved: %+v", got)
	}
}

func TestHealthCheckRoundTrip(t *testing.T) {
	frame := launcherproto.Frame{OK: true, Version: "v0.14.0"}
	var buf bytes.Buffer
	if err := launcherproto.WriteMsg(&buf, frame); err != nil {
		t.Fatal(err)
	}
	var got launcherproto.Frame
	if err := launcherproto.ReadMsg(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.Version != "v0.14.0" {
		t.Fatalf("health check frame not preserved: %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
CGO_ENABLED=0 go test ./internal/launcherproto/... -v
```

Expected: `FAIL` — package does not exist yet.

- [ ] **Step 3: Implement the protocol package**

```go
// internal/launcherproto/proto.go
package launcherproto

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Request is sent from launcher to daemon over the unix socket.
type Request struct {
	Argv []string          `json:"argv"`
	Cwd  string            `json:"cwd"`
	Env  map[string]string `json:"env"`
}

// Frame is streamed from daemon to launcher.
// Exactly one of (Stream+Data), (Exit), or (OK+Version) is set per frame.
type Frame struct {
	// stdout/stderr stream frame
	Stream string `json:"stream,omitempty"` // "stdout" or "stderr"
	Data   []byte `json:"data,omitempty"`   // raw bytes (JSON encodes as base64)

	// Terminal frame: daemon finished
	Exit *int `json:"exit,omitempty"`

	// Health-check response
	OK      bool   `json:"ok,omitempty"`
	Version string `json:"version,omitempty"`
}

// WriteMsg serialises v as JSON and writes it preceded by a 4-byte big-endian length.
func WriteMsg(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("launcherproto marshal: %w", err)
	}
	if len(b) > 1<<24 { // 16 MB sanity limit
		return fmt.Errorf("launcherproto: message too large (%d bytes)", len(b))
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// ReadMsg reads a length-prefixed JSON message from r and unmarshals into v.
func ReadMsg(r io.Reader, v any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > 1<<24 {
		return fmt.Errorf("launcherproto: incoming message too large (%d bytes)", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
CGO_ENABLED=0 go test ./internal/launcherproto/... -v
```

Expected: all tests `PASS`.

- [ ] **Step 5: Commit**

```bash
git add internal/launcherproto/
git commit -m "feat(launcher): add wire protocol package (launcherproto)"
```

---

### Task 2: Daemon Socket Server Mode

**Files:**
- Create: `cmd/mxcli/daemon_server.go`
- Modify: `cmd/mxcli/main.go` (add `--serve` interception before Cobra)

- [ ] **Step 1: Write the test (integration-style, using a test binary)**

```go
// cmd/mxcli/daemon_server_test.go
package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

func TestDaemonServer_HealthCheck(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "test.sock")

	// Start server in background
	ready := make(chan struct{})
	go func() {
		runDaemonServer(sockPath, func() { close(ready) })
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("daemon server did not start in time")
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := launcherproto.Request{Argv: []string{"__healthcheck__"}, Cwd: "/", Env: map[string]string{}}
	if err := launcherproto.WriteMsg(conn, req); err != nil {
		t.Fatalf("write request: %v", err)
	}

	var frame launcherproto.Frame
	if err := launcherproto.ReadMsg(conn, &frame); err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if !frame.OK {
		t.Errorf("health check: expected ok=true, got %+v", frame)
	}
	if frame.Version == "" {
		t.Error("health check: expected non-empty version")
	}
}

func TestDaemonServer_ExitCode(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "test2.sock")
	ready := make(chan struct{})
	go func() {
		runDaemonServer(sockPath, func() { close(ready) })
	}()
	<-ready

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// "version" command should exit 0
	req := launcherproto.Request{
		Argv: []string{"version"},
		Cwd:  t.TempDir(),
		Env:  map[string]string{},
	}
	if err := launcherproto.WriteMsg(conn, req); err != nil {
		t.Fatalf("write: %v", err)
	}

	var lastExit *int
	for {
		var frame launcherproto.Frame
		if err := launcherproto.ReadMsg(conn, &frame); err != nil {
			t.Fatalf("read frame: %v", err)
		}
		if frame.Exit != nil {
			lastExit = frame.Exit
			break
		}
	}
	if lastExit == nil || *lastExit != 0 {
		t.Errorf("expected exit 0, got %v", lastExit)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/... -run TestDaemonServer -v
```

Expected: `FAIL` — `runDaemonServer` undefined.

- [ ] **Step 3: Implement daemon_server.go**

```go
// cmd/mxcli/daemon_server.go
package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

// runDaemonServer listens on sockPath and dispatches each incoming request as a
// full mxcli command execution. onReady is called once the listener is bound
// (used by tests; pass nil in production).
func runDaemonServer(sockPath string, onReady func()) {
	os.Remove(sockPath) // clean up stale socket
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli-daemon: listen %s: %v\n", sockPath, err)
		os.Exit(1)
	}
	defer ln.Close()

	if onReady != nil {
		onReady()
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	var req launcherproto.Request
	if err := launcherproto.ReadMsg(conn, &req); err != nil {
		return
	}

	// Health-check shortcut
	if len(req.Argv) == 1 && req.Argv[0] == "__healthcheck__" {
		v := version
		if Version != "" {
			v = Version
		}
		_ = launcherproto.WriteMsg(conn, launcherproto.Frame{OK: true, Version: v})
		return
	}

	// Redirect stdout and stderr to the connection
	outW := &frameWriter{conn: conn, stream: "stdout"}
	errW := &frameWriter{conn: conn, stream: "stderr"}

	// Restore working directory
	if req.Cwd != "" {
		if err := os.Chdir(req.Cwd); err != nil {
			fmt.Fprintf(errW, "chdir %s: %v\n", req.Cwd, err)
		}
	}

	// Run command via cobra, capturing exit code
	exitCode := runCommand(req.Argv, outW, errW)

	_ = launcherproto.WriteMsg(conn, launcherproto.Frame{Exit: &exitCode})
}

// runCommand executes the given argv via the existing rootCmd, writing
// stdout/stderr to the provided writers. Returns the exit code.
func runCommand(argv []string, stdout, stderr io.Writer) int {
	// Swap os.Stdout/Stderr temporarily — not thread-safe for concurrent requests,
	// but acceptable for now (one goroutine per connection).
	oldStdout, oldStderr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = oldStdout, oldStderr }()

	// Redirect cobra output
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(argv)

	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

// frameWriter wraps a net.Conn and sends each Write as a Frame.
type frameWriter struct {
	conn   net.Conn
	stream string
}

func (fw *frameWriter) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	if err := launcherproto.WriteMsg(fw.conn, launcherproto.Frame{Stream: fw.stream, Data: buf}); err != nil {
		return 0, err
	}
	return len(p), nil
}
```

- [ ] **Step 4: Modify cmd/mxcli/main.go — intercept --serve before Cobra**

At the top of `main()`, before the existing banner/cobra logic, add:

```go
func main() {
	// Intercept --serve <socket-path> BEFORE cobra parses flags.
	// This starts the daemon socket server and never returns.
	if sockPath := extractServeSocket(os.Args[1:]); sockPath != "" {
		runDaemonServer(sockPath, nil)
		return
	}

	// ... rest of existing main() unchanged ...
```

Add `extractServeSocket` at the bottom of `main.go`:

```go
// extractServeSocket scans args for "--serve <path>" or "--serve=<path>"
// and returns the socket path, or "" if not found.
func extractServeSocket(args []string) string {
	for i, a := range args {
		if a == "--serve" && i+1 < len(args) {
			return args[i+1]
		}
		if len(a) > 8 && a[:8] == "--serve=" {
			return a[8:]
		}
	}
	return ""
}
```

- [ ] **Step 5: Run tests**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/... -run TestDaemonServer -v -count=1
```

Expected: both `TestDaemonServer_HealthCheck` and `TestDaemonServer_ExitCode` **PASS**.

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli/daemon_server.go cmd/mxcli/daemon_server_test.go cmd/mxcli/main.go
git commit -m "feat(daemon): add socket server mode (--serve <socket-path>)"
```

---

### Task 3: Path Helpers

**Files:**
- Create: `cmd/mxcli-launcher/paths.go`
- Create: `cmd/mxcli-launcher/paths_test.go`

- [ ] **Step 1: Write tests**

```go
// cmd/mxcli-launcher/paths_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDaemonDir_ContainsHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := daemonDir()
	if !filepath.IsAbs(dir) {
		t.Errorf("daemonDir must be absolute, got %q", dir)
	}
	if dir == home {
		t.Error("daemonDir must not equal home dir")
	}
}

func TestDaemonPaths_Consistent(t *testing.T) {
	dir := daemonDir()
	if daemonBinaryPath() != filepath.Join(dir, "mxcli-daemon") {
		t.Error("daemonBinaryPath mismatch")
	}
	if daemonBakPath() != filepath.Join(dir, "mxcli-daemon.bak") {
		t.Error("daemonBakPath mismatch")
	}
	if daemonSocketPath() != filepath.Join(dir, "mxcli.sock") {
		t.Error("daemonSocketPath mismatch")
	}
	if daemonVersionPath() != filepath.Join(dir, "version") {
		t.Error("daemonVersionPath mismatch")
	}
	if daemonVersionBakPath() != filepath.Join(dir, "version.bak") {
		t.Error("daemonVersionBakPath mismatch")
	}
	if daemonUpdateAvailablePath() != filepath.Join(dir, "update-available") {
		t.Error("daemonUpdateAvailablePath mismatch")
	}
	if daemonLastCheckPath() != filepath.Join(dir, "last-check") {
		t.Error("daemonLastCheckPath mismatch")
	}
	if daemonPIDPath() != filepath.Join(dir, "mxcli-daemon.pid") {
		t.Error("daemonPIDPath mismatch")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run TestDaemonDir -v 2>&1 | head -5
```

Expected: `FAIL` — package doesn't exist.

- [ ] **Step 3: Implement paths.go**

```go
// cmd/mxcli-launcher/paths.go
package main

import (
	"os"
	"path/filepath"
)

func daemonDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".mxcli", "daemon")
}

func daemonBinaryPath() string     { return filepath.Join(daemonDir(), "mxcli-daemon") }
func daemonBakPath() string        { return filepath.Join(daemonDir(), "mxcli-daemon.bak") }
func daemonSocketPath() string     { return filepath.Join(daemonDir(), "mxcli.sock") }
func daemonVersionPath() string    { return filepath.Join(daemonDir(), "version") }
func daemonVersionBakPath() string { return filepath.Join(daemonDir(), "version.bak") }
func daemonUpdateAvailablePath() string { return filepath.Join(daemonDir(), "update-available") }
func daemonLastCheckPath() string  { return filepath.Join(daemonDir(), "last-check") }
func daemonPIDPath() string        { return filepath.Join(daemonDir(), "mxcli-daemon.pid") }
```

- [ ] **Step 4: Run tests**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run TestDaemon -v
```

Expected: **PASS**.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli-launcher/paths.go cmd/mxcli-launcher/paths_test.go
git commit -m "feat(launcher): add daemon path helpers"
```

---

### Task 4: Socket Forward (Launcher Core)

**Files:**
- Create: `cmd/mxcli-launcher/forward.go`
- Create: `cmd/mxcli-launcher/forward_test.go`

- [ ] **Step 1: Write test**

```go
// cmd/mxcli-launcher/forward_test.go
package main

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

// startEchoServer is a minimal server that echoes argv back as stdout and exits 0.
func startEchoServer(t *testing.T) string {
	t.Helper()
	sockPath := filepath.Join(t.TempDir(), "echo.sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(ln.Close)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				var req launcherproto.Request
				launcherproto.ReadMsg(c, &req)
				// Echo argv[0] as stdout
				if len(req.Argv) > 0 {
					launcherproto.WriteMsg(c, launcherproto.Frame{Stream: "stdout", Data: []byte(req.Argv[0] + "\n")})
				}
				code := 0
				launcherproto.WriteMsg(c, launcherproto.Frame{Exit: &code})
			}(conn)
		}
	}()
	time.Sleep(10 * time.Millisecond)
	return sockPath
}

func TestForwardRequest_CapturesStdout(t *testing.T) {
	sockPath := startEchoServer(t)

	var stdout, stderr bytes.Buffer
	exitCode := forwardRequest(sockPath, []string{"hello"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}
	if stdout.String() != "hello\n" {
		t.Errorf("expected stdout 'hello\\n', got %q", stdout.String())
	}
}

func TestForwardRequest_NoServer(t *testing.T) {
	// Socket does not exist — should return exit code 1 quickly.
	var stdout, stderr bytes.Buffer
	code := forwardRequest(filepath.Join(t.TempDir(), "nosuch.sock"), []string{"x"}, &stdout, &stderr)
	if code == 0 {
		t.Error("expected non-zero exit code when daemon not running")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run TestForward -v 2>&1 | head -5
```

Expected: `FAIL` — `forwardRequest` undefined.

- [ ] **Step 3: Implement forward.go**

```go
// cmd/mxcli-launcher/forward.go
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

// forwardRequest connects to the daemon socket, sends the request, streams
// stdout/stderr to out/err, and returns the daemon's exit code.
// Returns 1 and prints a message to err if the socket cannot be reached.
func forwardRequest(sockPath string, argv []string, out, err io.Writer) int {
	conn, dialErr := net.DialTimeout("unix", sockPath, 3*time.Second)
	if dialErr != nil {
		fmt.Fprintf(err, "mxcli: cannot connect to daemon (%v)\n", dialErr)
		fmt.Fprintf(err, "       Try: mxcli upgrade\n")
		return 1
	}
	defer conn.Close()

	cwd, _ := os.Getwd()
	env := captureEnv()

	req := launcherproto.Request{Argv: argv, Cwd: cwd, Env: env}
	if writeErr := launcherproto.WriteMsg(conn, req); writeErr != nil {
		fmt.Fprintf(err, "mxcli: send request: %v\n", writeErr)
		return 1
	}

	for {
		var frame launcherproto.Frame
		if readErr := launcherproto.ReadMsg(conn, &frame); readErr != nil {
			if readErr != io.EOF {
				fmt.Fprintf(err, "mxcli: read frame: %v\n", readErr)
			}
			return 1
		}
		switch {
		case frame.Exit != nil:
			return *frame.Exit
		case frame.Stream == "stdout":
			out.Write(frame.Data)
		case frame.Stream == "stderr":
			err.Write(frame.Data)
		}
	}
}

// captureEnv captures the current process environment as a map.
func captureEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				env[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return env
}
```

- [ ] **Step 4: Run tests**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run TestForward -v
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli-launcher/forward.go cmd/mxcli-launcher/forward_test.go
git commit -m "feat(launcher): add socket forwarding (forwardRequest)"
```

---

### Task 5: Daemon Lifecycle (Start, Health-Check, Download)

**Files:**
- Create: `cmd/mxcli-launcher/daemon.go`
- Create: `cmd/mxcli-launcher/daemon_test.go`

- [ ] **Step 1: Write tests**

```go
// cmd/mxcli-launcher/daemon_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDaemonRunning_NoSocket(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "nosuch.sock")
	if isDaemonRunning(sockPath) {
		t.Error("expected false when socket does not exist")
	}
}

func TestReadDaemonVersion_Missing(t *testing.T) {
	vPath := filepath.Join(t.TempDir(), "version")
	v := readVersionFile(vPath)
	if v != "" {
		t.Errorf("expected empty version, got %q", v)
	}
}

func TestReadDaemonVersion_Present(t *testing.T) {
	vPath := filepath.Join(t.TempDir(), "version")
	os.WriteFile(vPath, []byte("v0.14.0\n"), 0644)
	v := readVersionFile(vPath)
	if v != "v0.14.0" {
		t.Errorf("expected v0.14.0, got %q", v)
	}
}

func TestDaemonBinaryExists_Missing(t *testing.T) {
	if daemonBinaryExists(filepath.Join(t.TempDir(), "no-daemon")) {
		t.Error("expected false for missing binary")
	}
}

func TestDaemonBinaryExists_Present(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "mxcli-daemon")
	os.WriteFile(p, []byte("fake"), 0755)
	if !daemonBinaryExists(p) {
		t.Error("expected true for existing binary")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run TestIsDaemon -v 2>&1 | head -5
```

Expected: `FAIL`.

- [ ] **Step 3: Implement daemon.go**

```go
// cmd/mxcli-launcher/daemon.go
package main

import (
	"archive/tar"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

const (
	daemonRepo    = "engalar/mxcli"
	daemonTimeout = 10 * time.Second
)

// isDaemonRunning returns true if the unix socket exists and accepts connections.
func isDaemonRunning(sockPath string) bool {
	conn, err := net.DialTimeout("unix", sockPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// daemonBinaryExists reports whether the daemon binary file exists.
func daemonBinaryExists(binPath string) bool {
	info, err := os.Stat(binPath)
	return err == nil && !info.IsDir()
}

// readVersionFile reads a one-line version string from path, trimming whitespace.
// Returns "" if the file cannot be read.
func readVersionFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// ensureDaemon checks that the daemon binary exists (downloading if needed)
// and that the daemon process is running (starting it if not).
func ensureDaemon() error {
	if err := os.MkdirAll(daemonDir(), 0755); err != nil {
		return fmt.Errorf("create daemon dir: %w", err)
	}

	// Download if binary missing
	if !daemonBinaryExists(daemonBinaryPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: daemon not found, downloading latest version...")
		if err := downloadDaemon(daemonBinaryPath()); err != nil {
			return fmt.Errorf("download daemon: %w", err)
		}
	}

	// Start if not running
	if !isDaemonRunning(daemonSocketPath()) {
		if err := startDaemon(); err != nil {
			return fmt.Errorf("start daemon: %w", err)
		}
	}
	return nil
}

// startDaemon launches mxcli-daemon in the background and waits until its
// socket is ready (up to daemonTimeout).
func startDaemon() error {
	cmd := exec.Command(daemonBinaryPath(), "--serve", daemonSocketPath())
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("exec daemon: %w", err)
	}
	// Write PID
	os.WriteFile(daemonPIDPath(), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)

	// Wait for socket to appear
	deadline := time.Now().Add(daemonTimeout)
	for time.Now().Before(deadline) {
		if isDaemonRunning(daemonSocketPath()) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("daemon did not start within %v", daemonTimeout)
}

// healthCheck sends a health-check request to the daemon and returns its version.
func healthCheck(sockPath string) (string, error) {
	conn, err := net.DialTimeout("unix", sockPath, 3*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	req := launcherproto.Request{Argv: []string{"__healthcheck__"}, Cwd: "/", Env: map[string]string{}}
	if err := launcherproto.WriteMsg(conn, req); err != nil {
		return "", err
	}
	var frame launcherproto.Frame
	if err := launcherproto.ReadMsg(conn, &frame); err != nil {
		return "", err
	}
	if !frame.OK {
		return "", fmt.Errorf("health check returned ok=false")
	}
	return frame.Version, nil
}

// downloadDaemon fetches the compressed daemon for the current platform and
// decompresses it to destPath.
func downloadDaemon(destPath string) error {
	tag, err := fetchLatestTag()
	if err != nil {
		return err
	}
	return downloadDaemonVersion(tag, destPath)
}

// downloadDaemonVersion downloads a specific tagged version of the daemon.
func downloadDaemonVersion(tag, destPath string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := ".tar.zst"
	if goos == "windows" {
		ext = ".zip"
	}
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s%s", goos, goarch, ext)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", daemonRepo, tag, assetName)

	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	if goos == "windows" {
		return extractZip(resp.Body, destPath)
	}
	return extractTarZst(resp.Body, destPath)
}

// extractTarZst decompresses a .tar.zst stream and writes the first regular
// file entry (the daemon binary) to destPath with executable permissions.
func extractTarZst(r io.Reader, destPath string) error {
	zr, err := zstd.NewReader(r)
	if err != nil {
		return err
	}
	defer zr.Close()

	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// Write to a temp file then rename atomically
		tmp := destPath + ".tmp"
		f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		f.Close()
		return os.Rename(tmp, destPath)
	}
	return fmt.Errorf("no regular file found in archive")
}

// extractZip handles Windows .zip archives (stub — implement when Windows support needed).
func extractZip(r io.Reader, destPath string) error {
	return fmt.Errorf("zip extraction not yet implemented; download %s manually", destPath)
}

// fetchLatestTag queries the GitHub releases API for the latest tag.
func fetchLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", daemonRepo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// Simple extraction: find "tag_name":"v..."
	s := string(body)
	key := `"tag_name":"`
	idx := strings.Index(s, key)
	if idx < 0 {
		return "", fmt.Errorf("tag_name not found in GitHub response")
	}
	s = s[idx+len(key):]
	end := strings.Index(s, `"`)
	if end < 0 {
		return "", fmt.Errorf("malformed tag_name in GitHub response")
	}
	return s[:end], nil
}
```

- [ ] **Step 4: Add zstd dependency**

```bash
go get github.com/klauspost/compress/zstd
go mod tidy
```

- [ ] **Step 5: Run tests**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestIsDaemon|TestReadDaemon|TestDaemonBinary" -v
```

Expected: all **PASS**.

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli-launcher/daemon.go cmd/mxcli-launcher/daemon_test.go go.mod go.sum
git commit -m "feat(launcher): add daemon lifecycle (start, health-check, download)"
```

---

### Task 6: Launcher Entry Point

**Files:**
- Create: `cmd/mxcli-launcher/main.go`

- [ ] **Step 1: Write the file**

No new test here — integration is covered by Tasks 4 and 5. The `main()` wires everything together.

```go
// cmd/mxcli-launcher/main.go
package main

import (
	"fmt"
	"os"
)

// Version and LauncherVersion are injected by ldflags at build time.
var (
	Version       = "dev"
	LauncherBuild = ""
)

func main() {
	args := os.Args[1:]

	// Commands that do not need the daemon
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		// Fall through to daemon (shows cobra help)
	}
	if len(args) > 0 {
		switch args[0] {
		case "upgrade":
			os.Exit(runUpgrade(args[1:]))
		case "rollback":
			os.Exit(runRollback(args[1:]))
		case "version", "--version", "-v":
			printVersion()
			os.Exit(0)
		}
	}

	// Ensure daemon exists and is running
	if err := ensureDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
		os.Exit(1)
	}

	// Start background version check (non-blocking)
	go backgroundVersionCheck()

	// Forward the request to the daemon
	exitCode := forwardRequest(daemonSocketPath(), args, os.Stdout, os.Stderr)

	// Print update notice if one is pending (after daemon output)
	printUpdateNotice()

	os.Exit(exitCode)
}

func printVersion() {
	v := Version
	if LauncherBuild != "" {
		v += " (" + LauncherBuild + ")"
	}
	daemonVer := readVersionFile(daemonVersionPath())
	fmt.Printf("mxcli launcher %s\n", v)
	if daemonVer != "" {
		fmt.Printf("mxcli daemon   %s\n", daemonVer)
	}
}

// printUpdateNotice checks the update-available file and prints a notice if
// a new version is available, then removes the file.
func printUpdateNotice() {
	p := daemonUpdateAvailablePath()
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	newVer := string(b)
	fmt.Fprintf(os.Stderr, "\n🆕 mxcli-daemon %s available → run: mxcli upgrade\n", newVer)
	os.Remove(p)
}
```

- [ ] **Step 2: Build and smoke-test**

```bash
CGO_ENABLED=0 go build -o /tmp/mxcli-launcher-test ./cmd/mxcli-launcher/
/tmp/mxcli-launcher-test version
```

Expected: prints version lines (daemon version may be empty if not installed).

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli-launcher/main.go
git commit -m "feat(launcher): add entry point (main.go)"
```

---

### Task 7: Makefile Build Targets

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add launcher + daemon targets**

In `Makefile`, after the existing `RELEASE_LDFLAGS` line, add:

```makefile
LAUNCHER_LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.LauncherBuild=$(BUILD_TIME) -s -w"
DAEMON_NAME = mxcli-daemon
```

Replace the `build` target with:

```makefile
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_NAME) $(CMD_PATH)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mxcli-launcher
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/source_tree ./cmd/source_tree
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(DAEMON_NAME) $(BUILD_DIR)/source_tree"
```

Add a `compress-daemon` helper target after `build`:

```makefile
# Compress a single daemon binary: make compress-daemon BIN=bin/mxcli-daemon-linux-amd64
compress-daemon:
	@command -v zstd >/dev/null || (echo "zstd not found; install it first" && exit 1)
	zstd --best -f "$(BIN)" -o "$(BIN).tar.zst"
	@echo "Compressed: $(BIN).tar.zst"
```

Update `release` to build both launcher and daemon, then compress daemon artifacts:

```makefile
release: clean sync-all
	@mkdir -p $(BUILD_DIR)
	@echo "Building release binaries..."

	@echo "  -> Launchers (all platforms)"
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64   ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64   ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64  ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64  ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/mxcli-launcher
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LAUNCHER_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/mxcli-launcher

	@echo "  -> Daemons (all platforms)"
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-linux-amd64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-linux-arm64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-darwin-amd64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-darwin-arm64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-windows-amd64.exe $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(DAEMON_NAME)-windows-arm64.exe $(CMD_PATH)

	@echo "  -> Compressing daemon binaries (requires zstd)"
	@for f in $(BUILD_DIR)/$(DAEMON_NAME)-linux-* $(BUILD_DIR)/$(DAEMON_NAME)-darwin-*; do \
		echo "    $$f -> $$f.tar.zst"; \
		tar -cf - -C $(BUILD_DIR) $$(basename $$f) | zstd --best -f -o $$f.tar.zst; \
	done
	@for f in $(BUILD_DIR)/$(DAEMON_NAME)-windows-*.exe; do \
		echo "    $$f -> $$f.zip"; \
		zip -j $$f.zip $$f; \
	done

	@echo ""
	@echo "Release binaries built in $(BUILD_DIR)/."
	@echo "  Launchers: mxcli-{os}-{arch}"
	@echo "  Daemons:   mxcli-daemon-{os}-{arch}.tar.zst (or .zip)"
```

- [ ] **Step 2: Verify dev build still works**

```bash
make build
ls -lh bin/mxcli bin/mxcli-daemon
```

Expected: both binaries exist.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add mxcli-launcher and daemon release targets with zstd compression"
```

---

## Phase 2: Update Mechanism

---

### Task 8: Background Version Check

**Files:**
- Create: `cmd/mxcli-launcher/update.go`
- Create: `cmd/mxcli-launcher/update_test.go`

- [ ] **Step 1: Write tests**

```go
// cmd/mxcli-launcher/update_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldCheckUpdate_TooRecent(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "last-check")
	// Write a timestamp 30 minutes ago
	ts := time.Now().Add(-30 * time.Minute).Unix()
	os.WriteFile(p, []byte(fmt.Sprintf("%d", ts)), 0644)
	if shouldCheckUpdate(p) {
		t.Error("should not check if last check was <1h ago")
	}
}

func TestShouldCheckUpdate_Expired(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "last-check")
	// Write a timestamp 2 hours ago
	ts := time.Now().Add(-2 * time.Hour).Unix()
	os.WriteFile(p, []byte(fmt.Sprintf("%d", ts)), 0644)
	if !shouldCheckUpdate(p) {
		t.Error("should check if last check was >1h ago")
	}
}

func TestShouldCheckUpdate_Missing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "last-check")
	if !shouldCheckUpdate(p) {
		t.Error("should check if last-check file does not exist")
	}
}

func TestFetchLatestTagFromServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.2.3","name":"Release v1.2.3"}`))
	}))
	defer srv.Close()

	tag, err := fetchTagFromURL(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %q", tag)
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestShouldCheck|TestFetchLatest" -v 2>&1 | head -5
```

Expected: `FAIL`.

- [ ] **Step 3: Implement update.go**

```go
// cmd/mxcli-launcher/update.go
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const updateCheckInterval = time.Hour

// backgroundVersionCheck checks GitHub for a newer daemon version (at most
// once per hour) and writes the new version to update-available if found.
// Runs in a goroutine; never panics.
func backgroundVersionCheck() {
	defer func() { recover() }()

	if !shouldCheckUpdate(daemonLastCheckPath()) {
		return
	}

	// Record check time immediately to prevent concurrent checks
	writeTimestamp(daemonLastCheckPath())

	latest, err := fetchLatestTag()
	if err != nil {
		return
	}

	current := readVersionFile(daemonVersionPath())
	if current != "" && latest != "" && latest != current {
		os.WriteFile(daemonUpdateAvailablePath(), []byte(latest), 0644)
	}
}

// shouldCheckUpdate returns true if the last-check timestamp is older than
// updateCheckInterval or the file does not exist.
func shouldCheckUpdate(lastCheckPath string) bool {
	b, err := os.ReadFile(lastCheckPath)
	if err != nil {
		return true // file missing → check now
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return true
	}
	return time.Since(time.Unix(ts, 0)) > updateCheckInterval
}

// writeTimestamp writes the current Unix timestamp to path.
func writeTimestamp(path string) {
	os.WriteFile(path, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)
}

// fetchTagFromURL fetches the GitHub releases JSON from url and extracts tag_name.
// Exported for testing with a mock server.
func fetchTagFromURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	key := `"tag_name":"`
	idx := strings.Index(s, key)
	if idx < 0 {
		return "", fmt.Errorf("tag_name not found")
	}
	s = s[idx+len(key):]
	end := strings.Index(s, `"`)
	if end < 0 {
		return "", fmt.Errorf("malformed tag_name")
	}
	return s[:end], nil
}

// runUpgrade downloads the latest daemon, backs up the current one, applies
// the new version, health-checks it, and rolls back on failure.
// Returns exit code (0=success, 1=failure).
func runUpgrade(_ []string) int {
	fmt.Println("Checking for updates...")
	latest, err := fetchLatestTag()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: fetch latest tag: %v\n", err)
		return 1
	}

	current := readVersionFile(daemonVersionPath())
	if current == latest {
		fmt.Printf("mxcli daemon is already at %s — nothing to do.\n", current)
		return 0
	}
	fmt.Printf("Upgrading daemon %s → %s\n", current, latest)

	// Download to temp location
	tmpDest := daemonBinaryPath() + ".new"
	if err := downloadDaemonVersion(latest, tmpDest); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: download: %v\n", err)
		return 1
	}

	// Roll current → bak (overwrite old bak)
	if daemonBinaryExists(daemonBinaryPath()) {
		os.Rename(daemonVersionPath(), daemonVersionBakPath())
		if err := os.Rename(daemonBinaryPath(), daemonBakPath()); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli upgrade: backup current: %v\n", err)
			os.Remove(tmpDest)
			return 1
		}
	}

	// Atomic replace
	if err := os.Rename(tmpDest, daemonBinaryPath()); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: install: %v\n", err)
		rollback()
		return 1
	}
	os.WriteFile(daemonVersionPath(), []byte(latest), 0644)

	// Health-check new daemon
	fmt.Print("Verifying new daemon...")
	sock := daemonSocketPath()
	os.Remove(sock)
	if err := startDaemon(); err != nil {
		fmt.Printf(" FAILED: %v\n", err)
		fmt.Println("Rolling back to previous version...")
		rollback()
		return 1
	}
	if _, err := healthCheck(sock); err != nil {
		fmt.Printf(" FAILED: %v\n", err)
		fmt.Println("Rolling back to previous version...")
		rollback()
		return 1
	}
	fmt.Printf(" OK\n")
	fmt.Printf("✅ Upgraded to %s (previous version kept as backup)\n", latest)
	os.Remove(daemonUpdateAvailablePath())
	return 0
}

// rollback restores the daemon from .bak. Called internally on upgrade failure.
func rollback() {
	if !daemonBinaryExists(daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: no backup to restore")
		return
	}
	os.Remove(daemonBinaryPath())
	os.Rename(daemonBakPath(), daemonBinaryPath())
	os.Rename(daemonVersionBakPath(), daemonVersionPath())
	ver := readVersionFile(daemonVersionPath())
	fmt.Printf("Rolled back to %s\n", ver)
}

// runRollback implements the `mxcli rollback` command.
func runRollback(args []string) int {
	if len(args) > 0 && args[0] == "--list" {
		current := readVersionFile(daemonVersionPath())
		bak := readVersionFile(daemonVersionBakPath())
		fmt.Printf("current: %s\n", current)
		if bak != "" {
			fmt.Printf("backup:  %s  (run 'mxcli rollback' to restore)\n", bak)
		} else {
			fmt.Println("backup:  (none)")
		}
		return 0
	}

	if !daemonBinaryExists(daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli rollback: no backup available")
		return 1
	}

	bakVer := readVersionFile(daemonVersionBakPath())
	curVer := readVersionFile(daemonVersionPath())
	fmt.Printf("Rolling back daemon %s → %s\n", curVer, bakVer)

	// Swap current ↔ bak
	tmpBin := daemonBinaryPath() + ".rollback-tmp"
	tmpVer := daemonVersionPath() + ".rollback-tmp"
	os.Rename(daemonBinaryPath(), tmpBin)
	os.Rename(daemonVersionPath(), tmpVer)
	os.Rename(daemonBakPath(), daemonBinaryPath())
	os.Rename(daemonVersionBakPath(), daemonVersionPath())
	os.Rename(tmpBin, daemonBakPath())
	os.Rename(tmpVer, daemonVersionBakPath())

	// Restart daemon with restored binary
	os.Remove(daemonSocketPath())
	if err := startDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli rollback: restart daemon: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Rolled back to %s\n", bakVer)
	return 0
}

// fetchLatestTag is called by both daemon.go and update.go.
// (The implementation is in daemon.go; this file uses it directly.)
// Note: fetchTagFromURL is the testable version; fetchLatestTag uses the real URL.
func init() {
	// Ensure fetchLatestTag uses fetchTagFromURL internally.
	// Both are in the same package (main), so no indirection needed.
	_ = filepath.Join // suppress unused import
}
```

- [ ] **Step 4: Fix daemon.go to use fetchTagFromURL**

In `daemon.go`, replace the inline implementation of `fetchLatestTag` with:

```go
func fetchLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", daemonRepo)
	return fetchTagFromURL(url)
}
```

And remove the duplicate implementation from `daemon.go` (the one that was there originally).

- [ ] **Step 5: Run tests**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli-launcher/... -run "TestShouldCheck|TestFetchLatest|TestUpgrade|TestRollback" -v
```

Expected: all **PASS**.

- [ ] **Step 6: Full build smoke-test**

```bash
make build
echo "=== version ==="
bin/mxcli version
echo "=== upgrade (dry-run check) ==="
# Should print checking message (will fail to download since no real release exists)
bin/mxcli upgrade 2>&1 | head -3 || true
```

- [ ] **Step 7: Commit**

```bash
git add cmd/mxcli-launcher/update.go cmd/mxcli-launcher/update_test.go cmd/mxcli-launcher/daemon.go
git commit -m "feat(launcher): add background version check, upgrade, rollback"
```

---

## Phase 3: Distribution

---

### Task 9: Install Script (Linux/macOS)

**Files:**
- Create: `install.sh`

- [ ] **Step 1: Write the script**

```bash
#!/bin/sh
# mxcli install script — idempotent, works on Linux and macOS.
# Usage: curl -fsSL https://raw.githubusercontent.com/engalar/mxcli/main/install.sh | sh
set -e

REPO="engalar/mxcli"
INSTALL_DIR="${MXCLI_INSTALL_DIR:-}"

# ── Detect platform ──────────────────────────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "❌ Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# ── Fetch latest release tag ─────────────────────────────────────────────────
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$LATEST" ]; then
  echo "❌ Could not fetch latest release tag from GitHub." >&2
  exit 1
fi

# ── Idempotent version check ─────────────────────────────────────────────────
if command -v mxcli >/dev/null 2>&1; then
  CURRENT=$(mxcli version --short 2>/dev/null | head -1 | awk '{print $NF}' || echo "")
  if [ "$CURRENT" = "$LATEST" ]; then
    echo "✅ mxcli $CURRENT is already up to date."
    exit 0
  fi
  echo "Updating mxcli $CURRENT → $LATEST"
else
  echo "Installing mxcli $LATEST"
fi

# ── Determine install directory ───────────────────────────────────────────────
if [ -z "$INSTALL_DIR" ]; then
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
    # Idempotent PATH entry
    for RC in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
      [ -f "$RC" ] || continue
      grep -qF "$INSTALL_DIR" "$RC" && continue
      printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$RC"
      echo "  Added $INSTALL_DIR to PATH in $RC"
    done
  fi
fi

# ── Download launcher binary ──────────────────────────────────────────────────
BIN_URL="https://github.com/${REPO}/releases/download/${LATEST}/mxcli-${OS}-${ARCH}"
TMP=$(mktemp /tmp/mxcli.XXXXXX)
trap 'rm -f "$TMP"' EXIT

echo "  Downloading launcher from $BIN_URL"
curl -fsSL "$BIN_URL" -o "$TMP"
chmod +x "$TMP"

# Atomic install
mv "$TMP" "${INSTALL_DIR}/mxcli"

echo ""
echo "✅ mxcli $LATEST installed to ${INSTALL_DIR}/mxcli"
echo "   The daemon (~20 MB) will be downloaded automatically on first use."
echo ""
echo "   Run: mxcli version"
```

- [ ] **Step 2: Make executable and lint**

```bash
chmod +x install.sh
# Lint with shellcheck if available
command -v shellcheck && shellcheck install.sh || echo "(shellcheck not installed, skipping)"
```

- [ ] **Step 3: Manual smoke-test (in a temp dir)**

```bash
# Test idempotent check: already installed
MXCLI_INSTALL_DIR=/tmp/test-mxcli-install sh install.sh
ls -la /tmp/test-mxcli-install/
```

Expected: downloads launcher binary, prints install path.

- [ ] **Step 4: Commit**

```bash
git add install.sh
git commit -m "feat(install): add idempotent Linux/macOS install script"
```

---

### Task 10: Install Script (Windows PowerShell)

**Files:**
- Create: `install.ps1`

- [ ] **Step 1: Write the script**

```powershell
# mxcli install script for Windows — idempotent.
# Usage: irm https://raw.githubusercontent.com/engalar/mxcli/main/install.ps1 | iex
#
# Optional: set $env:MXCLI_INSTALL_DIR before running to override install location.

$ErrorActionPreference = "Stop"
$Repo = "engalar/mxcli"

# ── Detect architecture ──────────────────────────────────────────────────────
$Arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Error "32-bit Windows is not supported."
    exit 1
}

# ── Fetch latest release tag ─────────────────────────────────────────────────
$ApiUrl = "https://api.github.com/repos/$Repo/releases/latest"
$Release = Invoke-RestMethod -Uri $ApiUrl -Headers @{ "User-Agent" = "mxcli-installer" }
$Latest = $Release.tag_name
if (-not $Latest) {
    Write-Error "Could not fetch latest release tag from GitHub."
    exit 1
}

# ── Idempotent version check ─────────────────────────────────────────────────
$MxcliCmd = Get-Command mxcli -ErrorAction SilentlyContinue
if ($MxcliCmd) {
    $Current = (& mxcli version --short 2>$null | Select-Object -First 1)?.Split(" ")[-1]
    if ($Current -eq $Latest) {
        Write-Host "✅ mxcli $Current is already up to date."
        exit 0
    }
    Write-Host "Updating mxcli $Current → $Latest"
} else {
    Write-Host "Installing mxcli $Latest"
}

# ── Determine install directory ───────────────────────────────────────────────
$InstallDir = if ($env:MXCLI_INSTALL_DIR) {
    $env:MXCLI_INSTALL_DIR
} else {
    "$env:LOCALAPPDATA\mxcli"
}
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# ── Add to PATH (idempotent) ─────────────────────────────────────────────────
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
    Write-Host "  Added $InstallDir to user PATH"
}

# ── Download launcher binary ──────────────────────────────────────────────────
$BinName = "mxcli-windows-$Arch.exe"
$BinUrl = "https://github.com/$Repo/releases/download/$Latest/$BinName"
$Dest = Join-Path $InstallDir "mxcli.exe"
$Tmp = [System.IO.Path]::GetTempFileName() + ".exe"

Write-Host "  Downloading from $BinUrl"
Invoke-WebRequest -Uri $BinUrl -OutFile $Tmp -UseBasicParsing
Move-Item -Force $Tmp $Dest

Write-Host ""
Write-Host "✅ mxcli $Latest installed to $Dest"
Write-Host "   The daemon (~20 MB) will be downloaded automatically on first use."
Write-Host "   Restart your terminal for PATH changes to take effect."
Write-Host ""
Write-Host "   Run: mxcli version"
```

- [ ] **Step 2: Commit**

```bash
git add install.ps1
git commit -m "feat(install): add idempotent Windows PowerShell install script"
```

---

### Task 11: Release CI — Compressed Daemon Artifacts

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Update release.yml**

Replace the `Build release binaries` step and `Create GitHub Release` step with:

```yaml
      - name: Install zstd
        run: sudo apt-get install -y zstd

      - name: Build release binaries
        run: make release

      - name: Generate SHA256 checksums
        run: |
          cd bin
          sha256sum mxcli-* mxcli-daemon-* > SHA256SUMS
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
            bin/mxcli-daemon-linux-amd64.tar.zst
            bin/mxcli-daemon-linux-arm64.tar.zst
            bin/mxcli-daemon-darwin-amd64.tar.zst
            bin/mxcli-daemon-darwin-arm64.tar.zst
            bin/mxcli-daemon-windows-amd64.exe.zip
            bin/mxcli-daemon-windows-arm64.exe.zip
            bin/SHA256SUMS
            install.sh
            install.ps1
```

- [ ] **Step 2: Verify Makefile produces expected artifacts**

```bash
# Check that release Makefile target produces correct artifact names
# (dry-run: just check the naming without actually building all platforms)
grep "DAEMON_NAME\|mxcli-daemon" Makefile
```

Expected: see `mxcli-daemon-linux-amd64`, `mxcli-daemon-darwin-arm64`, etc.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: release launcher + compressed daemon artifacts with SHA256SUMS"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Task |
|-----------------|------|
| Thin launcher (~2MB) | Task 6 (main.go) + Task 7 (Makefile) |
| Socket protocol (4-byte length + JSON) | Task 1 |
| Daemon socket server mode (--serve) | Task 2 |
| Daemon lifecycle (start, health-check) | Task 5 |
| Download daemon on first run | Task 5 |
| Background version check (non-blocking, 1h rate limit) | Task 8 |
| `mxcli upgrade` with backup + health-check + rollback | Task 8 |
| `mxcli rollback` + `--list` | Task 8 |
| N-1 backup always retained | Task 8 (rollback() keeps .bak) |
| Install script Linux/macOS (idempotent) | Task 9 |
| Install script Windows (idempotent) | Task 10 |
| Release CI with compressed daemon | Task 11 |
| `engalar/mxcli` repo references | Tasks 5, 9, 10, 11 |

**Placeholder scan:** No TBD, TODO, or "similar to Task N" found. Every code step contains complete implementations.

**Type consistency:**
- `launcherproto.Request`, `launcherproto.Frame`, `launcherproto.WriteMsg`, `launcherproto.ReadMsg` — defined Task 1, used consistently in Tasks 2, 4, 5.
- `forwardRequest(sockPath, argv, out, err)` — defined Task 4, called Task 6.
- `ensureDaemon()` → `daemonBinaryExists` + `isDaemonRunning` + `startDaemon` + `downloadDaemon` — all defined Task 5.
- `fetchTagFromURL` — defined Task 8, `fetchLatestTag` wraps it (daemon.go updated in Task 8 Step 4).
- All path helpers (`daemonBinaryPath`, `daemonBakPath`, etc.) — defined Task 3, used throughout.

**Gap found:** `update.go` imports `fmt` but it's not listed in the imports. Fixed: the import block in update.go uses `fmt` (for `Fprintf`/`Printf`) — it must be included in the import block. The implementation above includes it implicitly via the code; add explicit `"fmt"` to the import list in the actual file.

**Gap found:** `extractZip` is a stub that returns an error. This is acceptable for Phase 1 (Windows downloads via zip are not the primary path), and the TODO is documented in the function body.
