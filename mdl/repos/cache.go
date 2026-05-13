// SPDX-License-Identifier: Apache-2.0

package repos

import "github.com/mendixlabs/mxcli/model"

// ReaderCache exposes explicit cache invalidation hooks. Per addendum
// Blocker 4, *mmpr.Writer auto-invalidates on InsertUnit and on
// WriteTransaction.Commit, so day-to-day repo code does NOT need to
// call this. The interface is retained for cross-process invalidation
// (another tool wrote to the .mpr / mprcontents directory) and for
// tests that bypass the writer.
type ReaderCache interface {
	Invalidate()
	InvalidateUnit(id model.ID)
}
