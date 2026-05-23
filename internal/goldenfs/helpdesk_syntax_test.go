// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelpdeskMDL_SyntaxCheck runs `mxcli check` against both helpdesk MDL
// source files and expects zero errors from each. The test skips gracefully
// when the mxcli binary is not yet built (set MXCLI_BINARY or run make build).
func TestHelpdeskMDL_SyntaxCheck(t *testing.T) {
	mxcliBin := findMxcliBinaryForTest(t)
	if mxcliBin == "" {
		t.Skip("mxcli binary not available — build first with `make build` or set MXCLI_BINARY")
	}

	root := repoRoot(t)
	files := []string{
		filepath.Join(root, "mdl-examples", "use-cases", "helpdesk", "helpdesk-app.mdl"),
		filepath.Join(root, "testdata", "helpdesk-golden", "describe-snapshot.mdl"),
	}

	for _, f := range files {
		f := f
		name := filepath.Base(filepath.Dir(f)) + "/" + filepath.Base(f)
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(f); err != nil {
				t.Fatalf("MDL file not found: %s", f)
			}
			cmd := exec.Command(mxcliBin, "check", f)
			output, _ := cmd.CombinedOutput()
			out := string(output)
			t.Logf("mxcli check output:\n%s", out)
			var errLines []string
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "[error]") || strings.HasPrefix(line, "Error") {
					errLines = append(errLines, line)
				}
			}
			if len(errLines) > 0 {
				t.Errorf("mxcli check found %d error(s) in %s:\n%s",
					len(errLines), filepath.Base(f), strings.Join(errLines, "\n"))
			}
		})
	}
}
