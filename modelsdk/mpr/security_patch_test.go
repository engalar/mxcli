// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// buildTestDomainModel constructs a minimal DomainModel BSON document for
// testing PatchReconcileMemberAccesses.  entityID is used as the $ID for the
// single entity; the entity owns every association whose ParentPointer equals
// entityID (from assocs) and every cross-module association whose ParentPointer
// equals entityID (from crossAssocs).
//
// attrNames lists attribute names already present in the entity's Attributes
// array.  ruleMA is the raw MemberAccesses bson.A to embed in the AccessRule
// (pass nil or empty for "empty access rule").
func buildTestDomainModel(
	entityName string,
	entityID string,
	attrNames []string,
	ruleMA bson.A,
	assocs []bson.D,
	crossAssocs []bson.D,
) []byte {
	attrs := bson.A{int32(3)}
	for _, name := range attrNames {
		attrs = append(attrs, bson.D{
			{Key: "$ID", Value: entityID + "-attr-" + name},
			{Key: "$Type", Value: "DomainModels$StoredValue"},
			{Key: "Name", Value: name},
		})
	}

	maArr := bson.A{int32(3)}
	if ruleMA != nil {
		for _, item := range ruleMA {
			maArr = append(maArr, item)
		}
	}

	rule := bson.D{
		{Key: "$ID", Value: entityID + "-rule"},
		{Key: "$Type", Value: "DomainModels$AccessRule"},
		{Key: "AllowCreate", Value: true},
		{Key: "AllowDelete", Value: false},
		{Key: "DefaultMemberAccessRights", Value: "ReadWrite"},
		{Key: "AllowedModuleRoles", Value: bson.A{int32(1), "TestModule.User"}},
		{Key: "MemberAccesses", Value: maArr},
	}

	entity := bson.D{
		{Key: "$ID", Value: entityID},
		{Key: "$Type", Value: "DomainModels$Entity"},
		{Key: "Name", Value: entityName},
		{Key: "Attributes", Value: attrs},
		{Key: "AccessRules", Value: bson.A{int32(3), rule}},
	}

	assocsArr := bson.A{int32(3)}
	for _, a := range assocs {
		assocsArr = append(assocsArr, a)
	}

	crossArr := bson.A{int32(3)}
	for _, ca := range crossAssocs {
		crossArr = append(crossArr, ca)
	}

	dm := bson.D{
		{Key: "$Type", Value: "DomainModels$DomainModel"},
		{Key: "Entities", Value: bson.A{int32(3), entity}},
		{Key: "Associations", Value: assocsArr},
		{Key: "CrossAssociations", Value: crossArr},
	}

	out, err := bson.Marshal(dm)
	if err != nil {
		panic(fmt.Sprintf("buildTestDomainModel marshal: %v", err))
	}
	return out
}

func assocDoc(name, parentID string) bson.D {
	return bson.D{
		{Key: "$ID", Value: name + "-id"},
		{Key: "$Type", Value: "DomainModels$Association"},
		{Key: "Name", Value: name},
		{Key: "ParentPointer", Value: parentID},
		{Key: "ChildPointer", Value: "some-child-id"},
	}
}

func crossAssocDoc(name, parentID string) bson.D {
	return bson.D{
		{Key: "$ID", Value: name + "-id"},
		{Key: "$Type", Value: "DomainModels$CrossAssociation"},
		{Key: "Name", Value: name},
		{Key: "ParentPointer", Value: parentID},
		{Key: "Child", Value: "Other.Entity"},
	}
}

