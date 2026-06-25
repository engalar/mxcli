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

func TestReadAllUnits_MissingPath(t *testing.T) {
	t.Parallel()
	_, err := bsoncompare.ReadAllUnits("/nonexistent/path.mpr")
	if err == nil {
		t.Error("expected error for missing path")
	}
}
