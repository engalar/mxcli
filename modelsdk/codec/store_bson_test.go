// SPDX-License-Identifier: Apache-2.0

//go:build integration

package codec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/mpr"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestStore_InsertDelete_BSONShape inserts a unit for each document type
// via Store.InsertUnit, reads back its BSON to verify $Type and Name fields,
// then deletes it and confirms it is unreachable through a fresh reader.
//
// This replaces the FUSE-based goldenfs BSON comparison tests with a direct
// modelsdk/codec-level validation that is faster, cross-platform, and pinpoints
// Writer-layer issues without executor pipeline noise.
func TestStore_InsertDelete_BSONShape(t *testing.T) {
	srcDir := filepath.Join("..", "..", "testdata", "expr-checker")

	for _, tc := range []struct {
		docType         string // test case name
		unitType        string // BSON $Type
		containmentName string // ContainmentName in Unit table
	}{
		{"Microflow", "Microflows$Microflow", "Documents"},
		{"Enumeration", "Enumerations$Enumeration", "Documents"},
		{"Page", "Forms$Page", "Documents"},
	} {
		t.Run(tc.docType, func(t *testing.T) {
			tmpDir := t.TempDir()
			if err := os.CopyFS(tmpDir, os.DirFS(srcDir)); err != nil {
				t.Fatalf("copy testdata: %v", err)
			}
			mprPath := filepath.Join(tmpDir, "minimal.mpr")

			store, err := OpenForWriting(mprPath)
			if err != nil {
				t.Fatalf("OpenForWriting: %v", err)
			}
			defer store.Close()

			// Baseline unit count
			before := store.ListUnits()
			baseline := len(before)

			// Container: MyFirstModule
			modInfo, err := store.GetModuleByName("MyFirstModule")
			if err != nil {
				t.Fatalf("GetModuleByName: %v", err)
			}
			containerID := modInfo.ID

			unitName := "ACT_BSONTest_" + tc.docType
			unitID := mpr.GenerateID()

			// Build minimal BSON with $Type and Name
			contents, err := bson.Marshal(bson.D{
				{Key: "$Type", Value: tc.unitType},
				{Key: "Name", Value: unitName},
			})
			if err != nil {
				t.Fatalf("bson.Marshal: %v", err)
			}

			// Insert
			if err := store.InsertUnit(unitID, containerID, tc.containmentName, tc.unitType, contents); err != nil {
				t.Fatalf("InsertUnit: %v", err)
			}

			// LoadUnit and verify $Type + Name
			raw, err := store.LoadUnit(element.ID(unitID))
			if err != nil {
				t.Fatalf("LoadUnit after insert: %v", err)
			}
			checkBSONField(t, raw, "$Type", tc.unitType)
			checkBSONField(t, raw, "Name", unitName)

			// ListUnits must show exactly one more unit
			after := store.ListUnits()
			if len(after) != baseline+1 {
				t.Fatalf("after insert: got %d units, want %d", len(after), baseline+1)
			}

			// Delete
			if err := store.DeleteUnit(unitID); err != nil {
				t.Fatalf("DeleteUnit: %v", err)
			}

			store.Close()

			// Fresh reader — no cache cross-contamination
			store2, err := Open(mprPath)
			if err != nil {
				t.Fatalf("Open after delete: %v", err)
			}
			defer store2.Close()

			final := store2.ListUnits()
			if len(final) != baseline {
				t.Fatalf("after delete: got %d units, want %d (baseline)", len(final), baseline)
			}
			if _, err := store2.LoadUnit(element.ID(unitID)); err == nil {
				t.Fatal("unit still readable after DeleteUnit")
			}
		})
	}
}

// checkBSONField asserts that raw contains a string field key with value want.
func checkBSONField(t *testing.T, raw bson.Raw, key, want string) {
	t.Helper()
	val, err := raw.LookupErr(key)
	if err != nil {
		t.Fatalf("BSON missing field %q: %v", key, err)
	}
	got, ok := val.StringValueOK()
	if !ok {
		t.Fatalf("BSON field %q is not a string", key)
	}
	if got != want {
		t.Fatalf("BSON field %q = %q, want %q", key, got, want)
	}
}


