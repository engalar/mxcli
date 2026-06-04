// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

func TestTTYEnv_ContainsLauncherPath(t *testing.T) {
	env := ttyEnv("/usr/local/bin/mxcli")
	want := launcherproto.EnvLauncherPath + "=/usr/local/bin/mxcli"
	for _, e := range env {
		if e == want {
			return
		}
	}
	t.Errorf("%q not found in ttyEnv output", want)
}

func TestTTYEnv_IncludesCurrentEnv(t *testing.T) {
	t.Setenv("MXCLI_TEST_MARKER", "hello")
	env := ttyEnv("/some/mxcli")
	for _, e := range env {
		if e == "MXCLI_TEST_MARKER=hello" {
			return
		}
	}
	t.Error("current process env not propagated by ttyEnv")
}

func TestTTYEnv_LauncherPathOverridesExisting(t *testing.T) {
	// If caller already had MXCLI_LAUNCHER_PATH set to something else,
	// ttyEnv must set the correct (new) launcher path.
	t.Setenv(launcherproto.EnvLauncherPath, "/old/path")
	env := ttyEnv("/new/path")
	want := launcherproto.EnvLauncherPath + "=/new/path"
	for _, e := range env {
		if strings.HasPrefix(e, launcherproto.EnvLauncherPath+"=") {
			if e == want {
				return
			}
			t.Errorf("got %q, want %q", e, want)
			return
		}
	}
	t.Errorf("%q not found in ttyEnv output", want)
	_ = os.Getenv // suppress unused import lint noise
}
