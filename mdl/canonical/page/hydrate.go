// SPDX-License-Identifier: Apache-2.0

package page

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/mdl/types"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// Hydrate converts a gen-typed Page to a PageDocument.
//
// It extracts the page's top-level metadata (module, name, layout). moduleName
// is supplied by the caller because the gen Page does not carry its owning
// module. canonical/page imports modelsdk/gen/pages directly (the gen package
// imports nothing from canonical, so there is no cycle).
//
// Known gap: the widget tree is not extracted. Full widget extraction needs the
// backend's BSON read path (pageDocToModel consumes bson.D, not a gen Page);
// Hydrate is used primarily for diff, where ToMDL() output of the metadata is
// what matters.
func Hydrate(moduleName string, p *genPg.Page) (*PageDocument, []canonical.Warning, error) {
	if p == nil {
		return nil, nil, fmt.Errorf("page.Hydrate: nil page")
	}

	pm := &types.PageModel{
		ModuleName: moduleName,
		Name:       p.Name(),
	}

	var warns []canonical.Warning
	if lc, ok := p.LayoutCall().(*genPg.LayoutCall); ok && lc != nil {
		pm.Layout = lc.LayoutQualifiedName()
	} else {
		warns = append(warns, canonical.Warning{
			Field:   "Layout",
			Message: "page has no resolvable LayoutCall; layout left empty",
		})
	}

	return &PageDocument{pm: pm}, warns, nil
}
