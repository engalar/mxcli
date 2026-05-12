// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/model"
	_ "modernc.org/sqlite"
)

// makeSecurityTestMPR creates a minimal v1 MPR SQLite file in a temp dir
// and inserts one Security$ProjectSecurity unit with SecurityLevel = Off.
// Returns the file path and the unit ID.
func makeSecurityTestMPR(t *testing.T) (mprPath string, unitID model.ID) {
	t.Helper()
	dir := t.TempDir()
	mprPath = filepath.Join(dir, "test.mpr")

	db, err := sql.Open("sqlite", mprPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	// Minimal schema required by both sdk/mpr and modelsdk/mpr readers.
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

	unitIDStr := "11111111-1111-1111-1111-111111111111"
	unitID = model.ID(unitIDStr)

	idBlob := secTestUUIDBlob(unitIDStr)
	secDoc := bson.D{
		{Key: "$Type", Value: "Security$ProjectSecurity"},
		{Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: idBlob}},
		{Key: "SecurityLevel", Value: "Security$SecurityLevel_Off"},
		{Key: "CheckSecurity", Value: true},
		{Key: "EnableDemoUsers", Value: false},
	}
	secBytes, err := bson.Marshal(secDoc)
	if err != nil {
		t.Fatalf("marshal security BSON: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		 VALUES (?, ?, ?, 0, '', '', ?)`,
		idBlob,
		make([]byte, 16),
		"ProjectSecurity",
		secBytes,
	); err != nil {
		t.Fatalf("insert security unit: %v", err)
	}

	return mprPath, unitID
}

// secTestUUIDBlob converts a UUID string to the 16-byte Mendix GUID blob
// (little-endian groups 1-3, big-endian groups 4-5).
func secTestUUIDBlob(uuid string) []byte {
	hex := ""
	for _, ch := range uuid {
		if ch != '-' {
			hex += string(ch)
		}
	}
	blob := make([]byte, 16)
	for i := 0; i < 16; i++ {
		var b byte
		fmt.Sscanf(hex[i*2:i*2+2], "%02x", &b)
		blob[i] = b
	}
	blob[0], blob[1], blob[2], blob[3] = blob[3], blob[2], blob[1], blob[0]
	blob[4], blob[5] = blob[5], blob[4]
	blob[6], blob[7] = blob[7], blob[6]
	return blob
}

func TestSetProjectSecurityLevel_ViaModelsdk(t *testing.T) {
	mprPath, unitID := makeSecurityTestMPR(t)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	const wantLevel = "Security$SecurityLevel_Production"
	if err := b.SetProjectSecurityLevel(unitID, wantLevel); err != nil {
		t.Fatalf("SetProjectSecurityLevel: %v", err)
	}

	// Read back via modelsdk to verify the field persisted.
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes after write: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(rawBytes, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := ""
	for _, e := range doc {
		if e.Key == "SecurityLevel" {
			got, _ = e.Value.(string)
		}
	}
	if got != wantLevel {
		t.Errorf("SecurityLevel = %q, want %q", got, wantLevel)
	}
}
