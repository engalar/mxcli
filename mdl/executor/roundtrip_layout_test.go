// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"
)

// TestRoundtrip_Layout_Syntax verifies that DESCRIBE LAYOUT returns a
// non-error response for a built-in Atlas layout. Layouts cannot be created
// via MDL (only in Studio Pro), so the describe output is informational
// comments only — no full-round-trip syntax test is possible.
func TestRoundtrip_Layout_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	out := env.rtDescribe("describe layout Atlas_Core.Atlas_Default")
	// The output should mention the layout name; it is comment-only MDL (no
	// executable statements), so we only verify a non-empty, name-containing
	// response rather than full parse correctness.
	if strings.TrimSpace(out) == "" {
		t.Error("describe layout returned empty output")
	}
	if !strings.Contains(out, "Atlas_Default") {
		t.Errorf("describe layout output does not mention layout name:\n%s", out)
	}
}

// TestRoundtrip_Layout_Semantic is skipped because layouts cannot be created
// or modified via MDL — they must be authored in Mendix Studio Pro.
// Full semantic round-trip testing is therefore not applicable.
func TestRoundtrip_Layout_Semantic(t *testing.T) {
	t.Skip("layouts are read-only via MDL — no CREATE statement available for semantic roundtrip")
}
