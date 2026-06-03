// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
)

// BeginScriptTransaction starts a script-level write buffer. No SQL transaction
// is opened; writes accumulate in ScriptBuffer and are committed atomically at
// the end via a single BatchWrite call.
func (b *MprBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
	if b.scriptBuf != nil {
		return nil, fmt.Errorf("script transaction already active")
	}
	b.scriptBuf = newScriptBuffer(b.reader)
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
