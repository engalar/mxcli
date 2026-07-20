// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.2.A0: cache helpers for javaactions / javascriptactions.
// Mirrors helpers_gen.go (microflow) and helpers_security_gen.go shape
// but uses the generic ContainerWithGen[T] factory introduced for the
// Stage 3.3 marathon (see helpers_gen_container.go).

package executor

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// listJavaActionsWithContainerGen returns every Java action paired with
// its container UUID, using the domain-cached listing.
func listJavaActionsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genJA.JavaAction], error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache.javaActionsWithContainerGen == nil {
		ctx.Cache.javaActionsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genJA.JavaAction], error) {
			return loadJavaActionsWithContainerGen(ctx.Deps)
		})
	}
	return ctx.Cache.javaActionsWithContainerGen.Get()
}

// loadJavaActionsWithContainerGen loads Java actions without caching.
func loadJavaActionsWithContainerGen(deps *HandlerDeps) ([]ContainerWithGen[*genJA.JavaAction], error) {
	if deps == nil {
		return nil, nil
	}
	return listUnitsWithContainerGen(
		func() ([]*genJA.JavaAction, error) {
			if deps.JavaActionRepo != nil {
				return deps.JavaActionRepo.ListAll()
			}
			if deps.JavaActionReader != nil {
				return deps.JavaActionReader.ListJavaActionsGen()
			}
			return nil, nil
		},
		func(id element.ID) (element.ID, error) {
			if deps.JavaActionRepo != nil {
				c, err := deps.JavaActionRepo.GetContainerUUID(model.ID(id))
				return element.ID(c), err
			}
			return "", nil
		},
		func() ([]ContainerWithGen[*genJA.JavaAction], bool) { return nil, false },
		func([]ContainerWithGen[*genJA.JavaAction]) {},
	)
}

// listJavaScriptActionsWithContainerGen mirrors listJavaActionsWithContainerGen
// for JavaScript actions.
func listJavaScriptActionsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genJSA.JavaScriptAction], error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.Cache.javaScriptActionsWithContainerGen == nil {
		ctx.Cache.javaScriptActionsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genJSA.JavaScriptAction], error) {
			return loadJavaScriptActionsWithContainerGen(ctx.Deps)
		})
	}
	return ctx.Cache.javaScriptActionsWithContainerGen.Get()
}

// loadJavaScriptActionsWithContainerGen loads JavaScript actions without caching.
func loadJavaScriptActionsWithContainerGen(deps *HandlerDeps) ([]ContainerWithGen[*genJSA.JavaScriptAction], error) {
	if deps == nil || deps.JavaScriptActionRepo == nil {
		return nil, nil
	}
	return listUnitsWithContainerGen(
		func() ([]*genJSA.JavaScriptAction, error) { return deps.JavaScriptActionRepo.ListAll() },
		func(id element.ID) (element.ID, error) {
			c, err := deps.JavaScriptActionRepo.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		},
		func() ([]ContainerWithGen[*genJSA.JavaScriptAction], bool) { return nil, false },
		func([]ContainerWithGen[*genJSA.JavaScriptAction]) {},
	)
}

// listJavaActionsWithContainerGenDeps is the HandlerDeps version of listJavaActionsWithContainerGen.
func listJavaActionsWithContainerGenDeps(deps *HandlerDeps) ([]ContainerWithGen[*genJA.JavaAction], error) {
	if deps == nil || deps.Cache == nil {
		return nil, nil
	}
	if deps.Cache.javaActionsWithContainerGen == nil {
		deps.Cache.javaActionsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genJA.JavaAction], error) {
			return loadJavaActionsWithContainerGen(deps)
		})
	}
	return deps.Cache.javaActionsWithContainerGen.Get()
}

// listJavaScriptActionsWithContainerGenDeps is the HandlerDeps version of listJavaScriptActionsWithContainerGen.
func listJavaScriptActionsWithContainerGenDeps(deps *HandlerDeps) ([]ContainerWithGen[*genJSA.JavaScriptAction], error) {
	if deps == nil || deps.Cache == nil {
		return nil, nil
	}
	if deps.Cache.javaScriptActionsWithContainerGen == nil {
		deps.Cache.javaScriptActionsWithContainerGen = newDomainCache(func() ([]ContainerWithGen[*genJSA.JavaScriptAction], error) {
			return loadJavaScriptActionsWithContainerGen(deps)
		})
	}
	return deps.Cache.javaScriptActionsWithContainerGen.Get()
}

// invalidateJavaActionsCache clears the cached Java action listing.
func invalidateJavaActionsCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.Invalidate(CacheDomainJavaActions)
}

// invalidateJavaScriptActionsCache clears the cached JavaScript action listing.
func invalidateJavaScriptActionsCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.Invalidate(CacheDomainJavaScriptActions)
}
