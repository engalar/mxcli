package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestDiffArray_SetDiff_AddedRef(t *testing.T) {
	t.Parallel()
	golden := []any{"<ref:ModuleRole:Admin>", "<ref:ModuleRole:User>"}
	actual := []any{"<ref:ModuleRole:Admin>", "<ref:ModuleRole:Manager>"}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Roles", golden, actual, &diffs)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs (add Manager, remove User), got %d: %v", len(diffs), diffs)
	}
}

func TestDiffArray_SetDiff_NoChange(t *testing.T) {
	t.Parallel()
	golden := []any{"<ref:ModuleRole:Admin>", "<ref:ModuleRole:User>"}
	actual := []any{"<ref:ModuleRole:User>", "<ref:ModuleRole:Admin>"}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Roles", golden, actual, &diffs)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for same set (different order), got %d", len(diffs))
	}
}

func TestDiffArray_ByName_Changed(t *testing.T) {
	t.Parallel()
	golden := []any{
		map[string]any{"Name": "Param1", "Type": "String"},
		map[string]any{"Name": "Param2", "Type": "Integer"},
	}
	actual := []any{
		map[string]any{"Name": "Param1", "Type": "Boolean"},
		map[string]any{"Name": "Param2", "Type": "Integer"},
	}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Parameters", golden, actual, &diffs)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (Param1.Type), got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != ".Parameters[Param1].Type" {
		t.Errorf("unexpected path: %s", diffs[0].Path)
	}
}

func TestDiffArray_ByName_Added(t *testing.T) {
	t.Parallel()
	golden := []any{map[string]any{"Name": "P1", "Type": "String"}}
	actual := []any{
		map[string]any{"Name": "P1", "Type": "String"},
		map[string]any{"Name": "P2", "Type": "Integer"},
	}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Parameters", golden, actual, &diffs)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (added P2), got %d", len(diffs))
	}
	if diffs[0].Kind != bsoncompare.DiffAdded {
		t.Errorf("expected DiffAdded, got %s", diffs[0].Kind)
	}
}

func TestDiffArray_ByPosition_LengthOnly(t *testing.T) {
	t.Parallel()
	golden := []any{"x", "y"}
	actual := []any{"x", "y", "z"}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Flows", golden, actual, &diffs)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 length diff, got %d", len(diffs))
	}
	if diffs[0].Golden != "2" || diffs[0].Actual != "3" {
		t.Errorf("unexpected length diff: %v", diffs[0])
	}
}
