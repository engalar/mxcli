// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// TestResolveMemberChangeGen_AssocNotInCachedDM is a regression test for the
// bug where a 1-dot association name (Module.Assoc) was incorrectly stored in
// the BSON Attribute field instead of the Association field.
//
// Trigger condition: the backend returns a valid domain model but the
// association is absent from AssociationsItems() — simulating a stale reader
// cache that doesn't yet include an association written earlier in the same
// import session.
//
// Without the fix, resolveMemberChangeGenStandalone reaches the
// `strings.Contains(memberName, ".")` guard (line 237 pre-fix), returns
// {attributeQN: "Module.Assoc"}, and SetAttributeQualifiedName is called
// instead of SetAssociationQualifiedName. Studio Pro rejects the result with
// StorageLoadException: "not a valid AttributeIdentifier".
func TestResolveMemberChangeGen_AssocNotInCachedDM(t *testing.T) {
	// Arrange: backend returns a module + domain model, but the domain model
	// has NO associations (simulating stale cache after write).
	mod := &model.Module{Name: "EndCustomerRegistration"}
	mod.BaseElement.ID = "mod-1"
	emptyDM := genDm.NewDomainModel()

	b := &mock.MockBackend{
		GetModuleByNameFunc: func(name string) (*model.Module, error) {
			if name == "EndCustomerRegistration" {
				return mod, nil
			}
			return nil, nil
		},
		GetDomainModelGenFunc: func(moduleID model.ID) (*genDm.DomainModel, error) {
			return emptyDM, nil
		},
	}

	// Act: resolve a 1-dot association name not present in the (empty) DM.
	memberName := "EndCustomerRegistration.EndCustomer_ApplicationCommonHeader"
	entityQN := "EndCustomerRegistration.EndCustomer"
	got := resolveMemberChangeGenStandalone(b, memberName, entityQN, nil)

	// Assert: must return associationQN, NOT attributeQN.
	if got.attributeQN != "" {
		t.Errorf("got attributeQN=%q, want empty — association name must NOT be stored as attribute",
			got.attributeQN)
	}
	if got.associationQN != memberName {
		t.Errorf("got associationQN=%q, want %q", got.associationQN, memberName)
	}
}

// TestResolveMemberChangeGen_TwoDotAttributePreserved verifies that a 2-dot
// attribute name (Module.Entity.Attr) is correctly classified as an attribute
// even when the domain model is loaded but the attribute lookup fails — this
// is the intended use-case for the `strings.Count >= 2` guard.
func TestResolveMemberChangeGen_TwoDotAttributePreserved(t *testing.T) {
	mod := &model.Module{Name: "MyModule"}
	mod.BaseElement.ID = "mod-1"
	emptyDM := genDm.NewDomainModel()

	b := &mock.MockBackend{
		GetModuleByNameFunc: func(name string) (*model.Module, error) {
			if name == "MyModule" {
				return mod, nil
			}
			return nil, nil
		},
		GetDomainModelGenFunc: func(moduleID model.ID) (*genDm.DomainModel, error) {
			return emptyDM, nil
		},
	}

	memberName := "MyModule.MyEntity.MyAttr"
	entityQN := "MyModule.MyEntity"
	got := resolveMemberChangeGenStandalone(b, memberName, entityQN, nil)

	if got.attributeQN != memberName {
		t.Errorf("got attributeQN=%q, want %q — 2-dot name must be preserved as attribute",
			got.attributeQN, memberName)
	}
	if got.associationQN != "" {
		t.Errorf("got associationQN=%q, want empty", got.associationQN)
	}
}

// TestMemberChangeFallback_OneDotIsAssociation verifies the fallback rules.
func TestMemberChangeFallback_OneDotIsAssociation(t *testing.T) {
	cases := []struct {
		memberName  string
		entityQN    string
		wantAssocQN string
		wantAttrQN  string
	}{
		{
			memberName:  "MyMod.MyAssoc",
			entityQN:    "MyMod.MyEntity",
			wantAssocQN: "MyMod.MyAssoc",
		},
		{
			memberName: "BareAttr",
			entityQN:   "MyMod.MyEntity",
			wantAttrQN: "MyMod.MyEntity.BareAttr",
		},
		{
			memberName: "MyMod.MyEntity.MyAttr",
			entityQN:   "MyMod.MyEntity",
			wantAttrQN: "MyMod.MyEntity.MyAttr",
		},
	}

	for _, tc := range cases {
		got := memberChangeFallback(tc.memberName, tc.entityQN)
		if got.associationQN != tc.wantAssocQN {
			t.Errorf("memberChangeFallback(%q, %q).associationQN = %q, want %q",
				tc.memberName, tc.entityQN, got.associationQN, tc.wantAssocQN)
		}
		if got.attributeQN != tc.wantAttrQN {
			t.Errorf("memberChangeFallback(%q, %q).attributeQN = %q, want %q",
				tc.memberName, tc.entityQN, got.attributeQN, tc.wantAttrQN)
		}
	}
}

// TestResolveMemberChangeGen_FetchesDMTwiceForBareAttribute proves
// the performance bug: resolveMemberChangeGenStandalone fetches the
// domain model TWICE for a single bare-attribute member change — once
// directly (line 258) and once inside resolveAttributeInEntityHierarchyGen
// (line 203). The fix must reduce this to a single fetch.
func TestResolveMemberChangeGen_FetchesDMTwiceForBareAttribute(t *testing.T) {
	mod := &model.Module{Name: "HD"}
	mod.BaseElement.ID = "mod-1"

	dm := genDm.NewDomainModel()
	entity := mkEntityGen("Ticket")
	attr := genDm.NewAttribute()
	attr.SetName("Status")
	entity.AddAttributes(attr)
	dm.AddEntities(entity)

	callCount := 0
	b := &mock.MockBackend{
		GetModuleByNameFunc: func(name string) (*model.Module, error) {
			return mod, nil
		},
		GetDomainModelGenFunc: func(moduleID model.ID) (*genDm.DomainModel, error) {
			callCount++
			return dm, nil
		},
	}

	// Act: resolve a single bare-attribute member change
	_ = resolveMemberChangeGenStandalone(b, "Status", "HD.Ticket", nil)

	// Assert: GetDomainModelGen must be called exactly once.
	// Current bug: called 2x (double-fetch).
	if callCount != 1 {
		t.Errorf("GetDomainModelGen called %d times; want 1 (double-fetch bug: line 258 + line 203)", callCount)
	}
}
