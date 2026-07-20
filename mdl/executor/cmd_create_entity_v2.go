// SPDX-License-Identifier: Apache-2.0

// execCreateEntityGen — gen-typed CREATE ENTITY executor. Builds a gen
// *Entity via buildEntityFromAST and writes it through the backend
// (Create/UpdateEntityGen), handling MODIFY semantics, custom error
// messages, and position auto-layout.

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// execCreateEntityGen handles CREATE [OR MODIFY] ENTITY on the gen-typed
// write path. Builds the gen *Entity via buildEntityFromAST (attributes,
// indexes, validation rules, event handlers) and writes via
// ctx.Backend.Create/UpdateEntityGen, then invalidates the domainmodel
// cache so subsequent reads see the change.
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

	if err := persistEntityDirect(ctx, s, dm, existingEntity, module); err != nil {
		return err
	}
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

// persistEntityDirect builds a gen *Entity from the AST and writes it via the
// backend (Create/UpdateEntityGen). Pre-injects two Mendix-specific invariants
// the parser doesn't carry: a Boolean attribute without an explicit default
// gets `default false`, and an entity without an explicit @Position gets stacked
// auto-layout based on the current DM entity count.
func persistEntityDirect(ctx *ExecContext, s *ast.CreateEntityStmt, dm *genDm.DomainModel, existing element.Element, module *model.Module) error {
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
			// Temporary origin; RelayoutDomainModel below will assign real position.
			stmt.Position = &ast.Position{X: 0, Y: 0}
		}
	}
	// autoTypeToSystemMember maps pseudo-type kinds to their canonical system member name.
	autoTypeToSystemMember := map[ast.DataTypeKind]string{
		ast.TypeAutoOwner:       "owner",
		ast.TypeAutoChangedBy:   "changedBy",
		ast.TypeAutoCreatedDate: "createdDate",
		ast.TypeAutoChangedDate: "changedDate",
	}

	if len(stmt.Attributes) > 0 {
		// Collect existing SystemMembers into a set to avoid duplicates.
		existingSystemMembers := make(map[string]bool, len(stmt.SystemMembers))
		for _, m := range stmt.SystemMembers {
			existingSystemMembers[m] = true
		}

		var filteredAttrs []ast.Attribute
		extraSystemMembers := stmt.SystemMembers
		for _, attr := range stmt.Attributes {
			if memberName, isAuto := autoTypeToSystemMember[attr.Type.Kind]; isAuto {
				// Promote to system member; skip adding to attributes.
				if !existingSystemMembers[memberName] {
					extraSystemMembers = append(extraSystemMembers, memberName)
					existingSystemMembers[memberName] = true
				}
				continue
			}
			filteredAttrs = append(filteredAttrs, attr)
		}
		stmt.SystemMembers = extraSystemMembers

		attrs := make([]ast.Attribute, len(filteredAttrs))
		copy(attrs, filteredAttrs)
		for i := range attrs {
			if attrs[i].Type.Kind == ast.TypeBoolean && !attrs[i].HasDefault {
				attrs[i].HasDefault = true
				attrs[i].DefaultValue = false
			}
		}
		stmt.Attributes = attrs
	}

	gen, err := buildEntityFromAST(stmt.Name.Module, &stmt)
	if err != nil {
		return mdlerrors.NewBackend("CREATE ENTITY: build gen entity", err)
	}

	if existing != nil {
		if elem, ok := existing.(interface{ ID() element.ID }); ok {
			gen.SetID(elem.ID())
		}
		if err := ctx.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), gen); err != nil {
			return mdlerrors.NewBackend("CREATE ENTITY: update", err)
		}
	} else {
		if err := ctx.DomainModelWriter.CreateEntityGen(model.ID(dm.ID()), gen); err != nil {
			return mdlerrors.NewBackend("CREATE ENTITY: create", err)
		}
	}

	invalidateDomainModelGenForModule(ctx, module.ID)
	invalidateDomainModelsCache(ctx)
	invalidateHierarchy(ctx)

	if s.Position == nil {
		if err := ctx.DomainModelWriter.RelayoutDomainModel(model.ID(dm.ID())); err != nil {
			fmt.Fprintf(ctx.Output, "warning: auto-layout failed: %v\n", err)
		} else {
			invalidateDomainModelGenForModule(ctx, module.ID)
		}
	}

	if existing != nil {
		fmt.Fprintf(ctx.Output, "Modified entity: %s\n", s.Name)
	} else {
		fmt.Fprintf(ctx.Output, "Created entity: %s\n", s.Name)
	}
	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

// execCreateEntityGenDeps is the HandlerDeps implementation for CREATE ENTITY.
func execCreateEntityGenDeps(ctx context.Context, s *ast.CreateEntityStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("CREATE ENTITY requires a qualified name (Module.Entity)")
	}

	module, err := findOrCreateModuleDeps(ctx, deps, s.Name.Module)
	if err != nil {
		return err
	}
	if err := validateCreateEntityTypeRefsGenDeps(ctx, deps, s); err != nil {
		return err
	}

	dm, err := getDomainModelGenCachedDeps(ctx, deps, module.ID)
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

	if err := persistEntityDirectDeps(ctx, s, dm, existingEntity, module, deps); err != nil {
		return err
	}
	return nil
}

