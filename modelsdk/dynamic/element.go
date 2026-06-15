package dynamic

import (
	"sync"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

func KnownTypes() []*codec.TypeDesc {
	return codec.DefaultDescRegistry.All()
}

func LookupType(typeName string) (*codec.TypeDesc, bool) {
	return codec.DefaultDescRegistry.Lookup(typeName)
}

type Element struct {
	elem   element.Element
	once   sync.Once
	cached []*Property
	byName map[string]*Property
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

func (e *Element) Property(name string) *Property {
	e.init()
	return e.byName[name]
}

func (e *Element) Properties() []*Property {
	e.init()
	return e.cached
}

func (e *Element) GetString(name string) (string, bool) {
	p := e.Property(name)
	if p == nil {
		return "", false
	}
	return p.String()
}

func (e *Element) SetString(name, val string) bool {
	p := e.Property(name)
	if p == nil {
		return false
	}
	return p.SetValue(val) == nil
}

func (e *Element) GetBool(name string) (bool, bool) {
	p := e.Property(name)
	if p == nil {
		return false, false
	}
	return p.Bool()
}

func (e *Element) SetBool(name string, val bool) bool {
	p := e.Property(name)
	if p == nil {
		return false
	}
	return p.SetValue(val) == nil
}

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
