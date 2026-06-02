// SPDX-License-Identifier: Apache-2.0

package page

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// RegisterCodec wires the page Lift / Hydrate codecs into r.
func RegisterCodec(r *canonical.DefaultRegistry) {
	codec := canonical.Codec{
		LiftFn: func(stmt any) (canonical.Persistable, error) {
			s, ok := stmt.(*ast.CreatePageStmtV3)
			if !ok {
				return nil, fmt.Errorf("page codec: expected *ast.CreatePageStmtV3, got %T", stmt)
			}
			return Lift(s, "")
		},
		HydrateFn: func(el any, hctx canonical.HydrateCtx) (canonical.Document, []canonical.Warning, error) {
			p, ok := el.(*genPg.Page)
			if !ok {
				return nil, nil, fmt.Errorf("page codec: expected *genPg.Page, got %T", el)
			}
			return Hydrate(hctx.ModuleName, p)
		},
	}
	r.Register((*ast.CreatePageStmtV3)(nil), "Forms$Page", codec)
	r.RegisterGenType("Forms$PageImpl", codec)
}
