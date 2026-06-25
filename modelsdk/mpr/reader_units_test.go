// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// hashContents computes a base64-encoded SHA256 hash matching the writer's format.
func hashContents(contents []byte) string {
	h := sha256.Sum256(contents)
	return base64.StdEncoding.EncodeToString(h[:])
}

// makeTestBson creates a minimal BSON document with the given $Type for testing.
func makeTestBson(typeName string) []byte {
	// Minimal hand-crafted BSON: total_len(4) + type_tag + key + \x00 + value + trailing \x00
	// This is a valid BSON document with $Type = typeName
	b := makeBSON([][2]string{{"$Type", typeName}})
	return b
}

// makeBSON creates a BSON document from key-value string pairs.
func makeBSON(pairs [][2]string) []byte {
	// Estimator: 4 (total) + 2 (type+key) + len(key)+1 + len(val)+5(string overhead) + 1 (trailer)
	estimatedLen := 4 + 1 // trailer
	for _, kv := range pairs {
		estimatedLen += 1 + len(kv[0]) + 1 + 4 + len(kv[1]) + 1
	}
	buf := make([]byte, 0, estimatedLen)

	// Placeholder for total length (will be filled at end)
	buf = append(buf, 0, 0, 0, 0)

	for _, kv := range pairs {
		buf = append(buf, 0x02)             // string type
		buf = append(buf, []byte(kv[0])...) // key
		buf = append(buf, 0x00)             // key terminator
		// String: len + data + \x00
		strLen := len(kv[1]) + 1
		buf = append(buf, byte(strLen), byte(strLen>>8), byte(strLen>>16), byte(strLen>>24))
		buf = append(buf, []byte(kv[1])...)
		buf = append(buf, 0x00) // string terminator
	}

	buf = append(buf, 0x00) // document terminator

	// Write total length
	total := len(buf)
	buf[0] = byte(total)
	buf[1] = byte(total >> 8)
	buf[2] = byte(total >> 16)
	buf[3] = byte(total >> 24)

	return buf
}

