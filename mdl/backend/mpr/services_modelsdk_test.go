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

func TestUpdateBusinessEventServiceViaModelsdk(t *testing.T) {
	const unitID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	contents := makeBSONUnit(t, unitID, "BusinessEvents$BusinessEventService", bson.D{
		{Key: "Name", Value: "OldService"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "Definition", Value: nil},
		{Key: "SourceApi", Value: nil},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	svc := &model.BusinessEventService{
		Name:          "NewService",
		Documentation: "Updated docs",
		ExportLevel:   "Hidden",
	}
	svc.ID = id
	if err := b.updateBusinessEventServiceViaModelsdk(svc); err != nil {
		t.Fatalf("updateBusinessEventServiceViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewService" {
		t.Errorf("Name = %q, want %q", got, "NewService")
	}
	if got := readBSONField(t, raw, "Documentation"); got != "Updated docs" {
		t.Errorf("Documentation = %q, want %q", got, "Updated docs")
	}
}

func TestUpdateImportMappingViaModelsdk(t *testing.T) {
	const unitID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	contents := makeBSONUnit(t, unitID, "ImportMappings$ImportMapping", bson.D{
		{Key: "Name", Value: "OldMapping"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "XmlSchema", Value: ""},
		{Key: "JsonStructure", Value: ""},
		{Key: "RootElementName", Value: ""},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	im := &model.ImportMapping{
		Name:          "NewMapping",
		Documentation: "docs",
		ExportLevel:   "Hidden",
		JsonStructure: "MyModule.MyStructure",
	}
	im.ID = id
	if err := b.updateImportMappingViaModelsdk(im); err != nil {
		t.Fatalf("updateImportMappingViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewMapping" {
		t.Errorf("Name = %q, want %q", got, "NewMapping")
	}
	if got := readBSONField(t, raw, "JsonStructure"); got != "MyModule.MyStructure" {
		t.Errorf("JsonStructure = %q, want %q", got, "MyModule.MyStructure")
	}
}

func TestUpdateExportMappingViaModelsdk(t *testing.T) {
	const unitID = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	contents := makeBSONUnit(t, unitID, "ExportMappings$ExportMapping", bson.D{
		{Key: "Name", Value: "OldExport"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	em := &model.ExportMapping{
		Name:        "NewExport",
		ExportLevel: "Hidden",
	}
	em.ID = id
	if err := b.updateExportMappingViaModelsdk(em); err != nil {
		t.Fatalf("updateExportMappingViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewExport" {
		t.Errorf("Name = %q, want %q", got, "NewExport")
	}
}

func TestUpdateDataTransformerViaModelsdk(t *testing.T) {
	const unitID = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	contents := makeBSONUnit(t, unitID, "DataTransformers$DataTransformer", bson.D{
		{Key: "Name", Value: "OldTransformer"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "SourceType", Value: "XML"},
		{Key: "SourceJson", Value: ""},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	dt := &model.DataTransformer{
		Name: "NewTransformer",
	}
	dt.ID = id
	if err := b.updateDataTransformerViaModelsdk(dt); err != nil {
		t.Fatalf("updateDataTransformerViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewTransformer" {
		t.Errorf("Name = %q, want %q", got, "NewTransformer")
	}
}

func TestUpdateDatabaseConnectionViaModelsdk(t *testing.T) {
	const unitID = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	contents := makeBSONUnit(t, unitID, "DatabaseConnector$DatabaseConnection", bson.D{
		{Key: "Name", Value: "OldConn"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "DatabaseType", Value: "PostgreSQL"},
		{Key: "ConnectionString", Value: "MyModule.OldConn"},
		{Key: "UserName", Value: "MyModule.OldUser"},
		{Key: "Password", Value: "MyModule.OldPass"},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	conn := &model.DatabaseConnection{
		Name:             "NewConn",
		DatabaseType:     "MySQL",
		ConnectionString: "MyModule.NewConn",
		UserName:         "MyModule.NewUser",
		Password:         "MyModule.NewPass",
		ExportLevel:      "Hidden",
	}
	conn.ID = id
	if err := b.updateDatabaseConnectionViaModelsdk(conn); err != nil {
		t.Fatalf("updateDatabaseConnectionViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "DatabaseType"); got != "MySQL" {
		t.Errorf("DatabaseType = %q, want %q", got, "MySQL")
	}
	if got := readBSONField(t, raw, "ConnectionString"); got != "MyModule.NewConn" {
		t.Errorf("ConnectionString = %q, want %q", got, "MyModule.NewConn")
	}
}

func TestUpdateConsumedRestServiceViaModelsdk(t *testing.T) {
	const unitID = "11111111-2222-2222-2222-222222222222"
	contents := makeBSONUnit(t, unitID, "Rest$ConsumedRestService", bson.D{
		{Key: "Name", Value: "OldRest"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	svc := &model.ConsumedRestService{
		Name:          "NewRest",
		Documentation: "Updated docs",
	}
	svc.ID = id
	if err := b.updateConsumedRestServiceViaModelsdk(svc); err != nil {
		t.Fatalf("updateConsumedRestServiceViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewRest" {
		t.Errorf("Name = %q, want %q", got, "NewRest")
	}
	if got := readBSONField(t, raw, "Documentation"); got != "Updated docs" {
		t.Errorf("Documentation = %q, want %q", got, "Updated docs")
	}
}

func TestUpdatePublishedRestServiceViaModelsdk(t *testing.T) {
	const unitID = "11111111-3333-3333-3333-333333333333"
	contents := makeBSONUnit(t, unitID, "Rest$PublishedRestService", bson.D{
		{Key: "Name", Value: "OldPub"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "Version", Value: "1"},
		{Key: "Path", Value: "/old"},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	svc := &model.PublishedRestService{
		Name:    "NewPub",
		Version: "2",
		Path:    "/new",
	}
	svc.ID = id
	if err := b.updatePublishedRestServiceViaModelsdk(svc); err != nil {
		t.Fatalf("updatePublishedRestServiceViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Version"); got != "2" {
		t.Errorf("Version = %q, want %q", got, "2")
	}
	if got := readBSONField(t, raw, "Path"); got != "/new" {
		t.Errorf("Path = %q, want %q", got, "/new")
	}
}

func TestUpdateAllowedRolesViaModelsdk_Microflow(t *testing.T) {
	const unitID = "44444444-4444-4444-4444-444444444444"
	contents := makeBSONUnit(t, unitID, "Microflows$Microflow", bson.D{
		{Key: "Name", Value: "TestMF"},
		{Key: "AllowedModuleRoles", Value: bson.A{int32(3)}},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	roles := []string{"MyModule.UserRole", "MyModule.AdminRole"}
	if err := b.updateAllowedRolesViaModelsdk(id, roles); err != nil {
		t.Fatalf("updateAllowedRolesViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var gotRoles []string
	for _, e := range doc {
		if e.Key == "AllowedModuleRoles" {
			if arr, ok := e.Value.(bson.A); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						gotRoles = append(gotRoles, s)
					}
				}
			}
		}
	}
	if len(gotRoles) != 2 || gotRoles[0] != "MyModule.UserRole" || gotRoles[1] != "MyModule.AdminRole" {
		t.Errorf("AllowedModuleRoles = %v, want %v", gotRoles, roles)
	}
}

func TestUpdateAllowedRolesViaModelsdk_Page(t *testing.T) {
	const unitID = "55555555-5555-5555-5555-555555555555"
	contents := makeBSONUnit(t, unitID, "Forms$Page", bson.D{
		{Key: "Name", Value: "TestPage"},
		{Key: "AllowedRoles", Value: bson.A{int32(3)}},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	roles := []string{"MyModule.UserRole"}
	if err := b.updateAllowedRolesViaModelsdk(id, roles); err != nil {
		t.Fatalf("updateAllowedRolesViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, e := range doc {
		if e.Key == "AllowedRoles" {
			if arr, ok := e.Value.(bson.A); ok {
				var found bool
				for _, item := range arr {
					if s, ok := item.(string); ok && s == "MyModule.UserRole" {
						found = true
					}
				}
				if !found {
					t.Errorf("AllowedRoles does not contain MyModule.UserRole: %v", arr)
				}
				return
			}
		}
	}
	t.Error("AllowedRoles field not found in BSON")
}

func TestRemoveFromAllowedRolesViaModelsdk_Microflow(t *testing.T) {
	const unitID = "44444444-5555-5555-5555-555555555555"
	contents := makeBSONUnit(t, unitID, "Microflows$Microflow", bson.D{
		{Key: "Name", Value: "TestMF"},
		{Key: "AllowedModuleRoles", Value: bson.A{int32(3), "MyModule.UserRole", "MyModule.AdminRole"}},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	removed, err := b.removeFromAllowedRolesViaModelsdk(id, "MyModule.UserRole")
	if err != nil {
		t.Fatalf("removeFromAllowedRolesViaModelsdk: %v", err)
	}
	if !removed {
		t.Errorf("removed = false, want true")
	}

	raw, _ := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	var doc bson.D
	bson.Unmarshal(raw, &doc)
	for _, e := range doc {
		if e.Key == "AllowedModuleRoles" {
			if arr, ok := e.Value.(bson.A); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok && s == "MyModule.UserRole" {
						t.Errorf("AllowedModuleRoles still contains MyModule.UserRole: %v", arr)
					}
				}
				return
			}
		}
	}
}

func TestUpdatePublishedRestServiceRolesViaModelsdk(t *testing.T) {
	const unitID = "66666666-6666-6666-6666-666666666666"
	contents := makeBSONUnit(t, unitID, "Rest$PublishedRestService", bson.D{
		{Key: "Name", Value: "TestREST"},
		{Key: "AllowedRoles", Value: bson.A{int32(3)}},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	roles := []string{"MyModule.UserRole"}
	if err := b.updatePublishedRestServiceRolesViaModelsdk(id, roles); err != nil {
		t.Fatalf("updatePublishedRestServiceRolesViaModelsdk: %v", err)
	}

	raw, _ := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	var doc bson.D
	bson.Unmarshal(raw, &doc)
	for _, e := range doc {
		if e.Key == "AllowedRoles" {
			if arr, ok := e.Value.(bson.A); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok && s == "MyModule.UserRole" {
						return
					}
				}
			}
		}
	}
	t.Error("AllowedRoles does not contain expected role")
}

func TestUpdateJavaActionGen(t *testing.T) {
	const unitID = "11111111-4444-4444-4444-444444444444"
	contents := makeBSONUnit(t, unitID, "JavaActions$JavaAction", bson.D{
		{Key: "Name", Value: "OldAction"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Stage 3.3.2.E1: gen-native UpdateJavaActionGen via repo.
	ja, err := b.ReadJavaActionByNameGen("MyModule.OldAction")
	if err != nil || ja == nil {
		// FindByQualifiedName needs the module/container join; fall
		// back to reading the raw unit and decoding.
		all, _ := b.ListJavaActionsGen()
		for _, candidate := range all {
			if string(candidate.ID()) == unitID {
				ja = candidate
				break
			}
		}
		if ja == nil {
			t.Fatalf("could not locate test JavaAction by ID %q", unitID)
		}
	}
	ja.SetName("NewAction")

	if err := b.UpdateJavaActionGen(ja); err != nil {
		t.Fatalf("UpdateJavaActionGen: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewAction" {
		t.Errorf("Name = %q, want %q", got, "NewAction")
	}
}

func TestUpdateConsumedODataServiceViaModelsdk(t *testing.T) {
	const unitID = "11111111-5555-5555-5555-555555555555"
	contents := makeBSONUnit(t, unitID, "Rest$ConsumedODataService", bson.D{
		{Key: "Name", Value: "OldOData"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "Version", Value: "1.0"},
		{Key: "ServiceName", Value: "OldSvc"},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	svc := &model.ConsumedODataService{
		Name:         "NewOData",
		Version:      "2.0",
		ServiceName:  "NewSvc",
		ODataVersion: "V4",
	}
	svc.ID = id
	if err := b.updateConsumedODataServiceViaModelsdk(svc); err != nil {
		t.Fatalf("updateConsumedODataServiceViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewOData" {
		t.Errorf("Name = %q, want %q", got, "NewOData")
	}
	if got := readBSONField(t, raw, "Version"); got != "2.0" {
		t.Errorf("Version = %q, want %q", got, "2.0")
	}
}

func TestUpdatePublishedODataServiceViaModelsdk(t *testing.T) {
	const unitID = "11111111-6666-6666-6666-666666666666"
	contents := makeBSONUnit(t, unitID, "ODataPublish$PublishedODataService2", bson.D{
		{Key: "Name", Value: "OldPubOData"},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "Version", Value: "1"},
		{Key: "Path", Value: "/old"},
	})
	mprPath, id := makeServiceTestMPR(t, unitID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	svc := &model.PublishedODataService{
		Name:    "NewPubOData",
		Version: "2",
		Path:    "/new",
	}
	svc.ID = id
	if err := b.updatePublishedODataServiceViaModelsdk(svc); err != nil {
		t.Fatalf("updatePublishedODataServiceViaModelsdk: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "NewPubOData" {
		t.Errorf("Name = %q, want %q", got, "NewPubOData")
	}
	if got := readBSONField(t, raw, "Path"); got != "/new" {
		t.Errorf("Path = %q, want %q", got, "/new")
	}
}