// TestPatchReconcile_EmptyMemberAccesses_AddsAssociation is the core regression
// test for the len(maArr)<=1 silent-skip bug.
//
// When an entity has an AccessRule whose MemberAccesses contains ONLY the
// int32 version-prefix (len==1), the reconciler must still add MemberAccess
// entries for every owned association — it must NOT silently skip.
//
// Before the fix this test fails because the guard `if len(maArr) <= 1 { break }`
// exits before any work is done, returning 0 changes and leaving the BSON
// untouched.  After the fix it passes.
func TestPatchReconcile_EmptyMemberAccesses_AddsAssociation(t *testing.T) {
	const moduleName = "TestModule"
	const entityName = "MySetting"
	const entityID = "entity-mysetting"
	const assocName = "MySetting_Account"

	// MemberAccesses has ONLY the version prefix — the silent-skip scenario.
	// Use a regular (same-module) association so we test the add-assoc path
	// without interference from the CrossAssociation exclusion.
	raw := buildTestDomainModel(
		entityName, entityID,
		[]string{"Theme", "Mode"},               // entity has attributes
		nil,                                     // MemberAccesses is empty (will be [v3])
		[]bson.D{assocDoc(assocName, entityID)}, // regular assoc
		nil,
	)

	patched, changes, err := PatchReconcileMemberAccesses(raw, moduleName)
	if err != nil {
		t.Fatalf("PatchReconcileMemberAccesses returned error: %v", err)
	}

	if len(changes) == 0 {
		t.Fatalf(
			"PatchReconcileMemberAccesses returned 0 changes for entity %q with owned association %q and empty MemberAccesses.\n"+
				"Expected at least one ReconcileChange — the len(maArr)<=1 guard is silently skipping this entity.\n"+
				"Fix: remove the guard so reconciliation runs even when MemberAccesses contains only the version prefix.",
			entityName, assocName,
		)
	}

	found := false
	for _, ch := range changes {
		if ch.Entity == entityName && ch.Member == assocName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf(
			"changes does not contain entry {Entity:%q, Member:%q}; got: %v",
			entityName, assocName, changes,
		)
	}

	// Verify the association actually appears in the patched BSON.
	assocRef := moduleName + "." + assocName
	if !patchedBSONContainsMemberAccess(t, patched, entityName, assocRef) {
		t.Errorf(
			"patched BSON: entity %q MemberAccesses does not contain Association=%q.\n"+
				"The association was reported as a change but wasn't written to the BSON.",
			entityName, assocRef,
		)
	}
}

// TestPatchReconcile_ExistingAttrs_AddsNewAssociation verifies that when an
// entity already has explicit attribute MemberAccesses the reconciler appends
// the missing association and reports it in the changes list.
func TestPatchReconcile_ExistingAttrs_AddsNewAssociation(t *testing.T) {
	const moduleName = "TestModule"
	const entityName = "Attachment"
	const entityID = "entity-attachment"
	const attrName = "FileType"
	const assocName = "Attachment_FileMetadata"

	existingMA := bson.A{
		bson.D{
			{Key: "$ID", Value: "ma-001"},
			{Key: "$Type", Value: "DomainModels$MemberAccess"},
			{Key: "AccessRights", Value: "ReadWrite"},
			{Key: "Attribute", Value: moduleName + "." + entityName + "." + attrName},
			{Key: "Association", Value: ""},
		},
	}

	raw := buildTestDomainModel(
		entityName, entityID,
		[]string{attrName},
		existingMA,
		[]bson.D{assocDoc(assocName, entityID)},
		nil,
	)

	patched, changes, err := PatchReconcileMemberAccesses(raw, moduleName)
	if err != nil {
		t.Fatalf("PatchReconcileMemberAccesses returned error: %v", err)
	}

	// The association must be in the changes list.
	found := false
	for _, ch := range changes {
		if ch.Entity == entityName && ch.Member == assocName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf(
			"changes missing {Entity:%q, Member:%q}; got: %v\n"+
				"Existing attribute MemberAccesses should not prevent reconciler from adding the association.",
			entityName, assocName, changes,
		)
	}

	assocRef := moduleName + "." + assocName
	if !patchedBSONContainsMemberAccess(t, patched, entityName, assocRef) {
		t.Errorf(
			"patched BSON: entity %q MemberAccesses missing Association=%q",
			entityName, assocRef,
		)
	}

	// Existing attribute entry must be preserved.
	attrRef := moduleName + "." + entityName + "." + attrName
	if !patchedBSONContainsMemberAccess(t, patched, entityName, attrRef) {
		t.Errorf(
			"patched BSON: entity %q lost existing Attribute=%q after reconcile",
			entityName, attrRef,
		)
	}
}

// TestPatchReconcile_NoAccessRules_NoChanges ensures entities without any
// AccessRules are not touched.
func TestPatchReconcile_NoAccessRules_NoChanges(t *testing.T) {
	// Build a DomainModel BSON with no entities that have AccessRules.
	dm := bson.D{
		{Key: "$Type", Value: "DomainModels$DomainModel"},
		{Key: "Entities", Value: bson.A{int32(3)}},
		{Key: "Associations", Value: bson.A{int32(3)}},
		{Key: "CrossAssociations", Value: bson.A{int32(3)}},
	}
	raw, _ := bson.Marshal(dm)

	_, changes, err := PatchReconcileMemberAccesses(raw, "TestModule")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for empty domain model, got %d: %v", len(changes), changes)
	}
}

