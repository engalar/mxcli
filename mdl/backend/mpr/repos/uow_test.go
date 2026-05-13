// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// TestUoW_BeginCommit_Empty verifies the factory wires correctly and an
// empty transaction round-trips cleanly.
func TestUoW_BeginCommit_Empty(t *testing.T) {
	w := openTestWriter(t)
	tx := NewTransactionFactory(w)
	uow, err := tx.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatalf("Commit (empty): %v", err)
	}
}

// TestUoW_BeginRollback_Empty mirrors the empty-Commit test for Rollback.
func TestUoW_BeginRollback_Empty(t *testing.T) {
	w := openTestWriter(t)
	tx := NewTransactionFactory(w)
	uow, err := tx.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := uow.Rollback(); err != nil {
		t.Fatalf("Rollback (empty): %v", err)
	}
}

// TestUoW_Microflows_Update_RoundTrip seeds a microflow outside the
// transaction, then opens a UoW and uses Microflows().Update to rewrite
// it. After Commit, the new bytes must be visible via GetRawUnitBytes.
func TestUoW_Microflows_Update_RoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()

	// Seed a microflow we will subsequently update via the UoW.
	mf := genMf.NewMicroflow()
	mfID := element.ID(mmpr.GenerateID())
	containerID := element.ID(mmpr.GenerateID())
	mf.SetID(mfID)
	mf.SetTypeName("Microflows$Microflow")
	mf.SetName("UoWSeed")

	enc := &codec.Encoder{}
	seed, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode seed: %v", err)
	}
	if err := w.InsertUnit(string(mfID), string(containerID), "Documents", "Microflows$Microflow", seed); err != nil {
		t.Fatalf("InsertUnit (seed): %v", err)
	}

	// Now mutate via the UoW.
	mf.SetName("UoWUpdated")

	tx := NewTransactionFactory(w)
	uow, err := tx.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := uow.Microflows().Update(mf); err != nil {
		_ = uow.Rollback()
		t.Fatalf("Microflows().Update: %v", err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := r.GetRawUnitBytes(string(mfID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes after commit: %v", err)
	}
	expected, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode expected: %v", err)
	}
	if !bytes.Equal(got, expected) {
		t.Errorf("post-commit bytes (%d) != expected (%d) — Update not persisted via UoW", len(got), len(expected))
	}
}

// TestUoW_Microflows_Create_NotSupported asserts the documented Stage 2
// limitation: Create inside a transaction surfaces an explicit error
// rather than failing silently.
func TestUoW_Microflows_Create_NotSupported(t *testing.T) {
	w := openTestWriter(t)
	tx := NewTransactionFactory(w)
	uow, err := tx.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer func() { _ = uow.Rollback() }()

	mf := genMf.NewMicroflow()
	mf.SetID(element.ID(mmpr.GenerateID()))
	mf.SetName("ShouldNotInsertInsideTx")

	err = uow.Microflows().Create(string(element.ID(mmpr.GenerateID())), "Documents", mf)
	if err == nil {
		t.Fatal("expected Create-inside-tx to error, got nil")
	}
	if !strings.Contains(err.Error(), "not supported inside a transaction") {
		t.Errorf("error %q does not mention the Stage 2 limitation", err)
	}
}

// TestUoW_Microflows_Rollback_NoChange verifies Rollback discards an
// Update that was issued inside the transaction.
func TestUoW_Microflows_Rollback_NoChange(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()

	// Seed.
	mf := genMf.NewMicroflow()
	mfID := element.ID(mmpr.GenerateID())
	containerID := element.ID(mmpr.GenerateID())
	mf.SetID(mfID)
	mf.SetTypeName("Microflows$Microflow")
	mf.SetName("UoWRollbackOriginal")

	enc := &codec.Encoder{}
	original, err := enc.Encode(mf)
	if err != nil {
		t.Fatalf("Encode original: %v", err)
	}
	if err := w.InsertUnit(string(mfID), string(containerID), "Documents", "Microflows$Microflow", original); err != nil {
		t.Fatalf("InsertUnit (seed): %v", err)
	}

	// Update inside tx, then Rollback.
	mf.SetName("UoWRollbackChanged")

	tx := NewTransactionFactory(w)
	uow, err := tx.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := uow.Microflows().Update(mf); err != nil {
		_ = uow.Rollback()
		t.Fatalf("Microflows().Update: %v", err)
	}
	if err := uow.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	got, err := r.GetRawUnitBytes(string(mfID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes after rollback: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("post-rollback bytes differ from seed — Rollback did not discard the Update")
	}
}
