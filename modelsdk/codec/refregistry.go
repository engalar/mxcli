package codec

// RefKind identifies the kind of reference property.
type RefKind string

const (
	RefByName     RefKind = "byname"
	RefByNameList RefKind = "bynamelist"
	RefById       RefKind = "byid"
)

// RefMeta describes a single reference property on a generated type.
type RefMeta struct {
	Prop   string  // BSON key (e.g. "Entity", "ParentPointer")
	Kind   RefKind // byname, bynamelist, or byid
	Target string  // TargetType for ByNameRef (e.g. "DomainModels$Entity"); empty for ByIdRef
}

// RefRegistry stores reference metadata for all generated types.
type RefRegistry struct {
	refs map[string][]RefMeta // structureTypeName -> reference properties
}

// DefaultRefRegistry is the package-level registry populated by generated init() functions.
var DefaultRefRegistry = NewRefRegistry()

// NewRefRegistry returns an empty RefRegistry.
func NewRefRegistry() *RefRegistry {
	return &RefRegistry{refs: map[string][]RefMeta{}}
}

// RegisterRefs records reference properties for a type.
func (r *RefRegistry) RegisterRefs(typeName string, refs []RefMeta) {
	r.refs[typeName] = refs
}

// Refs returns the reference properties for a type.
func (r *RefRegistry) Refs(typeName string) []RefMeta {
	return r.refs[typeName]
}

// AllTargetTypes returns all unique ByNameRef target types across all registered types.
func (r *RefRegistry) AllTargetTypes() []string {
	set := map[string]bool{}
	for _, refs := range r.refs {
		for _, ref := range refs {
			if ref.Target != "" {
				set[ref.Target] = true
			}
		}
	}
	result := make([]string, 0, len(set))
	for t := range set {
		result = append(result, t)
	}
	return result
}
