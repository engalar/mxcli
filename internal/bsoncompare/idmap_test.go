package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestBuildIDMap_CorpusB(t *testing.T) {
	t.Parallel()
	units, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatal(err)
	}
	m := bsoncompare.BuildIDMap(units)
	if len(m) < 10000 {
		t.Errorf("expected >= 10000 IDMap entries for corpus-b, got %d", len(m))
	}
	for k, v := range m {
		if v == "" {
			t.Errorf("IDMap key %s has empty label", k)
		}
	}
}

func TestBuildIDMap_Empty(t *testing.T) {
	m := bsoncompare.BuildIDMap(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}
