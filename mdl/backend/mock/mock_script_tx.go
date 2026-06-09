// SPDX-License-Identifier: Apache-2.0

package mock

import "github.com/mendixlabs/mxcli/mdl/backend"

// BeginScriptTransaction delegates to BeginScriptTransactionFunc if set;
// otherwise returns a no-op ScriptTransaction. Tests that care about
// script transaction lifecycle should configure the Func field.
func (m *MockBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
	if m.BeginScriptTransactionFunc != nil {
		return m.BeginScriptTransactionFunc()
	}
	return &noopScriptTx{}, nil
}

type noopScriptTx struct{}

func (t *noopScriptTx) Commit() error   { return nil }
func (t *noopScriptTx) Rollback() error { return nil }

// BeginImportBuffer delegates to BeginImportBufferFunc if set; otherwise
// returns nil (no buffering). Implements backend.ImportBufferBackend.
func (m *MockBackend) BeginImportBuffer() backend.ImportBuffer {
	if m.BeginImportBufferFunc != nil {
		return m.BeginImportBufferFunc()
	}
	return nil
}

// DisableImportBuffer delegates to DisableImportBufferFunc if set; otherwise
// no-op. Implements backend.ImportBufferBackend.
func (m *MockBackend) DisableImportBuffer() {
	if m.DisableImportBufferFunc != nil {
		m.DisableImportBufferFunc()
	}
}
