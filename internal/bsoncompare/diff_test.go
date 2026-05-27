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
