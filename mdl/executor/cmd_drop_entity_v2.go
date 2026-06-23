// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D4: execDropEntityGen — gen-typed DROP ENTITY executor.

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// execDropEntityGen handles DROP ENTITY on the gen-typed read+write
// path. Mirrors ExecDropEntity in cmd_entities.go: locates the entity
// via the gen DomainModel, optionally drops the associated
// ViewEntitySourceDocument for OQL view entities, then calls
// ctx.Backend.DeleteEntity (the sdk-id-only path — DeleteEntity
// signature already takes only IDs and needs no gen variant).
func execDropEntityGen(ctx *ExecContext, s *ast.DropEntityStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("DROP ENTITY requires a qualified name (Module.Entity)")
	}

	module, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := getDomainModelGenCached(ctx, module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFoundMsg("entity", fmt.Sprint(s.Name), fmt.Sprintf("entity not found: %s", s.Name))
	}

	for _, e := range dm.EntitiesItems() {
		ent, ok := e.(*genDm.Entity)
		if !ok {
			continue
		}
		if ent.Name() != s.Name.Name {
			continue
		}
		warnEntityReferences(ctx, s.Name.String())
		warnMicroflowEntityParamRefs(ctx, s.Name.String())

		if src := ent.Source(); src != nil && src.TypeName() == "DomainModels$OqlViewEntitySource" {
			if err := ctx.DomainModelWriter.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
				return mdlerrors.NewBackend("delete view entity source document", err)
			}
		}
		if err := ctx.DomainModelWriter.DeleteEntity(model.ID(dm.ID()), model.ID(ent.ID())); err != nil {
			return mdlerrors.NewBackend("delete entity", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Dropped entity: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFoundMsg("entity", fmt.Sprint(s.Name), fmt.Sprintf("entity not found: %s", s.Name))
}
