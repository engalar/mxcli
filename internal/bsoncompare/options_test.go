package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestDefaultOptions(t *testing.T) {
	t.Parallel()
	opts := bsoncompare.DefaultOptions()
	if !opts.IgnoreDocumentation {
		t.Error("IgnoreDocumentation must default to true")
	}
	if !opts.IgnoreLayout {
		t.Error("IgnoreLayout must default to true")
	}
	if !opts.IgnoreStableId {
		t.Error("IgnoreStableId must default to true")
	}
}
