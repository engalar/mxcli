// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
	_ "modernc.org/sqlite"
)

func TestGetProjectSecurityGen_ReturnsSingleton(t *testing.T) {
	mprPath, _ := makeSecurityTestMPR(t)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	ps, err := b.GetProjectSecurityGen()
	if err != nil {
		t.Fatalf("GetProjectSecurityGen: %v", err)
	}
	if ps == nil {
		t.Fatal("GetProjectSecurityGen returned nil")
	}
}

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
		{Key: "$ID", Value: bson.Binary{Subtype: 0x00, Data: idBlob}},
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

// TestAddDemoUser_PreservesVersionPrefix2 asserts that AddDemoUser writes a
// DemoUsers BSON array whose first element (the versioned-array prefix) is
// int32(2), matching the format Studio Pro uses. Version 3 (the default from
// property.NewPartList) causes the Mendix runtime to skip demo-user account
// creation, producing "Login FAILED: unknown user" errors at runtime.
//
// This is a regression guard: if the test fails, DemoUsers are broken at runtime.
func TestAddDemoUser_PreservesVersionPrefix2(t *testing.T) {
	// Build a minimal ProjectSecurity fixture that already has one Studio-Pro-
	// style DemoUsers array (prefix=2, one existing user).
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "test.mpr")

	db, err := sql.Open("sqlite", mprPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE _MetaData (_FormatVersion INTEGER, _ProductVersion TEXT, _BuildVersion TEXT, _SchemaHash TEXT);
		INSERT INTO _MetaData VALUES (1,'10.18.0','10.18.0.0','testhash');
		CREATE TABLE _Transaction (LastTransactionID TEXT);
		INSERT INTO _Transaction VALUES ('00000000-0000-0000-0000-000000000000');
		CREATE TABLE Unit (UnitID BLOB PRIMARY KEY NOT NULL, ContainerID BLOB, ContainmentName TEXT,
		  TreeConflict LONG, ContentsHash TEXT, ContentsConflicts TEXT, Contents BLOB);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	unitIDStr := "22222222-2222-2222-2222-222222222222"
	unitID := model.ID(unitIDStr)
	idBlob := secTestUUIDBlob(unitIDStr)

	existingUserBlob := secTestUUIDBlob("33333333-3333-3333-3333-333333333333")
	// Construct a Studio-Pro-style ProjectSecurity: DemoUsers prefix=2 with one entry.
	secDoc := bson.D{
		{Key: "$Type", Value: "Security$ProjectSecurity"},
		{Key: "$ID", Value: bson.Binary{Subtype: 0x00, Data: idBlob}},
		{Key: "EnableDemoUsers", Value: true},
		{Key: "DemoUsers", Value: bson.A{
			int32(2), // Studio Pro writes version prefix 2
			bson.D{
				{Key: "$Type", Value: "Security$DemoUserImpl"},
				{Key: "$ID", Value: bson.Binary{Subtype: 0x00, Data: existingUserBlob}},
				{Key: "UserName", Value: "demo_administrator"},
				{Key: "Password", Value: "lmLpVVF38A1x"},
				{Key: "Entity", Value: "Administration.Account"},
				{Key: "UserRoles", Value: bson.A{int32(1), "Administrator"}},
			},
		}},
	}
	secBytes, _ := bson.Marshal(secDoc)
	db.Exec(
		`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		 VALUES (?, ?, ?, 0, '', '', ?)`,
		idBlob, make([]byte, 16), "ProjectSecurity", secBytes,
	)
	db.Close()

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Add a new demo user via mxcli — this triggers addDemoUserViaModelsdk.
	if err := b.AddDemoUser(unitID, "demo_agent@helpdesk.test", "Demo1234!", "Administration.Account", []string{"Agent"}); err != nil {
		t.Fatalf("AddDemoUser: %v", err)
	}

	// Read back raw BSON and check the DemoUsers version prefix.
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(rawBytes, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, e := range doc {
		if e.Key != "DemoUsers" {
			continue
		}
		arr, ok := e.Value.(bson.A)
		if !ok || len(arr) == 0 {
			t.Fatal("DemoUsers is not a BSON array or is empty")
		}
		prefix, ok := arr[0].(int32)
		if !ok {
			t.Fatalf("DemoUsers[0] is %T, want int32", arr[0])
		}
		// Studio Pro always writes prefix=2 for DemoUsers.
		// Mendix runtime skips demo-user creation when prefix=3.
		if prefix != 2 {
			t.Errorf("DemoUsers version prefix = %d, want 2 (Studio Pro format)\n"+
				"Hint: change demoUsers=property.NewPartList to property.NewPartListV2 in modelsdk/gen/security/types.go:initProjectSecurity()", prefix)
		}
		// Also verify the new user was actually appended (2 items total: existing + new).
		if got := len(arr) - 1; got != 2 {
			t.Errorf("DemoUsers has %d items, want 2 (1 existing + 1 new)", got)
		}
		return
	}
	t.Fatal("DemoUsers key not found in ProjectSecurity BSON")
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