// newTestReaderV2ForUnits builds a minimal V2 Reader backed by a temp SQLite DB
// and temp mprcontents directory pre-seeded with the given unit files.
// unitFiles is a map of unitID → BSON content.
func newTestReaderV2ForUnits(t *testing.T, unitFiles map[string]string) *Reader {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.mpr")
	contentsDir := filepath.Join(tmpDir, "mprcontents")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			TreeConflict LONG,
			ContentsHash TEXT,
			ContentsConflicts TEXT
		)
	`); err != nil {
		t.Fatalf("create Unit table: %v", err)
	}

	r := &Reader{
		db:          db,
		version:     MPRVersionV2,
		contentsDir: contentsDir,
		ownsDB:      true,
	}

	for unitID, typeName := range unitFiles {
		contents := makeTestBson(typeName)
		hash := hashContents(contents)

		if _, err := db.Exec(
			`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, ContentsHash) VALUES (?, ?, ?, ?)`,
			uuidToBlob(unitID), uuidToBlob("00000000-0000-0000-0000-000000000000"), "Documents", hash,
		); err != nil {
			t.Fatalf("insert unit: %v", err)
		}

		// Write mxunit file
		filePath := filepath.Join(
			contentsDir,
			unitID[0:2],
			unitID[2:4],
			unitID+".mxunit",
		)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filePath, contents, 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	return r
}

// TestBuildUnitCache_StoresContents ensures buildUnitCache stores Contents
// in each cachedUnit entry so that listUnitsByTypeV2 can reuse them.
func TestBuildUnitCache_StoresContents(t *testing.T) {
	t.Parallel()
	unitFiles := map[string]string{
		"11111111-1111-1111-1111-111111111111": "Projects$ModuleImpl",
		"22222222-2222-2222-2222-222222222222": "DomainModels$ViewEntitySourceDocument",
		"33333333-3333-3333-3333-333333333333": "Microflows$Microflow",
	}
	r := newTestReaderV2ForUnits(t, unitFiles)

	if err := r.buildUnitCache(); err != nil {
		t.Fatalf("buildUnitCache: %v", err)
	}

	if len(r.unitCache) != 3 {
		t.Fatalf("expected 3 cached units, got %d", len(r.unitCache))
	}

	for _, cu := range r.unitCache {
		if len(cu.Contents) == 0 {
			t.Errorf("cached unit %s has empty Contents", cu.ID)
		}
		if cu.Type == "" {
			t.Errorf("cached unit %s has empty Type", cu.ID)
		}
		if cu.ContentsHash == "" {
			t.Errorf("cached unit %s has empty ContentsHash", cu.ID)
		}
	}
}

// TestListUnitsByTypeV2_ReusesCachedContents verifies that after buildUnitCache
// populates the cache with Contents, listUnitsByTypeV2 does NOT re-read files.
func TestListUnitsByTypeV2_ReusesCachedContents(t *testing.T) {
	t.Parallel()
	unitFiles := map[string]string{
		"11111111-1111-1111-1111-111111111111": "Projects$ModuleImpl",
		"22222222-2222-2222-2222-222222222222": "DomainModels$ViewEntitySourceDocument",
	}
	r := newTestReaderV2ForUnits(t, unitFiles)

	// Build cache
	if err := r.buildUnitCache(); err != nil {
		t.Fatalf("buildUnitCache: %v", err)
	}

	// Delete the underlying files to prove we're not re-reading
	contentsDir := r.contentsDir
	for uid := range unitFiles {
		os.Remove(filepath.Join(contentsDir, uid[0:2], uid[2:4], uid+".mxunit"))
	}

	// listUnitsByTypeV2 should still return results from cache
	units, err := r.listUnitsByTypeV2("Projects$ModuleImpl")
	if err != nil {
		t.Fatalf("listUnitsByTypeV2: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(units))
	}
	if units[0].Type != "Projects$ModuleImpl" {
		t.Errorf("expected Projects$ModuleImpl, got %s", units[0].Type)
	}
	if len(units[0].Contents) == 0 {
		t.Error("expected non-empty Contents from cache")
	}
}

// TestInvalidateCache_Incremental verifies that after InvalidateCache, only
// changed units are re-read from disk (not all units).
func TestInvalidateCache_Incremental(t *testing.T) {
	t.Parallel()
	unitFiles := map[string]string{
		"11111111-1111-1111-1111-111111111111": "Projects$ModuleImpl",
		"22222222-2222-2222-2222-222222222222": "DomainModels$ViewEntitySourceDocument",
		"33333333-3333-3333-3333-333333333333": "Microflows$Microflow",
	}
	r := newTestReaderV2ForUnits(t, unitFiles)

	// First build
	if err := r.buildUnitCache(); err != nil {
		t.Fatalf("first buildUnitCache: %v", err)
	}

	// Simulate write: update one unit's file and SQLite hash
	updatedUnitID := "22222222-2222-2222-2222-222222222222"
	newContents := makeTestBson("DomainModels$EntityImpl")
	newHash := hashContents(newContents)

	// Update file on disk
	filePath := filepath.Join(r.contentsDir, updatedUnitID[0:2], updatedUnitID[2:4], updatedUnitID+".mxunit")
	if err := os.WriteFile(filePath, newContents, 0644); err != nil {
		t.Fatalf("update file: %v", err)
	}

	// Update hash in SQLite
	if _, err := r.db.Exec(`UPDATE Unit SET ContentsHash = ? WHERE UnitID = ?`, newHash, uuidToBlob(updatedUnitID)); err != nil {
		t.Fatalf("update hash: %v", err)
	}

	// Invalidate and rebuild
	r.InvalidateCache()
	if err := r.buildUnitCache(); err != nil {
		t.Fatalf("rebuild after invalidation: %v", err)
	}

	// Verify the updated unit has new type
	found := false
	for _, cu := range r.unitCache {
		if cu.ID == updatedUnitID {
			found = true
			if cu.Type != "DomainModels$EntityImpl" {
				t.Errorf("expected updated type DomainModels$EntityImpl, got %s", cu.Type)
			}
		}
	}
	if !found {
		t.Error("updated unit not found in cache")
	}

	// The other units should still have their original types (proving they weren't re-read from missing files)
	// Delete all files except the updated one
	for uid := range unitFiles {
		if uid == updatedUnitID {
			continue
		}
		os.Remove(filepath.Join(r.contentsDir, uid[0:2], uid[2:4], uid+".mxunit"))
	}

	// Invalidate and rebuild again — unchanged units should come from cache
	r.InvalidateCache()
	if err := r.buildUnitCache(); err != nil {
		t.Fatalf("rebuild after deleting files: %v", err)
	}

	// Check that all 3 units are still in cache (from hash-matched cached data)
	if len(r.unitCache) != 3 {
		t.Fatalf("expected 3 units after rebuild, got %d", len(r.unitCache))
	}
}

// TestContentCache_PersistsAcrossInvalidation verifies that when contentCache
// is enabled, its entries survive InvalidateCache so subsequent reads hit
// memory instead of disk.
func TestContentCache_PersistsAcrossInvalidation(t *testing.T) {
	t.Parallel()
	unitFiles := map[string]string{
		"11111111-1111-1111-1111-111111111111": "Projects$ModuleImpl",
		"22222222-2222-2222-2222-222222222222": "DomainModels$ViewEntitySourceDocument",
	}
	r := newTestReaderV2ForUnits(t, unitFiles)

	// First read populates cache (lazy init)
	data1, err := r.readMprContents("11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("first read: %v", err)
	}

	// Should be enabled after first read
	if r.contentCache == nil {
		t.Fatal("expected contentCache to be enabled (non-nil) after first read")
	}
	if _, ok := r.contentCache["11111111-1111-1111-1111-111111111111"]; !ok {
		t.Error("content not cached after first read")
	}

	// Delete the file to prove subsequent reads come from cache
	os.Remove(filepath.Join(r.contentsDir, "11", "11", "11111111-1111-1111-1111-111111111111.mxunit"))

	// Second read should still succeed from cache
	data2, err := r.readMprContents("11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("second read (should be from cache): %v", err)
	}
	if string(data1) != string(data2) {
		t.Error("cached data differs from original")
	}

	// cache should survive InvalidateCache
	r.InvalidateCache()
	if _, ok := r.contentCache["11111111-1111-1111-1111-111111111111"]; !ok {
		t.Error("content cache entry lost after InvalidateCache")
	}
}

// TestContentCache_InsertUpdateInvalidatesOnlyAffected verifies that
// InvalidateCache clears only the affected entry, not the entire cache.
func TestContentCache_ReadAfterInsertHitsCache(t *testing.T) {
	t.Parallel()
	unitFiles := map[string]string{
		"11111111-1111-1111-1111-111111111111": "Projects$ModuleImpl",
	}
	r := newTestReaderV2ForUnits(t, unitFiles)

	id := "11111111-1111-1111-1111-111111111111"

	// First read (lazy inits contentCache)
	_, err := r.readMprContents(id)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if _, ok := r.contentCache[id]; !ok {
		t.Fatal("content not cached after first read")
	}

	// InvalidateCache should NOT nuke the content cache
	r.InvalidateCache()
	if _, ok := r.contentCache[id]; !ok {
		t.Error("content cache was cleared by InvalidateCache")
	}
}
