// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// BeginScriptTransaction starts a script-level write buffer. No SQL transaction
// is opened; writes accumulate in ScriptBuffer and are committed atomically at
// the end via a single BatchWrite call.
func (b *MprBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
	if b.scriptBuf != nil {
		return nil, fmt.Errorf("script transaction already active")
	}
	b.scriptBuf = newScriptBuffer(b.reader)
	b.scriptDirtyDMs = make(map[string]*genDm.DomainModel)
	return &mprScriptTx{b: b}, nil
}

type mprScriptTx struct {
	b *MprBackend
}

// Commit flushes all dirty domain models and buffered writes in one atomic SQL transaction.
func (tx *mprScriptTx) Commit() error {
	if tx.b.scriptBuf == nil {
		return fmt.Errorf("script transaction already closed")
	}
	if err := tx.b.flushScriptDirtyDMs(); err != nil {
		return err
	}
	return tx.b.commitScriptBuffer()
}

// Rollback discards all buffered writes and dirty domain models.
func (tx *mprScriptTx) Rollback() error {
	if tx.b.scriptBuf == nil {
		return nil
	}
	tx.b.scriptDirtyDMs = nil
	tx.b.scriptBuf.Rollback()
	tx.b.scriptBuf = nil
	return nil
}

// flushScriptDirtyDMs serialises all accumulated in-memory domain model
// mutations and routes each through writeUnitContents (which writes to
// scriptBuf + sets the reader overlay). Called once before finalization and
// once at Commit — safe to call multiple times (idempotent after first flush
// because scriptDirtyDMs is cleared).
func (b *MprBackend) flushScriptDirtyDMs() error {
	if len(b.scriptDirtyDMs) == 0 {
		return nil
	}
	for _, dm := range b.scriptDirtyDMs {
		if err := b.UpdateDomainModelGen(dm); err != nil {
			return fmt.Errorf("flush dirty domain model: %w", err)
		}
	}
	b.scriptDirtyDMs = make(map[string]*genDm.DomainModel) // keep map alive (transaction still open)
	return nil
}

// FlushScriptDirtyDMs is the exported hook called by ExecuteProgram before
// finalizeProgramExecution so that raw BSON reads (e.g. ReconcileMemberAccesses)
// see the accumulated entity/association changes via the reader overlay.
func (b *MprBackend) FlushScriptDirtyDMs() error {
	return b.flushScriptDirtyDMs()
}
