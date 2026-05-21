// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"testing"
)

// TestRoundtrip_UserRole_Syntax verifies that DESCRIBE USER ROLE produces
// valid, parseable MDL for the BasicUser and PowerUser roles seeded in
// testdata/roundtrip/roundtrip.mpr.
func TestRoundtrip_UserRole_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	for _, role := range []string{"BasicUser", "PowerUser"} {
		mdl := env.rtDescribe("describe user role '" + role + "'")
		rtAssertParseOK(t, mdl)
	}
}

// TestRoundtrip_UserRole_Semantic verifies that DROP + CREATE produces
// identical DESCRIBE output. We cannot use rtAssertSemantic directly because
// "create user role" is not idempotent — there is no "create or modify" form.
func TestRoundtrip_UserRole_Semantic(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	for _, role := range []string{"BasicUser", "PowerUser"} {
		mdl1 := env.rtDescribe("describe user role '" + role + "'")
		rtAssertParseOK(t, mdl1)

		// Drop then recreate so the CREATE succeeds on re-import.
		if err := env.executeMDL("drop user role " + role + ";"); err != nil {
			t.Fatalf("drop user role %s: %v", role, err)
		}
		if err := env.executeMDL(mdl1); err != nil {
			t.Fatalf("re-import user role %s: %v\nMDL:\n%s", role, err, mdl1)
		}

		mdl2 := env.rtDescribe("describe user role '" + role + "'")
		n1 := normalizeRoundtrip(mdl1)
		n2 := normalizeRoundtrip(mdl2)
		if n1 != n2 {
			diff := diffStrings(n1, n2)
			t.Errorf("user role %s: round-trip not idempotent:\n%s", role, diff)
		}
	}
}
