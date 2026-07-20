// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.5.A0: cache helpers for the pages domain.
// Mirrors helpers_workflows_gen.go shape — Pages, Layouts and Snippets
// each get a dedicated list-with-container helper backed by the
// generic ContainerWithGen[T] factory (helpers_gen_container.go).
//
// Per memory `feedback_executor_cache_pattern`: list calls in
// mdl/executor/ MUST go through these helpers to avoid O(N²) container
// lookups when iterating across the project.

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// listPagesWithContainerGen returns every Page paired with its
// container UUID, caching the result on
// ctx.Cache.pagesWithContainerGen for the session.
func listPagesWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genPg.Page], error) {
	if ctx == nil {
		return nil, nil
	}
	listFn := func() ([]*genPg.Page, error) {
		if ctx.Pages == nil {
			return nil, nil
		}
		all, err := ctx.Pages.ListAll()
		if err != nil {
			return nil, err
		}
		// Per helpers_gen_container.go contract: list callers MUST
		// filter nil entries — the generic factory dereferences
		// e.ID() unconditionally.
		filtered := all[:0]
		for _, p := range all {
			if p != nil {
				filtered = append(filtered, p)
			}
		}
		return filtered, nil
	}
	resolveFn := func(id element.ID) (element.ID, error) {
		if ctx.Pages != nil {
			c, err := ctx.Pages.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		}
		return "", nil
	}
	return listUnitsWithContainerGen(
		listFn,
		resolveFn,
		func() ([]ContainerWithGen[*genPg.Page], bool) {
			if ctx.Cache != nil && ctx.Cache.pagesWithContainerGen != nil {
				return ctx.Cache.pagesWithContainerGen, true
			}
			return nil, false
		},
		func(s []ContainerWithGen[*genPg.Page]) {
			if ctx.Cache != nil {
				ctx.Cache.pagesWithContainerGen = s
			}
		},
	)
}

// listLayoutsWithContainerGen mirrors listPagesWithContainerGen for
// layouts.
func listLayoutsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genPg.Layout], error) {
	if ctx == nil {
		return nil, nil
	}
	listFn := func() ([]*genPg.Layout, error) {
		if ctx.Layouts == nil {
			return nil, nil
		}
		all, err := ctx.Layouts.ListAll()
		if err != nil {
			return nil, err
		}
		filtered := all[:0]
		for _, l := range all {
			if l != nil {
				filtered = append(filtered, l)
			}
		}
		return filtered, nil
	}
	resolveFn := func(id element.ID) (element.ID, error) {
		if ctx.Layouts != nil {
			c, err := ctx.Layouts.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		}
		return "", nil
	}
	return listUnitsWithContainerGen(
		listFn,
		resolveFn,
		func() ([]ContainerWithGen[*genPg.Layout], bool) {
			if ctx.Cache != nil && ctx.Cache.layoutsWithContainerGen != nil {
				return ctx.Cache.layoutsWithContainerGen, true
			}
			return nil, false
		},
		func(s []ContainerWithGen[*genPg.Layout]) {
			if ctx.Cache != nil {
				ctx.Cache.layoutsWithContainerGen = s
			}
		},
	)
}

// listSnippetsWithContainerGen mirrors listPagesWithContainerGen for
// snippets.
func listSnippetsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genPg.Snippet], error) {
	if ctx == nil {
		return nil, nil
	}
	listFn := func() ([]*genPg.Snippet, error) {
		if ctx.Snippets == nil {
			return nil, nil
		}
		all, err := ctx.Snippets.ListAll()
		if err != nil {
			return nil, err
		}
		filtered := all[:0]
		for _, s := range all {
			if s != nil {
				filtered = append(filtered, s)
			}
		}
		return filtered, nil
	}
	resolveFn := func(id element.ID) (element.ID, error) {
		if ctx.Snippets != nil {
			c, err := ctx.Snippets.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		}
		return "", nil
	}
	return listUnitsWithContainerGen(
		listFn,
		resolveFn,
		func() ([]ContainerWithGen[*genPg.Snippet], bool) {
			if ctx.Cache != nil && ctx.Cache.snippetsWithContainerGen != nil {
				return ctx.Cache.snippetsWithContainerGen, true
			}
			return nil, false
		},
		func(s []ContainerWithGen[*genPg.Snippet]) {
			if ctx.Cache != nil {
				ctx.Cache.snippetsWithContainerGen = s
			}
		},
	)
}

