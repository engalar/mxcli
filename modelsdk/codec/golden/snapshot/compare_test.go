package snapshot

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCompareCanonical_Identical(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Test"},
	})
	snap, _ := NewUnitSnapshot("Microflows$Nanoflow", raw)
	result, err := CompareCanonical(raw, snap, CompareStructure)
	if err != nil {
		t.Fatalf("CompareCanonical: %v", err)
	}
	if len(result.Diffs) != 0 {
		t.Fatalf("expected 0 diffs, got %d: %v", len(result.Diffs), result.Diffs)
	}
}

func TestCompareCanonical_ValueMismatch(t *testing.T) {
	golden := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Expected"},
	})
	got := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Got"},
	})
	snap, _ := NewUnitSnapshot("Microflows$Nanoflow", golden)
	result, _ := CompareCanonical(got, snap, CompareStructure)
	if len(result.Diffs) == 0 {
		t.Fatal("expected diffs")
	}
}

func TestCompareBinary_Identical(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Test"},
	})
	snap, _ := NewUnitSnapshot("Microflows$Nanoflow", raw)
	result, err := CompareCanonical(raw, snap, CompareBinary)
	if err != nil {
		t.Fatalf("CompareCanonical: %v", err)
	}
	if len(result.Diffs) != 0 {
		t.Fatalf("expected 0 diffs, got %d: %v", len(result.Diffs), result.Diffs)
	}
}
