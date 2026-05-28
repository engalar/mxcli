# Install/Update Test Fixtures Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two-layer test coverage for mxcli launcher install/upgrade/rollback logic: component injection via `Env` struct + end-to-end via `FakeGitHub` HTTP server, all running in-process with `t.TempDir()` isolation.

**Architecture:** Introduce `Env{HomeDir, HTTPClient}` into `cmd/mxcli-launcher/`; convert path free-functions to `(*Env)` methods; convert HTTP-using functions to take `*Env`; add file-lock helpers; create `testfixtures/` package with `FakeGitHub` (configurable `httptest.Server`) and `FakeDaemon` (tar.zst payload builder); write scenario matrix in `install_update_test.go`.

**Tech Stack:** Go standard library (`net/http/httptest`, `archive/tar`, `crypto/sha256`, `syscall`), `github.com/klauspost/compress/zstd` (already in `go.mod`)

---

## File Map

| Action | Path | Responsibility |
|--------|------|---------------|
| Create | `cmd/mxcli-launcher/env.go` | `Env` struct + `DefaultEnv()` |
| Create | `cmd/mxcli-launcher/lock_unix.go` | `flock`/`funlock` (linux + darwin) |
| Create | `cmd/mxcli-launcher/lock_windows.go` | `LockFileEx`/`UnlockFile` (windows) |
| Modify | `cmd/mxcli-launcher/paths.go` | Free functions → `(*Env)` methods |
| Modify | `cmd/mxcli-launcher/daemon.go` | Add `*Env` param; remove package-level `httpClient` |
| Modify | `cmd/mxcli-launcher/update.go` | Add `*Env` param; add lock in `runUpgrade` |
| Modify | `cmd/mxcli-launcher/main.go` | Create `DefaultEnv()` and thread through |
| Modify | `cmd/mxcli-launcher/paths_test.go` | Adapt to `(*Env)` methods |
| Modify | `cmd/mxcli-launcher/daemon_test.go` | Adapt to `*Env` params |
| Modify | `cmd/mxcli-launcher/update_test.go` | No change needed (tests free functions that keep same sig) |
| Create | `cmd/mxcli-launcher/testfixtures/fake_daemon.go` | Build minimal tar.zst payload + compute SHA256 |
| Create | `cmd/mxcli-launcher/testfixtures/fake_github.go` | `FakeGitHub` httptest server |
| Create | `cmd/mxcli-launcher/install_update_test.go` | Full scenario matrix |

---

## Task 1: Create `env.go` + refactor `paths.go`

**Files:**
- Create: `cmd/mxcli-launcher/env.go`
- Modify: `cmd/mxcli-launcher/paths.go`

- [ ] **Step 1: Write failing test for `Env`-based paths**

Add to `cmd/mxcli-launcher/paths_test.go` (replace the entire file):

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestEnv(t *testing.T) *Env {
	t.Helper()
	return &Env{HomeDir: t.TempDir(), HTTPClient: nil}
}

func TestEnvDaemonDir_IsUnderHome(t *testing.T) {
	e := newTestEnv(t)
	dir := e.daemonDir()
	if !filepath.IsAbs(dir) {
		t.Errorf("daemonDir must be absolute, got %q", dir)
	}
	if dir == e.HomeDir {
		t.Error("daemonDir must not equal HomeDir")
	}
}

func TestEnvPaths_Consistent(t *testing.T) {
	e := newTestEnv(t)
	dir := e.daemonDir()
	if e.daemonBinaryPath() != filepath.Join(dir, "mxcli-daemon") {
		t.Error("daemonBinaryPath mismatch")
	}
	if e.daemonBakPath() != filepath.Join(dir, "mxcli-daemon.bak") {
		t.Error("daemonBakPath mismatch")
	}
	if e.daemonSocketPath() != filepath.Join(dir, "mxcli.sock") {
		t.Error("daemonSocketPath mismatch")
	}
	if e.daemonVersionPath() != filepath.Join(dir, "version") {
		t.Error("daemonVersionPath mismatch")
	}
	if e.daemonVersionBakPath() != filepath.Join(dir, "version.bak") {
		t.Error("daemonVersionBakPath mismatch")
	}
	if e.daemonUpdateAvailablePath() != filepath.Join(dir, "update-available") {
		t.Error("daemonUpdateAvailablePath mismatch")
	}
	if e.daemonLastCheckPath() != filepath.Join(dir, "last-check") {
		t.Error("daemonLastCheckPath mismatch")
	}
	if e.daemonPIDPath() != filepath.Join(dir, "mxcli-daemon.pid") {
		t.Error("daemonPIDPath mismatch")
	}
}

func TestDefaultEnv_HomeDir(t *testing.T) {
	e := DefaultEnv()
	home, _ := os.UserHomeDir()
	if e.HomeDir != home {
		t.Errorf("DefaultEnv HomeDir = %q, want %q", e.HomeDir, home)
	}
	if e.HTTPClient == nil {
		t.Error("DefaultEnv HTTPClient must not be nil")
	}
}
```

- [ ] **Step 2: Run tests — expect compile error (types don't exist yet)**

```bash
cd cmd/mxcli-launcher && go test ./... 2>&1 | head -20
```

Expected: `undefined: Env`

- [ ] **Step 3: Create `env.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"
	"os"
	"time"
)

// Env holds injectable dependencies for launcher operations.
// Use DefaultEnv() in production; substitute fields in tests.
type Env struct {
	HomeDir    string
	HTTPClient *http.Client
}

