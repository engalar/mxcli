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
	canonicalmodel "github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
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

	dm, err := getDomainModelGenCached(ctx, module.ID)
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

	// Canonical path: applicable when no event handlers and the codec
	// registry is wired. Event handlers are not modelled by EntityModel
	// yet — fall back to the legacy gen builder for entities that use
	// them, so behaviour is preserved end-to-end.
	if ctx.ModelCodecs != nil && len(s.EventHandlers) == 0 {
		if err := persistEntityCanonical(ctx, s, dm, existingEntity, module); err != nil {
			return err
		}
		return nil
	}

	// Legacy path (event handlers, or absent codec registry).
	entity := astToEntityGen(s)
	if entity == nil {
		return mdlerrors.NewValidation("failed to build gen entity from AST")
	}
	if entity.Location() == "" {
		entity.SetLocation(layoutPos(100+len(dm.EntitiesItems())*150, 100))
	}

	if existingEntity != nil {
		entity.SetID(element.ID(existingEntity.ID()))
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("update entity (gen)", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		invalidateHierarchy(ctx)
		fmt.Fprintf(ctx.Output, "Modified entity: %s\n", s.Name)
		ctx.trackModifiedDomainModel(module.ID, module.Name)
		return nil
	}

	if err := ctx.Backend.CreateEntityGen(model.ID(dm.ID()), entity); err != nil {
		return mdlerrors.NewBackend("create entity (gen)", err)
	}

	invalidateDomainModelGenForModule(ctx, module.ID)
	invalidateDomainModelsCache(ctx)
	invalidateHierarchy(ctx)
	fmt.Fprintf(ctx.Output, "Created entity: %s\n", s.Name)
	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

// parseEntityLocation parses a Mendix entity Location string ("X;Y" or legacy "X Y").
func parseEntityLocation(loc string) (x, y int, ok bool) {
	if _, err := fmt.Sscanf(loc, "%d;%d", &x, &y); err == nil {
		return x, y, true
	}
	_, err := fmt.Sscanf(loc, "%d %d", &x, &y)
	return x, y, err == nil
}

// persistEntityCanonical routes CREATE ENTITY through the canonical model
// pipeline: Lift the AST -> EntityModel -> Persist. Pre-injects two
// Mendix-specific invariants the parser doesn't carry: a Boolean attribute
// without an explicit default gets `default false`, and an entity without
// an explicit @Position gets stacked auto-layout based on the current DM
// entity count.
func persistEntityCanonical(ctx *ExecContext, s *ast.CreateEntityStmt, dm *genDm.DomainModel, existing element.Element, module *model.Module) error {
	stmt := *s
	if stmt.Position == nil {
		if existing != nil {
			// Modify case: preserve the existing entity's canvas position so we don't
			// overwrite it with an auto-layout default. Read Location from the gen element.
			if ent, ok := existing.(interface{ Location() string }); ok {
				if loc := ent.Location(); loc != "" {
					// parseLocation handles both "X;Y" and legacy "X Y" formats.
					if x, y, parsed := parseEntityLocation(loc); parsed {
						stmt.Position = &ast.Position{X: x, Y: y}
					}
				}
			}
		}
		if stmt.Position == nil {
			// New entity: auto-layout based on current entity count.
			stmt.Position = &ast.Position{X: 100 + len(dm.EntitiesItems())*150, Y: 100}
		}
	}
	if len(stmt.Attributes) > 0 {
		attrs := make([]ast.Attribute, len(stmt.Attributes))
		copy(attrs, stmt.Attributes)
		for i := range attrs {
			if attrs[i].Type.Kind == ast.TypeBoolean && !attrs[i].HasDefault {
				attrs[i].HasDefault = true
				attrs[i].DefaultValue = false
			}
		}
		stmt.Attributes = attrs
	}

	doc, err := ctx.ModelCodecs.LiftFrom(&stmt)
	if err != nil {
		return mdlerrors.NewBackend("CREATE ENTITY: lift", err)
	}

	var existingID model.ID
	if existing != nil {
		if elem, ok := existing.(interface{ ID() element.ID }); ok {
			existingID = model.ID(elem.ID())
		}
	}

	pCtx := canonicalmodel.PersistContext{
		DomainModelID:    model.ID(dm.ID()),
		ExistingEntityID: existingID,
		Backend:          ctx.Backend,
	}
	if err := doc.Persist(pCtx); err != nil {
		return mdlerrors.NewBackend("CREATE ENTITY: persist", err)
	}

	invalidateDomainModelGenForModule(ctx, module.ID)
	invalidateDomainModelsCache(ctx)
	invalidateHierarchy(ctx)
	if existing != nil {
		fmt.Fprintf(ctx.Output, "Modified entity: %s\n", s.Name)
	} else {
		fmt.Fprintf(ctx.Output, "Created entity: %s\n", s.Name)
	}
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
