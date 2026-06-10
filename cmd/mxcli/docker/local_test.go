// cmd/mxcli/docker/local_test.go
// SPDX-License-Identifier: Apache-2.0

package docker_test

import (
	"fmt"
	"net"
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
