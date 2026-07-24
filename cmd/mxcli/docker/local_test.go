// SPDX-License-Identifier: Apache-2.0

package docker_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

// CaptureStarter records the exec.Cmd passed to it and returns nil.
type CaptureStarter struct {
	Cmd *exec.Cmd
}

func (c *CaptureStarter) Run(cmd *exec.Cmd) error {
	c.Cmd = cmd
	return nil
}

// createDeployLayout creates a minimal deployment/ directory structure for tests.
// Returns the deployment directory path.
func createDeployLayout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	modelDir := filepath.Join(dir, "model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatalf("mkdir model: %v", err)
	}

	configJSON := `{"Configuration":{"DatabaseType":"HSQLDB","DatabaseName":"default"},"Constants":{},"AdminPassword":""}`
	if err := os.WriteFile(filepath.Join(modelDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	modelMDP := []byte("fake model.mdp")
	if err := os.WriteFile(filepath.Join(modelDir, "model.mdp"), modelMDP, 0644); err != nil {
		t.Fatalf("write model.mdp: %v", err)
	}

	return dir
}

func TestStartLocal_MissingDeploy_ReturnsError(t *testing.T) {
	cs := &CaptureStarter{}
	err := docker.StartLocal(docker.LocalRunOptions{
		DeployDir: t.TempDir(), // empty dir, no deployment layout
		Starter:   cs,
	})
	if err == nil {
		t.Fatal("expected error for missing deployment")
	}
}

func TestStartLocal_GeneratesHoconConfig(t *testing.T) {
	deployDir := createDeployLayout(t)
	cs := &CaptureStarter{}
	appPort, adminPort := docker.FindAvailablePorts(8080, 8090)

	err := docker.StartLocal(docker.LocalRunOptions{
		DeployDir: deployDir,
		Starter:   cs,
		AppPort:   appPort,
		AdminPort: adminPort,
	})
	if err != nil {
		t.Fatalf("StartLocal: %v", err)
	}
	if cs.Cmd == nil {
		t.Fatal("expected Cmd to be set")
	}

	// The last argument should be a HOCON config file path
	args := cs.Cmd.Args
	if len(args) < 2 {
		t.Fatalf("expected at least 2 args, got %d: %v", len(args), args)
	}
	hoconArg := args[len(args)-1]
	if !strings.HasSuffix(hoconArg, ".conf") {
		t.Errorf("expected last arg to be a .conf file, got %s", hoconArg)
	}
}

func TestStartLocal_LaunchesJavaWithLauncher(t *testing.T) {
	deployDir := createDeployLayout(t)
	cs := &CaptureStarter{}
	appPort, adminPort := docker.FindAvailablePorts(8080, 8090)

	err := docker.StartLocal(docker.LocalRunOptions{
		DeployDir: deployDir,
		Starter:   cs,
		AppPort:   appPort,
		AdminPort: adminPort,
	})
	if err != nil {
		t.Fatalf("StartLocal: %v", err)
	}

	// Should be java -jar runtimelauncher.jar <deployDir> <config>
	if cs.Cmd == nil {
		t.Fatal("expected Cmd")
	}
	if !strings.Contains(cs.Cmd.Path, "java") && !strings.Contains(cs.Cmd.Path, "java") {
		t.Errorf("expected java executable, got %s", cs.Cmd.Path)
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

func TestResolveRunDir_ReturnsDeployment(t *testing.T) {
	projectDir := t.TempDir()
	want := filepath.Join(projectDir, "deployment")
	got := docker.ResolveRunDir(projectDir)
	if got != want {
		t.Errorf("ResolveRunDir(%s) = %s, want %s", projectDir, got, want)
	}
}

func TestFindAvailablePorts_ReturnsFreePair(t *testing.T) {
	appPort, adminPort := docker.FindAvailablePorts(8080, 8090)
	if appPort == 0 || adminPort == 0 {
		t.Error("expected non-zero ports")
	}
	if appPort >= adminPort {
		t.Errorf("expected appPort < adminPort, got %d >= %d", appPort, adminPort)
	}
}

func ExampleFindAvailablePorts() {
	appPort, adminPort := docker.FindAvailablePorts(8080, 8090)
	fmt.Printf("app=%d admin=%d\n", appPort, adminPort)
}
