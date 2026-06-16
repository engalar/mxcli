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
	b.scriptDMCache = make(map[string]*genDm.DomainModel)
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
	tx.b.scriptDMCache = nil
	return tx.b.commitScriptBuffer()
}

// Rollback discards all buffered writes. No SQL rollback needed.
func (tx *mprScriptTx) Rollback() error {
	if tx.b.scriptBuf == nil {
		return nil
	}
	tx.b.scriptDMCache = nil
	tx.b.scriptBuf.Rollback()
	tx.b.scriptBuf = nil
	return nil
}

// scriptDMCacheGet returns the cached DomainModel for id, or nil if not cached.
func (b *MprBackend) scriptDMCacheGet(id string) *genDm.DomainModel {
	if b.scriptDMCache == nil {
		return nil
	}
	return b.scriptDMCache[id]
}

// scriptDMCachePut stores a DomainModel in the per-script cache.
func (b *MprBackend) scriptDMCachePut(id string, dm *genDm.DomainModel) {
	if b.scriptDMCache != nil {
		b.scriptDMCache[id] = dm
	}
}
