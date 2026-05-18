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
