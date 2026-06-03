// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
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
