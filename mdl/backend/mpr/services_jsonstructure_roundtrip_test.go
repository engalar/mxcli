package mprbackend

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	_ "modernc.org/sqlite"
)

// makeModuleMPR creates a minimal v1 MPR with a module and one seed unit.
func makeModuleMPR(t *testing.T, moduleName string) (mprPath string) {
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

	moduleIDBlob := secTestUUIDBlob("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	moduleBSON := makeBSONUnit(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Projects$ModuleImpl", bson.D{
		{Key: "Name", Value: moduleName},
		{Key: "ModuleName", Value: moduleName},
	})
	if _, err := db.Exec(
		`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		 VALUES (?, x'00000000000000000000000000000000', 'Modules', 0, '', '', ?)`,
		moduleIDBlob, moduleBSON,
	); err != nil {
		t.Fatalf("insert module: %v", err)
	}

	return mprPath
}

func TestJsonStructureWriteThenQualifiedRead(t *testing.T) {
	t.Parallel()

	mprPath := makeModuleMPR(t, "TestModule")

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	jsonSnippet := `{"priority_no": 1, "Process_variant_rule_id": "abc"}`
	elements, err := types.BuildJsonElementsFromSnippet(jsonSnippet, nil)
	if err != nil {
		t.Fatalf("BuildJsonElementsFromSnippet: %v", err)
	}

	js := &types.JsonStructure{
		BaseElement: model.BaseElement{
			ID: model.ID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		},
		ContainerID: model.ID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name:        "TestSchema",
		JsonSnippet: jsonSnippet,
		Elements:    elements,
	}
	if err := b.createJsonStructureGen(js); err != nil {
		t.Fatalf("createJsonStructureGen: %v", err)
	}

	got, err := b.GetJsonStructureByQualifiedName("TestModule", "TestSchema")
	if err != nil {
		t.Fatalf("GetJsonStructureByQualifiedName: %v", err)
	}
	if got == nil {
		t.Fatal("GetJsonStructureByQualifiedName returned nil")
	}
	if len(got.Elements) == 0 {
		t.Fatal("Elements empty — causes CE5015: element tree not found on read-back")
	}
	root := got.Elements[0]
	if root.ExposedName != "Root" {
		t.Errorf("root ExposedName = %q, want Root", root.ExposedName)
	}
	if len(root.Children) != 2 {
		t.Fatalf("root Children count = %d, want 2", len(root.Children))
	}
}

func TestJsonStructureScriptTxThenQualifiedRead(t *testing.T) {
	t.Parallel()

	mprPath := makeModuleMPR(t, "TestModule")

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	tx, err := b.BeginScriptTransaction()
	if err != nil {
		t.Fatalf("BeginScriptTransaction: %v", err)
	}

	jsonSnippet := `{"priority_no": 1, "Process_variant_rule_id": "abc"}`
	elements, err := types.BuildJsonElementsFromSnippet(jsonSnippet, nil)
	if err != nil {
		t.Fatalf("BuildJsonElementsFromSnippet: %v", err)
	}

	js := &types.JsonStructure{
		BaseElement: model.BaseElement{
			ID: model.ID("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		},
		ContainerID: model.ID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name:        "ScriptSchema",
		JsonSnippet: jsonSnippet,
		Elements:    elements,
	}
	if err := b.createJsonStructureGen(js); err != nil {
		t.Fatalf("createJsonStructureGen inside script: %v", err)
	}

	got, err := b.GetJsonStructureByQualifiedName("TestModule", "ScriptSchema")
	if err != nil {
		t.Fatalf("GetJsonStructureByQualifiedName inside script: %v", err)
	}
	if got == nil {
		t.Fatal("GetJsonStructureByQualifiedName inside script returned nil")
	}
	if len(got.Elements) == 0 {
		t.Fatal("Elements empty inside script")
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err = b.GetJsonStructureByQualifiedName("TestModule", "ScriptSchema")
	if err != nil {
		t.Fatalf("GetJsonStructureByQualifiedName after commit: %v", err)
	}
	if got == nil {
		t.Fatal("GetJsonStructureByQualifiedName after commit returned nil")
	}
	if len(got.Elements) == 0 {
		t.Fatal("Elements empty after commit — CE5015 risk")
	}
}
