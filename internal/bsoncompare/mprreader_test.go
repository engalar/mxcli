package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestReadAllUnits_CorpusB(t *testing.T) {
	t.Parallel()
	units, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatalf("ReadAllUnits: %v", err)
	}
	if len(units) < 100 {
		t.Errorf("expected at least 100 units, got %d", len(units))
	}
	found := false
	for _, u := range units {
		if u.QualifiedName != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one unit with a QualifiedName")
	}
}

func TestReadAllUnits_ContentHashPopulated(t *testing.T) {
	t.Parallel()
	units, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatal(err)
	}
	if len(units) == 0 {
		t.Fatal("no units loaded")
	}
	for _, u := range units {
		if u.ContentHash == 0 {
			t.Errorf("unit %s: ContentHash is zero", u.QualifiedName)
		}
	}
}

func TestReadAllUnits_ContentHashStable(t *testing.T) {
	t.Parallel()
	units1, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatal(err)
	}
	// Second call — returns cached result; verify ContentHash survived
	units2, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatal(err)
	}
	if len(units1) != len(units2) {
		t.Fatalf("length mismatch: %d vs %d", len(units1), len(units2))
	}
	for i := range units1 {
		if units1[i].QualifiedName == "" {
			continue
		}
		if units1[i].QualifiedName != units2[i].QualifiedName {
			continue
		}
		if units1[i].ContentHash != units2[i].ContentHash {
			t.Errorf("unit %s: ContentHash unstable across cache (%d vs %d)",
				units1[i].QualifiedName, units1[i].ContentHash, units2[i].ContentHash)
		}
	}
}

func TestReadAllUnits_MissingPath(t *testing.T) {
	t.Parallel()
	_, err := bsoncompare.ReadAllUnits("/nonexistent/path.mpr")
	if err == nil {
		t.Error("expected error for missing path")
	}
}
