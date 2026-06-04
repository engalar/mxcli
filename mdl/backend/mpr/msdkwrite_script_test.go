// SPDX-License-Identifier: Apache-2.0

// Regression test for msdkWrite bypassing scriptBuf during EXECUTE SCRIPT.
//
// Bug: msdkWrite used UpdateRawUnit which skips scriptBuf. This caused two
// problems when a script transaction was active:
//   1. ScriptOverlay was never updated → each msdkWrite read stale data
//   2. commitScriptBuffer BatchWrite overwrote the UpdateRawUnit SQLite writes
//      with the old ScriptBuffer content (pre-GRANT domain model)
//
// Net result: ALL security grants applied inside EXECUTE SCRIPT were silently
// discarded on commit, producing CE2729 "No access" errors in mx check.
package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// TestMsdkWrite_ScriptTransaction_CumulativeGrants verifies that two consecutive
// addEntityAccessRuleViaModelsdk calls inside a BeginScriptTransaction block
// both survive Commit — i.e., neither overwrites the other.
//
// The entity is created INSIDE the transaction so that the domain model write
// goes through scriptBuf (SetScriptOverlay). Before the fix, subsequent
// msdkWrite calls (for GRANTs) use UpdateRawUnit which bypasses scriptBuf,
// leaving the ScriptOverlay stale. This causes:
//   1. Each GRANT reads the pre-GRANT ScriptOverlay, so they overwrite each other.
//   2. commitScriptBuffer BatchWrite overwrites the SQLite writes with the
//      stale ScriptBuffer content, wiping all GRANTs.
func TestMsdkWrite_ScriptTransaction_CumulativeGrants(t *testing.T) {
	// Create a bare MPR (no entity yet) so we can add the entity inside the tx.
	mprPath, dmID := makeDomainModelTestMPR(t)
	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = b.Disconnect() })

	tx, err := b.BeginScriptTransaction()
	if err != nil {
		t.Fatalf("BeginScriptTransaction: %v", err)
	}

	// CREATE ENTITY inside the transaction: this triggers writeUnitContents →
	// scriptBuf.AddUpdate → SetScriptOverlay. The domain model is now only
	// visible through the ScriptOverlay.
	entity := genDm.NewEntity()
	entity.SetName("Customer")
	ng := genDm.NewNoGeneralization()
	ng.SetPersistable(true)
	entity.SetGeneralization(ng)
	if err := b.CreateEntityGen(dmID, entity); err != nil {
		t.Fatalf("CreateEntityGen inside tx: %v", err)
	}

	// First GRANT: RoleA — reads from ScriptOverlay (correct), writes v2.
	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.RoleA"},
		false, false, "ReadOnly", "",
		[]types.EntityMemberAccess{},
	); err != nil {
		t.Fatalf("addEntityAccessRuleViaModelsdk RoleA: %v", err)
	}

	// Second GRANT: RoleB — must see v2 (with RoleA), not the stale ScriptOverlay (v1).
	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.RoleB"},
		false, false, "ReadWrite", "",
		[]types.EntityMemberAccess{},
	); err != nil {
		t.Fatalf("addEntityAccessRuleViaModelsdk RoleB: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// After commit, ScriptOverlay is cleared; reads go to SQLite/disk.
	// findCustomerAccessRules filters out non-bson.D items (including the
	// versioned-array int32 prefix), so len(rules) == number of real access rules.
	rules := findCustomerAccessRules(t, b, dmID)
	if got := len(rules); got != 2 {
		t.Fatalf("AccessRules count after Commit = %d, want 2 (both RoleA and RoleB must survive)", got)
	}

	rolesFound := map[string]bool{}
	for _, rule := range rules {
		mr, _ := ruleField(t, rule, "AllowedModuleRoles").(bson.A)
		for _, r := range mr {
			if s, ok := r.(string); ok {
				rolesFound[s] = true
			}
		}
	}
	if !rolesFound["TestModule.RoleA"] {
		t.Error("RoleA access rule missing after Commit")
	}
	if !rolesFound["TestModule.RoleB"] {
		t.Error("RoleB access rule missing after Commit")
	}
}