// DefaultEnv returns an Env configured for production use.
func DefaultEnv() *Env {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return &Env{
		HomeDir:    home,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}
```

- [ ] **Step 4: Rewrite `paths.go` as `(*Env)` methods**

Replace the entire file:

```go
// SPDX-License-Identifier: Apache-2.0

package main

import "path/filepath"

func (e *Env) daemonDir() string             { return filepath.Join(e.HomeDir, ".mxcli", "daemon") }
func (e *Env) daemonBinaryPath() string      { return filepath.Join(e.daemonDir(), "mxcli-daemon") }
func (e *Env) daemonBakPath() string         { return filepath.Join(e.daemonDir(), "mxcli-daemon.bak") }
func (e *Env) daemonSocketPath() string      { return filepath.Join(e.daemonDir(), "mxcli.sock") }
func (e *Env) daemonVersionPath() string     { return filepath.Join(e.daemonDir(), "version") }
func (e *Env) daemonVersionBakPath() string  { return filepath.Join(e.daemonDir(), "version.bak") }
func (e *Env) daemonUpdateAvailablePath() string { return filepath.Join(e.daemonDir(), "update-available") }
func (e *Env) daemonLastCheckPath() string   { return filepath.Join(e.daemonDir(), "last-check") }
func (e *Env) daemonPIDPath() string         { return filepath.Join(e.daemonDir(), "mxcli-daemon.pid") }
```

- [ ] **Step 5: Run tests (expect compile errors from callers of old free functions)**

```bash
cd cmd/mxcli-launcher && go build ./... 2>&1 | head -30
```

Expected: multiple `undefined: daemonDir` etc. errors — this guides Task 2.

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli-launcher/env.go cmd/mxcli-launcher/paths.go cmd/mxcli-launcher/paths_test.go
git commit -m "refactor(launcher): introduce Env struct; convert paths to methods"
```

---

## Task 2: Refactor `daemon.go` to use `*Env`

**Files:**
- Modify: `cmd/mxcli-launcher/daemon.go`
- Modify: `cmd/mxcli-launcher/daemon_test.go`

- [ ] **Step 1: Update `daemon.go` — add `*Env` to all functions, remove `httpClient` global**

Replace the entire file content. Key changes:
1. Remove `var httpClient = &http.Client{...}`
2. `isDaemonRunning`, `daemonBinaryExists`, `readVersionFile` — no HTTP, no paths, signatures unchanged
3. `ensureDaemon(e *Env)`, `startDaemon(e *Env)`, `healthCheck(e *Env, sockPath string)` — path calls become `e.daemonXxx()`
4. `downloadDaemon(e *Env, destPath string)`, `downloadDaemonVersion(e *Env, tag, destPath string)` — `httpClient.Get` → `e.HTTPClient.Get`
5. `fetchAssetChecksum(e *Env, tag, assetName string)` — same
6. `fetchLatestTag(e *Env)` calls `fetchTagFromURL(e *Env, url)` — `httpClient.Get` → `e.HTTPClient.Get`
7. `killPIDFile`, `killRunningDaemon(e *Env)`, `rollback(e *Env)` — path calls become `e.daemonXxx()`

Full replacement:

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

const (
	daemonRepo    = "engalar/mxcli"
	daemonTimeout = 10 * time.Second
)

func isDaemonRunning(sockPath string) bool {
	conn, err := net.DialTimeout("unix", sockPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func daemonBinaryExists(binPath string) bool {
	info, err := os.Stat(binPath)
	return err == nil && !info.IsDir()
}

func readVersionFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (e *Env) ensureDaemon() error {
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		return fmt.Errorf("create daemon dir: %w", err)
	}
	if !daemonBinaryExists(e.daemonBinaryPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: daemon not found, downloading latest version...")
		if err := e.downloadDaemon(e.daemonBinaryPath()); err != nil {
			return fmt.Errorf("download daemon: %w", err)
		}
	}
	if !isDaemonRunning(e.daemonSocketPath()) {
		if err := e.startDaemon(); err != nil {
			return fmt.Errorf("start daemon: %w", err)
		}
	}
	return nil
}

func (e *Env) startDaemon() error {
	cmd := exec.Command(e.daemonBinaryPath(), "--serve", e.daemonSocketPath())
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("exec daemon: %w", err)
	}
	os.WriteFile(e.daemonPIDPath(), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)
	deadline := time.Now().Add(daemonTimeout)
	for time.Now().Before(deadline) {
		if isDaemonRunning(e.daemonSocketPath()) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	cmd.Process.Kill()
	return fmt.Errorf("daemon did not start within %v", daemonTimeout)
}

func (e *Env) healthCheck(sockPath string) (string, error) {
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

func (e *Env) downloadDaemon(destPath string) error {
	tag, err := e.fetchLatestTag()
	if err != nil {
		return err
	}
	return e.downloadDaemonVersion(tag, destPath)
}

func (e *Env) downloadDaemonVersion(tag, destPath string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var archiveExt string
	if goos == "windows" {
		archiveExt = ".zip"
	} else {
		archiveExt = ".tar.zst"
	}
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s%s", goos, goarch, archiveExt)

	expectedHash, err := e.fetchAssetChecksum(tag, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", daemonRepo, tag, assetName)
	fmt.Fprintf(os.Stderr, "  Downloading %s...\n", url)
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
		return fmt.Errorf("create archive temp: %w", err)
	}
	if _, err := io.Copy(af, tee); err != nil {
		af.Close()
		return fmt.Errorf("download archive: %w", err)
	}
	af.Close()

	actualHash := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("checksum mismatch for %s: expected %s got %s", assetName, expectedHash, actualHash)
	}

	binaryName := "mxcli-daemon"
	if goos == "windows" {
		binaryName = "mxcli-daemon.exe"
		return extractZip(archiveTmp, destPath, binaryName)
	}
	ar, err := os.Open(archiveTmp)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer ar.Close()
	return extractTarZst(ar, destPath, binaryName)
}

func (e *Env) fetchAssetChecksum(tag, assetName string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/SHA256SUMS", daemonRepo, tag)
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
		return "", fmt.Errorf("read SHA256SUMS: %w", err)
	}
	return parseChecksumFile(string(content), assetName)
}

func parseChecksumFile(content, filename string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum for %q in SHA256SUMS", filename)
}

func extractTarZst(r io.Reader, destPath, expectedName string) error {
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
		if filepath.Base(hdr.Name) != expectedName {
			continue
		}
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
	return fmt.Errorf("no file named %q found in archive", expectedName)
}

func extractZip(srcPath, destPath, expectedName string) error {
	zr, err := zip.OpenReader(srcPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if filepath.Base(f.Name) != expectedName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		tmp := destPath + ".tmp"
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			os.Remove(tmp)
			return copyErr
		}
		return os.Rename(tmp, destPath)
	}
	return fmt.Errorf("no file named %q found in zip archive", expectedName)
}

