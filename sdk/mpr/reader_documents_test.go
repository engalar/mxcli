// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
	"go.mongodb.org/mongo-driver/bson"
)

const viewEntitySourceDocSchema = `
CREATE TABLE Unit (
	UnitID BLOB PRIMARY KEY NOT NULL,
	ContainerID BLOB,
	ContainmentName TEXT,
	ContentsHash TEXT,
	ContentsConflicts TEXT,
	Contents BLOB
)`

func newTestReaderWithDB(t *testing.T) (*Reader, *sql.DB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.mpr")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(viewEntitySourceDocSchema); err != nil {
		t.Fatalf("failed to create Unit table: %v", err)
	}

	return &Reader{db: db, version: MPRVersionV1}, db
}

func insertBSONUnit(t *testing.T, db *sql.DB, unitID, containerID, containmentName string, contents map[string]any) {
	t.Helper()
	data, err := bson.Marshal(contents)
	if err != nil {
		t.Fatalf("failed to marshal BSON: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO Unit (UnitID, ContainerID, ContainmentName, ContentsHash, ContentsConflicts, Contents)
		VALUES (?, ?, ?, '', '', ?)
	`, uuidToBlob(unitID), uuidToBlob(containerID), containmentName, data)
	if err != nil {
		t.Fatalf("failed to insert unit: %v", err)
	}
}

func TestReader_FindViewEntitySourceDocumentID_NotFound(t *testing.T) {
	r, _ := newTestReaderWithDB(t)
	id, err := r.FindViewEntitySourceDocumentID("NonExistentModule", "NonExistentDoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "" {
		t.Fatalf("expected empty ID, got %s", id)
	}
}

func TestReader_FindViewEntitySourceDocumentID_Found(t *testing.T) {
	r, db := newTestReaderWithDB(t)

	moduleID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	docID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	insertBSONUnit(t, db, moduleID, moduleID, "ProjectDocuments", map[string]any{
		"$Type": "Projects$ModuleImpl",
		"Name":  "MyModule",
	})
	insertBSONUnit(t, db, docID, moduleID, "Documents", map[string]any{
		"$Type": "DomainModels$ViewEntitySourceDocument",
		"Name":  "MyViewDoc",
	})

	got, err := r.FindViewEntitySourceDocumentID("MyModule", "MyViewDoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != docID {
		t.Fatalf("expected ID %s, got %s", docID, got)
	}
}

func TestReader_FindAllViewEntitySourceDocumentIDs_MultipleFound(t *testing.T) {
	r, db := newTestReaderWithDB(t)

	moduleID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	docID1 := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	docID2 := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"

	insertBSONUnit(t, db, moduleID, moduleID, "ProjectDocuments", map[string]any{
		"$Type": "Projects$ModuleImpl",
		"Name":  "MyModule",
	})
	insertBSONUnit(t, db, docID1, moduleID, "Documents", map[string]any{
		"$Type": "DomainModels$ViewEntitySourceDocument",
		"Name":  "MyViewDoc",
	})
	insertBSONUnit(t, db, docID2, moduleID, "Documents", map[string]any{
		"$Type": "DomainModels$ViewEntitySourceDocument",
		"Name":  "MyViewDoc",
	})

	ids, err := r.FindAllViewEntitySourceDocumentIDs("MyModule", "MyViewDoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
}

func TestReader_FindAllViewEntitySourceDocumentIDs_NotFound(t *testing.T) {
	r, _ := newTestReaderWithDB(t)
	ids, err := r.FindAllViewEntitySourceDocumentIDs("NoModule", "NoDoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected 0 IDs, got %d", len(ids))
	}
}
