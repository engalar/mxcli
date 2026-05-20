// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D2: execCreateEntityGen — gen-typed CREATE ENTITY executor.
// Composes D1 builders (astToEntityGen, astToValidationRulesGen) with
// the D8 backend write bridge (CreateEntityGen). The legacy
// execCreateEntity in cmd_entities.go is left in place until D9 cuts
// over the dispatcher; intermediate sub-features (custom error
// messages, MODIFY semantics, position auto-layout) follow in D3.

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// execCreateEntityGen handles CREATE ENTITY on the gen-typed write path.
// Builds the gen *Entity via astToEntityGen and writes via
// ctx.Backend.CreateEntityGen. Invalidates the domainmodel cache so
// subsequent reads see the new entity.
//
// Limitations vs legacy execCreateEntity (filled in by D3 sub-commits):
//   - CREATE OR MODIFY semantics not yet wired (always treats as
//     pure CREATE; ALREADY_EXISTS error matches legacy on conflict).
//   - Position auto-layout (X = 100 + N*150 etc.) deferred — caller-
//     supplied @Position annotation is honored, otherwise location
//     defaults to (0,0) which Studio Pro auto-positions on first open.
//   - Indexes / event handlers / validation rules attached via
//     entity-level builders (D1.d, D1.e) but not yet wired here —
//     follow-up D3.add-index / D3.add-event-handler land them.
func execCreateEntityGen(ctx *ExecContext, s *ast.CreateEntityStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("CREATE ENTITY requires a qualified name (Module.Entity)")
	}

	module, err := findOrCreateModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}
	if err := validateCreateEntityTypeRefsGen(ctx, s); err != nil {
		return err
	}

	dm, err := ctx.Backend.GetDomainModelGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewBackend("get domain model", nil)
	}

	var existingEntity element.Element
	for _, e := range dm.EntitiesItems() {
		ent, ok := e.(interface{ Name() string })
		if !ok {
			continue
		}
		if ent.Name() == s.Name.Name && !s.CreateOrModify {
			return mdlerrors.NewAlreadyExistsMsg("entity", s.Name.Module+"."+s.Name.Name,
				"entity already exists: "+s.Name.Module+"."+s.Name.Name+" (use create or modify to update)")
		}
		if ent.Name() == s.Name.Name && s.CreateOrModify {
			existingEntity = e
			break
		}
	}

	entity := astToEntityGen(s)
	if entity == nil {
		return mdlerrors.NewValidation("failed to build gen entity from AST")
	}
	if entity.Location() == "" {
		entity.SetLocation(fmt.Sprintf("%d;%d", 100+len(dm.EntitiesItems())*150, 100))
	}

	if existingEntity != nil {
		entity.SetID(element.ID(existingEntity.ID()))
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("update entity (gen)", err)
		}
		invalidateDomainModelsCache(ctx)
		invalidateHierarchy(ctx)
		fmt.Fprintf(ctx.Output, "Modified entity: %s\n", s.Name)
		ctx.trackModifiedDomainModel(module.ID, module.Name)
		return nil
	}

	if err := ctx.Backend.CreateEntityGen(model.ID(dm.ID()), entity); err != nil {
		return mdlerrors.NewBackend("create entity (gen)", err)
	}

	invalidateDomainModelsCache(ctx)
	invalidateHierarchy(ctx)
	fmt.Fprintf(ctx.Output, "Created entity: %s\n", s.Name)
	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

func canExecCreateEntityGen(ctx *ExecContext, s *ast.CreateEntityStmt) bool {
	return ctx != nil && s != nil
}

func validateCreateEntityTypeRefsGen(ctx *ExecContext, s *ast.CreateEntityStmt) error {
	if ctx == nil || s == nil {
		return nil
	}
	for _, a := range s.Attributes {
		if a.Type.Kind != ast.TypeEnumeration || a.Type.EnumRef == nil {
			continue
		}
		refModule := a.Type.EnumRef.Module
		refName := a.Type.EnumRef.Name
		if findEnumeration(ctx, refModule, refName) != nil {
			continue
		}
		if _, err := findEntity(ctx, refModule, refName); err == nil {
			continue
		}
		return mdlerrors.NewValidationf(
			"attribute '%s': unknown type '%s' — not a primitive, enumeration, or entity",
			a.Name, a.Type.EnumRef.String())
	}
	return nil
}
