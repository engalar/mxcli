// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"database/sql"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	_ "modernc.org/sqlite"
)

// makeEnumRefMPR creates a minimal MPR with one domain model containing
// one entity with one EnumerationAttributeType attribute referencing oldRef.
func makeEnumRefMPR(t *testing.T, oldRef string) (mprPath string, dmID model.ID) {
	t.Helper()
	dir := t.TempDir()
	mprPath = filepath.Join(dir, "test.mpr")

	db, err := sql.Open("sqlite", mprPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE _MetaData (
			_FormatVersion INTEGER,
			_ProductVersion TEXT,
			_BuildVersion TEXT,
			_SchemaHash TEXT
		);
		INSERT INTO _MetaData VALUES (1, '10.18.0', '10.18.0.0', 'testhash');

		CREATE TABLE _Transaction (LastTransactionID TEXT);
		INSERT INTO _Transaction VALUES ('00000000-0000-0000-0000-000000000000');

		CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			TreeConflict LONG,
			ContentsHash TEXT,
			ContentsConflicts TEXT,
			Contents BLOB
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	dmIDStr := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	dmID = model.ID(dmIDStr)
	dmBlob := secTestUUIDBlob(dmIDStr)

	entityIDStr := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	entityBlob := secTestUUIDBlob(entityIDStr)

	attrIDStr := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	attrBlob := secTestUUIDBlob(attrIDStr)

	// Build BSON matching what SerializeDomainModel produces for an entity
	// with one EnumerationAttributeType attribute.
	enumTypeDoc := bson.D{
		{Key: "$Type", Value: "DomainModels$EnumerationAttributeType"},
		{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: make([]byte, 16)}},
		{Key: "Enumeration", Value: oldRef},
	}
	valueDoc := bson.D{
		{Key: "$Type", Value: "DomainModels$StoredValue"},
		{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: make([]byte, 16)}},
		{Key: "DefaultValue", Value: ""},
	}
	attrDoc := bson.D{
		{Key: "$Type", Value: "DomainModels$Attribute"},
		{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: attrBlob}},
		{Key: "Name", Value: "Status"},
		{Key: "Documentation", Value: ""},
		{Key: "Exposed", Value: false},
		{Key: "Type", Value: enumTypeDoc},
		{Key: "Value", Value: valueDoc},
	}
	entityDoc := bson.D{
		{Key: "$Type", Value: "DomainModels$Entity"},
		{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: entityBlob}},
		{Key: "Name", Value: "MyEntity"},
		{Key: "Documentation", Value: ""},
		{Key: "Persistable", Value: "DomainModels$StorageBehavior_Persist"},
		{Key: "DataStorageGuid", Value: ""},
		{Key: "Generalization", Value: bson.D{
			{Key: "$Type", Value: "DomainModels$NoGeneralization"},
			{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: make([]byte, 16)}},
		}},
		{Key: "Attributes", Value: bson.A{attrDoc}},
		{Key: "EventHandlers", Value: bson.A{}},
		{Key: "Indexes", Value: bson.A{}},
		{Key: "AccessRules", Value: bson.A{}},
	}
	dmDoc := bson.D{
		{Key: "$Type", Value: "DomainModels$DomainModel"},
		{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: dmBlob}},
		{Key: "Entities", Value: bson.A{entityDoc}},
		{Key: "Associations", Value: bson.A{}},
		{Key: "CrossAssociations", Value: bson.A{}},
	}
	dmBytes, err := bson.Marshal(dmDoc)
	if err != nil {
		t.Fatalf("marshal domain model: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		 VALUES (?, ?, ?, 0, '', '', ?)`,
		dmBlob, make([]byte, 16), "DomainModel", dmBytes,
	); err != nil {
		t.Fatalf("insert domain model unit: %v", err)
	}

	return mprPath, dmID
}

