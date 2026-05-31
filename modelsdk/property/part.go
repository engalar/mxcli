package property

import "github.com/mendixlabs/mxcli/modelsdk/element"

// Part[T] holds a single contained child element.
type Part[T element.Element] struct {
	propertyBase
	val T
	set bool
}

func NewPart[T element.Element](name string) *Part[T] {
	return &Part[T]{propertyBase: propertyBase{name: name}}
}

func (p *Part[T]) Get() T {
	if !p.set {
		var zero T
		return zero
	}
	return p.val
}

func (p *Part[T]) Set(v T) {
	p.val = v
	p.set = true
	p.markDirty()
}

// BSONValue returns nil — Part children must be encoded recursively by the Encoder.
// The Encoder checks for ChildProperty interface instead.
func (p *Part[T]) BSONValue() any { return nil }

// ChildElement returns the contained element for recursive encoding.
func (p *Part[T]) ChildElement() element.Element {
	if !p.set {
		return nil
	}
	return p.val
}

func (p *Part[T]) SetFromDecode(v T) {
	p.val = v
	p.set = true
	if setter, ok := any(v).(interface{ SetContainer(element.Element) }); ok && p.owner != nil {
		setter.SetContainer(p.owner)
	}
}

// PartList[T] holds a list of contained child elements.
type PartList[T element.Element] struct {
	propertyBase
	items         []T
	versionMarker int32 // BSON array version prefix: 3 (default), 2 for legacy Java action params
}

func NewPartList[T element.Element](name string) *PartList[T] {
	return &PartList[T]{propertyBase: propertyBase{name: name}, versionMarker: 3}
}

// NewPartListV2 creates a PartList that writes version marker int32(2) instead of 3.
// Required for JavaActions$JavaAction.Parameters and TypeParameters to match the
// Mendix-native BSON format (Studio Pro writes [2, ...] for these lists).
// Using version 3 causes MprTool to crash with InvalidCastException.
func NewPartListV2[T element.Element](name string) *PartList[T] {
	return &PartList[T]{propertyBase: propertyBase{name: name}, versionMarker: 2}
}

// VersionMarker returns the BSON array version prefix written before the list items.
func (p *PartList[T]) VersionMarker() int32 {
	if p.versionMarker == 0 {
		return 3
	}
	return p.versionMarker
}

func (p *PartList[T]) Items() []T     { return p.items }
func (p *PartList[T]) Len() int       { return len(p.items) }
func (p *PartList[T]) BSONValue() any { return nil } // handled by ChildElements

// ChildElements returns all contained elements for recursive encoding.
func (p *PartList[T]) ChildElements() []element.Element {
	out := make([]element.Element, len(p.items))
	for i, v := range p.items {
		out[i] = v
	}
	return out
}

func (p *PartList[T]) Append(v T) {
	p.items = append(p.items, v)
	p.markDirty()
	if setter, ok := any(v).(interface{ SetContainer(element.Element) }); ok && p.owner != nil {
		setter.SetContainer(p.owner)
	}
}

func (p *PartList[T]) Remove(index int) {
	if index < 0 || index >= len(p.items) {
		return
	}
	p.items = append(p.items[:index], p.items[index+1:]...)
	p.markDirty()
}

// InsertAt inserts an element at the given index, shifting subsequent items right.
// If index <= 0 the item is prepended; if index >= len it is appended.
func (p *PartList[T]) InsertAt(index int, v T) {
	if index <= 0 {
		p.items = append([]T{v}, p.items...)
	} else if index >= len(p.items) {
		p.items = append(p.items, v)
	} else {
		p.items = append(p.items[:index], append([]T{v}, p.items[index:]...)...)
	}
	p.markDirty()
	if setter, ok := any(v).(interface{ SetContainer(element.Element) }); ok && p.owner != nil {
		setter.SetContainer(p.owner)
	}
}

func (p *PartList[T]) AppendFromDecode(v T) {
	p.items = append(p.items, v)
	if setter, ok := any(v).(interface{ SetContainer(element.Element) }); ok && p.owner != nil {
		setter.SetContainer(p.owner)
	}
}
