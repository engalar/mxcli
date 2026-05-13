// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// seedEntityForAccessTest creates an MPR with one Customer entity inside the
// pre-built DomainModel and returns the path + dmID. Reuses
// makeDomainModelTestMPR (defined in domainmodel_modelsdk_test.go) for the
// SQLite + Module + DomainModel skeleton, then adds a Customer entity through
// the production createEntityViaModelsdk helper so the test exercises the same
// gen-native write path the access-rule funcs target.
func seedEntityForAccessTest(t *testing.T) (b *MprBackend, dmID model.ID) {
	t.Helper()
	mprPath, dmID := makeDomainModelTestMPR(t)

	b = New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = b.Disconnect() })

	if err := b.createEntityViaModelsdk(dmID, &domainmodel.Entity{
		Name:        "Customer",
		Persistable: true,
	}); err != nil {
		t.Fatalf("createEntityViaModelsdk: %v", err)
	}
	return b, dmID
}

// findCustomerAccessRules walks the freshly serialized DomainModel BSON and
// returns the AccessRules entries of the Customer entity, with the leading
// int32(3) version prefix stripped. Mendix encodes arrays as
// [versionTag, item0, item1, ...] — only the items after index 0 are real
// rules.
func findCustomerAccessRules(t *testing.T, b *MprBackend, dmID model.ID) []bson.D {
	t.Helper()
	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(dmID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal DM: %v", err)
	}
	for _, top := range doc {
		if top.Key != "Entities" {
			continue
		}
		entities, ok := top.Value.(bson.A)
		if !ok {
			t.Fatalf("Entities is %T, want bson.A", top.Value)
		}
		for _, ent := range entities {
			entDoc, ok := ent.(bson.D)
			if !ok {
				continue
			}
			isCustomer := false
			for _, f := range entDoc {
				if f.Key == "Name" && f.Value == "Customer" {
					isCustomer = true
					break
				}
			}
			if !isCustomer {
				continue
			}
			for _, f := range entDoc {
				if f.Key != "AccessRules" {
					continue
				}
				arr, ok := f.Value.(bson.A)
				if !ok {
					t.Fatalf("AccessRules is %T, want bson.A", f.Value)
				}
				out := make([]bson.D, 0, len(arr))
				for _, item := range arr {
					if rule, ok := item.(bson.D); ok {
						out = append(out, rule)
					}
				}
				return out
			}
			return nil
		}
	}
	t.Fatalf("Customer entity not found in DomainModel")
	return nil
}

// ruleField returns the first matching field value in a rule bson.D. Returns
// nil if missing.
func ruleField(t *testing.T, rule bson.D, key string) interface{} {
	t.Helper()
	for _, f := range rule {
		if f.Key == key {
			return f.Value
		}
	}
	return nil
}

// TestAddEntityAccessRuleViaModelsdk_GenNative is the regression sentinel for
// addEntityAccessRuleViaModelsdk. It locks in the BSON shape (AllowedModuleRoles
// key, AllowCreate/AllowDelete bools, DefaultMemberAccessRights, XPathConstraint,
// MemberAccesses array) so the gen-native rewrite cannot silently regress.
func TestAddEntityAccessRuleViaModelsdk_GenNative(t *testing.T) {
	b, dmID := seedEntityForAccessTest(t)

	memberAccesses := []mpr.EntityMemberAccess{
		{AttributeRef: "TestModule.Customer.Name", AccessRights: "ReadOnly"},
	}
	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.UserRole"},
		true, false,
		"ReadOnly", "[Active = true()]",
		memberAccesses,
	); err != nil {
		t.Fatalf("addEntityAccessRuleViaModelsdk: %v", err)
	}

	rules := findCustomerAccessRules(t, b, dmID)
	if len(rules) != 1 {
		t.Fatalf("rules count = %d, want 1", len(rules))
	}
	rule := rules[0]
	if got := ruleField(t, rule, "AllowedModuleRoles"); got == nil {
		t.Errorf("AllowedModuleRoles missing — gen rewrite regressed BSON key")
	}
	if got := ruleField(t, rule, "AllowCreate"); got != true {
		t.Errorf("AllowCreate = %v, want true", got)
	}
	if got := ruleField(t, rule, "AllowDelete"); got != false {
		t.Errorf("AllowDelete = %v, want false", got)
	}
	if got := ruleField(t, rule, "DefaultMemberAccessRights"); got != "ReadOnly" {
		t.Errorf("DefaultMemberAccessRights = %v, want ReadOnly", got)
	}
	if got := ruleField(t, rule, "XPathConstraint"); got != "[Active = true()]" {
		t.Errorf("XPathConstraint = %v, want [Active = true()]", got)
	}
	mas, ok := ruleField(t, rule, "MemberAccesses").(bson.A)
	if !ok {
		t.Fatalf("MemberAccesses is %T, want bson.A", ruleField(t, rule, "MemberAccesses"))
	}
	// MemberAccesses (like AccessRules) is a versioned array: [int32(1), entry0, ...]
	if got := countVersionedEntries(mas); got != 1 {
		t.Fatalf("MemberAccesses entries = %d, want 1 (raw=%v)", got, mas)
	}
}

