// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// TestRoundtrip_Layout_Syntax verifies that DESCRIBE LAYOUT emits executable
// `create or modify layout` MDL (not comment-only informational output) and
// that the emitted MDL parses cleanly back through the visitor.
func TestRoundtrip_Layout_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()

	out := env.rtDescribe("describe layout Atlas_Core.Atlas_Default")
	if strings.TrimSpace(out) == "" {
		t.Fatal("describe layout returned empty output")
	}
	if !strings.Contains(out, "create or modify layout") {
		t.Errorf("describe layout is not executable MDL:\n%s", out)
	}
	if !strings.Contains(out, "scrollcontainer") {
		t.Errorf("describe layout missing scrollcontainer:\n%s", out)
	}
	if !strings.Contains(out, "placeholder") {
		t.Errorf("describe layout missing placeholder:\n%s", out)
	}

	// The describe output must itself parse as valid MDL.
	if _, errs := visitor.Build(out); len(errs) > 0 {
		t.Errorf("describe layout output is not valid MDL: %v\n%s", errs, out)
	}
}
