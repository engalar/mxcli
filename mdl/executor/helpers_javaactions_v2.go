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
// its container UUID, caching the result on
// ctx.Cache.javaActionsWithContainerGen for the session.
//
// Per memory `feedback_executor_cache_pattern`: list calls in
// mdl/executor/ MUST go through this helper to avoid O(N²) container
// lookups when iterating across the project.
func listJavaActionsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genJA.JavaAction], error) {
	if ctx == nil {
		return nil, nil
	}
	// Production path: ctx.JavaActions populated via duck-type provider
	// (MprBackend). Tests using MockBackend lack the provider, so fall
	// back to ctx.Backend.ListJavaActionsGen which goes through the
	// regular FullBackend interface.
	listFn := func() ([]*genJA.JavaAction, error) {
		if ctx.JavaActions != nil {
			return ctx.JavaActions.ListAll()
		}
		if ctx.JavaActionReader != nil {
			return ctx.JavaActionReader.ListJavaActionsGen()
		}
		return nil, nil
	}
	resolveFn := func(id element.ID) (element.ID, error) {
		if ctx.JavaActions != nil {
			c, err := ctx.JavaActions.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		}
		// MockBackend: caller wires container UUIDs separately via
		// withContainer / hierarchy. Returning empty here is fine — the
		// hierarchy walk later resolves Module from element ID directly.
		return "", nil
	}
	return listUnitsWithContainerGen(
		listFn,
		resolveFn,
		func() ([]ContainerWithGen[*genJA.JavaAction], bool) {
			if ctx.Cache != nil && ctx.Cache.javaActionsWithContainerGen != nil {
				return ctx.Cache.javaActionsWithContainerGen, true
			}
			return nil, false
		},
		func(s []ContainerWithGen[*genJA.JavaAction]) {
			if ctx.Cache != nil {
				ctx.Cache.javaActionsWithContainerGen = s
			}
		},
	)
}

// listJavaScriptActionsWithContainerGen mirrors listJavaActionsWithContainerGen
// for JavaScript actions.
func listJavaScriptActionsWithContainerGen(ctx *ExecContext) ([]ContainerWithGen[*genJSA.JavaScriptAction], error) {
	if ctx == nil || ctx.JavaScriptActions == nil {
		return nil, nil
	}
	return listUnitsWithContainerGen(
		func() ([]*genJSA.JavaScriptAction, error) { return ctx.JavaScriptActions.ListAll() },
		func(id element.ID) (element.ID, error) {
			c, err := ctx.JavaScriptActions.GetContainerUUID(model.ID(id))
			return element.ID(c), err
		},
		func() ([]ContainerWithGen[*genJSA.JavaScriptAction], bool) {
			if ctx.Cache != nil && ctx.Cache.javaScriptActionsWithContainerGen != nil {
				return ctx.Cache.javaScriptActionsWithContainerGen, true
			}
			return nil, false
		},
		func(s []ContainerWithGen[*genJSA.JavaScriptAction]) {
			if ctx.Cache != nil {
				ctx.Cache.javaScriptActionsWithContainerGen = s
			}
		},
	)
}

// invalidateJavaActionsCache clears the cached gen-typed Java action
// listing. Call from any write path that creates, drops, or otherwise
// mutates Java action units.
func invalidateJavaActionsCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.javaActionsWithContainerGen = nil
}

// invalidateJavaScriptActionsCache mirrors invalidateJavaActionsCache.
// JavaScript actions have no MDL write surface today so callers are
// limited to test fixture rebuilds, but the helper is wired for
// symmetry with the Java side.
func invalidateJavaScriptActionsCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.javaScriptActionsWithContainerGen = nil
}
