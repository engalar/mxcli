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
// container UUID, using the domain-cached listing.
func listPagesWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genPg.Page], error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache.pagesWithContainerGen == nil {
		ctx.Cache.pagesWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Page], error) {
			return loadPagesWithContainerGen(ctx.Deps)
		})
	}
	return ctx.Cache.pagesWithContainerGen.Get()
}

// loadPagesWithContainerGen loads pages without caching.
func loadPagesWithContainerGen(deps *HandlerDeps) ([]ContainerWithGen[*genPg.Page], error) {
	if deps == nil || deps.PageRepo == nil {
		return nil, nil
	}
	return listUnitsWithContainerGen(
		func() ([]*genPg.Page, error) {
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
			return filtered, nil
		},
		func(id element.ID) (element.ID, error) {
			c, err := deps.PageRepo.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		},
		func() ([]ContainerWithGen[*genPg.Page], bool) { return nil, false },
		func([]ContainerWithGen[*genPg.Page]) {},
	)
}

// listLayoutsWithContainerGen mirrors listPagesWithContainerGen for layouts.
func listLayoutsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genPg.Layout], error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache.layoutsWithContainerGen == nil {
		ctx.Cache.layoutsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Layout], error) {
			return loadLayoutsWithContainerGen(ctx.Deps)
		})
	}
	return ctx.Cache.layoutsWithContainerGen.Get()
}

// loadLayoutsWithContainerGen loads layouts without caching.
func loadLayoutsWithContainerGen(deps *HandlerDeps) ([]ContainerWithGen[*genPg.Layout], error) {
	if deps == nil || deps.LayoutRepo == nil {
		return nil, nil
	}
	return listUnitsWithContainerGen(
		func() ([]*genPg.Layout, error) {
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
			return filtered, nil
		},
		func(id element.ID) (element.ID, error) {
			c, err := deps.LayoutRepo.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		},
		func() ([]ContainerWithGen[*genPg.Layout], bool) { return nil, false },
		func([]ContainerWithGen[*genPg.Layout]) {},
	)
}

// listSnippetsWithContainerGen mirrors listPagesWithContainerGen for snippets.
func listSnippetsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genPg.Snippet], error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache.snippetsWithContainerGen == nil {
		ctx.Cache.snippetsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Snippet], error) {
			return loadSnippetsWithContainerGen(ctx.Deps)
		})
	}
	return ctx.Cache.snippetsWithContainerGen.Get()
}

// loadSnippetsWithContainerGen loads snippets without caching.
func loadSnippetsWithContainerGen(deps *HandlerDeps) ([]ContainerWithGen[*genPg.Snippet], error) {
	if deps == nil || deps.SnippetRepo == nil {
		return nil, nil
	}
	return listUnitsWithContainerGen(
		func() ([]*genPg.Snippet, error) {
			all, err := deps.SnippetRepo.ListAll()
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
		},
		func(id element.ID) (element.ID, error) {
			c, err := deps.SnippetRepo.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		},
		func() ([]ContainerWithGen[*genPg.Snippet], bool) { return nil, false },
		func([]ContainerWithGen[*genPg.Snippet]) {},
	)
}

// listPagesWithContainerGenDeps is the HandlerDeps version of listPagesWithContainerGen.
func listPagesWithContainerGenDeps(ctx context.Context, deps *HandlerDeps) ([]ContainerWithGen[*genPg.Page], error) {
	if deps == nil || deps.Cache == nil {
		return nil, nil
	}
	if deps.Cache.pagesWithContainerGen == nil {
		deps.Cache.pagesWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Page], error) {
			return loadPagesWithContainerGen(deps)
		})
	}
	return deps.Cache.pagesWithContainerGen.Get()
}

// listSnippetsWithContainerGenDeps is the HandlerDeps version of listSnippetsWithContainerGen.
func listSnippetsWithContainerGenDeps(ctx context.Context, deps *HandlerDeps) ([]ContainerWithGen[*genPg.Snippet], error) {
	if deps == nil || deps.Cache == nil {
		return nil, nil
	}
	if deps.Cache.snippetsWithContainerGen == nil {
		deps.Cache.snippetsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Snippet], error) {
			return loadSnippetsWithContainerGen(deps)
		})
	}
	return deps.Cache.snippetsWithContainerGen.Get()
}

// listLayoutsWithContainerGenDeps is the HandlerDeps version of listLayoutsWithContainerGen.
func listLayoutsWithContainerGenDeps(deps *HandlerDeps) ([]ContainerWithGen[*genPg.Layout], error) {
	if deps == nil || deps.Cache == nil {
		return nil, nil
	}
	if deps.Cache.layoutsWithContainerGen == nil {
		deps.Cache.layoutsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genPg.Layout], error) {
			return loadLayoutsWithContainerGen(deps)
		})
	}
	return deps.Cache.layoutsWithContainerGen.Get()
}

// invalidatePagesGenCache clears the cached page/layout/snippet listings.
func invalidatePagesGenCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.Invalidate(CacheDomainPages, CacheDomainLayouts, CacheDomainSnippets)
}

// invalidatePagesGenCacheDeps is the HandlerDeps version of invalidatePagesGenCache.
func invalidatePagesGenCacheDeps(deps *HandlerDeps) {
	if deps == nil || deps.Cache == nil {
		return
	}
	deps.Cache.Invalidate(CacheDomainPages, CacheDomainLayouts, CacheDomainSnippets)
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
	// Check session-created pages first (MPR v2 may not immediately
	// surface newly written files via ListAll).
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