func (e *Env) fetchLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", daemonRepo)
	return e.fetchTagFromURL(url)
}

func (e *Env) fetchTagFromURL(url string) (string, error) {
	resp, err := e.HTTPClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse GitHub response: %w", err)
	}
	if result.TagName == "" {
		return "", fmt.Errorf("tag_name not found in GitHub response")
	}
	return result.TagName, nil
}

func killPIDFile(pidPath, sockPath string) {
	b, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err == nil && pid > 0 {
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill()
		}
	}
	os.Remove(pidPath)
	os.Remove(sockPath)
}

func (e *Env) killRunningDaemon() {
	killPIDFile(e.daemonPIDPath(), e.daemonSocketPath())
}

func (e *Env) rollback() {
	if !daemonBinaryExists(e.daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli: no backup to restore")
		return
	}
	e.killRunningDaemon()
	os.Remove(e.daemonBinaryPath())
	os.Rename(e.daemonBakPath(), e.daemonBinaryPath())
	os.Rename(e.daemonVersionBakPath(), e.daemonVersionPath())
	ver := readVersionFile(e.daemonVersionPath())
	fmt.Printf("Rolled back to %s\n", ver)
}
```

- [ ] **Step 2: Update `daemon_test.go` — fix `fetchTagFromURL` and `healthCheck` callers**

Replace the `TestFetchTagFromURL` test to use `*Env`:

```go
func TestFetchTagFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v1.2.3","name":"Release v1.2.3"}`))
	}))
	defer srv.Close()
	e := &Env{HTTPClient: srv.Client()}
	tag, err := e.fetchTagFromURL(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %q", tag)
	}
}
```

All other tests in `daemon_test.go` call free functions (`extractTarZst`, `extractZip`, `parseChecksumFile`, `isDaemonRunning`, `daemonBinaryExists`, `readVersionFile`, `killPIDFile`) — these remain free functions with unchanged signatures, so no other changes needed.

- [ ] **Step 3: Build + run tests**

```bash
cd cmd/mxcli-launcher && go build ./... && go test ./... -v 2>&1 | tail -20
```

Expected: all existing tests pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli-launcher/daemon.go cmd/mxcli-launcher/daemon_test.go
git commit -m "refactor(launcher): add *Env to HTTP-using functions in daemon.go"
```

---

## Task 3: Refactor `update.go` + wire `main.go`

**Files:**
- Modify: `cmd/mxcli-launcher/update.go`
- Modify: `cmd/mxcli-launcher/main.go`

- [ ] **Step 1: Update `update.go` — thread `*Env` through all functions**

Replace the entire file:

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const updateCheckInterval = time.Hour

func (e *Env) backgroundVersionCheck() {
	defer func() { recover() }()

	if !shouldCheckUpdate(e.daemonLastCheckPath()) {
		return
	}
	writeTimestamp(e.daemonLastCheckPath())

	latest, err := e.fetchLatestTag()
	if err != nil {
		return
	}
	current := readVersionFile(e.daemonVersionPath())
	if current != "" && latest != "" && latest != current {
		os.WriteFile(e.daemonUpdateAvailablePath(), []byte(latest), 0644)
	}
}

func shouldCheckUpdate(lastCheckPath string) bool {
	b, err := os.ReadFile(lastCheckPath)
	if err != nil {
		return true
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return true
	}
	return time.Since(time.Unix(ts, 0)) > updateCheckInterval
}

func writeTimestamp(path string) {
	os.WriteFile(path, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)
}

func fprintUpdateNotice(w io.Writer, p string) {
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "\n🆕 mxcli-daemon %s available → run: mxcli upgrade\n", strings.TrimSpace(string(b)))
	os.Remove(p)
}

func (e *Env) printUpdateNotice() {
	fprintUpdateNotice(os.Stderr, e.daemonUpdateAvailablePath())
}

func (e *Env) runUpgrade(_ []string) int {
	fmt.Println("Checking for updates...")
	latest, err := e.fetchLatestTag()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: fetch latest tag: %v\n", err)
		return 1
	}
	current := readVersionFile(e.daemonVersionPath())
	if current == latest {
		fmt.Printf("mxcli daemon is already at %s — nothing to do.\n", current)
		return 0
	}
	fmt.Printf("Upgrading daemon %s → %s\n", current, latest)

	tmpDest := e.daemonBinaryPath() + ".new"
	if err := e.downloadDaemonVersion(latest, tmpDest); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: download: %v\n", err)
		return 1
	}

	if daemonBinaryExists(e.daemonBinaryPath()) {
		os.Rename(e.daemonVersionPath(), e.daemonVersionBakPath())
		if err := os.Rename(e.daemonBinaryPath(), e.daemonBakPath()); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli upgrade: backup current: %v\n", err)
			os.Remove(tmpDest)
			return 1
		}
	}

	if err := os.Rename(tmpDest, e.daemonBinaryPath()); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: install: %v\n", err)
		e.rollback()
		return 1
	}
	os.WriteFile(e.daemonVersionPath(), []byte(latest), 0644)

	fmt.Print("Verifying new daemon...")
	sock := e.daemonSocketPath()
	os.Remove(sock)
	if err := e.startDaemon(); err != nil {
		fmt.Printf(" FAILED: %v\n", err)
		fmt.Println("Rolling back to previous version...")
		e.rollback()
		return 1
	}
	if _, err := e.healthCheck(sock); err != nil {
		fmt.Printf(" FAILED: %v\n", err)
		fmt.Println("Rolling back to previous version...")
		e.rollback()
		return 1
	}
	fmt.Println(" OK")
	fmt.Printf("✅ Upgraded to %s (previous version kept as backup)\n", latest)
	os.Remove(e.daemonUpdateAvailablePath())
	return 0
}