// persistEntityDirectDeps is the HandlerDeps version of persistEntityDirect.
func persistEntityDirectDeps(ctx context.Context, s *ast.CreateEntityStmt, dm *genDm.DomainModel, existing element.Element, module *model.Module, deps *HandlerDeps) error {
	stmt := *s
	if stmt.Position == nil {
		if existing != nil {
			if ent, ok := existing.(interface{ Location() string }); ok {
				if loc := ent.Location(); loc != "" {
					if x, y, parsed := parseEntityLocation(loc); parsed {
						stmt.Position = &ast.Position{X: x, Y: y}
					}
				}
			}
		}
		if stmt.Position == nil {
			stmt.Position = &ast.Position{X: 0, Y: 0}
		}
	}
	autoTypeToSystemMember := map[ast.DataTypeKind]string{
		ast.TypeAutoOwner:       "owner",
		ast.TypeAutoChangedBy:   "changedBy",
		ast.TypeAutoCreatedDate: "createdDate",
		ast.TypeAutoChangedDate: "changedDate",
	}

	if len(stmt.Attributes) > 0 {
		existingSystemMembers := make(map[string]bool, len(stmt.SystemMembers))
		for _, m := range stmt.SystemMembers {
			existingSystemMembers[m] = true
		}

		var filteredAttrs []ast.Attribute
		extraSystemMembers := stmt.SystemMembers
		for _, attr := range stmt.Attributes {
			if memberName, isAuto := autoTypeToSystemMember[attr.Type.Kind]; isAuto {
				if !existingSystemMembers[memberName] {
					extraSystemMembers = append(extraSystemMembers, memberName)
					existingSystemMembers[memberName] = true
				}
				continue
			}
			filteredAttrs = append(filteredAttrs, attr)
		}
		stmt.SystemMembers = extraSystemMembers

		attrs := make([]ast.Attribute, len(filteredAttrs))
		copy(attrs, filteredAttrs)
		for i := range attrs {
			if attrs[i].Type.Kind == ast.TypeBoolean && !attrs[i].HasDefault {
				attrs[i].HasDefault = true
				attrs[i].DefaultValue = false
			}
		}
		stmt.Attributes = attrs
	}

	gen, err := buildEntityFromAST(stmt.Name.Module, &stmt)
	if err != nil {
		return mdlerrors.NewBackend("CREATE ENTITY: build gen entity", err)
	}

	if existing != nil {
		if elem, ok := existing.(interface{ ID() element.ID }); ok {
			gen.SetID(elem.ID())
		}
		if err := deps.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), gen); err != nil {
			return mdlerrors.NewBackend("CREATE ENTITY: update", err)
		}
	} else {
		if err := deps.DomainModelWriter.CreateEntityGen(model.ID(dm.ID()), gen); err != nil {
			return mdlerrors.NewBackend("CREATE ENTITY: create", err)
		}
	}

	invalidateDomainModelGenForModuleDeps(deps, module.ID)
	invalidateDomainModelsCacheDeps(deps)
	invalidateHierarchyDeps(deps)

	if s.Position == nil {
		if err := deps.DomainModelWriter.RelayoutDomainModel(model.ID(dm.ID())); err != nil {
			fmt.Fprintf(deps.Output, "warning: auto-layout failed: %v\n", err)
		} else {
			invalidateDomainModelGenForModuleDeps(deps, module.ID)
		}
	}

	if existing != nil {
		fmt.Fprintf(deps.Output, "Modified entity: %s\n", s.Name)
	} else {
		fmt.Fprintf(deps.Output, "Created entity: %s\n", s.Name)
	}
	trackModifiedDomainModelDeps(deps, module.ID, module.Name)
	return nil
}

// validateCreateEntityTypeRefsGenDeps is the HandlerDeps version of validateCreateEntityTypeRefsGen.
func validateCreateEntityTypeRefsGenDeps(ctx context.Context, deps *HandlerDeps, s *ast.CreateEntityStmt) error {
	if deps == nil || s == nil {
		return nil
	}
	for _, a := range s.Attributes {
		if a.Type.Kind != ast.TypeEnumeration || a.Type.EnumRef == nil {
			continue
		}
		refModule := a.Type.EnumRef.Module
		refName := a.Type.EnumRef.Name
		if findEnumerationDeps(deps, refModule, refName) != nil {
			continue
		}
		if _, err := findEntityDeps(ctx, deps, refModule, refName); err == nil {
			continue
		}
		return mdlerrors.NewValidationf(
			"attribute '%s': unknown type '%s' — not a primitive, enumeration, or entity",
			a.Name, a.Type.EnumRef.String())
	}
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
