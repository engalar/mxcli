// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// writeSink hides whether writes go straight to the Writer (direct mode)
// or through an active WriteTransaction (txn mode). Repository
// implementations take a sink rather than the Writer directly so the
// same code path serves both.
//
// Direct mode (writerSink): every method delegates to *mmpr.Writer.
// Tx mode (txnSink): UpdateRawUnit routes through
// WriteTransaction.WriteUnit; the other three methods return an explicit
// "not supported in transaction" error (Stage 2 limitation — see uow.go).
type writeSink interface {
	InsertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error
	UpdateRawUnit(unitID string, contents []byte) error
	DeleteUnit(unitID string) error
	UpdateUnitContainer(unitID, newContainerID string) error
}

// writerSink delegates to *mmpr.Writer (direct mode — no transaction).
type writerSink struct{ w *mmpr.Writer }

func newWriterSink(w *mmpr.Writer) writerSink { return writerSink{w: w} }

func (s writerSink) InsertUnit(u, c, cn, t string, b []byte) error {
	return s.w.InsertUnit(u, c, cn, t, b)
}
func (s writerSink) UpdateRawUnit(u string, b []byte) error { return s.w.UpdateRawUnit(u, b) }
func (s writerSink) DeleteUnit(u string) error              { return s.w.DeleteUnit(u) }
func (s writerSink) UpdateUnitContainer(u, nc string) error { return s.w.UpdateUnitContainer(u, nc) }

// txnSink defers Update writes through WriteTransaction.WriteUnit.
// Inserts / Deletes / container moves return an explicit error: mmpr's
// WriteTransaction API only exposes WriteUnit, so we surface the
// limitation rather than fail silently. Stage 3 may extend mmpr to
// support staged inserts inside a transaction.
type txnSink struct{ tx *mmpr.WriteTransaction }

func newTxnSink(tx *mmpr.WriteTransaction) txnSink { return txnSink{tx: tx} }

func (s txnSink) InsertUnit(_, _, _, _ string, _ []byte) error {
	return fmt.Errorf("UnitOfWork: InsertUnit not supported inside a transaction (Stage 2 limitation)")
}
func (s txnSink) UpdateRawUnit(u string, b []byte) error { return s.tx.WriteUnit(u, b) }
func (s txnSink) DeleteUnit(_ string) error {
	return fmt.Errorf("UnitOfWork: DeleteUnit not supported inside a transaction (Stage 2 limitation)")
}
func (s txnSink) UpdateUnitContainer(_, _ string) error {
	return fmt.Errorf("UnitOfWork: UpdateUnitContainer not supported inside a transaction (Stage 2 limitation)")
}