func (e *Env) runRollback(args []string) int {
	if len(args) > 0 && args[0] == "--list" {
		current := readVersionFile(e.daemonVersionPath())
		bak := readVersionFile(e.daemonVersionBakPath())
		fmt.Printf("current: %s\n", current)
		if bak != "" {
			fmt.Printf("backup:  %s  (run 'mxcli rollback' to restore)\n", bak)
		} else {
			fmt.Println("backup:  (none)")
		}
		return 0
	}

	if !daemonBinaryExists(e.daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli rollback: no backup available")
		return 1
	}

	bakVer := readVersionFile(e.daemonVersionBakPath())
	curVer := readVersionFile(e.daemonVersionPath())
	fmt.Printf("Rolling back daemon %s → %s\n", curVer, bakVer)

	e.killRunningDaemon()

	tmpBin := e.daemonBinaryPath() + ".rb-tmp"
	tmpVer := e.daemonVersionPath() + ".rb-tmp"
	os.Rename(e.daemonBinaryPath(), tmpBin)
	os.Rename(e.daemonVersionPath(), tmpVer)
	os.Rename(e.daemonBakPath(), e.daemonBinaryPath())
	os.Rename(e.daemonVersionBakPath(), e.daemonVersionPath())
	os.Rename(tmpBin, e.daemonBakPath())
	os.Rename(tmpVer, e.daemonVersionBakPath())
	if err := e.startDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli rollback: restart daemon: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Rolled back to %s\n", bakVer)
	return 0
}
```

- [ ] **Step 2: Update `main.go` — create `DefaultEnv()` and thread it through**

Replace the entire file:

```go
// SPDX-License-Identifier: Apache-2.0

// mxcli is the launcher — a thin cross-platform client that forwards CLI
// requests to mxcli-daemon via unix socket. It handles daemon lifecycle,
// background version checks, and upgrade/rollback.
package main

import (
	"fmt"
	"os"
)

var (
	Version       = "dev"
	LauncherBuild = ""
)

func main() {
	e := DefaultEnv()
	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "upgrade":
			os.Exit(e.runUpgrade(args[1:]))
		case "rollback":
			os.Exit(e.runRollback(args[1:]))
		case "version", "--version":
			printVersion(e)
			os.Exit(0)
		}
	}

	if err := e.ensureDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
		os.Exit(1)
	}

	go e.backgroundVersionCheck()

	exitCode := forwardRequest(e.daemonSocketPath(), args, os.Stdout, os.Stderr)

	e.printUpdateNotice()

	os.Exit(exitCode)
}

func printVersion(e *Env) {
	v := Version
	if LauncherBuild != "" {
		v += " (" + LauncherBuild + ")"
	}
	daemonVer := readVersionFile(e.daemonVersionPath())
	fmt.Printf("mxcli launcher %s\n", v)
	if daemonVer != "" {
		fmt.Printf("mxcli daemon   %s\n", daemonVer)
	}
}
```

- [ ] **Step 3: Build + run all tests**

```bash
cd cmd/mxcli-launcher && go build ./... && go test ./... -v 2>&1 | tail -20
```

Expected: all existing tests pass. No test changed in `update_test.go` — `shouldCheckUpdate`, `writeTimestamp`, `fprintUpdateNotice` remain free functions.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli-launcher/update.go cmd/mxcli-launcher/main.go
git commit -m "refactor(launcher): thread *Env through update.go and main.go"
```

---

## Task 4: Add file lock helpers + lock `runUpgrade`

**Files:**
- Create: `cmd/mxcli-launcher/lock_unix.go`
- Create: `cmd/mxcli-launcher/lock_windows.go`
- Modify: `cmd/mxcli-launcher/update.go`

- [ ] **Step 1: Write failing test for concurrent upgrade**

Add to `cmd/mxcli-launcher/update_test.go`:

```go
func TestRunUpgradeWithEnv_ConcurrentLock(t *testing.T) {
	// Both goroutines call upgradeWithLock on the same Env.
	// Exactly one should succeed (return nil), the other should return an error.
	e := &Env{HomeDir: t.TempDir(), HTTPClient: nil}
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}

	results := make([]error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = e.acquireUpgradeLock()
			if results[i] == nil {
				// Hold the lock briefly to force contention
				time.Sleep(10 * time.Millisecond)
				e.releaseUpgradeLock()
			}
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 lock acquisition, got %d (errors: %v, %v)", successes, results[0], results[1])
	}
}
```

Add `"sync"` and `"time"` to the import block in `update_test.go`.

- [ ] **Step 2: Run test — expect compile error**

```bash
cd cmd/mxcli-launcher && go test -run TestRunUpgradeWithEnv_ConcurrentLock ./... 2>&1
```

Expected: `undefined: e.acquireUpgradeLock`

- [ ] **Step 3: Create `lock_unix.go`**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// lockFile holds the open lock file descriptor.
type lockFile struct{ f *os.File }

