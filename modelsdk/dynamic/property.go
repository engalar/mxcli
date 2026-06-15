package dynamic

import (
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/property"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type PropertyKind uint8

const (
	KindString     PropertyKind = iota
	KindBool
	KindInt32
	KindFloat64
	KindPart
	KindPartList
	KindByID
	KindStringList
	KindBinary
	KindUnknown
)

func (k PropertyKind) String() string {
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
	case KindBinary:
		return "binary"
	default:
		return "unknown"
	}
}

type Property struct {
	prop element.Property
	kind PropertyKind
}

func newProperty(p element.Property) *Property {
	d := &Property{prop: p}
	d.inferKind()
	return d
}

func (p *Property) Name() string        { return p.prop.Name() }
func (p *Property) Kind() PropertyKind  { return p.kind }

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
		p.kind = KindStringList
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

func (p *Property) String() (string, bool) {
	v := p.Value()
	s, ok := v.(string)
	return s, ok
}

func (p *Property) Bool() (bool, bool) {
	v := p.Value()
	b, ok := v.(bool)
	return b, ok
}

func (p *Property) Int32() (int32, bool) {
	v := p.Value()
	i, ok := v.(int32)
	return i, ok
}
