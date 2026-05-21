// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// newTestReaderV1WithUnit builds a minimal V1 Reader backed by a temp SQLite DB
// pre-seeded with a single Unit row, so GetRawUnitBytes' disk path is testable.
func newTestReaderV1WithUnit(t *testing.T, unitID string, diskContents []byte) *Reader {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.mpr")
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
			ContentsConflicts TEXT,
			Contents BLOB
		)
	`); err != nil {
		t.Fatalf("create Unit table: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO Unit (UnitID, Contents) VALUES (?, ?)`,
		uuidToBlob(unitID), diskContents,
	); err != nil {
		t.Fatalf("insert unit row: %v", err)
	}

	return &Reader{
		db:      db,
		version: MPRVersionV1,
	}
}

func TestReader_OverlayTakesPrecedenceOverDisk(t *testing.T) {
	unitID := "00000000-0000-0000-0000-000000000001"
	diskData := []byte("disk-data")
	overlayData := []byte("overlay-data")

	r := newTestReaderV1WithUnit(t, unitID, diskData)

	got, err := r.GetRawUnitBytes(unitID)
	if err != nil {
		t.Fatalf("baseline disk read: %v", err)
	}
	if string(got) != string(diskData) {
		t.Fatalf("baseline = %q, want %q", got, diskData)
	}

	r.SetOverlay(unitID, overlayData)
	got, err = r.GetRawUnitBytes(unitID)
	if err != nil {
		t.Fatalf("overlay read: %v", err)
	}
	if string(got) != string(overlayData) {
		t.Errorf("overlay = %q, want %q", got, overlayData)
	}

	r.ClearOverlay(unitID)
	got, err = r.GetRawUnitBytes(unitID)
	if err != nil {
		t.Fatalf("post-clear disk read: %v", err)
	}
	if string(got) != string(diskData) {
		t.Errorf("after ClearOverlay = %q, want disk %q", got, diskData)
	}
}

func TestReader_ClearAllOverlaysDropsEverything(t *testing.T) {
	unitID := "00000000-0000-0000-0000-000000000002"
	diskData := []byte("disk")
	overlayData := []byte("overlay")

	r := newTestReaderV1WithUnit(t, unitID, diskData)

	r.SetOverlay(unitID, overlayData)
	r.SetOverlay("00000000-0000-0000-0000-000000000099", []byte("other"))
	r.ClearAllOverlays()

	if r.overlay != nil {
		t.Errorf("expected overlay to be nil after ClearAllOverlays, got %v", r.overlay)
	}

	got, err := r.GetRawUnitBytes(unitID)
	if err != nil {
		t.Fatalf("disk read after ClearAllOverlays: %v", err)
	}
	if string(got) != string(diskData) {
		t.Errorf("after ClearAllOverlays = %q, want disk %q", got, diskData)
	}
}

func TestReader_OverlayZeroCostWhenNil(t *testing.T) {
	unitID := "00000000-0000-0000-0000-000000000003"
	diskData := []byte("disk-only")

	r := newTestReaderV1WithUnit(t, unitID, diskData)

	if r.overlay != nil {
		t.Fatalf("fresh Reader has unexpected overlay: %v", r.overlay)
	}

	got, err := r.GetRawUnitBytes(unitID)
	if err != nil {
		t.Fatalf("disk read on fresh reader: %v", err)
	}
	if string(got) != string(diskData) {
		t.Errorf("got %q, want %q", got, diskData)
	}
}
