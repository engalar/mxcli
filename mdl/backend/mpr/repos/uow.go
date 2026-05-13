// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// txFactory is the concrete TransactionFactory.
type txFactory struct {
	w   *mmpr.Writer
	enc *encoder
}

func NewTransactionFactory(w *mmpr.Writer) repos.TransactionFactory {
	return &txFactory{w: w, enc: newEncoder()}
}

func (f *txFactory) Begin() (repos.UnitOfWork, error) {
	wt, err := f.w.BeginWriteTransaction()
	if err != nil {
		return nil, err
	}
	sink := newTxnSink(wt)
	return &uow{
		mfWriter:   newMicroflowWriterWithSink(sink, f.enc),
		pageWriter: newPageWriterWithSink(sink, f.enc),
		tx:         wt,
	}, nil
}

// uow groups all UoW state. Microflows() / Pages() return the
// pre-constructed sink-backed writers; Commit / Rollback delegate to
// the underlying *mmpr.WriteTransaction.
type uow struct {
	mfWriter   repos.MicroflowWriter
	pageWriter repos.PageWriter
	tx         *mmpr.WriteTransaction
}

func (u *uow) Microflows() repos.MicroflowWriter { return u.mfWriter }
func (u *uow) Pages() repos.PageWriter           { return u.pageWriter }
func (u *uow) Commit() error                     { return u.tx.Commit() }
func (u *uow) Rollback() error                   { return u.tx.Rollback() }
