// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// ContainerWithGen pairs a gen-typed element with its container UUID,
// resolved from the MPR Unit table. The factory listUnitsWithContainerGen
// produces these so per-domain helpers (microflow, entity, page, ...)
// don't each rewrite the cache+resolve loop.
type ContainerWithGen[T element.Element] struct {
	Elem        T
	ContainerID element.ID
}

// listUnitsWithContainerGen is the generic factory used by per-domain
// helpers (e.g. listMicroflowsWithContainerGen, listEntitiesWithContainerGen).
// It enforces the cache-or-build pattern that prevents O(N²) regressions
// per memory `feedback_executor_cache_pattern`.
//
// list:             returns the gen-typed elements (typically backed by
//                   ctx.<Domain>.ListAll()).
// resolveContainer: per-element container UUID lookup. Errors are
//                   swallowed — ContainerID stays zero — to match the
//                   existing microflow helper's tolerance for elements
//                   the backend cannot resolve (e.g. orphaned units).
// cacheGet/cachePut: read/write hooks bound by the caller to the
//                   appropriate ctx.Cache.<domain>WithContainerGen field.
//                   cacheGet returns (slice, true) on hit; on miss the
//                   factory builds, calls cachePut, and returns.
func listUnitsWithContainerGen[T element.Element](
	list func() ([]T, error),
	resolveContainer func(element.ID) (element.ID, error),
	cacheGet func() ([]ContainerWithGen[T], bool),
	cachePut func([]ContainerWithGen[T]),
) ([]ContainerWithGen[T], error) {
	if cached, ok := cacheGet(); ok {
		return cached, nil
	}
	elems, err := list()
	if err != nil {
		return nil, err
	}
	out := make([]ContainerWithGen[T], 0, len(elems))
	for _, e := range elems {
		var cid element.ID
		if c, err := resolveContainer(e.ID()); err == nil {
			cid = c
		}
		out = append(out, ContainerWithGen[T]{Elem: e, ContainerID: cid})
	}
	cachePut(out)
	return out, nil
}
