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
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
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
