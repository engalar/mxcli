// SPDX-License-Identifier: Apache-2.0

// convert_roundtrip_test.go — tests for the remaining non-trivial convert
// functions in convert.go. Most mpr.* types are now type aliases to types.*,
// so their "conversions" are identity operations and don't need testing.
// Only convertRawCustomWidgetType* require actual field-copy logic (bson.D→any).
package mprbackend

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/sdk/mpr"
	"go.mongodb.org/mongo-driver/bson"
)

var errTest = errors.New("test error")

func TestConvertRawCustomWidgetTypePtr(t *testing.T) {
	in := &mpr.RawCustomWidgetType{
		WidgetID: "w1", RawType: bson.D{{Key: "k", Value: "v"}}, RawObject: bson.D{{Key: "k2", Value: "v2"}},
		UnitID: "u1", UnitName: "Unit", WidgetName: "Widget",
	}
	out, err := convertRawCustomWidgetTypePtr(in, nil)
	if err != nil || out == nil {
		t.Fatalf("unexpected: out=%v err=%v", out, err)
	}
	if out.WidgetID != "w1" || out.WidgetName != "Widget" {
		t.Errorf("field mismatch: %+v", out)
	}
}

func TestConvertRawCustomWidgetTypePtr_Error(t *testing.T) {
	out, err := convertRawCustomWidgetTypePtr(nil, errTest)
	if out != nil || err != errTest {
		t.Errorf("expected nil/errTest, got out=%v err=%v", out, err)
	}
}

func TestConvertRawCustomWidgetTypeSlice(t *testing.T) {
	in := []*mpr.RawCustomWidgetType{
		{WidgetID: "w1", UnitName: "U1"},
		{WidgetID: "w2", UnitName: "U2"},
	}
	out, err := convertRawCustomWidgetTypeSlice(in, nil)
	if err != nil || len(out) != 2 {
		t.Fatalf("unexpected: out=%v err=%v", out, err)
	}
	if out[0].WidgetID != "w1" || out[1].WidgetID != "w2" {
		t.Errorf("slice mismatch: %+v", out)
	}
}

// TestFieldCountDrift guards that field counts between mpr.* type aliases
// and types.* definitions stay aligned. If a struct gains a field, update
// both sides (they should be identical since they're type aliases).
func TestFieldCountDrift(t *testing.T) {
	assertFieldCount(t, "mpr.FolderInfo", mpr.FolderInfo{}, 3)
	assertFieldCount(t, "types.FolderInfo", types.FolderInfo{}, 3)
	assertFieldCount(t, "mpr.UnitInfo", mpr.UnitInfo{}, 4)
	assertFieldCount(t, "types.UnitInfo", types.UnitInfo{}, 4)
	assertFieldCount(t, "mpr.RenameHit", mpr.RenameHit{}, 4)
	assertFieldCount(t, "types.RenameHit", types.RenameHit{}, 4)
	assertFieldCount(t, "mpr.RawUnit", mpr.RawUnit{}, 4)
	assertFieldCount(t, "types.RawUnit", types.RawUnit{}, 4)
	assertFieldCount(t, "mpr.RawUnitInfo", mpr.RawUnitInfo{}, 5)
	assertFieldCount(t, "types.RawUnitInfo", types.RawUnitInfo{}, 5)
	assertFieldCount(t, "mpr.RawCustomWidgetType", mpr.RawCustomWidgetType{}, 6)
	assertFieldCount(t, "types.RawCustomWidgetType", types.RawCustomWidgetType{}, 6)
	assertFieldCount(t, "mpr.NavigationDocument", mpr.NavigationDocument{}, 4)
	assertFieldCount(t, "types.NavigationDocument", types.NavigationDocument{}, 4)
	assertFieldCount(t, "mpr.JsonStructure", mpr.JsonStructure{}, 8)
	assertFieldCount(t, "types.JsonStructure", types.JsonStructure{}, 8)
	assertFieldCount(t, "mpr.JsonElement", mpr.JsonElement{}, 14)
	assertFieldCount(t, "types.JsonElement", types.JsonElement{}, 14)
	assertFieldCount(t, "mpr.ImageCollection", mpr.ImageCollection{}, 6)
	assertFieldCount(t, "types.ImageCollection", types.ImageCollection{}, 6)
	assertFieldCount(t, "mpr.EntityMemberAccess", mpr.EntityMemberAccess{}, 3)
	assertFieldCount(t, "types.EntityMemberAccess", types.EntityMemberAccess{}, 3)
	assertFieldCount(t, "mpr.EntityAccessRevocation", mpr.EntityAccessRevocation{}, 6)
	assertFieldCount(t, "types.EntityAccessRevocation", types.EntityAccessRevocation{}, 6)
}

func assertFieldCount(t *testing.T, name string, v any, want int) {
	t.Helper()
	got := reflect.TypeOf(v).NumField()
	if got != want {
		t.Errorf("%s: field count = %d, want %d", name, got, want)
	}
}
