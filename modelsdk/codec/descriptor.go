package codec

import "sync"

type PropKind uint8

const (
	PropKindString     PropKind = iota
	PropKindBool
	PropKindInt32
	PropKindFloat64
	PropKindPart
	PropKindPartList
	PropKindByNameRef
	PropKindByNameList
	PropKindByIdRef
	PropKindEnum
	PropKindEnumList
	PropKindStringList
	PropKindBinaryUUID
)

type PropDesc struct {
	Name    string
	BSONKey string
	Kind    PropKind
	RefType string
}

type TypeDesc struct {
	TypeName   string
	Properties []PropDesc
}

type DescRegistry struct {
	mu    sync.RWMutex
	descs map[string]*TypeDesc
}

func NewDescRegistry() *DescRegistry {
	return &DescRegistry{descs: map[string]*TypeDesc{}}
}

func (r *DescRegistry) Register(typeName string, td *TypeDesc) {
	r.mu.Lock()
	r.descs[typeName] = td
	r.mu.Unlock()
}

func (r *DescRegistry) Lookup(typeName string) (*TypeDesc, bool) {
	r.mu.RLock()
	td, ok := r.descs[typeName]
	r.mu.RUnlock()
	return td, ok
}

func (r *DescRegistry) All() []*TypeDesc {
	r.mu.RLock()
	out := make([]*TypeDesc, 0, len(r.descs))
	for _, td := range r.descs {
		out = append(out, td)
	}
	r.mu.RUnlock()
	return out
}

var DefaultDescRegistry = NewDescRegistry()
