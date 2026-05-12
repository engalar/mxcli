// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"database/sql"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	_ "modernc.org/sqlite"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// makeSecurityTestMPR creates a minimal MPR v1 SQLite file containing one
// Security$ProjectSecurity unit with SecurityLevel="Off". Returns the path and
// the ProjectSecurity unit ID.
func makeSecurityTestMPR(t *testing.T) (string, model.ID) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "sec.mpr")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			TreeConflict LONG,
			ContentsHash TEXT,
			ContentsConflicts TEXT,
			Contents BLOB
		)`); err != nil {
		t.Fatalf("create Unit: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE _MetaData (
			_FormatVersion INTEGER,
			_ProductVersion TEXT,
			_BuildVersion TEXT,
			_SchemaHash TEXT
		)`); err != nil {
		t.Fatalf("create _MetaData: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO _MetaData VALUES (1, '11.6.0', '11.6.0', '')`); err != nil {
		t.Fatalf("seed _MetaData: %v", err)
	}

	const (
		unitIDStr      = "11111111-1111-1111-1111-111111111111"
		containerIDStr = "22222222-2222-2222-2222-222222222222"
	)

	doc := bson.D{
		{Key: "$Type", Value: "Security$ProjectSecurity"},
		{Key: "$ID", Value: mpr.IDToBsonBinary(unitIDStr)},
		{Key: "SecurityLevel", Value: "Off"},
		{Key: "CheckSecurity", Value: false},
		{Key: "EnableDemoUsers", Value: false},
		{Key: "UserRoles", Value: bson.A{int32(1)}},
		{Key: "DemoUsers", Value: bson.A{int32(1)}},
	}
	contents, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal ProjectSecurity: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		VALUES (?, ?, 'ProjectSecurity', 0, '', '', ?)`,
		mpr.IDToBsonBinary(unitIDStr).Data,
		mpr.IDToBsonBinary(containerIDStr).Data,
		contents,
	); err != nil {
		t.Fatalf("insert ProjectSecurity unit: %v", err)
	}

	return path, model.ID(unitIDStr)
}

// readSecurityLevel re-opens the MPR and returns the SecurityLevel field of the
// given unit by parsing its raw BSON.
func readSecurityLevel(t *testing.T, path string, unitID model.ID) string {
	t.Helper()

	r, err := mpr.Open(path)
	if err != nil {
		t.Fatalf("reopen mpr: %v", err)
	}
	defer r.Close()

	raw, err := r.GetRawUnitBytes(unitID)
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}

	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, f := range doc {
		if f.Key == "SecurityLevel" {
			s, _ := f.Value.(string)
			return s
		}
	}
	return ""
}

func TestSetProjectSecurityLevel_ViaModelsdk(t *testing.T) {
	path, unitID := makeSecurityTestMPR(t)

	b := New()
	if err := b.Connect(path); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if err := b.SetProjectSecurityLevel(unitID, "CheckEverything"); err != nil {
		_ = b.Disconnect()
		t.Fatalf("SetProjectSecurityLevel: %v", err)
	}

	if err := b.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	got := readSecurityLevel(t, path, unitID)
	if got != "CheckEverything" {
		t.Errorf("SecurityLevel after write = %q, want %q", got, "CheckEverything")
	}
}
