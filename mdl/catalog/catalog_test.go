// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	if cat.IsBuilt() {
		t.Error("New catalog should not be built")
	}
	if cat.ProjectID() != "default" {
		t.Errorf("Expected default project ID, got %q", cat.ProjectID())
	}
}

func TestSetProject(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	cat.SetProject("proj-1", "MyApp", "10.0.0")

	if cat.ProjectID() != "proj-1" {
		t.Errorf("Expected project ID 'proj-1', got %q", cat.ProjectID())
	}
	if cat.ProjectName() != "MyApp" {
		t.Errorf("Expected project name 'MyApp', got %q", cat.ProjectName())
	}
}

func TestSetBuilt(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	if cat.IsBuilt() {
		t.Error("New catalog should not be built")
	}

	cat.SetBuilt(true)
	if !cat.IsBuilt() {
		t.Error("Catalog should be built after SetBuilt(true)")
	}

	cat.SetBuilt(false)
	if cat.IsBuilt() {
		t.Error("Catalog should not be built after SetBuilt(false)")
	}
}

func TestTables(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	tables := cat.Tables()
	if len(tables) == 0 {
		t.Fatal("Expected non-empty table list")
	}

	// Verify some key tables exist
	expected := []string{
		"CATALOG.MODULES",
		"CATALOG.ENTITIES",
		"CATALOG.ATTRIBUTES",
		"CATALOG.MICROFLOWS",
		"CATALOG.PAGES",
		"CATALOG.WIDGETS",
		"CATALOG.DATABASE_CONNECTIONS",
	}
	tableSet := make(map[string]bool)
	for _, tbl := range tables {
		tableSet[tbl] = true
	}
	for _, exp := range expected {
		if !tableSet[exp] {
			t.Errorf("Expected table %q not found in Tables()", exp)
		}
	}
}

func TestQueryEmptyTable(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	result, err := cat.Query("SELECT * FROM modules")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Expected 0 rows in empty table, got %d", result.Count)
	}
	if len(result.Columns) == 0 {
		t.Error("Expected column names even for empty result")
	}
}

func TestQueryWithData(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	// Insert test data
	_, err = cat.CatalogDB().Exec(
		"INSERT INTO modules (Id, Name, QualifiedName, ModuleName) VALUES (?, ?, ?, ?)",
		"mod-1", "TestModule", "TestModule", "TestModule",
	)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	result, err := cat.Query("SELECT Name FROM modules WHERE Id = 'mod-1'")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count != 1 {
		t.Fatalf("Expected 1 row, got %d", result.Count)
	}
	if result.Rows[0][0] != "TestModule" {
		t.Errorf("Expected 'TestModule', got %v", result.Rows[0][0])
	}
}

func TestQueryError(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	_, err = cat.Query("SELECT * FROM nonexistent_table")
	if err == nil {
		t.Error("Expected error for nonexistent table")
	}
}

func TestMetadata(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	// Set and get metadata
	if err := cat.SetMeta("test_key", "test_value"); err != nil {
		t.Fatalf("SetMeta failed: %v", err)
	}

	val, err := cat.GetMeta("test_key")
	if err != nil {
		t.Fatalf("GetMeta failed: %v", err)
	}
	if val != "test_value" {
		t.Errorf("Expected 'test_value', got %q", val)
	}

	// Get nonexistent key returns empty string
	val, err = cat.GetMeta("nonexistent")
	if err != nil {
		t.Fatalf("GetMeta for nonexistent key failed: %v", err)
	}
	if val != "" {
		t.Errorf("Expected empty string for nonexistent key, got %q", val)
	}

	// Overwrite metadata
	if err := cat.SetMeta("test_key", "updated_value"); err != nil {
		t.Fatalf("SetMeta overwrite failed: %v", err)
	}
	val, _ = cat.GetMeta("test_key")
	if val != "updated_value" {
		t.Errorf("Expected 'updated_value', got %q", val)
	}
}

func TestCacheInfo(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	modTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	buildDuration := 2 * time.Second

	err = cat.SetCacheInfo("/path/to/app.mpr", modTime, "10.0.0", "full", buildDuration)
	if err != nil {
		t.Fatalf("SetCacheInfo failed: %v", err)
	}

	info, err := cat.GetCacheInfo()
	if err != nil {
		t.Fatalf("GetCacheInfo failed: %v", err)
	}

	if info.MprPath != "/path/to/app.mpr" {
		t.Errorf("Expected MprPath '/path/to/app.mpr', got %q", info.MprPath)
	}
	if info.MendixVersion != "10.0.0" {
		t.Errorf("Expected MendixVersion '10.0.0', got %q", info.MendixVersion)
	}
	if info.BuildMode != "full" {
		t.Errorf("Expected BuildMode 'full', got %q", info.BuildMode)
	}
	if info.BuildDuration != buildDuration {
		t.Errorf("Expected BuildDuration %v, got %v", buildDuration, info.BuildDuration)
	}
}

