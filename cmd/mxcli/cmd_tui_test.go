// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

func TestResolveMxcliPath_PrefersEnvVar(t *testing.T) {
	t.Setenv(launcherproto.EnvLauncherPath, "/custom/launcher/mxcli")
	got := resolveMxcliPath()
	if got != "/custom/launcher/mxcli" {
		t.Errorf("got %q, want /custom/launcher/mxcli", got)
	}
}

func TestResolveMxcliPath_FallsBackToExecutable(t *testing.T) {
	t.Setenv(launcherproto.EnvLauncherPath, "")
	got := resolveMxcliPath()
	exe, _ := os.Executable()
	if got != exe {
		t.Errorf("got %q, want os.Executable() = %q", got, exe)
	}
}
