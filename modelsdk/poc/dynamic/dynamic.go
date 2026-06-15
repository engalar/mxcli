package dynamic

import (
	"fmt"
	"sync"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/property"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Kind identifies the category of a property value.
type Kind uint8

const (
	KindString     Kind = iota // Primitive[string], Enum, ByNameRef (qualified name)
	KindBool                   // Primitive[bool]
	KindInt32                  // Primitive[int32]
	KindFloat64                // Primitive[float64]
	KindPart                   // Part[element.Element] — single child element
	KindPartList               // PartList[element.Element] — list of child elements
	KindByID                   // ByIdRef — element ID reference
	KindStringList             // StringListPrimitive, EnumList, ByNameRefList
	KindMixedArray             // versioned []any (ByNameRefList)
	KindBinary                 // BinaryUUIDPrimitive
	KindUnknown
)

func (k Kind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindBool:
		return "bool"
	case KindInt32:
		return "int32"
	case KindFloat64:
		return "float64"
	case KindPart:
		return "part"
	case KindPartList:
		return "partlist"
	case KindByID:
		return "byid"
	case KindStringList:
		return "stringlist"
	case KindMixedArray:
		return "mixedarray"
	case KindBinary:
		return "binary"
	default:
		return "unknown"
	}
}

// Property wraps an element.Property with dynamic access.
type Property struct {
	prop   element.Property
	kind   Kind
	target string // target type name for references
}

func newProperty(p element.Property) *Property {
	d := &Property{prop: p}
	d.inferKind()
	return d
}

func (p *Property) Name() string  { return p.prop.Name() }
func (p *Property) Kind() Kind    { return p.kind }
func (p *Property) TargetType() string { return p.target }

func (p *Property) inferKind() {
	switch p.prop.(type) {
	case element.ChildListProperty:
		p.kind = KindPartList
		return
	case element.ChildProperty:
		p.kind = KindPart
		return
	}

	wp, ok := p.prop.(element.WritableProperty)
	if !ok {
		p.kind = KindUnknown
		return
	}

	switch wp.BSONValue().(type) {
	case string:
		p.kind = KindString
	case bool:
		p.kind = KindBool
	case int32:
		p.kind = KindInt32
	case float64:
		p.kind = KindFloat64
	case element.ID:
		p.kind = KindByID
	case []string:
		p.kind = KindStringList
	case []any:
		p.kind = KindMixedArray
	case bson.A:
		p.kind = KindStringList
	case bson.Binary:
		p.kind = KindBinary
	default:
		switch p.prop.(type) {
		case *property.StringListPrimitive:
			p.kind = KindStringList
		case *property.BinaryUUIDPrimitive:
			p.kind = KindBinary
		default:
			p.kind = KindUnknown
		}
	}
}

// Value returns the current value. Returns nil for Part/PartList.
func (p *Property) Value() any {
	switch p.kind {
	case KindPart, KindPartList:
		return nil
	default:
		if wp, ok := p.prop.(element.WritableProperty); ok {
			return wp.BSONValue()
		}
		return nil
	}
}

// String returns the string value if this is a string-like property.
func (p *Property) String() (string, bool) {
	v := p.Value()
	s, ok := v.(string)
	return s, ok
}

// Bool returns the bool value if this is a bool property.
func (p *Property) Bool() (bool, bool) {
	v := p.Value()
	b, ok := v.(bool)
	return b, ok
}

// Int32 returns the int32 value if this is an int32 property.
func (p *Property) Int32() (int32, bool) {
	v := p.Value()
	i, ok := v.(int32)
	return i, ok
}

// Children returns child elements for Part/PartList properties.
func (p *Property) Children() []element.Element {
	switch p.kind {
	case KindPart:
		if cp, ok := p.prop.(element.ChildProperty); ok {
			if ch := cp.ChildElement(); ch != nil {
				return []element.Element{ch}
			}
		}
	case KindPartList:
		if cl, ok := p.prop.(element.ChildListProperty); ok {
			return cl.ChildElements()
		}
	}
	return nil
}