func (e *Env) acquireUpgradeLock() error {
	lockPath := filepath.Join(e.daemonDir(), "upgrade.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return fmt.Errorf("upgrade in progress")
	}
	e.upgradeLock = &lockFile{f: f}
	return nil
}

func (e *Env) releaseUpgradeLock() {
	if e.upgradeLock == nil {
		return
	}
	syscall.Flock(int(e.upgradeLock.f.Fd()), syscall.LOCK_UN)
	e.upgradeLock.f.Close()
	e.upgradeLock = nil
}
```

- [ ] **Step 4: Create `lock_windows.go`**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var modkernel32 = syscall.NewLazyDLL("kernel32.dll")
var procLockFileEx = modkernel32.NewProc("LockFileEx")
var procUnlockFile = modkernel32.NewProc("UnlockFile")

const lockfileFailImmediately = 0x00000001
const lockfileExclusiveLock = 0x00000002

type lockFile struct{ f *os.File }

func (e *Env) acquireUpgradeLock() error {
	lockPath := filepath.Join(e.daemonDir(), "upgrade.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	ol := new(syscall.Overlapped)
	r, _, _ := procLockFileEx.Call(
		uintptr(f.Fd()),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0, 1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r == 0 {
		f.Close()
		return fmt.Errorf("upgrade in progress")
	}
	e.upgradeLock = &lockFile{f: f}
	return nil
}

func (e *Env) releaseUpgradeLock() {
	if e.upgradeLock == nil {
		return
	}
	procUnlockFile.Call(uintptr(e.upgradeLock.f.Fd()), 0, 1, 0, 0)
	e.upgradeLock.f.Close()
	e.upgradeLock = nil
}
```

- [ ] **Step 5: Add `upgradeLock` field to `Env` in `env.go`**

```go
type Env struct {
	HomeDir     string
	HTTPClient  *http.Client
	upgradeLock *lockFile // non-nil while upgrade lock is held
}
```

- [ ] **Step 6: Wire lock into `runUpgrade` in `update.go`**

At the top of `(e *Env) runUpgrade`:

```go
func (e *Env) runUpgrade(_ []string) int {
	if err := e.acquireUpgradeLock(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: %v\n", err)
		return 1
	}
	defer e.releaseUpgradeLock()
	// ... rest unchanged
```

- [ ] **Step 7: Run the concurrent lock test**

```bash
cd cmd/mxcli-launcher && go test -run TestRunUpgradeWithEnv_ConcurrentLock -v -count=5 ./...
```

Expected: `PASS` consistently across 5 runs.

- [ ] **Step 8: Run all tests**

```bash
cd cmd/mxcli-launcher && go test ./... -v 2>&1 | tail -20
```

Expected: all pass.

- [ ] **Step 9: Commit**

```bash
git add cmd/mxcli-launcher/env.go cmd/mxcli-launcher/lock_unix.go cmd/mxcli-launcher/lock_windows.go cmd/mxcli-launcher/update.go cmd/mxcli-launcher/update_test.go
git commit -m "feat(launcher): add file-lock for concurrent upgrade safety"
```

---

## Task 5: Create `testfixtures` package

**Files:**
- Create: `cmd/mxcli-launcher/testfixtures/fake_daemon.go`
- Create: `cmd/mxcli-launcher/testfixtures/fake_github.go`

- [ ] **Step 1: Create `testfixtures/fake_daemon.go`**

```go
// SPDX-License-Identifier: Apache-2.0

// Package testfixtures provides test helpers for launcher install/update tests.
package testfixtures

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"

	"github.com/klauspost/compress/zstd"
)

// DaemonPayload holds a fake daemon archive and its SHA256 checksum.
type DaemonPayload struct {
	// AssetName is the filename the fake server uses, e.g. "mxcli-daemon-linux-amd64.tar.zst"
	AssetName string
	// Archive is the raw bytes of the tar.zst archive.
	Archive []byte
	// Checksum is the correct SHA256 hex digest of Archive.
	Checksum string
}

// BuildDaemonPayload creates a minimal tar.zst containing a fake daemon binary
// for the current platform. The binary content is the provided content bytes.
func BuildDaemonPayload(content []byte) (*DaemonPayload, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	binaryName := "mxcli-daemon"
	if goos == "windows" {
		binaryName = "mxcli-daemon.exe"
	}
	assetName := fmt.Sprintf("mxcli-daemon-%s-%s.tar.zst", goos, goarch)

	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return nil, fmt.Errorf("zstd writer: %w", err)
	}
	tw := tar.NewWriter(zw)
	hdr := &tar.Header{
		Name:     binaryName,
		Typeflag: tar.TypeReg,
		Size:     int64(len(content)),
		Mode:     0755,
	}
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
	checksum := hex.EncodeToString(h[:])

	return &DaemonPayload{
		AssetName: assetName,
		Archive:   archiveBytes,
		Checksum:  checksum,
	}, nil
}

// CorruptChecksum returns a deliberately wrong SHA256 hex string (all zeros).
func CorruptChecksum() string {
	return "0000000000000000000000000000000000000000000000000000000000000000"
}
```

