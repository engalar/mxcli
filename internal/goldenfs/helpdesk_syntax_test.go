// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestHelpdeskMDL_SyntaxCheck runs `mxcli check` against both helpdesk MDL
// source files and expects a zero exit code from each. The test skips
// gracefully when the mxcli binary is not yet built (set MXCLI_BINARY or
// run make build).
func TestHelpdeskMDL_SyntaxCheck(t *testing.T) {
	mxcliBin := findMxcliBinaryForTest(t)
	if mxcliBin == "" {
		t.Skip("mxcli binary not available — build first with `make build` or set MXCLI_BINARY")
	}

	root := repoRoot(t)
	versions := []string{"11.6.6", "11.10.0", "11.12.1"}
	files := []string{
		filepath.Join(root, "mdl-examples", "use-cases", "helpdesk", "helpdesk-app.mdl"),
	}
	for _, v := range versions {
		files = append(files, filepath.Join(root, "testdata", "helpdesk-golden-"+v, "describe-snapshot.mdl"))
	}

	for _, f := range files {
		f := f
		name := filepath.Base(filepath.Dir(f)) + "/" + filepath.Base(f)
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(f); err != nil {
				t.Skipf("MDL file not found: %s — skip (expected pre-Phase 2)", f)
			}
			cmd := exec.Command(mxcliBin, "check", f)
			output, _ := cmd.CombinedOutput()
			t.Logf("mxcli check output:\n%s", output)
			if cmd.ProcessState.ExitCode() != 0 {
				t.Errorf("mxcli check failed for %s (exit %d) — see output above",
					filepath.Base(f), cmd.ProcessState.ExitCode())
			}
		})
	}
}
