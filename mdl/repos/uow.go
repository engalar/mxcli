// SPDX-License-Identifier: Apache-2.0

package repos

// UnitOfWork groups multi-domain writes into a single atomic commit.
// Per addendum Blocker 4, the underlying *mmpr.WriteTransaction commits
// the SQL row changes and renames the temp BSON files atomically; cache
// invalidation is automatic on Commit.
//
// Stage 2 only wires Microflows() and Pages() — the remaining accessors
// are reserved for Stage 3.
//
// Stage 2 limitation (documented): InsertUnit / DeleteUnit /
// UpdateUnitContainer are NOT supported inside an active UnitOfWork
// because mmpr.WriteTransaction only exposes WriteUnit. Calling
// Create/Delete/Move on a UoW-derived Writer returns an explicit error
// rather than failing silently. Stage 3 may extend mmpr to support
// staged inserts inside a transaction.
type UnitOfWork interface {
	Microflows() MicroflowWriter
	Pages() PageWriter
	// Stage 3 wiring:
	// Nanoflows()    NanoflowWriter
	// DomainModels() DomainModelWriter
	// Modules()      ModuleWriter
	// ... etc.
	Commit() error
	Rollback() error
}

type TransactionFactory interface {
	Begin() (UnitOfWork, error)
}
