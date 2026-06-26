package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestCompare_NoChange(t *testing.T) {
	t.Parallel()
	diffs, err := bsoncompare.Compare(
		"../../testdata/corpus-b/app.mpr",
		"../../testdata/corpus-b/app.mpr",
		bsoncompare.DefaultOptions(),
	)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("comparing MPR with itself: expected 0 diffs, got %d", len(diffs))
		n := len(diffs)
		if n > 3 {
			n = 3
		}
		for _, d := range diffs[:n] {
			t.Logf("  diff: %s %s", d.Kind, d.QualifiedName)
		}
	}
}

func TestCompare_ContentHashSkip(t *testing.T) {
	t.Parallel()
	// Read the same file via two separate cache-returned slices to verify
	// the ContentHash skip works when different []UnitDoc slices carry
	// identical hashes.
	unitsA, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatal(err)
	}
	unitsB, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatal(err)
	}
	if len(unitsA) != len(unitsB) {
		t.Fatalf("length mismatch: %d vs %d", len(unitsA), len(unitsB))
	}
	for i := range unitsA {
		if unitsA[i].ContentHash != unitsB[i].ContentHash {
			t.Errorf("unit %s: ContentHash mismatch across cached reads (%d vs %d)",
				unitsA[i].QualifiedName, unitsA[i].ContentHash, unitsB[i].ContentHash)
		}
	}
}
