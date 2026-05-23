// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// RegisterCodec wires the entity Lift / Hydrate codecs into r. Call once
// during executor initialisation; safe to call against multiple registries.
//
// The Hydrate side does not yet know the owning module name (the gen Entity
// does not carry it). Callers that need the qualified name should wrap the
// returned Document and overlay the module after dispatch, or call
// entity.Hydrate(moduleName, e) directly.
func RegisterCodec(r *model.DefaultRegistry) {
	r.Register(
		(*ast.CreateEntityStmt)(nil),
		"DomainModels$EntityImpl",
		model.Codec{
			LiftFn: func(stmt any) (model.Persistable, error) {
				s, ok := stmt.(*ast.CreateEntityStmt)
				if !ok {
					return nil, fmt.Errorf("entity codec: expected *ast.CreateEntityStmt, got %T", stmt)
				}
				return Lift(s)
			},
			HydrateFn: func(el any) (model.Document, []model.Warning, error) {
				e, ok := el.(*genDm.Entity)
				if !ok {
					return nil, nil, fmt.Errorf("entity codec: expected *genDm.Entity, got %T", el)
				}
				return Hydrate("", e)
			},
		},
	)
}
