// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// =============================================================================
// Issue #50: CrossAssociation must NOT include ParentConnection/ChildConnection
// =============================================================================

// TestSerializeCrossAssociation_NoConnectionFields verifies that
// serializeCrossAssociation does NOT emit ParentConnection or ChildConnection.
// These properties only exist on DomainModels$Association, not on
// DomainModels$CrossAssociation. Writing them causes Studio Pro to crash with
// System.InvalidOperationException: Sequence contains no matching element.
func TestSerializeCrossAssociation_NoConnectionFields(t *testing.T) {
	ca := &domainmodel.CrossModuleAssociation{
		Name:     "Child_Parent",
		ParentID: "parent-entity-id",
		ChildRef: "OtherModule.Parent",
		Type:     domainmodel.AssociationTypeReference,
		Owner:    domainmodel.AssociationOwnerDefault,
	}
	ca.ID = "test-cross-assoc-id"

	result := serializeCrossAssociation(ca)

	// Must NOT contain these fields
	for key := range result {
		if key == "ParentConnection" {
			t.Error("serializeCrossAssociation must NOT include ParentConnection (only valid for Association)")
		}
		if key == "ChildConnection" {
			t.Error("serializeCrossAssociation must NOT include ChildConnection (only valid for Association)")
		}
	}

	// Must contain all expected fields (exhaustive structural contract)
	expectedKeys := []string{"$ID", "$Type", "Name", "Child", "ParentPointer", "Type", "Owner",
		"Documentation", "ExportLevel", "GUID", "StorageFormat", "Source", "DeleteBehavior"}
	for _, key := range expectedKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("serializeCrossAssociation missing expected field %q", key)
		}
	}

	// $Type must be CrossAssociation
	if got := result["$Type"]; got != "DomainModels$CrossAssociation" {
		t.Errorf("$Type = %q, want %q", got, "DomainModels$CrossAssociation")
	}
}

// TestSerializeAssociation_HasConnectionFields verifies that the regular
// serializeAssociation DOES include ParentConnection and ChildConnection
// (to ensure we didn't accidentally remove them from the wrong function).
func TestSerializeAssociation_HasConnectionFields(t *testing.T) {
	a := &domainmodel.Association{
		Name:     "Child_Parent",
		ParentID: "parent-entity-id",
		ChildID:  "child-entity-id",
		Type:     domainmodel.AssociationTypeReference,
		Owner:    domainmodel.AssociationOwnerDefault,
	}
	a.ID = "test-assoc-id"

	result := serializeAssociation(a)

	hasParentConn := false
	hasChildConn := false
	for key := range result {
		if key == "ParentConnection" {
			hasParentConn = true
		}
		if key == "ChildConnection" {
			hasChildConn = true
		}
	}

	if !hasParentConn {
		t.Error("serializeAssociation must include ParentConnection")
	}
	if !hasChildConn {
		t.Error("serializeAssociation must include ChildConnection")
	}
}