// patchedBSONContainsMemberAccess is a test helper that walks the patched BSON
// and returns true when the named entity has a MemberAccess referencing ref
// (as either Attribute or Association).
func patchedBSONContainsMemberAccess(t *testing.T, patched []byte, entityName, ref string) bool {
	t.Helper()
	var doc bson.D
	if err := bson.Unmarshal(patched, &doc); err != nil {
		t.Fatalf("unmarshal patched bytes: %v", err)
	}
	for _, top := range doc {
		if top.Key != "Entities" {
			continue
		}
		entities, ok := top.Value.(bson.A)
		if !ok {
			continue
		}
		for _, entItem := range entities {
			entDoc, ok := entItem.(bson.D)
			if !ok {
				continue
			}
			name := ""
			for _, f := range entDoc {
				if f.Key == "Name" {
					name, _ = f.Value.(string)
					break
				}
			}
			if name != entityName {
				continue
			}
			for _, f := range entDoc {
				if f.Key != "AccessRules" {
					continue
				}
				rules, ok := f.Value.(bson.A)
				if !ok {
					continue
				}
				for _, ruleItem := range rules {
					ruleDoc, ok := ruleItem.(bson.D)
					if !ok {
						continue
					}
					for _, rf := range ruleDoc {
						if rf.Key != "MemberAccesses" {
							continue
						}
						mas, ok := rf.Value.(bson.A)
						if !ok {
							continue
						}
						for _, maItem := range mas {
							maDoc, ok := maItem.(bson.D)
							if !ok {
								continue
							}
							for _, mf := range maDoc {
								if (mf.Key == "Attribute" || mf.Key == "Association") && mf.Value == ref {
									return true
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

// TestPatchReconcile_CrossAssoc_AddedToMemberAccesses verifies that
// PatchReconcileMemberAccesses correctly handles cross-module associations.
//
// Cross-module associations (CrossAssociation BSON type, stored in the
// CrossAssociations array of the DomainModel) owned by an entity ARE included
// in MemberAccesses — Studio Pro includes them when running "Update Security."
//
// NOTE: The original hypothesis that CrossAssociations should be excluded was
// wrong.  PayerRegistration entities have CrossAssociation MAs and no CE0066.
// CE0066 in Common_Utils has a different root cause (internal Studio Pro
// security stamp that mxcli cannot replicate).
func TestPatchReconcile_CrossAssoc_AddedToMemberAccesses(t *testing.T) {
	const moduleName = "Common_Utils"
	const entityName = "MySetting"
	const entityID = "entity-mysetting"
	const crossAssocName = "MySetting_Account" // cross-module: FROM MySetting TO Administration.Account

	// Scenario A: entity owns a CrossAssociation with empty MemberAccesses — must ADD it.
	t.Run("cross_assoc_added_when_missing", func(t *testing.T) {
		raw := buildTestDomainModel(
			entityName, entityID,
			[]string{"Theme"}, // one attribute
			nil,               // empty MemberAccesses
			nil,               // no regular associations
			[]bson.D{crossAssocDoc(crossAssocName, entityID)}, // one cross assoc
		)

		patched, changes, err := PatchReconcileMemberAccesses(raw, moduleName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Must report a change for the cross assoc.
		found := false
		for _, ch := range changes {
			if ch.Member == crossAssocName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf(
				"expected a ReconcileChange for CrossAssociation %q but got none.\n"+
					"CrossAssociations owned by an entity ARE included in MemberAccesses "+
					"(Studio Pro includes them in 'Update Security').\n"+
					"changes: %v",
				crossAssocName, changes,
			)
		}

		// Must appear in patched BSON.
		assocRef := moduleName + "." + crossAssocName
		if !patchedBSONContainsMemberAccess(t, patched, entityName, assocRef) {
			t.Errorf(
				"patched BSON missing MemberAccess for CrossAssociation %q",
				assocRef,
			)
		}
	})

	// Scenario B: existing MemberAccess for CrossAssociation is preserved.
	t.Run("cross_assoc_preserved_when_existing", func(t *testing.T) {
		existingCrossMA := bson.A{
			bson.D{
				{Key: "$ID", Value: "existing-ma-id"},
				{Key: "$Type", Value: "DomainModels$MemberAccess"},
				{Key: "AccessRights", Value: "ReadWrite"},
				{Key: "Association", Value: moduleName + "." + crossAssocName},
				{Key: "Attribute", Value: ""},
			},
		}

		raw := buildTestDomainModel(
			entityName, entityID,
			[]string{"Theme"},
			existingCrossMA,
			nil,
			[]bson.D{crossAssocDoc(crossAssocName, entityID)},
		)

		patched, _, err := PatchReconcileMemberAccesses(raw, moduleName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assocRef := moduleName + "." + crossAssocName
		if !patchedBSONContainsMemberAccess(t, patched, entityName, assocRef) {
			t.Errorf(
				"patched BSON lost existing MemberAccess for CrossAssociation %q — it should be preserved",
				assocRef,
			)
		}
	})
}