func TestSnapshots(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	// No active snapshot initially
	if snap := cat.GetActiveSnapshot(); snap != nil {
		t.Error("Expected no active snapshot initially")
	}

	// Create snapshot
	snap := cat.CreateSnapshot("test-snap", SnapshotSourceLive)
	if snap == nil {
		t.Fatal("CreateSnapshot returned nil")
	}
	if snap.Name != "test-snap" {
		t.Errorf("Expected snapshot name 'test-snap', got %q", snap.Name)
	}
	if snap.Source != SnapshotSourceLive {
		t.Errorf("Expected source LIVE, got %q", snap.Source)
	}

	// Active snapshot should be set
	active := cat.GetActiveSnapshot()
	if active == nil {
		t.Fatal("Expected active snapshot after CreateSnapshot")
	}
	if active.ID != snap.ID {
		t.Errorf("Expected active snapshot ID %q, got %q", snap.ID, active.ID)
	}

	// Create second snapshot replaces active
	snap2 := cat.CreateSnapshot("snap-2", SnapshotSourceGit)
	active = cat.GetActiveSnapshot()
	if active.ID != snap2.ID {
		t.Errorf("Expected active snapshot to be snap-2, got %q", active.ID)
	}
}

func TestSaveAndLoadFromFile(t *testing.T) {
	// Create and populate catalog
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	cat.SetProject("proj-1", "TestApp", "10.0.0")
	_, err = cat.CatalogDB().Exec(
		"INSERT INTO modules (Id, Name, QualifiedName, ModuleName) VALUES (?, ?, ?, ?)",
		"mod-1", "MyModule", "MyModule", "MyModule",
	)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	if err := cat.SetMeta("test_key", "test_value"); err != nil {
		t.Fatalf("SetMeta failed: %v", err)
	}

	// Save to file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "catalog.db")
	if err := cat.SaveToFile(filePath); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}
	cat.Close()

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Catalog file was not created")
	}

	// Load from file
	loaded, err := NewFromFile(filePath)
	if err != nil {
		t.Fatalf("NewFromFile failed: %v", err)
	}
	defer loaded.Close()

	// Loaded catalog should be marked as built
	if !loaded.IsBuilt() {
		t.Error("Loaded catalog should be marked as built")
	}

	// Query data from loaded catalog
	result, err := loaded.Query("SELECT Name FROM modules WHERE Id = 'mod-1'")
	if err != nil {
		t.Fatalf("Query on loaded catalog failed: %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("Expected 1 row from loaded catalog, got %d", result.Count)
	}
	if result.Rows[0][0] != "MyModule" {
		t.Errorf("Expected 'MyModule' from loaded catalog, got %v", result.Rows[0][0])
	}

	// Check metadata survived
	val, err := loaded.GetMeta("test_key")
	if err != nil {
		t.Fatalf("GetMeta on loaded catalog failed: %v", err)
	}
	if val != "test_value" {
		t.Errorf("Expected metadata 'test_value' from loaded catalog, got %q", val)
	}
}

func TestRoleMappingsTable(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	// Insert role mappings
	mappings := []struct {
		userRole   string
		moduleRole string
		module     string
	}{
		{"Administrator", "MyModule.Admin", "MyModule"},
		{"Administrator", "System.Administrator", "System"},
		{"User", "MyModule.User", "MyModule"},
		{"User", "System.User", "System"},
	}

	for _, m := range mappings {
		_, err := cat.CatalogDB().Exec(
			"INSERT INTO role_mappings (UserRoleName, ModuleRoleName, ModuleName) VALUES (?, ?, ?)",
			m.userRole, m.moduleRole, m.module,
		)
		if err != nil {
			t.Fatalf("Failed to insert role mapping: %v", err)
		}
	}

	// Query all mappings for Administrator
	result, err := cat.Query("SELECT ModuleRoleName FROM role_mappings WHERE UserRoleName = 'Administrator'")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Expected 2 mappings for Administrator, got %d", result.Count)
	}

	// Query distinct module roles
	result, err = cat.Query("SELECT DISTINCT ModuleRoleName FROM role_mappings WHERE ModuleName = 'MyModule'")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Expected 2 module roles in MyModule, got %d", result.Count)
	}

	// Verify ROLE_MAPPINGS is in Tables() list
	found := false
	for _, tbl := range cat.Tables() {
		if tbl == "CATALOG.ROLE_MAPPINGS" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected CATALOG.ROLE_MAPPINGS in Tables() list")
	}
}

func TestCreateTablesAreQueryable(t *testing.T) {
	cat, err := New()
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer cat.Close()

	// Verify all core tables can be queried
	coreTables := []string{
		"modules", "entities", "attributes", "microflows",
		"pages", "snippets", "layouts", "enumerations",
		"java_actions", "projects", "snapshots", "catalog_meta",
		"workflows", "odata_clients", "odata_services",
		"business_event_services", "database_connections",
		"jar_dependencies", "role_mappings",
	}
	for _, tbl := range coreTables {
		_, err := cat.Query("SELECT * FROM " + tbl)
		if err != nil {
			t.Errorf("Failed to query table %q: %v", tbl, err)
		}
	}
}
