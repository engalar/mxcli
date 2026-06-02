// SPDX-License-Identifier: Apache-2.0

// Package page implements the canonical model for Mendix page documents.
// It wraps the types.PageModel IR and satisfies the canonical.Document and
// canonical.Persistable interfaces.
package page

import (
	"bytes"

	"github.com/mendixlabs/mxcli/mdl/pagerender"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// PageDocument is the canonical model for a Mendix page.
// It wraps types.PageModel (the complete IR) and implements Document/Persistable.
type PageDocument struct {
	pm *types.PageModel
}

// ToMDL renders the page as deterministic MDL text via pagerender.PageModelToMDL.
// The renderer lives in the pagerender package so canonical/page does not import
// the executor package (which would create an import cycle: executor imports
// canonical/page to register its codec).
func (d *PageDocument) ToMDL() string {
	if d == nil || d.pm == nil {
		return ""
	}
	var buf bytes.Buffer
	pagerender.PageModelToMDL(&buf, d.pm, d.pm.ModuleName, d.pm.Name)
	return buf.String()
}

// PageModel returns the underlying IR for callers that need direct access.
func (d *PageDocument) PageModel() *types.PageModel {
	return d.pm
}
