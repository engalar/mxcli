package bsoncompare_test

import (
	"testing"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestBuildIDMapFromReader_CorpusB(t *testing.T) {
	t.Parallel()
	r, err := mmpr.OpenWithOptions("../../testdata/corpus-b/app.mpr", mmpr.OpenOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	m, err := bsoncompare.BuildIDMapFromReader(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) < 10000 {
		t.Errorf("expected >= 10000 IDMap entries for corpus-b, got %d", len(m))
	}
	for k, v := range m {
		if v == "" {
			t.Errorf("key %s has empty label", k)
		}
	}
}
