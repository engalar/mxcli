// SPDX-License-Identifier: Apache-2.0

package golden

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func mustMarshal(t *testing.T, doc bson.D) []byte {
	t.Helper()
	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

func TestCompareIdentical(t *testing.T) {
	data := mustMarshal(t, bson.D{
		{Key: "$ID", Value: bson.Binary{Subtype: 3, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}},
		{Key: "$Type", Value: "Test$T"},
		{Key: "Name", Value: "hello"},
	})
	diffs := CompareBSON(data, data, nil)
	if len(diffs) != 0 {
		t.Errorf("identical BSON produced %d diffs: %v", len(diffs), diffs)
	}
}

func TestCompareMissingField(t *testing.T) {
	got := mustMarshal(t, bson.D{
		{Key: "Name", Value: "hello"},
	})
	exp := mustMarshal(t, bson.D{
		{Key: "Name", Value: "hello"},
		{Key: "ExtraField", Value: "world"},
	})
	diffs := CompareBSON(got, exp, nil)
	missing := DiffsByKind(diffs, DiffMissing)
	if len(missing) == 0 {
		t.Fatal("expected MissingField diff")
	}
}

func TestCompareExtraField(t *testing.T) {
	got := mustMarshal(t, bson.D{
		{Key: "Name", Value: "hello"},
		{Key: "Extra", Value: "world"},
	})
	exp := mustMarshal(t, bson.D{
		{Key: "Name", Value: "hello"},
	})
	diffs := CompareBSON(got, exp, nil)
	extra := DiffsByKind(diffs, DiffExtra)
	if len(extra) == 0 {
		t.Fatal("expected ExtraField diff")
	}
}

func TestCompareValueMismatch(t *testing.T) {
	got := mustMarshal(t, bson.D{
		{Key: "Name", Value: "hello"},
	})
	exp := mustMarshal(t, bson.D{
		{Key: "Name", Value: "world"},
	})
	diffs := CompareBSON(got, exp, nil)
	val := DiffsByKind(diffs, DiffValue)
	if len(val) == 0 {
		t.Fatal("expected ValueMismatch diff")
	}
}

func TestCompareOrderMismatch(t *testing.T) {
	got := mustMarshal(t, bson.D{
		{Key: "B", Value: 1},
		{Key: "A", Value: 2},
	})
	exp := mustMarshal(t, bson.D{
		{Key: "A", Value: 2},
		{Key: "B", Value: 1},
	})
	diffs := CompareBSON(got, exp, nil)
	order := DiffsByKind(diffs, DiffOrder)
	if len(order) == 0 {
		t.Fatal("expected OrderMismatch diff")
	}
}

func TestCompareSkipsID(t *testing.T) {
	got := mustMarshal(t, bson.D{
		{Key: "$ID", Value: bson.Binary{Subtype: 3, Data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}},
		{Key: "Name", Value: "x"},
	})
	exp := mustMarshal(t, bson.D{
		{Key: "$ID", Value: bson.Binary{Subtype: 3, Data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}}},
		{Key: "Name", Value: "x"},
	})
	diffs := CompareBSON(got, exp, nil)
	if len(diffs) != 0 {
		t.Errorf("$ID differences should be skipped, got %d diffs: %v", len(diffs), diffs)
	}
}

func TestCompareArrayMarker(t *testing.T) {
	got := mustMarshal(t, bson.D{
		{Key: "Items", Value: bson.A{int32(3), bson.D{{Key: "Name", Value: "a"}}}},
	})
	exp := mustMarshal(t, bson.D{
		{Key: "Items", Value: bson.A{int32(2), bson.D{{Key: "Name", Value: "a"}}}},
	})
	diffs := CompareBSON(got, exp, nil)
	marker := DiffsByKind(diffs, DiffMarker)
	if len(marker) == 0 {
		t.Fatal("expected MarkerMismatch diff")
	}
}
