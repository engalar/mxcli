// cmd/mxcli/docker/local_test.go
// SPDX-License-Identifier: Apache-2.0

package docker_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func TestResolveMxInstallPath_MxBuildCacheExactVersion(t *testing.T) {
	home := t.TempDir()
	version := "99.0.0"
	launcherDir := filepath.Join(home, ".mxcli", "mxbuild", version, "runtime", "launcher")
	os.MkdirAll(launcherDir, 0755)
	os.WriteFile(filepath.Join(launcherDir, "runtimelauncher.jar"), []byte("fake"), 0644)

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MX_INSTALL_PATH", "")

	got, err := docker.ResolveMxInstallPathForVersion(version)
	if err != nil {
		t.Fatalf("expected to find fake cache, got error: %v", err)
	}
	want := filepath.Join(home, ".mxcli", "mxbuild", version)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMxInstallPath_MxBuildCacheNewest(t *testing.T) {
	home := t.TempDir()
	for _, v := range []string{"10.0.0", "11.0.0"} {
		launcherDir := filepath.Join(home, ".mxcli", "mxbuild", v, "runtime", "launcher")
		os.MkdirAll(launcherDir, 0755)
		os.WriteFile(filepath.Join(launcherDir, "runtimelauncher.jar"), []byte("fake"), 0644)
	}

	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MX_INSTALL_PATH", "")

	got, err := docker.ResolveMxInstallPathForVersion("")
	if err != nil {
		t.Fatalf("expected to find newest cache, got error: %v", err)
	}
	want := filepath.Join(home, ".mxcli", "mxbuild", "11.0.0")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
