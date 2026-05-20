package bsoncompare_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestFormatDiff_Changed(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{
		{
			QualifiedName: "MyFirstModule.ACT_Test",
			UnitType:      "Microflows$Microflow",
			Kind:          bsoncompare.DiffChanged,
			Fields: []bsoncompare.FieldDiff{
				{Path: ".ExportLevel", Golden: "Hidden", Actual: "Project", Kind: bsoncompare.DiffChanged},
			},
		},
	}
	out := bsoncompare.FormatDiff(diffs)
	if !strings.Contains(out, "[CHANGED]") {
		t.Errorf("missing [CHANGED] header: %s", out)
	}
	if !strings.Contains(out, "ACT_Test") {
		t.Errorf("missing unit name: %s", out)
	}
	if !strings.Contains(out, "Hidden") || !strings.Contains(out, "Project") {
		t.Errorf("missing field values: %s", out)
	}
}

func TestFormatDiff_Added(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{
		{QualifiedName: "MyFirstModule.NewMF", UnitType: "Microflows$Microflow", Kind: bsoncompare.DiffAdded},
	}
	out := bsoncompare.FormatDiff(diffs)
	if !strings.Contains(out, "[ADDED]") {
		t.Errorf("missing [ADDED]: %s", out)
	}
}

func TestFormatDiff_Empty(t *testing.T) {
	out := bsoncompare.FormatDiff(nil)
	if out != "" {
		t.Errorf("expected empty string for nil diffs, got %q", out)
	}
}
