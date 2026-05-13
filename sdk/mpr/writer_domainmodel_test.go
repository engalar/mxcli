// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/mpr/version"
	"go.mongodb.org/mongo-driver/bson"
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

// TestSerializeODataRemoteEntitySource_HasKeyAndMappedValues verifies that
// external entities serialized via serializeEntity produce:
//   - Rest$ODataKey with Parts (fixes CE6010 "Key cannot be empty")
//   - Rest$ODataMappedValue on each attribute (fixes CE6612 "attribute not supported")
//
// Regression guard for the bugs that caused 51+ Studio Pro errors when opening
// a project with external entities created by `CREATE EXTERNAL ENTITIES FROM`.
func TestSerializeODataRemoteEntitySource_HasKeyAndMappedValues(t *testing.T) {
	entity := &domainmodel.Entity{
		Name:              "Airlines",
		Source:            "Rest$ODataRemoteEntitySource",
		RemoteServiceName: "TripPinTest.TripPinRW",
		RemoteEntityName:  "Airline",
		RemoteEntitySet:   "Airlines",
		Persistable:       true,
		Creatable:         true,
		Countable:         true,
		SkipSupported:     true,
		TopSupported:      true,
		RemoteKeyParts: []*domainmodel.RemoteKeyPart{
			{
				Name:       "AirlineCode",
				RemoteName: "AirlineCode",
				RemoteType: "Edm.String",
				Type:       &domainmodel.StringAttributeType{Length: 100},
			},
		},
		Attributes: []*domainmodel.Attribute{
			{
				BaseElement: model.BaseElement{ID: "attr-airlinecode"},
				Name:        "AirlineCode",
				RemoteName:  "AirlineCode",
				RemoteType:  "Edm.String",
				Filterable:  true,
				Sortable:    true,
				Creatable:   true,
				Type:        &domainmodel.StringAttributeType{Length: 100},
			},
			{
				BaseElement: model.BaseElement{ID: "attr-name"},
				Name:        "Name",
				RemoteName:  "Name",
				RemoteType:  "Edm.String",
				Filterable:  true,
				Sortable:    true,
				Type:        &domainmodel.StringAttributeType{Length: 0},
			},
		},
	}
	entity.ID = "entity-test-id"

	doc := serializeEntity(entity, "TripPinTest", nil)

	// Marshal to BSON map for inspection
	raw, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var m map[string]any
	if err := bson.Unmarshal(raw, &m); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Source must be Rest$ODataRemoteEntitySource
	sourceRaw, ok := m["Source"]
	if !ok {
		t.Fatal("Source field missing from entity BSON")
	}
	source, ok := sourceRaw.(map[string]any)
	if !ok {
		t.Fatalf("Source: expected map, got %T", sourceRaw)
	}
	if got := source["$Type"]; got != "Rest$ODataRemoteEntitySource" {
		t.Errorf("Source.$Type = %v, want Rest$ODataRemoteEntitySource", got)
	}

	// CE6010: Key must be present with Rest$ODataKey type
	keyRaw, ok := source["Key"]
	if !ok {
		t.Fatal("CE6010: Source.Key missing — Studio Pro reports 'Key cannot be empty'")
	}
	key, ok := keyRaw.(map[string]any)
	if !ok {
		t.Fatalf("Source.Key: expected map, got %T", keyRaw)
	}
	if got := key["$Type"]; got != "Rest$ODataKey" {
		t.Errorf("Source.Key.$Type = %v, want Rest$ODataKey", got)
	}
	if key["Parts"] == nil {
		t.Error("Source.Key.Parts is nil")
	}

	// CE6612: Each attribute Value must be Rest$ODataMappedValue
	attrItems := extractBsonArray(m["Attributes"])
	if len(attrItems) == 0 {
		t.Fatal("Attributes array is empty")
	}
	for i, item := range attrItems {
		attrMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		valueMap, ok := attrMap["Value"].(map[string]any)
		if !ok {
			t.Errorf("Attribute[%d].Value: expected map, got %T", i, attrMap["Value"])
			continue
		}
		if vType := valueMap["$Type"]; vType != "Rest$ODataMappedValue" {
			t.Errorf("CE6612: Attribute[%d].Value.$Type = %v, want Rest$ODataMappedValue", i, vType)
		}
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

// =============================================================================
// ScanOqlQueryUpdates
// =============================================================================

func TestScanOqlQueryUpdates_NoMatch(t *testing.T) {
	w, db := newTestWriterV1(t, `
		CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			ContentsHash TEXT,
			ContentsConflicts TEXT,
			Contents BLOB
		)`)

	// Insert a ViewEntitySourceDocument with OQL that does NOT contain oldName
	docID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	modID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	contents, _ := bson.Marshal(map[string]any{
		"$Type": "DomainModels$ViewEntitySourceDocument",
		"Name":  "MyView",
		"Oql":   "from OtherModule.Customer as c return c",
	})
	if _, err := db.Exec(`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, ContentsHash, ContentsConflicts, Contents) VALUES (?,?,?,?,?,?)`,
		uuidToBlob(docID), uuidToBlob(modID), "Documents", "", "", contents); err != nil {
		t.Fatalf("insert: %v", err)
	}

	patches, count, err := w.ScanOqlQueryUpdates("MyModule.Customer", "NewModule.Customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 matches, got %d", count)
	}
	if len(patches) != 0 {
		t.Fatalf("expected 0 patches, got %d", len(patches))
	}
}

func TestScanOqlQueryUpdates_WithMatch(t *testing.T) {
	w, db := newTestWriterV1(t, `
		CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			ContentsHash TEXT,
			ContentsConflicts TEXT,
			Contents BLOB
		)`)

	docID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	modID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	contents, _ := bson.Marshal(map[string]any{
		"$Type": "DomainModels$ViewEntitySourceDocument",
		"Name":  "MyView",
		"Oql":   "from MyModule.Customer as c return c",
	})
	if _, err := db.Exec(`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, ContentsHash, ContentsConflicts, Contents) VALUES (?,?,?,?,?,?)`,
		uuidToBlob(docID), uuidToBlob(modID), "Documents", "", "", contents); err != nil {
		t.Fatalf("insert: %v", err)
	}

	patches, count, err := w.ScanOqlQueryUpdates("MyModule.Customer", "NewModule.Customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 match, got %d", count)
	}
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}

	// Verify the patch contains the updated OQL
	var raw map[string]any
	if err := bson.Unmarshal(patches[0].Contents, &raw); err != nil {
		t.Fatalf("unmarshal patch: %v", err)
	}
	oql, _ := raw["Oql"].(string)
	if oql != "from NewModule.Customer as c return c" {
		t.Fatalf("expected updated OQL, got %q", oql)
	}
}

// TestSerializeDomainModelStandalone verifies that SerializeDomainModel works
// as a standalone package-level function — no *Writer required. Callers pass
// the module name and project version explicitly so the encoder is decoupled
// from reader state. This is part of retiring Writer Serialize* receivers.
func TestSerializeDomainModelStandalone(t *testing.T) {
	dm := &domainmodel.DomainModel{
		ContainerID: "container-id",
	}
	dm.ID = model.ID("11111111-1111-1111-1111-111111111111")
	dm.TypeName = "DomainModels$DomainModel"

	pv := version.DefaultVersion()

	contents, err := SerializeDomainModel(dm, "MyModule", pv)
	if err != nil {
		t.Fatalf("SerializeDomainModel: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("expected non-empty BSON, got 0 bytes")
	}

	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["$Type"] != "DomainModels$DomainModel" {
		t.Fatalf("expected $Type=DomainModels$DomainModel, got %v", raw["$Type"])
	}
}
