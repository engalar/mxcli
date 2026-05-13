// SPDX-License-Identifier: Apache-2.0

// NOTE: Lives in package mprbackend (same package as MprBackend) so it
// can be reached from existing call sites without a new import. It
// imports the new mprrepos sub-package by its full path.
package mprbackend

import (
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/repos"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// NewExecutorContext wires a single *mmpr.Writer through every Stage 2
// repository. The returned context is owned by the caller; closing the
// underlying Writer is the caller's responsibility.
//
// Stage 3 will replace this with a path-taking constructor that opens
// the Writer internally, mirroring spec Section 7.
func NewExecutorContext(w *mmpr.Writer) *repos.ExecutorContext {
	return &repos.ExecutorContext{
		Microflows: mprrepos.NewMicroflowRepository(w),
		Pages:      mprrepos.NewPageRepository(w),
		IDs:        mprrepos.NewIDGenerator(),
		Tx:         mprrepos.NewTransactionFactory(w),
		Names:      mprrepos.NewQualifiedNameResolver(w),
		Cache:      mprrepos.NewReaderCache(w),
	}
}
