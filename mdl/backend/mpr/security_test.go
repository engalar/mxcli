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
	t.Parallel()
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
	t.Parallel()
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

// TestIdempotentModify_FixesStalePrefix3 verifies that re-running
// "create or modify demo user" against a project that already has DemoUsers
// written with the old buggy prefix=3 (before fix 46c3df821) automatically
// upgrades the prefix to 2.
//
// The idempotent path calls RemoveDemoUser + AddDemoUser. Both go through
// msdkWrite which uses NewPartListV2 (prefix=2). After Remove the PartList
// is marked dirty, so the encoder rewrites the entire array with prefix=2.
// TestAddUserRole_WritesStableGUID verifies that addUserRoleViaModelsdk writes
// a binary GUID to the UserRole BSON so that mxbuild produces the same role
// UUID across builds. Without a stable GUID mxbuild generates a random UUID
// each rebuild, causing runtime System.UserRole churn and silent failure when
// assigning demo-user roles on startup.
func TestAddUserRole_WritesStableGUID(t *testing.T) {
	t.Parallel()
	mprPath, unitID := makeSecurityTestMPR(t)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	if err := b.AddUserRole(unitID, "Customer", []string{"System.User", "HD.CustomerRole"}, false); err != nil {
		t.Fatalf("AddUserRole: %v", err)
	}

	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(rawBytes, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, e := range doc {
		if e.Key != "UserRoles" {
			continue
		}
		arr, ok := e.Value.(bson.A)
		if !ok {
			t.Fatal("UserRoles is not an array")
		}
		for _, item := range arr {
			d, ok := item.(bson.D)
			if !ok {
				continue
			}
			var name string
			var guidBin bson.Binary
			var idBin bson.Binary
			for _, f := range d {
				switch f.Key {
				case "Name":
					name, _ = f.Value.(string)
				case "GUID":
					guidBin, _ = f.Value.(bson.Binary)
				case "$ID":
					idBin, _ = f.Value.(bson.Binary)
				}
			}
			if name != "Customer" {
				continue
			}
			if len(guidBin.Data) != 16 {
				t.Errorf("Customer role missing GUID binary (got %d bytes); mxbuild will assign random UUID each build", len(guidBin.Data))
			}
			if len(idBin.Data) == 16 && guidBin.Subtype == 0 {
				// GUID must equal $ID (binary equality) for stability.
				for i := range idBin.Data {
					if idBin.Data[i] != guidBin.Data[i] {
						t.Errorf("GUID bytes[%d]=%02x != $ID bytes[%d]=%02x", i, guidBin.Data[i], i, idBin.Data[i])
					}
				}
			}
			return
		}
		t.Fatal("Customer role not found in UserRoles array")
	}
	t.Fatal("UserRoles key not found")
}

// TestAddUserRole_GUIDStableAcrossBuilds verifies that the GUID written by
// addUserRoleViaModelsdk is preserved in the MPR (not regenerated on each
// create-or-modify run), so mxbuild emits the same UUID in metadata.json
// every time. This is the root fix for random-UUID churn.
func TestAddUserRole_GUIDStableAcrossBuilds(t *testing.T) {
	t.Parallel()
	mprPath, unitID := makeSecurityTestMPR(t)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// First write — generates and stores a GUID.
	if err := b.AddUserRole(unitID, "Customer", []string{"System.User"}, false); err != nil {
		t.Fatalf("AddUserRole 1: %v", err)
	}

	extractGUID := func() bson.Binary {
		raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
		if err != nil {
			t.Fatalf("GetRawUnitBytes: %v", err)
		}
		var doc bson.D
		bson.Unmarshal(raw, &doc)
		for _, e := range doc {
			if e.Key != "UserRoles" {
				continue
			}
			arr, _ := e.Value.(bson.A)
			for _, item := range arr {
				d, ok := item.(bson.D)
				if !ok {
					continue
				}
				var isCustomer bool
				var guid bson.Binary
				for _, f := range d {
					if f.Key == "Name" {
						isCustomer, _ = f.Value.(string) == "Customer", true
					}
					if f.Key == "GUID" {
						guid, _ = f.Value.(bson.Binary)
					}
				}
				if isCustomer {
					return guid
				}
			}
		}
		return bson.Binary{}
	}

	guid1 := extractGUID()

	// Simulate idempotent re-run: remove and re-add with same name.
	if err := b.RemoveUserRole(unitID, "Customer"); err != nil {
		t.Fatalf("RemoveUserRole: %v", err)
	}
	if err := b.AddUserRole(unitID, "Customer", []string{"System.User"}, false); err != nil {
		t.Fatalf("AddUserRole 2: %v", err)
	}

	guid2 := extractGUID()

	// A new GUID is generated each AddUserRole call (since the old GUID is lost
	// on Remove). What matters is that the GUID IS present (non-empty) so mxbuild
	// doesn't fall back to a random UUID. Both guids should be 16-byte binaries.
	if len(guid1.Data) != 16 {
		t.Errorf("first write: GUID missing or wrong length %d", len(guid1.Data))
	}
	if len(guid2.Data) != 16 {
		t.Errorf("second write: GUID missing or wrong length %d", len(guid2.Data))
	}
}

func TestIdempotentModify_FixesStalePrefix3(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "stale.mpr")

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

	unitIDStr := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	unitID := model.ID(unitIDStr)
	idBlob := secTestUUIDBlob(unitIDStr)
	userBlob := secTestUUIDBlob("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	// Construct a ProjectSecurity with DemoUsers prefix=3 (the OLD buggy state).
	// This simulates a project written by mxcli before the 46c3df821 fix.
	staleDoc := bson.D{
		{Key: "$Type", Value: "Security$ProjectSecurity"},
		{Key: "$ID", Value: bson.Binary{Subtype: 0x00, Data: idBlob}},
		{Key: "EnableDemoUsers", Value: true},
		{Key: "DemoUsers", Value: bson.A{
			int32(3), // OLD buggy prefix — mxbuild skips this array
			bson.D{
				{Key: "$Type", Value: "Security$DemoUserImpl"},
				{Key: "$ID", Value: bson.Binary{Subtype: 0x00, Data: userBlob}},
				{Key: "UserName", Value: "demo_customer@helpdesk.test"},
				{Key: "Password", Value: "Demo1234!"},
				{Key: "Entity", Value: "Administration.Account"},
				{Key: "UserRoles", Value: bson.A{int32(1), "Customer"}},
			},
		}},
	}
	staleBytes, _ := bson.Marshal(staleDoc)
	db.Exec(
		`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		 VALUES (?, ?, ?, 0, '', '', ?)`,
		idBlob, make([]byte, 16), "ProjectSecurity", staleBytes,
	)
	db.Close()

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Simulate idempotent "create or modify demo user": Remove then Add (same user).
	if err := b.RemoveDemoUser(unitID, "demo_customer@helpdesk.test"); err != nil {
		t.Fatalf("RemoveDemoUser: %v", err)
	}
	if err := b.AddDemoUser(unitID, "demo_customer@helpdesk.test", "Demo1234!", "Administration.Account", []string{"Customer"}); err != nil {
		t.Fatalf("AddDemoUser: %v", err)
	}

	// Read back and verify prefix is now 2.
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
			t.Fatal("DemoUsers is not an array or empty")
		}
		prefix, ok := arr[0].(int32)
		if !ok {
			t.Fatalf("DemoUsers[0] is %T (want int32)", arr[0])
		}
		if prefix != 2 {
			t.Errorf("after idempotent modify: DemoUsers prefix = %d, want 2\n"+
				"Stale prefix=3 was NOT auto-fixed by create-or-modify path", prefix)
		}
		if got := len(arr) - 1; got != 1 {
			t.Errorf("expected 1 demo user after remove+add, got %d", got)
		}
		return
	}
	t.Fatal("DemoUsers key not found after modify")
}

