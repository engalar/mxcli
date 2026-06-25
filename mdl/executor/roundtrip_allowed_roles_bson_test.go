// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestPageAllowedModuleRolesBSON verifies that the AllowedModuleRoles BSON field
// is injected when granting view access on a page, and that it is preserved by
// create or replace page (regression guard for pageWriter injecting this field).
//
// Regression test for CE0557: Mendix's CE0557 validator checks
// AllowedModuleRoles (BSON version int32(1)), not AllowedRoles (version 3).
// If AllowedModuleRoles is missing from the page unit, mx check reports CE0557
// even when AllowedRoles is correctly set.
func TestPageAllowedModuleRolesBSON(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	backend := env.executor.Backend()
	pageName := "RoundtripTest.AllowedRolesBSON"

	// ── Step 1: Create a module role ────────────────────────────────────────
	if err := env.executeMDL("create module role RoundtripTest.Viewer;"); err != nil {
		t.Fatalf("create module role: %v", err)
	}

	// ── Step 2: Create a page (no security set) ────────────────────────────
	if err := env.executeMDL(`create page ` + pageName + ` (Title: 'Test', Layout: Atlas_Core.Atlas_Default) {
		container c { dynamictext txt (Content: 'Hello') }
	}`); err != nil {
		t.Fatalf("create page: %v", err)
	}

	// ── Step 3: Read raw BSON before grant ─────────────────────────────────
	info, err := backend.GetRawUnitByName("page", pageName)
	if err != nil {
		t.Fatalf("GetRawUnitByName: %v", err)
	}
	if info == nil {
		t.Fatal("page info is nil")
	}

	rawBefore, err := backend.GetRawUnitBytes(model.ID(info.ID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}

	hasAllowedRolesBefore := rawFieldExists(rawBefore, "AllowedRoles")
	hasAllowedModuleRolesBefore := rawFieldExists(rawBefore, "AllowedModuleRoles")
	t.Logf("Before grant: AllowedRoles=%v AllowedModuleRoles=%v",
		hasAllowedRolesBefore, hasAllowedModuleRolesBefore)

	// ── Step 4: Grant view access ──────────────────────────────────────────
	if err := env.executeMDL(`grant view on page ` + pageName + ` to RoundtripTest.Viewer;`); err != nil {
		t.Fatalf("grant view: %v", err)
	}

	// ── Step 5: Read raw BSON after grant ──────────────────────────────────
	rawAfter, err := backend.GetRawUnitBytes(model.ID(info.ID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes after grant: %v", err)
	}

	hasAllowedRolesAfter := rawFieldExists(rawAfter, "AllowedRoles")
	hasAllowedModuleRolesAfter := rawFieldExists(rawAfter, "AllowedModuleRoles")
	t.Logf("After grant: AllowedRoles=%v AllowedModuleRoles=%v",
		hasAllowedRolesAfter, hasAllowedModuleRolesAfter)

	if !hasAllowedRolesAfter {
		t.Error("AllowedRoles BSON field missing after GRANT VIEW ON PAGE")
	}
	if !hasAllowedModuleRolesAfter {
		t.Error("CE0557 ROOT CAUSE: AllowedModuleRoles BSON field missing after GRANT VIEW ON PAGE — mx check will reject this page")
	}

	// Dump the BSON fields for debugging
	logBSONFields(t, rawAfter)

	// ── Step 6: Create or replace page (simulates second deployment) ───────
	if err := env.executeMDL(`create or replace page ` + pageName + ` (Title: 'Test', Layout: Atlas_Core.Atlas_Default) {
		container c { dynamictext txt (Content: 'World') }
	}`); err != nil {
		t.Fatalf("create or replace page: %v", err)
	}

	// ── Step 7: Read raw BSON after replace (before re-grant) ──────────────
	rawAfterReplace, err := backend.GetRawUnitBytes(model.ID(info.ID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes after replace: %v", err)
	}

	hasAllowedRolesAfterReplace := rawFieldExists(rawAfterReplace, "AllowedRoles")
	hasAllowedModuleRolesAfterReplace := rawFieldExists(rawAfterReplace, "AllowedModuleRoles")
	t.Logf("After replace (no re-grant): AllowedRoles=%v AllowedModuleRoles=%v",
		hasAllowedRolesAfterReplace, hasAllowedModuleRolesAfterReplace)

	if !hasAllowedRolesAfterReplace {
		t.Error("AllowedRoles missing after create or replace page — preserveAllowedRoles not working")
	}
	if !hasAllowedModuleRolesAfterReplace {
		t.Error("CE0557 FIX BROKEN: AllowedModuleRoles stripped by create or replace page — pageWriter must inject this field")
	}

	// ── Step 8: Re-grant view access ───────────────────────────────────────
	if err := env.executeMDL(`grant view on page ` + pageName + ` to RoundtripTest.Viewer;`); err != nil {
		t.Fatalf("re-grant view: %v", err)
	}

	// ── Step 9: Read raw BSON after re-grant ───────────────────────────────
	rawAfterRegrant, err := backend.GetRawUnitBytes(model.ID(info.ID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes after re-grant: %v", err)
	}

	hasAllowedRolesAfterRegrant := rawFieldExists(rawAfterRegrant, "AllowedRoles")
	hasAllowedModuleRolesAfterRegrant := rawFieldExists(rawAfterRegrant, "AllowedModuleRoles")
	t.Logf("After re-grant: AllowedRoles=%v AllowedModuleRoles=%v",
		hasAllowedRolesAfterRegrant, hasAllowedModuleRolesAfterRegrant)

	if !hasAllowedModuleRolesAfterRegrant {
		t.Error("CE0557 ROOT CAUSE: AllowedModuleRoles missing after create or replace + re-grant")
	}
}

// rawFieldExists checks if a named field exists in a raw BSON document.
func rawFieldExists(raw []byte, field string) bool {
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return false
	}
	for _, e := range doc {
		if e.Key == field {
			return true
		}
	}
	return false
}

// logBSONFields logs all top-level BSON field keys for debugging.
func logBSONFields(t *testing.T, raw []byte) {
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Logf("bson unmarshal error: %v", err)
		return
	}
	for _, e := range doc {
		t.Logf("  BSON field: %s (type: %T)", e.Key, e.Value)
	}
}
