// internal/fkg/concepts/registry.go
package concepts

import "github.com/mendixlabs/mxcli/internal/mxgraph"

var registry []mxgraph.IndexAdapter

// Register adds a concept adapter to the global registry.
// Called from each adapter's init() function.
func Register(a mxgraph.IndexAdapter) {
	registry = append(registry, a)
}

// All returns all registered concept adapters.
func All() []mxgraph.IndexAdapter {
	return registry
}
