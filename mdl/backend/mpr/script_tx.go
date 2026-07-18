// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
)

// BeginScriptTransaction starts a script-level write buffer. No SQL transaction
// is opened; writes accumulate in ScriptBuffer and are committed atomically at
// the end via a single BatchWrite call.
//
// While the script transaction is active, Writer.InsertUnit and Writer.UpdateRawUnit
// are intercepted via SetScriptBuf so all CREATEs and UPDATEs route through the
// ScriptBuffer instead of doing per-statement file I/O + SQL + cache invalidation.
func (b *MprBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
	if b.scriptBuf != nil {
		return nil, fmt.Errorf("script transaction already active")
	}
	b.scriptBuf = newScriptBuffer(b.reader)
	// Install interceptors on the Writer so repo CREATE/UPDATE operations
	// route through the ScriptBuffer instead of direct file I/O + SQL.
	if b.writer != nil {
		b.writer.SetScriptBuf(b.scriptBuf.AddInsert, b.scriptBuf.AddUpdate, b.scriptBuf.AddContainerUpdate)
	}
	return &mprScriptTx{b: b}, nil
}

type mprScriptTx struct {
	b *MprBackend
}

// Commit flushes all buffered writes in one atomic SQL transaction.
func (tx *mprScriptTx) Commit() error {
	if tx.b.scriptBuf == nil {
		return fmt.Errorf("script transaction already closed")
	}
	return tx.b.commitScriptBuffer()
}

// Rollback discards all buffered writes. No SQL rollback needed.
func (tx *mprScriptTx) Rollback() error {
	if tx.b.scriptBuf == nil {
		return nil
	}
	tx.b.scriptBuf.Rollback()
	tx.b.scriptBuf = nil
	return nil
}