// invalidatePagesGenCache clears the cached gen-typed page/layout/
// snippet listings. Call from any write path that creates, drops, or
// otherwise mutates page-family units.
func invalidatePagesGenCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.pagesWithContainerGen = nil
	ctx.Cache.layoutsWithContainerGen = nil
	ctx.Cache.snippetsWithContainerGen = nil
}

// parameterEntityNameGen extracts the qualified entity name from a
// gen-typed page/snippet parameter type element. Mirrors the legacy
// pages.PageParameter.EntityName fallback chain. Returns "" when the
// parameter is a primitive type (caller dispatches to MDL primitives via
// pageParamTypeMDL or similar) or when no entity is referenced.
// listPagesWithContainerGenDeps is the HandlerDeps version of listPagesWithContainerGen.
func listPagesWithContainerGenDeps(ctx context.Context, deps *HandlerDeps) ([]ContainerWithGen[*genPg.Page], error) {
	if deps == nil || deps.PageRepo == nil {
		return nil, nil
	}
	if deps.Cache != nil && deps.Cache.pagesWithContainerGen != nil {
		return deps.Cache.pagesWithContainerGen, nil
	}

	all, err := deps.PageRepo.ListAll()
	if err != nil {
		return nil, err
	}
	filtered := all[:0]
	for _, p := range all {
		if p != nil {
			filtered = append(filtered, p)
		}
	}

	var out []ContainerWithGen[*genPg.Page]
	for _, p := range filtered {
		cid, err := deps.PageRepo.GetContainerUUID(model.ID(p.ID()))
		if err != nil {
			continue
		}
		out = append(out, ContainerWithGen[*genPg.Page]{Elem: p, ContainerID: element.ID(cid)})
	}
	if deps.Cache != nil {
		deps.Cache.pagesWithContainerGen = out
	}
	return out, nil
}

// listSnippetsWithContainerGenDeps is the HandlerDeps version of listSnippetsWithContainerGen.
func listSnippetsWithContainerGenDeps(ctx context.Context, deps *HandlerDeps) ([]ContainerWithGen[*genPg.Snippet], error) {
	if deps == nil || deps.SnippetRepo == nil {
		return nil, nil
	}
	if deps.Cache != nil && deps.Cache.snippetsWithContainerGen != nil {
		return deps.Cache.snippetsWithContainerGen, nil
	}

	all, err := deps.SnippetRepo.ListAll()
	if err != nil {
		return nil, err
	}
	filtered := all[:0]
	for _, p := range all {
		if p != nil {
			filtered = append(filtered, p)
		}
	}

	var out []ContainerWithGen[*genPg.Snippet]
	for _, p := range filtered {
		cid, err := deps.SnippetRepo.GetContainerUUID(model.ID(p.ID()))
		if err != nil {
			continue
		}
		out = append(out, ContainerWithGen[*genPg.Snippet]{Elem: p, ContainerID: element.ID(cid)})
	}
	if deps.Cache != nil {
		deps.Cache.snippetsWithContainerGen = out
	}
	return out, nil
}

// listLayoutsWithContainerGenDeps is the HandlerDeps version of listLayoutsWithContainerGen.
func listLayoutsWithContainerGenDeps(deps *HandlerDeps) ([]ContainerWithGen[*genPg.Layout], error) {
	if deps == nil || deps.LayoutRepo == nil {
		return nil, nil
	}
	if deps.Cache != nil && deps.Cache.layoutsWithContainerGen != nil {
		return deps.Cache.layoutsWithContainerGen, nil
	}

	all, err := deps.LayoutRepo.ListAll()
	if err != nil {
		return nil, err
	}
	filtered := all[:0]
	for _, l := range all {
		if l != nil {
			filtered = append(filtered, l)
		}
	}

	var out []ContainerWithGen[*genPg.Layout]
	for _, l := range filtered {
		cid, err := deps.LayoutRepo.GetContainerUUID(model.ID(l.ID()))
		if err != nil {
			continue
		}
		out = append(out, ContainerWithGen[*genPg.Layout]{Elem: l, ContainerID: element.ID(cid)})
	}
	if deps.Cache != nil {
		deps.Cache.layoutsWithContainerGen = out
	}
	return out, nil
}

