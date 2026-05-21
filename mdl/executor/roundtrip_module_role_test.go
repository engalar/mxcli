// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestRoundtrip_ModuleRole_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	for _, role := range []string{"RoundtripModule.Viewer", "RoundtripModule.Editor"} {
		mdl := env.rtDescribe("describe module role " + role)
		rtAssertParseOK(t, mdl)
	}
}

// TestRoundtrip_ModuleRole_Semantic verifies that drop+recreate produces
// identical describe output. We cannot use rtAssertSemantic directly because
// "create module role" is not idempotent — the grammar has no
// "create or modify module role" form.
func TestRoundtrip_ModuleRole_Semantic(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	for _, role := range []string{"RoundtripModule.Viewer", "RoundtripModule.Editor"} {
		mdl1 := env.rtDescribe("describe module role " + role)
		rtAssertParseOK(t, mdl1)

		// Drop then recreate so the create succeeds on re-import.
		dropMDL := "drop module role " + role + ";"
		if err := env.executeMDL(dropMDL); err != nil {
			t.Fatalf("drop module role %s: %v", role, err)
		}
		if err := env.executeMDL(mdl1); err != nil {
			t.Fatalf("re-import module role %s: %v\nMDL:\n%s", role, err, mdl1)
		}

		mdl2 := env.rtDescribe("describe module role " + role)
		n1 := normalizeRoundtrip(mdl1)
		n2 := normalizeRoundtrip(mdl2)
		if n1 != n2 {
			diff := diffStrings(n1, n2)
			t.Errorf("module role %s: round-trip not idempotent:\n%s", role, diff)
		}
	}
}

func TestRoundtrip_ModuleRole_Storage(t *testing.T) {
	env := setupRoundtripEnv(t)
	snap := snapshotMPR(t, env.projectPath)
	for _, role := range []string{"RoundtripModule.Viewer", "RoundtripModule.Editor"} {
		mdl := env.rtDescribe("describe module role " + role)
		// Drop then recreate so the create does not fail with "already exists".
		dropMDL := "drop module role " + role + ";"
		if err := env.executeMDL(dropMDL); err != nil {
			t.Fatalf("drop module role %s before re-import: %v", role, err)
		}
		if err := env.executeMDL(mdl); err != nil {
			t.Fatalf("re-import module role %s: %v", role, err)
		}
	}
	env.teardown()
	bsoncompare.AssertEqual(t, snap, env.projectPath,
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)
}

// TestRoundtrip_ModuleRole_GrantNotSkipped is a regression test for Bug 4:
// verifies that GRANT statements in entity MDL are not silently skipped when
// module roles already exist.
func TestRoundtrip_ModuleRole_GrantNotSkipped(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	entityMDL := env.rtDescribe("describe entity RoundtripModule.Item")
	if err := env.executeMDL(entityMDL); err != nil {
		t.Fatalf("re-import entity MDL: %v", err)
	}
	entityMDL2 := env.rtDescribe("describe entity RoundtripModule.Item")
	if !strings.Contains(entityMDL2, "grant") {
		t.Error("expected GRANT statements in re-described entity — grants may have been dropped")
	}
}
