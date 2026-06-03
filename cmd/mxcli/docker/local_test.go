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
