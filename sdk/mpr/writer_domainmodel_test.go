// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// =============================================================================
// ScanOqlQueryUpdates
// =============================================================================

func TestScanOqlQueryUpdates_NoMatch(t *testing.T) {
	w, db := newTestWriterV1(t, `
		CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			ContentsHash TEXT,
			ContentsConflicts TEXT,
			Contents BLOB
		)`)

	// Insert a ViewEntitySourceDocument with OQL that does NOT contain oldName
	docID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	modID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	contents, _ := bson.Marshal(map[string]any{
		"$Type": "DomainModels$ViewEntitySourceDocument",
		"Name":  "MyView",
		"Oql":   "from OtherModule.Customer as c return c",
	})
	if _, err := db.Exec(`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, ContentsHash, ContentsConflicts, Contents) VALUES (?,?,?,?,?,?)`,
		uuidToBlob(docID), uuidToBlob(modID), "Documents", "", "", contents); err != nil {
		t.Fatalf("insert: %v", err)
	}

	patches, count, err := w.ScanOqlQueryUpdates("MyModule.Customer", "NewModule.Customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 matches, got %d", count)
	}
	if len(patches) != 0 {
		t.Fatalf("expected 0 patches, got %d", len(patches))
	}
}

func TestScanOqlQueryUpdates_WithMatch(t *testing.T) {
	w, db := newTestWriterV1(t, `
		CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			ContentsHash TEXT,
			ContentsConflicts TEXT,
			Contents BLOB
		)`)

	docID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	modID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	contents, _ := bson.Marshal(map[string]any{
		"$Type": "DomainModels$ViewEntitySourceDocument",
		"Name":  "MyView",
		"Oql":   "from MyModule.Customer as c return c",
	})
	if _, err := db.Exec(`INSERT INTO Unit (UnitID, ContainerID, ContainmentName, ContentsHash, ContentsConflicts, Contents) VALUES (?,?,?,?,?,?)`,
		uuidToBlob(docID), uuidToBlob(modID), "Documents", "", "", contents); err != nil {
		t.Fatalf("insert: %v", err)
	}

	patches, count, err := w.ScanOqlQueryUpdates("MyModule.Customer", "NewModule.Customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 match, got %d", count)
	}
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}

	// Verify the patch contains the updated OQL
	var raw map[string]any
	if err := bson.Unmarshal(patches[0].Contents, &raw); err != nil {
		t.Fatalf("unmarshal patch: %v", err)
	}
	oql, _ := raw["Oql"].(string)
	if oql != "from NewModule.Customer as c return c" {
		t.Fatalf("expected updated OQL, got %q", oql)
	}
}