// countVersionedEntries returns the number of real entries in a Mendix
// versioned-array (the leading int32 version tag is ignored).
func countVersionedEntries(arr bson.A) int {
	count := 0
	for _, item := range arr {
		switch item.(type) {
		case int32, int64:
			// version tag — skip
		default:
			count++
		}
	}
	return count
}

// TestRemoveEntityAccessRuleViaModelsdk_GenNative locks in the role-match
// removal semantics: only rules whose AllowedModuleRoles equal the input
// (order-independent) are removed.
func TestRemoveEntityAccessRuleViaModelsdk_GenNative(t *testing.T) {
	b, dmID := seedEntityForAccessTest(t)

	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.RoleA"},
		false, false, "ReadOnly", "", nil,
	); err != nil {
		t.Fatalf("seed RoleA rule: %v", err)
	}
	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.RoleB"},
		false, false, "ReadOnly", "", nil,
	); err != nil {
		t.Fatalf("seed RoleB rule: %v", err)
	}

	if rules := findCustomerAccessRules(t, b, dmID); len(rules) != 2 {
		t.Fatalf("pre-remove rules count = %d, want 2", len(rules))
	}

	removed, err := b.removeEntityAccessRuleViaModelsdk(dmID, "Customer", []string{"TestModule.RoleA"})
	if err != nil {
		t.Fatalf("removeEntityAccessRuleViaModelsdk: %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	rules := findCustomerAccessRules(t, b, dmID)
	if len(rules) != 1 {
		t.Fatalf("post-remove rules count = %d, want 1", len(rules))
	}
	mr, _ := ruleField(t, rules[0], "AllowedModuleRoles").(bson.A)
	if got := countVersionedEntries(mr); got != 1 {
		t.Fatalf("surviving rule role list entries = %d, want 1 (raw=%v)", got, mr)
	}
}

// TestRemoveRoleFromAllEntitiesViaModelsdk_GenNative locks in the per-rule
// role-strip semantics: occurrences of the named role are filtered out of every
// rule's AllowedModuleRoles list, and the count returned matches the number of
// rules touched.
func TestRemoveRoleFromAllEntitiesViaModelsdk_GenNative(t *testing.T) {
	b, dmID := seedEntityForAccessTest(t)

	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.RoleA", "TestModule.RoleB"},
		false, false, "ReadOnly", "", nil,
	); err != nil {
		t.Fatalf("seed rule with two roles: %v", err)
	}
	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.RoleC"},
		false, false, "ReadOnly", "", nil,
	); err != nil {
		t.Fatalf("seed rule with RoleC: %v", err)
	}

	removed, err := b.removeRoleFromAllEntitiesViaModelsdk(dmID, "TestModule.RoleA")
	if err != nil {
		t.Fatalf("removeRoleFromAllEntitiesViaModelsdk: %v", err)
	}
	if removed != 1 {
		t.Errorf("rules touched = %d, want 1", removed)
	}

	rules := findCustomerAccessRules(t, b, dmID)
	if len(rules) != 2 {
		t.Fatalf("rules count after strip = %d, want 2", len(rules))
	}
	for _, rule := range rules {
		mr, _ := ruleField(t, rule, "AllowedModuleRoles").(bson.A)
		for _, role := range mr {
			if role == "TestModule.RoleA" {
				t.Errorf("RoleA still present in rule: %v", mr)
			}
		}
	}
}

// TestRevokeEntityMemberAccessViaModelsdk_GenNative locks in the partial
// revoke semantics: matching rules have their flags downgraded according to
// the EntityAccessRevocation struct.
func TestRevokeEntityMemberAccessViaModelsdk_GenNative(t *testing.T) {
	b, dmID := seedEntityForAccessTest(t)

	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.UserRole"},
		true, true, "ReadWrite", "",
		nil,
	); err != nil {
		t.Fatalf("seed rule: %v", err)
	}

	revoked, err := b.revokeEntityMemberAccessViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.UserRole"},
		mpr.EntityAccessRevocation{
			RevokeCreate:   true,
			RevokeDelete:   true,
			RevokeWriteAll: true,
		},
	)
	if err != nil {
		t.Fatalf("revokeEntityMemberAccessViaModelsdk: %v", err)
	}
	if revoked != 1 {
		t.Errorf("revoked = %d, want 1", revoked)
	}

	rules := findCustomerAccessRules(t, b, dmID)
	if len(rules) != 1 {
		t.Fatalf("rules count = %d, want 1", len(rules))
	}
	rule := rules[0]
	if got := ruleField(t, rule, "AllowCreate"); got != false {
		t.Errorf("AllowCreate = %v, want false", got)
	}
	if got := ruleField(t, rule, "AllowDelete"); got != false {
		t.Errorf("AllowDelete = %v, want false", got)
	}
	if got := ruleField(t, rule, "DefaultMemberAccessRights"); got != "ReadOnly" {
		t.Errorf("DefaultMemberAccessRights = %v, want ReadOnly", got)
	}
}
