// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"testing"
)

// TestRoundtrip_Settings_Syntax verifies that DESCRIBE SETTINGS produces
// valid, parseable MDL for the project settings that every Mendix project
// contains by default (model, configuration, language, workflow sections).
func TestRoundtrip_Settings_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	mdl := env.rtDescribe("describe settings")
	rtAssertParseOK(t, mdl)
}

// TestRoundtrip_Settings_Semantic verifies that re-applying the described
// settings MDL produces identical output (ALTER SETTINGS is idempotent).
func TestRoundtrip_Settings_Semantic(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	env.rtAssertSemantic("describe settings")
}
