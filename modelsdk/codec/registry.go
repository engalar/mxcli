package codec

import (
	"sync"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// TypeRegistry maps BSON $Type strings to element factory functions.
type TypeRegistry struct {
	mu        sync.RWMutex
	factories map[string]func() element.Element
}

// NewRegistry returns an empty TypeRegistry.
func NewRegistry() *TypeRegistry {
	return &TypeRegistry{factories: map[string]func() element.Element{}}
}

// Register adds or replaces the factory for the given typeName.
func (r *TypeRegistry) Register(typeName string, factory func() element.Element) {
	r.mu.Lock()
	r.factories[typeName] = factory
	r.mu.Unlock()
}

// Lookup returns the factory for typeName, and whether it was found.
func (r *TypeRegistry) Lookup(typeName string) (func() element.Element, bool) {
	r.mu.RLock()
	f, ok := r.factories[typeName]
	r.mu.RUnlock()
	return f, ok
}

// RegisterAlias registers storageName as an alias for qualifiedName.
// When the decoder encounters storageName in $Type, it uses the factory
// registered for qualifiedName.
func (r *TypeRegistry) RegisterAlias(storageName, qualifiedName string) {
	r.mu.RLock()
	f, ok := r.factories[qualifiedName]
	r.mu.RUnlock()
	if ok {
		r.Register(storageName, f)
	}
}

// DefaultRegistry is the package-level registry used by generated types.
var DefaultRegistry = NewRegistry()

// No runtime aliases needed — all storage name mappings are handled at
// codegen time via dual registration in gen/*/types.go init() functions.
// See cmd/modelsdk-codegen/main.go storageAliases and propertyKeyOverrides.