// invalidatePagesGenCacheDeps is the HandlerDeps version of invalidatePagesGenCache.
func invalidatePagesGenCacheDeps(deps *HandlerDeps) {
	if deps == nil || deps.Cache == nil {
		return
	}
	deps.Cache.pagesWithContainerGen = nil
	deps.Cache.layoutsWithContainerGen = nil
	deps.Cache.snippetsWithContainerGen = nil
}

func parameterEntityNameGen(paramType element.Element) string {
	if paramType == nil {
		return ""
	}
	switch t := paramType.(type) {
	case *genDt.EntityType:
		if t == nil {
			return ""
		}
		return t.EntityQualifiedName()
	case *genDt.ListType:
		if t == nil {
			return ""
		}
		return t.EntityQualifiedName()
	case *genDt.ObjectType:
		if t == nil {
			return ""
		}
		return t.EntityQualifiedName()
	}
	return ""
}

// findPageIDGen resolves a Page by qualified name through the gen-typed
// listing and returns its model.ID. Used by consumers that only need the
// page identifier (e.g. raw-BSON widget tree readers) and want to skip
// the legacy sdk-typed Page lookup.
func findPageIDGen(ctx *ExecContext, name ast.QualifiedName, h *ContainerHierarchy) (model.ID, error) {
	// Check session-created pages first (FUSE-backed goldenfs and MPR v2
	// may not immediately surface newly written files via ListAll).
	qualifiedName := name.Module + "." + name.Name
	if info := ctx.getCreatedPage(qualifiedName); info != nil {
		return info.ID, nil
	}
	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list pages", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == name.Name && (name.Module == "" || modName == name.Module) {
			return model.ID(p.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("page", name.String())
}

// findSnippetIDGen mirrors findPageIDGen for snippets.
func findSnippetIDGen(ctx *ExecContext, name ast.QualifiedName, h *ContainerHierarchy) (model.ID, error) {
	pairs, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list snippets", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == name.Name && (name.Module == "" || modName == name.Module) {
			return model.ID(p.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("snippet", name.String())
}

// findPageIDGenDeps is the HandlerDeps version of findPageIDGen.
func findPageIDGenDeps(deps *HandlerDeps, name ast.QualifiedName, h *ContainerHierarchy) (model.ID, error) {
	pairs, err := listPagesWithContainerGenDeps(context.Background(), deps)
	if err != nil {
		return "", mdlerrors.NewBackend("list pages", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == name.Name && (name.Module == "" || modName == name.Module) {
			return model.ID(p.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("page", name.String())
}

// findSnippetIDGenDeps is the HandlerDeps version of findSnippetIDGen.
func findSnippetIDGenDeps(deps *HandlerDeps, name ast.QualifiedName, h *ContainerHierarchy) (model.ID, error) {
	pairs, err := listSnippetsWithContainerGenDeps(context.Background(), deps)
	if err != nil {
		return "", mdlerrors.NewBackend("list snippets", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == name.Name && (name.Module == "" || modName == name.Module) {
			return model.ID(p.Elem.ID()), nil
		}
	}
	return "", mdlerrors.NewNotFound("snippet", name.String())
}

// pageParamTypeMDLGen returns the MDL type string for a gen-typed page
// parameter. Mirrors pageParamTypeMDL: primitive types collapse to
// "String"/"Integer"/etc., entity types return the qualified name, and
// the raw BSON storage name is the last-resort fallback so unknown
// parameter shapes are still introspectable in DESCRIBE output.
func pageParamTypeMDLGen(paramType element.Element) string {
	if paramType == nil {
		return ""
	}
	if name := parameterEntityNameGen(paramType); name != "" {
		return name
	}
	switch paramType.TypeName() {
	case "DataTypes$StringType":
		return "String"
	case "DataTypes$IntegerType":
		return "Integer"
	case "DataTypes$LongType":
		return "Long"
	case "DataTypes$DecimalType":
		return "Decimal"
	case "DataTypes$BooleanType":
		return "Boolean"
	case "DataTypes$DateTimeType":
		return "DateTime"
	}
	return paramType.TypeName()
}