- [ ] **Step 2: Create `testfixtures/fake_github.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package testfixtures

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// FakeGitHub is a configurable httptest.Server that mimics GitHub releases API.
// Fields must be set before calling NewFakeGitHub; do not modify after creation.
type FakeGitHub struct {
	// LatestTag is the version returned by the releases/latest endpoint.
	LatestTag string
	// Payload holds the fake daemon archive; built by BuildDaemonPayload.
	Payload *DaemonPayload
	// CorruptBinary causes the archive download to return wrong content.
	CorruptBinary bool
	// DownloadCut truncates the download response after this many bytes (0 = no cut).
	DownloadCut int
	// StatusCode, if non-zero, overrides all responses with this HTTP status.
	StatusCode int

	server     *httptest.Server
	mu         sync.Mutex
	requestLog []string
}

// NewFakeGitHub starts the fake server and registers t.Cleanup to close it.
func NewFakeGitHub(t *testing.T, cfg *FakeGitHub) *FakeGitHub {
	t.Helper()
	cfg.server = httptest.NewServer(http.HandlerFunc(cfg.handle))
	t.Cleanup(cfg.server.Close)
	return cfg
}

// Client returns an *http.Client that redirects api.github.com and github.com
// traffic to the fake server. Inject into Env.HTTPClient.
func (f *FakeGitHub) Client() *http.Client {
	fakeURL, _ := url.Parse(f.server.URL)
	return &http.Client{
		Transport: &redirectTransport{
			base:    http.DefaultTransport,
			fakeURL: fakeURL,
		},
	}
}

// RequestLog returns a snapshot of all request paths received so far.
func (f *FakeGitHub) RequestLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.requestLog))
	copy(out, f.requestLog)
	return out
}

func (f *FakeGitHub) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requestLog = append(f.requestLog, r.URL.Path)
	f.mu.Unlock()

	if f.StatusCode != 0 {
		http.Error(w, "injected error", f.StatusCode)
		return
	}

	path := r.URL.Path
	switch {
	case strings.Contains(path, "/releases/latest"):
		fmt.Fprintf(w, `{"tag_name":%q}`, f.LatestTag)

	case strings.Contains(path, "SHA256SUMS"):
		checksum := f.Payload.Checksum
		if f.CorruptBinary {
			checksum = CorruptChecksum()
		}
		fmt.Fprintf(w, "%s  %s\n", checksum, f.Payload.AssetName)

	case strings.Contains(path, ".tar.zst") || strings.Contains(path, ".zip"):
		data := f.Payload.Archive
		if f.CorruptBinary {
			data = []byte("this is not a valid archive")
		}
		if f.DownloadCut > 0 && f.DownloadCut < len(data) {
			data = data[:f.DownloadCut]
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)

	default:
		http.NotFound(w, r)
	}
}

// redirectTransport rewrites requests targeting api.github.com or github.com
// to the fake server URL.
type redirectTransport struct {
	base    http.RoundTripper
	fakeURL *url.URL
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Host
	if host == "api.github.com" || host == "github.com" {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = t.fakeURL.Scheme
		clone.URL.Host = t.fakeURL.Host
		clone.Host = t.fakeURL.Host
		return t.base.RoundTrip(clone)
	}
	return t.base.RoundTrip(req)
}
```

- [ ] **Step 3: Build the testfixtures package**

```bash
cd cmd/mxcli-launcher && go build ./testfixtures/... 2>&1
```

