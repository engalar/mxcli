// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
)

// writeUnitContents commits already-serialized BSON bytes for a unit through a
// modelsdk WriteTransaction.
//
// Pipeline: BeginWriteTransaction → WriteUnit → Commit → InvalidateCache.
//
// This is the SOLE write primitive used by the modelsdk-backed paths. It
// avoids sdk/mpr's updateTransactionID(), which fails with
// SQLITE_READONLY_DBMOVED 1544 on hard-linked MPR files. New code that needs
// to commit raw BSON bytes (after either gen-type setters or low-level
// PatchXxx helpers) must call this function rather than introducing yet
// another wrapper.
func (b *MprBackend) writeUnitContents(unitID model.ID, contents []byte) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}

	// Import buffer path: write to memory and set reader overlay so subsequent
	// reads within the same file see the new bytes without a SQLite round-trip.
	// Flush() will batch-commit all pending units at file end.
	if b.unitBuf != nil {
		if err := b.unitBuf.Write(unitID, contents); err != nil {
			return fmt.Errorf("write unit to import buffer: %w", err)
		}
		b.msdkReader.SetOverlay(string(unitID), contents)
		if w, ok := b.concreteWriter(); ok {
			w.ConcreteReader().SetOverlay(string(unitID), contents)
		}
		return nil
	}

	// If a script-level buffer is active, route the write into it so the whole
	// EXECUTE SCRIPT block commits or rolls back as one atomic BatchWrite.
	if b.scriptBuf != nil {
		if err := b.scriptBuf.AddUpdate(string(unitID), contents); err != nil {
			return fmt.Errorf("write unit (in script buffer): %w", err)
		}
		return nil
	}

	// Default path: one transaction per write (interactive REPL / single exec).
	wtx, err := b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := wtx.WriteUnit(string(unitID), contents); err != nil {
		_ = wtx.Rollback()
		return fmt.Errorf("write unit: %w", err)
	}
	if err := wtx.Commit(); err != nil {
		return err
	}
	b.msdkReader.InvalidateCache()
	return nil
}