// makeDomainModelTestMPR creates a minimal MPR with one Module unit and one
// empty DomainModel unit owned by it. Used by TestCreateEntityViaModelsdk as a
// regression guard for writeDomainModel's transport tail.
func makeDomainModelTestMPR(t *testing.T) (mprPath string, dmID model.ID) {
	t.Helper()
	dir := t.TempDir()
	mprPath = filepath.Join(dir, "test.mpr")

	db, err := sql.Open("sqlite", mprPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE _MetaData (
			_FormatVersion INTEGER,
			_ProductVersion TEXT,
			_BuildVersion TEXT,
			_SchemaHash TEXT
		);
		INSERT INTO _MetaData VALUES (1, '10.18.0', '10.18.0.0', 'testhash');

		CREATE TABLE _Transaction (LastTransactionID TEXT);
		INSERT INTO _Transaction VALUES ('00000000-0000-0000-0000-000000000000');

		CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			TreeConflict LONG,
			ContentsHash TEXT,
			ContentsConflicts TEXT,
			Contents BLOB
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	moduleIDStr := "11111111-2222-3333-4444-555555555555"
	moduleBlob := secTestUUIDBlob(moduleIDStr)
	dmIDStr := "66666666-7777-8888-9999-aaaaaaaaaaaa"
	dmBlob := secTestUUIDBlob(dmIDStr)
	dmID = model.ID(dmIDStr)

	dmDoc := bson.D{
		{Key: "$Type", Value: "DomainModels$DomainModel"},
		{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: dmBlob}},
		{Key: "Entities", Value: bson.A{}},
		{Key: "Associations", Value: bson.A{}},
		{Key: "CrossAssociations", Value: bson.A{}},
	}
	dmBytes, err := bson.Marshal(dmDoc)
	if err != nil {
		t.Fatalf("marshal domain model: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		 VALUES (?, ?, ?, 0, '', '', ?)`,
		dmBlob, moduleBlob, "DomainModel", dmBytes,
	); err != nil {
		t.Fatalf("insert domain model unit: %v", err)
	}

	return mprPath, dmID
}

// TestCreateEntityViaModelsdk verifies createEntityViaModelsdk persists a new
// entity through writeDomainModel. Regression guard for the transport-level
// switch from msdkWriteRaw to BeginWriteTransaction/WriteUnit/Commit.
func TestCreateEntityViaModelsdk(t *testing.T) {
	mprPath, dmID := makeDomainModelTestMPR(t)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	entity := &domainmodel.Entity{
		Name:        "Customer",
		Persistable: true,
	}
	if err := b.createEntityViaModelsdk(dmID, entity); err != nil {
		t.Fatalf("createEntityViaModelsdk: %v", err)
	}

	dm, err := b.reader.GetDomainModelByID(dmID)
	if err != nil {
		t.Fatalf("GetDomainModelByID after create: %v", err)
	}
	if got, want := len(dm.Entities), 1; got != want {
		t.Fatalf("Entities count = %d, want %d", got, want)
	}
	if got := dm.Entities[0].Name; got != "Customer" {
		t.Errorf("Entities[0].Name = %q, want %q", got, "Customer")
	}
	if dm.Entities[0].ID == "" {
		t.Error("Entities[0].ID was not auto-generated")
	}
}

func TestUpdateEnumerationRefsInAllDomainModels_NoChange(t *testing.T) {
	mprPath, _ := makeEnumRefMPR(t, "OldModule.StatusEnum")

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Should succeed with no matches
	if err := b.updateEnumerationRefsInAllDomainModelsViaModelsdk("NonExistent.Enum", "Other.Enum"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateEnumerationRefsInAllDomainModels_UpdatesRef(t *testing.T) {
	mprPath, dmID := makeEnumRefMPR(t, "OldModule.StatusEnum")

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	if err := b.updateEnumerationRefsInAllDomainModelsViaModelsdk("OldModule.StatusEnum", "NewModule.StatusEnum"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read back and verify
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(dmID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}

	dm, err := b.reader.GetDomainModelByID(dmID)
	if err != nil {
		t.Fatalf("GetDomainModelByID after update: %v (raw len=%d)", err, len(rawBytes))
	}
	if len(dm.Entities) == 0 {
		t.Fatal("expected at least one entity")
	}
	attr := dm.Entities[0].Attributes[0]
	enumType, ok := attr.Type.(*domainmodel.EnumerationAttributeType)
	if !ok {
		t.Fatalf("expected EnumerationAttributeType, got %T", attr.Type)
	}
	if enumType.EnumerationRef != "NewModule.StatusEnum" {
		t.Fatalf("EnumerationRef = %q, want %q", enumType.EnumerationRef, "NewModule.StatusEnum")
	}
}
