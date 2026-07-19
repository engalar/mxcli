package snapshot

import (
	"bytes"
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

func TestToCanonicalJSON_Roundtrip(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Test"},
	})
	canonical, err := ToCanonicalJSON(raw)
	if err != nil {
		t.Fatalf("ToCanonicalJSON: %v", err)
	}
	recovered, err := FromCanonicalJSON(canonical)
	if err != nil {
		t.Fatalf("FromCanonicalJSON: %v", err)
	}
	if !bytes.Equal(raw, recovered) {
		t.Fatalf("roundtrip produced different bytes")
	}
}

func TestNewUnitSnapshot(t *testing.T) {
	raw := mustMarshal(t, bson.D{
		{Key: "$Type", Value: "Microflows$Nanoflow"},
		{Key: "Name", Value: "Test"},
	})
	snap, err := NewUnitSnapshot("Microflows$Nanoflow", raw)
	if err != nil {
		t.Fatalf("NewUnitSnapshot: %v", err)
	}
	if snap.Type != "Microflows$Nanoflow" {
		t.Fatalf("Type = %q", snap.Type)
	}
	if len(snap.Canonical) == 0 {
		t.Fatal("Canonical is empty")
	}
}
