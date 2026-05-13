// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"database/sql"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	_ "modernc.org/sqlite"
)

// makeServiceTestMPR creates a minimal v1 MPR SQLite file with one unit whose
// BSON is provided by the caller. Returns the path and the unit ID.
func makeServiceTestMPR(t *testing.T, unitIDStr string, contents []byte) (mprPath string, unitID model.ID) {
	t.Helper()
	dir := t.TempDir()
	mprPath = filepath.Join(dir, "test.mpr")

	db, err := sql.Open("sqlite", mprPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE _MetaData (_FormatVersion INTEGER, _ProductVersion TEXT, _BuildVersion TEXT, _SchemaHash TEXT);
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

	idBlob := secTestUUIDBlob(unitIDStr)
	if _, err := db.Exec(
		`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		 VALUES (?, ?, ?, 0, '', '', ?)`,
		idBlob, make([]byte, 16), "Documents", contents,
	); err != nil {
		t.Fatalf("insert unit: %v", err)
	}

	return mprPath, model.ID(unitIDStr)
}

// readBSONField unmarshals raw bytes and returns the string value of a named field.
func readBSONField(t *testing.T, raw []byte, field string) string {
	t.Helper()
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, e := range doc {
		if e.Key == field {
			s, _ := e.Value.(string)
			return s
		}
	}
	return ""
}

// makeBSONUnit constructs the BSON bytes for a unit with the given $Type, $ID,
// and any additional fields supplied via extra.
func makeBSONUnit(t *testing.T, unitIDStr, bsonType string, extra bson.D) []byte {
	t.Helper()
	idBlob := secTestUUIDBlob(unitIDStr)
	doc := bson.D{
		{Key: "$Type", Value: bsonType},
		{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: idBlob}},
	}
	doc = append(doc, extra...)
	b, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal unit: %v", err)
	}
	return b
}

func TestUpdateJsonStructureViaModelsdk(t *testing.T) {
	const unitID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	contents := makeBSONUnit(t, unitID, "JsonStructures$JsonStructure", bson.D{
		{Key: "Name", Value: "OldName"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "JsonSnippet", Value: `{"old": true}`},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	js := &types.JsonStructure{
		Name:        "NewName",
		JsonSnippet: `{"updated": true}`,
		ExportLevel: "Hidden",
	}
	js.ID = id
	if err := b.updateJsonStructureViaModelsdk(js); err != nil {
		t.Fatalf("updateJsonStructureViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewName" {
		t.Errorf("Name = %q, want %q", got, "NewName")
	}
	if got := readBSONField(t, raw, "JsonSnippet"); got != `{"updated": true}` {
		t.Errorf("JsonSnippet = %q, want updated value", got)
	}
}