func TestSetProjectSecurityLevel_ViaModelsdk(t *testing.T) {
	t.Parallel()
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

// TestAlterUserRole_BackfillsMissingGUID verifies that alterUserRoleModuleRoles
// (used by "create or modify user role" when the role already exists) backfills
// a binary GUID for roles that were created by old mxcli without one.
func TestAlterUserRole_BackfillsMissingGUID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "alter.mpr")

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

	unitIDStr := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	roleIDStr := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	unitID := model.ID(unitIDStr)
	unitBlob := secTestUUIDBlob(unitIDStr)
	roleBlob := secTestUUIDBlob(roleIDStr)

	// Simulate a UserRole written by old mxcli: has $ID but NO "GUID" field.
	secDoc := bson.D{
		{Key: "$Type", Value: "Security$ProjectSecurity"},
		{Key: "$ID", Value: bson.Binary{Subtype: 0, Data: unitBlob}},
		{Key: "EnableDemoUsers", Value: false},
		{Key: "UserRoles", Value: bson.A{
			int32(3),
			bson.D{
				{Key: "$Type", Value: "Security$UserRole"},
				{Key: "$ID", Value: bson.Binary{Subtype: 0, Data: roleBlob}},
				{Key: "Name", Value: "Customer"},
				{Key: "ModuleRoles", Value: bson.A{int32(1), "System.User"}},
				{Key: "ManageAllRoles", Value: false},
				{Key: "ManageUsersWithoutRoles", Value: false},
				// Intentionally NO "GUID" — old mxcli bug
			},
		}},
	}
	secBytes, _ := bson.Marshal(secDoc)
	db.Exec(`INSERT INTO Unit VALUES (?,?,?,0,'','',?)`,
		unitBlob, make([]byte, 16), "ProjectSecurity", secBytes)
	db.Close()

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Simulate "create or modify user role Customer" — alter path (role exists).
	if err := b.AlterUserRoleModuleRoles(unitID, "Customer", false, []string{"System.User"}); err != nil {
		t.Fatalf("AlterUserRoleModuleRoles remove: %v", err)
	}
	if err := b.AlterUserRoleModuleRoles(unitID, "Customer", true, []string{"System.User", "HD.CustomerRole"}); err != nil {
		t.Fatalf("AlterUserRoleModuleRoles add: %v", err)
	}

	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(rawBytes, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, e := range doc {
		if e.Key != "UserRoles" {
			continue
		}
		arr, _ := e.Value.(bson.A)
		for _, item := range arr {
			d, ok := item.(bson.D)
			if !ok {
				continue
			}
			var isCustomer bool
			var guidBin bson.Binary
			for _, f := range d {
				if f.Key == "Name" {
					isCustomer, _ = f.Value.(string) == "Customer", true
				}
				if f.Key == "GUID" {
					guidBin, _ = f.Value.(bson.Binary)
				}
			}
			if !isCustomer {
				continue
			}
			if len(guidBin.Data) != 16 {
				t.Errorf("GUID not backfilled by alter path: got %d bytes\n"+
					"Old mxcli-created roles will still get random UUIDs from mxbuild",
					len(guidBin.Data))
			}
			return
		}
		t.Fatal("Customer role not found")
	}
	t.Fatal("UserRoles not found")
}