Expected: clean build (no errors).

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli-launcher/testfixtures/
git commit -m "test(launcher): add testfixtures package — FakeGitHub + DaemonPayload builder"
```

---

## Task 6: Write core path + throttle tests

**Files:**
- Create: `cmd/mxcli-launcher/install_update_test.go`

- [ ] **Step 1: Create `install_update_test.go` with helper + core path tests**

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

// newInstallEnv creates a self-contained Env + FakeGitHub for install/update tests.
// daemonContent is what gets packaged as the fake binary (e.g. []byte("v0.15.0-binary")).
// Returns both Env (for launcher calls) and FakeGitHub (for RequestLog assertions).
func newInstallEnv(t *testing.T, cfg *testfixtures.FakeGitHub, daemonContent []byte) (*Env, *testfixtures.FakeGitHub) {
	t.Helper()
	payload, err := testfixtures.BuildDaemonPayload(daemonContent)
	if err != nil {
		t.Fatalf("BuildDaemonPayload: %v", err)
	}
	cfg.Payload = payload
	gh := testfixtures.NewFakeGitHub(t, cfg)
	e := &Env{
		HomeDir:    t.TempDir(),
		HTTPClient: gh.Client(),
	}
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}
	return e, gh
}

// writeFakeDaemon plants a fake daemon binary + version file in e's daemon dir.
func writeFakeDaemon(t *testing.T, e *Env, version string) {
	t.Helper()
	if err := os.WriteFile(e.daemonBinaryPath(), []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.daemonVersionPath(), []byte(version+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

// — Core path tests —

func TestDownloadDaemon_FreshInstall(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("v015-binary"))

	if err := e.downloadDaemon(e.daemonBinaryPath()); err != nil {
		t.Fatalf("downloadDaemon: %v", err)
	}

	got, err := os.ReadFile(e.daemonBinaryPath())
	if err != nil {
		t.Fatalf("daemon binary not written: %v", err)
	}
	if string(got) != "v015-binary" {
		t.Errorf("binary content = %q, want %q", got, "v015-binary")
	}
}

func TestDownloadDaemonVersion_AlreadyLatest(t *testing.T) {
	t.Parallel()
	const tag = "v0.15.0"
	e, gh := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: tag}, []byte("binary"))
	writeFakeDaemon(t, e, tag)

	// Simulate what runUpgrade does: fetch latest, compare, skip download if equal.
	latest, err := e.fetchLatestTag()
	if err != nil {
		t.Fatalf("fetchLatestTag: %v", err)
	}
	current := readVersionFile(e.daemonVersionPath())
	if current != latest {
		t.Fatalf("test setup: current=%q latest=%q should match", current, latest)
	}

	// Assert: only the releases/latest endpoint was called — no download or SHA256SUMS.
	for _, path := range gh.RequestLog() {
		if strings.Contains(path, ".tar.zst") || strings.Contains(path, "SHA256SUMS") {
			t.Errorf("unexpected download request when already at latest: %s", path)
		}
	}
}

func TestDownloadDaemonVersion_UpgradeOldToNew(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("v015-binary"))
	writeFakeDaemon(t, e, "v0.14.0")

	// Write bak so backup rename succeeds
	if err := os.Rename(e.daemonBinaryPath(), e.daemonBakPath()); err != nil {
		t.Fatal(err)
	}
	os.Rename(e.daemonVersionPath(), e.daemonVersionBakPath())

	tmpDest := e.daemonBinaryPath() + ".new"
	if err := e.downloadDaemonVersion("v0.15.0", tmpDest); err != nil {
		t.Fatalf("downloadDaemonVersion: %v", err)
	}

	// Simulate the rename that runUpgrade does
	if err := os.Rename(tmpDest, e.daemonBinaryPath()); err != nil {
		t.Fatalf("install rename: %v", err)
	}
	if err := os.WriteFile(e.daemonVersionPath(), []byte("v0.15.0"), 0644); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(e.daemonBinaryPath())
	if string(got) != "v015-binary" {
		t.Errorf("new binary content = %q, want %q", got, "v015-binary")
	}
	bakVer := readVersionFile(e.daemonVersionBakPath())
	if bakVer != "v0.14.0" {
		t.Errorf("bak version = %q, want v0.14.0", bakVer)
	}
	newVer := readVersionFile(e.daemonVersionPath())
	if newVer != "v0.15.0" {
		t.Errorf("new version = %q, want v0.15.0", newVer)
	}
}

// — Throttle tests —

func TestBackgroundVersionCheck_ThrottledWithinHour(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	// Write a last-check timestamp 30 minutes ago
	ts := time.Now().Add(-30 * time.Minute).Unix()
	os.WriteFile(e.daemonLastCheckPath(), []byte(strconv.FormatInt(ts, 10)), 0644)

	// shouldCheckUpdate should return false
	if shouldCheckUpdate(e.daemonLastCheckPath()) {
		t.Error("should not check within 1h")
	}
}

func TestBackgroundVersionCheck_AllowedAfterHour(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	// Write a last-check timestamp 2 hours ago
	ts := time.Now().Add(-2 * time.Hour).Unix()
	os.WriteFile(e.daemonLastCheckPath(), []byte(strconv.FormatInt(ts, 10)), 0644)

	if !shouldCheckUpdate(e.daemonLastCheckPath()) {
		t.Error("should check after 1h")
	}
}

func TestBackgroundVersionCheck_NoLastCheckFile(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))

	// No last-check file — should allow check and write one
	if !shouldCheckUpdate(e.daemonLastCheckPath()) {
		t.Error("should check when last-check absent")
	}
	writeTimestamp(e.daemonLastCheckPath())
	if _, err := os.Stat(e.daemonLastCheckPath()); err != nil {
		t.Error("last-check file should exist after writeTimestamp")
	}
}

func TestBackgroundVersionCheck_WritesUpdateAvailable(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	e.backgroundVersionCheck()

	content, err := os.ReadFile(e.daemonUpdateAvailablePath())
	if err != nil {
		t.Fatalf("update-available not written: %v", err)
	}
	if strings.TrimSpace(string(content)) != "v0.15.0" {
		t.Errorf("update-available content = %q, want v0.15.0", content)
	}
}

func TestBackgroundVersionCheck_SkipsWhenAlreadyLatest(t *testing.T) {
	t.Parallel()
	const tag = "v0.15.0"
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: tag}, []byte("bin"))
	writeFakeDaemon(t, e, tag)

	e.backgroundVersionCheck()

	if _, err := os.Stat(e.daemonUpdateAvailablePath()); !os.IsNotExist(err) {
		t.Error("update-available should not be written when already at latest")
	}
}

```

- [ ] **Step 2: Run core tests**

```bash
cd cmd/mxcli-launcher && go test -run "TestDownloadDaemon|TestBackground" -v ./... 2>&1
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli-launcher/install_update_test.go
git commit -m "test(launcher): core path + throttle scenario tests"
```

---

## Task 7: Write failure recovery tests

**Files:**
- Modify: `cmd/mxcli-launcher/install_update_test.go`

- [ ] **Step 1: Add failure recovery tests**

Append to `install_update_test.go`:

