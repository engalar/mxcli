// SPDX-License-Identifier: Apache-2.0

//go:build integration

package mpr

import (
	"os"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestWriter_InsertThenDeleteUnit_NoBSONDrift verifies that Writer.DeleteUnit
// cleanly removes a unit from both the SQLite Unit table and (for v2 format)
// the mprcontents/.mxunit file. The test uses a real MPR project to exercise
// the full write path — insert, read back, delete, verify gone — without the
// FUSE overlay infrastructure that goldenfs/bsoncompare_integration_test.go
// relied on.
func TestWriter_InsertThenDeleteUnit_NoBSONDrift(t *testing.T) {
	srcDir := filepath.Join("..", "..", "testdata", "expr-checker")
	tmpDir := t.TempDir()

    //nolint:errcheck
    _ = os.MkdirAll(tmpDir, 0755)
	if err := os.CopyFS(tmpDir, os.DirFS(srcDir)); err != nil {
		t.Fatalf("copy testdata: %v", err)
	}
	mprPath := filepath.Join(tmpDir, "minimal.mpr")

	w, err := NewWriter(mprPath)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	r := w.ConcreteReader()

	// Baseline unit count
	baseline, err := r.ListUnitsByType("")
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	baselineCount := len(baseline)

	// Find MyFirstModule container ID
	modules, err := r.ListModules()
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	var containerID string
	for _, m := range modules {
		if m.Name == "MyFirstModule" {
			containerID = m.ID
			break
		}
	}
	if containerID == "" {
		t.Fatal("MyFirstModule not found")
	}

	// Build minimal microflow BSON with $Type and Name fields so that
	// Reader.ListUnitsByType / Store.ListUnits can identify the unit.
	microflowBSON, err := bson.Marshal(bson.D{
		{Key: "$Type", Value: "Microflows$Microflow"},
		{Key: "Name", Value: "ACT_DeleteTest"},
	})
	if err != nil {
		t.Fatalf("bson.Marshal: %v", err)
	}

	unitID := GenerateID()

	// Insert — should succeed
	if err := w.InsertUnit(unitID, containerID, "Documents", "Microflows$Microflow", microflowBSON); err != nil {
		t.Fatalf("InsertUnit: %v", err)
	}

	// Verify inserted: unit count increased by 1
	after, err := r.ListUnitsByType("")
	if err != nil {
		t.Fatalf("ListUnitsByType after insert: %v", err)
	}
	if len(after) != baselineCount+1 {
		t.Fatalf("after insert: got %d units, want %d", len(after), baselineCount+1)
	}

	// Verify we can read the inserted unit back
	raw, err := r.GetRawUnitBytes(unitID)
	if err != nil {
		t.Fatalf("GetRawUnitBytes after insert: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("GetRawUnitBytes returned empty after insert")
	}

	// Delete
	if err := w.DeleteUnit(unitID); err != nil {
		t.Fatalf("DeleteUnit: %v", err)
	}

	// For v2 format, verify the .mxunit file is removed
	if r.version == MPRVersionV2 {
		unitIDBlob := uuidToBlob(unitID)
		swappedUUID := blobToUUIDSwapped(unitIDBlob)
		mxunitPath := filepath.Join(r.contentsDir, swappedUUID[0:2], swappedUUID[2:4], swappedUUID+".mxunit")
		if _, err := os.Stat(mxunitPath); err == nil {
			t.Error(".mxunit file still exists after DeleteUnit")
		}
	}

	// Close writer so all buffers/transactions are flushed
	if err := w.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	// Open a fresh reader — no cache cross-contamination
	r2, err := Open(mprPath)
	if err != nil {
		t.Fatalf("Open after delete: %v", err)
	}
	defer r2.Close()

	final, err := r2.ListUnitsByType("")
	if err != nil {
		t.Fatalf("ListUnitsByType final: %v", err)
	}
	if len(final) != baselineCount {
		t.Fatalf("after delete: got %d units, want %d (baseline)", len(final), baselineCount)
	}

	// The deleted unit must not be readable
	if _, err := r2.GetRawUnitBytes(unitID); err == nil {
		t.Fatal("unit still readable after DeleteUnit")
	}
}
