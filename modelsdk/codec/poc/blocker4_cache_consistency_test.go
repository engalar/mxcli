// SPDX-License-Identifier: Apache-2.0

package poc_test

import (
	"bytes"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// TestBlocker4_WriteThenRead confirms that after InsertUnit completes, a
// follow-up read via the same modelsdk/mpr.Reader sees the new unit. This
// determines whether repo Write methods need to call cache invalidation
// explicitly or whether the writer already does so.
//
// PoC validation: see docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md Section 9 Blocker 4.
func TestBlocker4_WriteThenRead(t *testing.T) {
	w, r := openTestMPR(t)

	baseRefs, err := r.ListUnitsByType("Microflows$Microflow")
	if err != nil {
		t.Fatalf("ListUnitsByType (baseline): %v", err)
	}
	baseCount := len(baseRefs)

	mf := genMf.NewMicroflow()
	mfID := newID()
	containerID := newID()
	mf.SetID(mfID)
	mf.SetName("PoCBlocker4Flow")

	enc := &codec.Encoder{}
	contents, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if err := w.InsertUnit(string(mfID), string(containerID), "Documents", "Microflows$Microflow", contents); err != nil {
		t.Fatalf("InsertUnit: %v", err)
	}

	afterRefs, err := r.ListUnitsByType("Microflows$Microflow")
	if err != nil {
		t.Fatalf("ListUnitsByType (after write): %v", err)
	}

	if len(afterRefs) != baseCount+1 {
		t.Errorf("after InsertUnit: count = %d, want %d (write not visible to reader)", len(afterRefs), baseCount+1)
	}

	found := false
	for _, ref := range afterRefs {
		if ref.ID == string(mfID) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("new microflow ID %q not visible in ListUnitsByType after InsertUnit", mfID)
	}

	got, err := r.GetRawUnitBytes(string(mfID))
	if err != nil {
		t.Errorf("GetRawUnitBytes(%q): %v", mfID, err)
	}
	if len(got) == 0 {
		t.Errorf("GetRawUnitBytes(%q) returned empty bytes", mfID)
	}
	if !bytes.Equal(got, contents) {
		t.Errorf("GetRawUnitBytes content (%d bytes) != written content (%d bytes)", len(got), len(contents))
	}
}

// TestBlocker4_TransactionWriteThenRead writes via BeginWriteTransaction +
// WriteUnit + Commit. Because WriteUnit updates an existing unit, the test
// first InsertUnits a unit, then opens a transaction to update it, then
// reads back to verify the updated bytes are visible.
func TestBlocker4_TransactionWriteThenRead(t *testing.T) {
	w, r := openTestMPR(t)

	mf := genMf.NewMicroflow()
	mfID := newID()
	containerID := newID()
	mf.SetID(mfID)
	mf.SetName("PoCBlocker4TxFlowOriginal")

	enc := &codec.Encoder{}
	originalContents, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode (original): %v", err)
	}
	if err := w.InsertUnit(string(mfID), string(containerID), "Documents", "Microflows$Microflow", originalContents); err != nil {
		t.Fatalf("InsertUnit (seed): %v", err)
	}

	mf.SetName("PoCBlocker4TxFlowUpdated")
	updatedContents, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode (updated): %v", err)
	}

	wtx, err := w.BeginWriteTransaction()
	if err != nil {
		t.Fatalf("BeginWriteTransaction: %v", err)
	}
	if err := wtx.WriteUnit(string(mfID), updatedContents); err != nil {
		_ = wtx.Rollback()
		t.Fatalf("WriteUnit: %v", err)
	}
	if err := wtx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := r.GetRawUnitBytes(string(mfID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes after commit: %v", err)
	}
	if !bytes.Equal(got, updatedContents) {
		t.Errorf("GetRawUnitBytes after commit = %d bytes, want %d bytes (updated content not visible)",
			len(got), len(updatedContents))
	}
}