```go
// — Failure recovery tests —

func TestDownloadDaemonVersion_CorruptBinary(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{
		LatestTag:     "v0.15.0",
		CorruptBinary: true,
	}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	originalBin, _ := os.ReadFile(e.daemonBinaryPath())
	originalVer := readVersionFile(e.daemonVersionPath())

	tmpDest := e.daemonBinaryPath() + ".new"
	err := e.downloadDaemonVersion("v0.15.0", tmpDest)
	if err == nil {
		t.Fatal("expected checksum error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("error should mention checksum mismatch, got: %v", err)
	}

	// Original binary and version must be unchanged
	gotBin, _ := os.ReadFile(e.daemonBinaryPath())
	if string(gotBin) != string(originalBin) {
		t.Error("original binary must be untouched after checksum failure")
	}
	if readVersionFile(e.daemonVersionPath()) != originalVer {
		t.Error("original version must be untouched after checksum failure")
	}
	// Temp file must be cleaned up
	if _, err := os.Stat(tmpDest); !os.IsNotExist(err) {
		t.Error("temp download file should be removed after failure")
	}
}

func TestDownloadDaemonVersion_DownloadTruncated(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{
		LatestTag:   "v0.15.0",
		DownloadCut: 128,
	}, []byte("bin"))

	tmpDest := e.daemonBinaryPath() + ".new"
	err := e.downloadDaemonVersion("v0.15.0", tmpDest)
	if err == nil {
		t.Fatal("expected error on truncated download, got nil")
	}
	// Temp files cleaned up
	if _, err2 := os.Stat(tmpDest); !os.IsNotExist(err2) {
		t.Error("temp download file should be removed after truncation")
	}
	archiveTmp := tmpDest + ".archive-dl"
	if _, err2 := os.Stat(archiveTmp); !os.IsNotExist(err2) {
		t.Error("archive temp file should be removed after truncation")
	}
}

func TestDownloadDaemonVersion_GitHub500(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{
		LatestTag:  "v0.15.0",
		StatusCode: 500,
	}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	originalBin, _ := os.ReadFile(e.daemonBinaryPath())

	tmpDest := e.daemonBinaryPath() + ".new"
	err := e.downloadDaemonVersion("v0.15.0", tmpDest)
	if err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}

	gotBin, _ := os.ReadFile(e.daemonBinaryPath())
	if string(gotBin) != string(originalBin) {
		t.Error("original binary must be untouched after HTTP 500")
	}
}

func TestRollback_RestoresBakVersion(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))

	// Set up: current = v0.15.0 binary, bak = v0.14.0 binary
	os.WriteFile(e.daemonBinaryPath(), []byte("new-binary"), 0755)
	os.WriteFile(e.daemonVersionPath(), []byte("v0.15.0"), 0644)
	os.WriteFile(e.daemonBakPath(), []byte("old-binary"), 0755)
	os.WriteFile(e.daemonVersionBakPath(), []byte("v0.14.0"), 0644)

	e.rollback()

	gotBin, _ := os.ReadFile(e.daemonBinaryPath())
	if string(gotBin) != "old-binary" {
		t.Errorf("after rollback binary = %q, want old-binary", gotBin)
	}
	if ver := readVersionFile(e.daemonVersionPath()); ver != "v0.14.0" {
		t.Errorf("after rollback version = %q, want v0.14.0", ver)
	}
}

func TestRollback_NoBak(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))
	os.WriteFile(e.daemonBinaryPath(), []byte("current"), 0755)

	// Should not panic; logs a message but leaves current binary intact
	e.rollback()

	got, _ := os.ReadFile(e.daemonBinaryPath())
	if string(got) != "current" {
		t.Error("current binary should be untouched when no backup exists")
	}
}

func TestFetchAssetChecksum_GitHub500(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{
		LatestTag:  "v0.15.0",
		StatusCode: 500,
	}, []byte("bin"))

	_, err := e.fetchAssetChecksum("v0.15.0", "mxcli-daemon-linux-amd64.tar.zst")
	if err == nil {
		t.Fatal("expected error on HTTP 500 for SHA256SUMS")
	}
}
```

- [ ] **Step 2: Run failure recovery tests**

```bash
cd cmd/mxcli-launcher && go test -run "TestDownloadDaemonVersion|TestRollback|TestFetchAsset" -v ./... 2>&1
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli-launcher/install_update_test.go
git commit -m "test(launcher): failure recovery scenario tests"
```

---

## Task 8: Write concurrent upgrade test

**Files:**
- Modify: `cmd/mxcli-launcher/install_update_test.go`

- [ ] **Step 1: Add concurrent upgrade scenario**

Append to `install_update_test.go`:

```go
// — Concurrent upgrade tests —

func TestRunUpgrade_ConcurrentOnlyOneWins(t *testing.T) {
	t.Parallel()

	// Build a payload that downloadDaemonVersion will accept
	payload, err := testfixtures.BuildDaemonPayload([]byte("v015-binary"))
	if err != nil {
		t.Fatalf("BuildDaemonPayload: %v", err)
	}

	// Two separate Envs sharing the same HomeDir — they race on the same lock file
	home := t.TempDir()
	makeEnv := func() *Env {
		gh := testfixtures.NewFakeGitHub(t, &testfixtures.FakeGitHub{
			LatestTag: "v0.15.0",
			Payload:   payload,
		})
		e := &Env{HomeDir: home, HTTPClient: gh.Client()}
		return e
	}

	e1 := makeEnv()
	e2 := makeEnv()
	if err := os.MkdirAll(e1.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}
	// Plant current version
	os.WriteFile(e1.daemonBinaryPath(), []byte("old"), 0755)
	os.WriteFile(e1.daemonVersionPath(), []byte("v0.14.0"), 0644)

	results := make([]error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); results[0] = e1.acquireUpgradeLock() }()
	go func() { defer wg.Done(); results[1] = e2.acquireUpgradeLock() }()
	wg.Wait()

	successes := 0
	for i, err := range results {
		if err == nil {
			successes++
			if i == 0 {
				e1.releaseUpgradeLock()
			} else {
				e2.releaseUpgradeLock()
			}
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 lock acquisition, got %d", successes)
	}
}
```

Add `"sync"` to the imports at the top of `install_update_test.go`.

- [ ] **Step 2: Run concurrent test 10 times to check for flakiness**

```bash
cd cmd/mxcli-launcher && go test -run TestRunUpgrade_ConcurrentOnlyOneWins -v -count=10 -race ./... 2>&1
```

Expected: 10 PASS, no race conditions.

- [ ] **Step 3: Run complete test suite**

```bash
cd cmd/mxcli-launcher && go test ./... -race 2>&1
```

Expected: all pass, no races.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli-launcher/install_update_test.go
git commit -m "test(launcher): concurrent upgrade lock scenario"
```

---

## Task 9: Final verification

- [ ] **Step 1: Build everything**

```bash
make build 2>&1 | tail -5
```

Expected: `bin/mxcli` present, no errors.

- [ ] **Step 2: Run all launcher tests with race detector**

```bash
cd cmd/mxcli-launcher && go test ./... -race -count=3 -v 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected: all `ok`, no `FAIL`.

- [ ] **Step 3: Run full test suite**

```bash
make test 2>&1 | tail -10
```

Expected: no failures.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test(launcher): install/update test fixture implementation complete"
```
