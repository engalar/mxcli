// SPDX-License-Identifier: Apache-2.0

package association

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// RegisterCodec wires the association Lift / Hydrate codecs into r.
func RegisterCodec(r *canonical.DefaultRegistry) {
	codec := canonical.Codec{
		LiftFn: func(stmt any) (canonical.Persistable, error) {
			s, ok := stmt.(*ast.CreateAssociationStmt)
			if !ok {
				return nil, fmt.Errorf("association codec: expected *ast.CreateAssociationStmt, got %T", stmt)
			}
			return Lift(s), nil
		},
		HydrateFn: func(el any, hctx canonical.HydrateCtx) (canonical.Document, []canonical.Warning, error) {
			a, ok := el.(*genDm.Association)
			if !ok {
				return nil, nil, fmt.Errorf("association codec: expected *genDm.Association, got %T", el)
			}
			return Hydrate(hctx, a)
		},
	}
	r.Register((*ast.CreateAssociationStmt)(nil), "DomainModels$Association", codec)
	r.RegisterGenType("DomainModels$AssociationImpl", codec)
}
