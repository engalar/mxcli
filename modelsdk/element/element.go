package element

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ID string

type Property interface {
	Name() string
}

// WritableProperty is implemented by properties that can report dirty state
// and encode their current value for BSON serialization.
type WritableProperty interface {
	Property
	Dirty() bool
	// BSONValue returns the current value suitable for bson.D insertion.
	// For Part/PartList properties, returns nil — use ChildProperty/ChildListProperty instead.
	BSONValue() any
}

// ChildProperty is a Part-like property holding a single child Element.
type ChildProperty interface {
	WritableProperty
	ChildElement() Element
}

// ChildListProperty is a PartList-like property holding a list of child Elements.
type ChildListProperty interface {
	WritableProperty
	ChildElements() []Element
}

type Element interface {
	ID() ID
	TypeName() string
	Container() Element
	SetContainer(c Element)
	Unit() Unit
	Raw() bson.Raw
	IsDirty() bool
	Properties() []Property
}

type Unit interface {
	Element
	UnitID() ID
	IsLoaded() bool
}

type ContainerUnit interface {
	Unit
	Units() []Unit
}

type Base struct {
	id         ID
	typeName   string
	container  Element
	unit       Unit
	raw        bson.Raw
	dirty      []uint64
	childDirty bool
	props      []Property
}

func (b *Base) ID() ID                 { return b.id }
func (b *Base) SetID(id ID)            { b.id = id }
func (b *Base) TypeName() string       { return b.typeName }
func (b *Base) SetTypeName(t string)   { b.typeName = t }
func (b *Base) Container() Element     { return b.container }
func (b *Base) SetContainer(c Element) { b.container = c }
func (b *Base) Unit() Unit             { return b.unit }
func (b *Base) SetUnit(u Unit)         { b.unit = u }
func (b *Base) Raw() bson.Raw          { return b.raw }
func (b *Base) SetRaw(r bson.Raw)      { b.raw = r }
func (b *Base) IsDirty() bool {
	if b.childDirty {
		return true
	}
	for _, w := range b.dirty {
		if w != 0 {
			return true
		}
	}
	return false
}
func (b *Base) MarkDirty(bit uint) {
	word := bit / 64
	for len(b.dirty) <= int(word) {
		b.dirty = append(b.dirty, 0)
	}
	b.dirty[word] |= 1 << (bit % 64)
	if b.container != nil {
		if mc, ok := b.container.(interface{ MarkChildDirty() }); ok {
			mc.MarkChildDirty()
		}
	}
}
func (b *Base) MarkChildDirty() {
	if b.childDirty {
		return // already marked, stop recursion
	}
	b.childDirty = true
	if b.container != nil {
		if mc, ok := b.container.(interface{ MarkChildDirty() }); ok {
			mc.MarkChildDirty()
		}
	}
}

// DirtyBits returns the raw dirty bitmap words.
func (b *Base) DirtyBits() []uint64 { return b.dirty }

// IsChildDirty returns true if a child element was modified.
func (b *Base) IsChildDirty() bool         { return b.childDirty }
func (b *Base) Properties() []Property     { return b.props }
func (b *Base) SetProperties(p []Property) { b.props = p }

// NameValue returns the Name field from the underlying raw BSON.
// Returns "" if the element has no Name field or no raw BSON.
func (b *Base) NameValue() string {
	if b.raw == nil {
		return ""
	}
	val, err := b.raw.LookupErr("Name")
	if err != nil {
		return ""
	}
	s, _ := val.StringValueOK()
	return s
}

// AddProperty appends a property and binds it to this element.
// Use this to inject inherited or ad-hoc properties that the codegen
// doesn't produce (e.g. Document.Name on Microflow).
func (b *Base) AddProperty(p Property, bit uint) {
	if binder, ok := p.(interface{ Bind(owner *Base, bit uint) }); ok {
		binder.Bind(b, bit)
	}
	b.props = append(b.props, p)
}
