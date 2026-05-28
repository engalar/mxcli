// SPDX-License-Identifier: Apache-2.0

package poc_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// TestBlocker1_EncodeFreshMicroflow confirms that a Microflow constructed
// from scratch (no prior raw BSON) encodes to valid BSON containing the
// expected $Type and Name fields.
//
// This is the load-bearing PoC: if encoding a fresh object doesn't produce
// the right $Type, the modelsdk-native write path is unbuildable.
//
// PoC validation: see docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md Section 9 Blocker 1.
func TestBlocker1_EncodeFreshMicroflow(t *testing.T) {
	mf := genMf.NewMicroflow()
	mf.SetID(newID()) // newID lives in blocker2_id_setters_test.go (same package)
	mf.SetName("FreshlyConstructed")

	enc := &codec.Encoder{} // Encoder is a zero-value struct, no constructor.
	out, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("Encode returned empty bytes")
	}

	var doc bson.D
	if err := bson.Unmarshal(out, &doc); err != nil {
		t.Fatalf("bson.Unmarshal: %v", err)
	}

	if got := lookupString(doc, "$Type"); got != "Microflows$Microflow" {
		t.Errorf("$Type = %q, want %q", got, "Microflows$Microflow")
	}
	if got := lookupString(doc, "Name"); got != "FreshlyConstructed" {
		t.Errorf("Name = %q, want FreshlyConstructed", got)
	}
}

// TestBlocker1_DecodeRoundTrip confirms encode-then-decode produces a
// Microflow whose Name and ID match the original.
func TestBlocker1_DecodeRoundTrip(t *testing.T) {
	mf := genMf.NewMicroflow()
	mfID := newID()
	mf.SetID(mfID)
	mf.SetName("RoundTrip")

	enc := &codec.Encoder{}
	encoded, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(encoded))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	mf2, ok := elem.(*genMf.Microflow)
	if !ok {
		t.Fatalf("Decode produced %T, want *genMf.Microflow", elem)
	}
	if got := mf2.Name(); got != "RoundTrip" {
		t.Errorf("decoded Name = %q, want RoundTrip", got)
	}
	if got := mf2.ID(); got != mfID {
		t.Errorf("decoded ID = %q, want %q", got, mfID)
	}
}

// lookupString fetches a top-level string field from a bson.D.
// Returns empty string if missing or not a string.
func lookupString(doc bson.D, key string) string {
	for _, e := range doc {
		if e.Key == key {
			if s, ok := e.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}
