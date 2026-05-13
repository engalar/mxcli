// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// TestWriteUnitContents_RoundTrip verifies writeUnitContents commits raw BSON
// bytes via the modelsdk WriteTransaction and that the reader cache is
// invalidated so the next read returns the new bytes.
func TestWriteUnitContents_RoundTrip(t *testing.T) {
	mprPath, unitID := makeSecurityTestMPR(t)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Read the existing unit, mutate one field, and write back.
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes (before): %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(rawBytes, &doc); err != nil {
		t.Fatalf("unmarshal (before): %v", err)
	}
	const wantLevel = "Security$SecurityLevel_Production"
	for i := range doc {
		if doc[i].Key == "SecurityLevel" {
			doc[i].Value = wantLevel
		}
	}
	patched, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := b.writeUnitContents(unitID, patched); err != nil {
		t.Fatalf("writeUnitContents: %v", err)
	}

	// Read back through the same backend (reader cache should have been invalidated).
	got, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes (after): %v", err)
	}
	var doc2 bson.D
	if err := bson.Unmarshal(got, &doc2); err != nil {
		t.Fatalf("unmarshal (after): %v", err)
	}
	gotLevel := ""
	for _, e := range doc2 {
		if e.Key == "SecurityLevel" {
			gotLevel, _ = e.Value.(string)
		}
	}
	if gotLevel != wantLevel {
		t.Errorf("SecurityLevel = %q, want %q", gotLevel, wantLevel)
	}
}

// TestWriteUnitContents_NilWriter ensures writeUnitContents fails fast with a
// helpful error when the modelsdk writer has not been initialized (i.e. the
// backend was never Connect()ed).
func TestWriteUnitContents_NilWriter(t *testing.T) {
	b := New()
	// Do not call Connect — msdkWriter stays nil.

	err := b.writeUnitContents("11111111-1111-1111-1111-111111111111", []byte{0})
	if err == nil {
		t.Fatal("writeUnitContents on nil msdkWriter: want error, got nil")
	}
	if !strings.Contains(err.Error(), "modelsdk writer not initialized") {
		t.Errorf("error message %q does not mention nil writer", err.Error())
	}
}
