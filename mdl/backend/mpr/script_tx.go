// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// BeginScriptTransaction opens a single modelsdk WriteTransaction and
// installs it as the backend's activeScriptTx. While set, all subsequent
// writeUnitContents calls reuse this transaction instead of opening a
// per-statement one, which is what gives EXECUTE SCRIPT its
// all-or-nothing semantics.
func (b *MprBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
	if b.msdkWriter == nil {
		return nil, fmt.Errorf("modelsdk writer not initialized")
	}
	if b.activeScriptTx != nil {
		return nil, fmt.Errorf("script transaction already active")
	}
	wtx, err := b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return nil, fmt.Errorf("begin script transaction: %w", err)
	}
	b.activeScriptTx = wtx
	return &mprScriptTx{b: b, wtx: wtx}, nil
}

type mprScriptTx struct {
	b   *MprBackend
	wtx *modelsdkmpr.WriteTransaction
}

// Commit finalises the script-level transaction. WriteTransaction.Commit
// already invalidates the reader cache on success.
func (tx *mprScriptTx) Commit() error {
	if tx.b.activeScriptTx == nil {
		return fmt.Errorf("script transaction already closed")
	}
	err := tx.wtx.Commit()
	tx.b.activeScriptTx = nil
	return err
}

// Rollback discards any pending writes from the script-level transaction.
func (tx *mprScriptTx) Rollback() error {
	if tx.b.activeScriptTx == nil {
		return nil
	}
	err := tx.wtx.Rollback()
	tx.b.activeScriptTx = nil
	return err
}
