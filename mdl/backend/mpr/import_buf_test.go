// SPDX-License-Identifier: Apache-2.0
package mprbackend_test

import (
	"path/filepath"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

func TestEnableImportBuffer_GenTypeWriteGoesToBuffer(t *testing.T) {
	mprPath := filepath.Join("..", "..", "..", "testdata", "roundtrip", "roundtrip.mpr")
	be, err := mprbackend.NewFromPath(mprPath)
	if err != nil {
		t.Skipf("testdata not available: %v", err)
	}
	defer be.Disconnect()

	importBuf := be.EnableImportBuffer()
	defer be.DisableImportBuffer()

	if n := importBuf.PendingCount(); n != 0 {
		t.Fatalf("expected 0 pending before write, got %d", n)
	}

	mods, err := be.ListModules()
	if err != nil || len(mods) == 0 {
		t.Skipf("no modules available: %v", err)
	}
	dm, err := be.GetDomainModelGen(mods[0].ID)
	if err != nil || dm == nil {
		t.Skipf("no domain model: %v", err)
	}
	if err := be.UpdateDomainModelGen(dm); err != nil {
		t.Fatalf("UpdateDomainModelGen: %v", err)
	}

	if n := importBuf.PendingCount(); n == 0 {
		t.Errorf("expected PendingCount > 0 after UpdateDomainModelGen, got 0 — SetSessionBuf not wired")
	}
}

func TestDisableImportBuffer_ClearsBufAndOverlay(t *testing.T) {
	mprPath := filepath.Join("..", "..", "..", "testdata", "roundtrip", "roundtrip.mpr")
	be, err := mprbackend.NewFromPath(mprPath)
	if err != nil {
		t.Skipf("testdata not available: %v", err)
	}
	defer be.Disconnect()

	importBuf := be.EnableImportBuffer()

	mods, _ := be.ListModules()
	if len(mods) > 0 {
		dm, _ := be.GetDomainModelGen(mods[0].ID)
		if dm != nil {
			_ = be.UpdateDomainModelGen(dm)
		}
	}

	be.DisableImportBuffer()

	if importBuf.PendingCount() != 0 {
		t.Errorf("expected 0 pending after DisableImportBuffer (Discard called), got %d", importBuf.PendingCount())
	}
}