// SetValue sets the value from an interface{} via type switch on the concrete property type.
func (p *Property) SetValue(v any) error {
	switch prop := p.prop.(type) {
	case *property.Primitive[string]:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", v)
		}
		prop.Set(s)
		return nil
	case *property.Primitive[bool]:
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("expected bool, got %T", v)
		}
		prop.Set(b)
		return nil
	case *property.Primitive[int32]:
		i, ok := v.(int32)
		if !ok {
			return fmt.Errorf("expected int32, got %T", v)
		}
		prop.Set(i)
		return nil
	case *property.Primitive[float64]:
		f, ok := v.(float64)
		if !ok {
			return fmt.Errorf("expected float64, got %T", v)
		}
		prop.Set(f)
		return nil
	case *property.Enum[string]:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", v)
		}
		prop.Set(s)
		return nil
	case *property.ByNameRef[element.Element]:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("expected string (qualified name), got %T", v)
		}
		prop.SetQualifiedName(s)
		return nil
	case *property.ByIdRef[element.Element]:
		id, ok := v.(element.ID)
		if !ok {
			return fmt.Errorf("expected element.ID, got %T", v)
		}
		prop.SetID(id)
		return nil
	case *property.StringListPrimitive:
		switch val := v.(type) {
		case string:
			prop.Set(val)
		case []string:
			prop.SetList(val)
		default:
			return fmt.Errorf("expected string or []string, got %T", v)
		}
		return nil
	case *property.BinaryUUIDPrimitive:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("expected string (UUID), got %T", v)
		}
		prop.Set(s)
		return nil
	default:
		return fmt.Errorf("unsupported property type %T", p.prop)
	}
}

// Element wraps an element.Element with cached dynamic property access.
type Element struct {
	elem   element.Element
	cached []*Property
	byName map[string]*Property
	once   sync.Once
}

func WrapElement(elem element.Element) *Element {
	return &Element{elem: elem}
}

func (e *Element) Element() element.Element { return e.elem }

func (e *Element) init() {
	e.once.Do(func() {
		props := e.elem.Properties()
		e.cached = make([]*Property, len(props))
		e.byName = make(map[string]*Property, len(props))
		for i, p := range props {
			dp := newProperty(p)
			e.cached[i] = dp
			e.byName[dp.Name()] = dp
		}
	})
}

// Property returns a dynamic view of the named property.
func (e *Element) Property(name string) *Property {
	e.init()
	return e.byName[name]
}

// Properties returns all cached dynamic properties.
func (e *Element) Properties() []*Property {
	e.init()
	return e.cached
}

// GetString is a convenience for reading a string property.
func (e *Element) GetString(name string) (string, bool) {
	p := e.Property(name)
	if p == nil {
		return "", false
	}
	return p.String()
}

// SetString is a convenience for setting a string property.
func (e *Element) SetString(name, val string) bool {
	p := e.Property(name)
	if p == nil {
		return false
	}
	return p.SetValue(val) == nil
}

// GetBool is a convenience for reading a bool property.
func (e *Element) GetBool(name string) (bool, bool) {
	p := e.Property(name)
	if p == nil {
		return false, false
	}
	return p.Bool()
}

// SetBool is a convenience for setting a bool property.
func (e *Element) SetBool(name string, val bool) bool {
	p := e.Property(name)
	if p == nil {
		return false
	}
	return p.SetValue(val) == nil
}

// DescribeElement returns a human-readable description of an element's structure.
func DescribeElement(elem element.Element) map[string]string {
	desc := map[string]string{}
	desc["$Type"] = elem.TypeName()
	desc["$ID"] = string(elem.ID())
	e := WrapElement(elem)
	for _, p := range e.Properties() {
		desc[p.Name()] = p.Kind().String()
	}
	return desc
}

// AllTypes returns all registered type names from the codec registry.
func AllTypes() []string {
	// Access the DefaultRegistry to enumerate types.
	// Using reflect-based approach since registry doesn't expose enumeration.
	type typeRegistry interface {
		Lookup(string) (func() element.Element, bool)
	}
	_ = codec.DefaultRegistry
	// For now, iterate RefRegistry for known types.
	return codec.DefaultRefRegistry.AllTargetTypes()
}

// RawString reads a string field directly from the element's raw BSON.
func RawString(elem element.Element, key string) (string, bool) {
	raw := elem.Raw()
	if raw == nil {
		return "", false
	}
	val, err := raw.LookupErr(key)
	if err != nil {
		return "", false
	}
	return val.StringValueOK()
}

// RawBool reads a bool field directly from raw BSON.
func RawBool(elem element.Element, key string) (bool, bool) {
	raw := elem.Raw()
	if raw == nil {
		return false, false
	}
	val, err := raw.LookupErr(key)
	if err != nil {
		return false, false
	}
	return val.BooleanOK()
}

// RawInt32 reads an int32 field directly from raw BSON.
func RawInt32(elem element.Element, key string) (int32, bool) {
	raw := elem.Raw()
	if raw == nil {
		return 0, false
	}
	val, err := raw.LookupErr(key)
	if err != nil {
		return 0, false
	}
	return val.Int32OK()
}
