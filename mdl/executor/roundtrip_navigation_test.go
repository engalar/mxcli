// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"testing"
)

// TestRoundtrip_Navigation_Syntax verifies that DESCRIBE NAVIGATION produces
// valid, parseable MDL for the Responsive profile that every Mendix project
// ships with by default.
func TestRoundtrip_Navigation_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	mdl := env.rtDescribe("describe navigation Responsive")
	rtAssertParseOK(t, mdl)
}

// TestRoundtrip_Navigation_Semantic verifies that re-importing the described
// MDL produces identical output (CREATE OR REPLACE is idempotent).
func TestRoundtrip_Navigation_Semantic(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	env.rtAssertSemantic("describe navigation Responsive")
}
