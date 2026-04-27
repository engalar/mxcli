package codec

import (
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// Decoder decodes a raw BSON document into an Element by dispatching on $Type.
type Decoder struct {
	registry *TypeRegistry
}

// NewDecoder returns a Decoder backed by the given registry.
func NewDecoder(r *TypeRegistry) *Decoder {
	return &Decoder{registry: r}
}

// RawInitializer is an optional interface that generated types can implement
// to populate their typed fields from the raw BSON bytes after construction.
type RawInitializer interface {
	InitFromRaw(raw bson.Raw)
}

// Decode parses raw and returns the appropriate Element.
// For types not found in the registry it returns a bare *element.Base that
// still carries the original raw bytes so the document round-trips safely.
func (d *Decoder) Decode(raw bson.Raw) (element.Element, error) {
	typeName := decodeTypeName(raw)
	if typeName == "" {
		return nil, fmt.Errorf("missing $Type in BSON document")
	}

	id := decodeID(raw)

	factory, ok := d.registry.Lookup(typeName)
	if !ok {
		// Unknown type — preserve raw bytes in a generic Base so the document
		// can be round-tripped without data loss.
		b := &element.Base{}
		b.SetTypeName(typeName)
		b.SetID(id)
		b.SetRaw(raw)
		return b, nil
	}

	elem := factory()

	// All generated types embed element.Base, which exposes these setters.
	if base, ok := elem.(interface{ SetTypeName(string) }); ok {
		base.SetTypeName(typeName)
	}
	if base, ok := elem.(interface{ SetID(element.ID) }); ok {
		base.SetID(id)
	}
	if base, ok := elem.(interface{ SetRaw(bson.Raw) }); ok {
		base.SetRaw(raw)
	}

	// Let the element parse its own typed fields if it knows how.
	if ri, ok := elem.(RawInitializer); ok {
		ri.InitFromRaw(raw)
	}

	return elem, nil
}

// decodeTypeName extracts the $Type string from a raw BSON document.
func decodeTypeName(raw bson.Raw) string {
	val, err := raw.LookupErr("$Type")
	if err != nil {
		return ""
	}
	s, _ := val.StringValueOK()
	return s
}

// fieldAliases maps SDK property names to their BSON storage names where they
// differ. Mendix historically used "Form" for "Layout" and "Page".
var fieldAliases = map[string]string{
	"LayoutCall":   "FormCall",
	"PageSettings": "FormSettings",
	"Layout":       "Form",
}

// DecodeChild decodes a single embedded document child from raw BSON by key.
// It uses DefaultRegistry to dispatch on $Type. Falls back to fieldAliases
// when the primary key is not found.
func DecodeChild(raw bson.Raw, key string) (element.Element, error) {
	val, err := raw.LookupErr(key)
	if err != nil {
		if alias, ok := fieldAliases[key]; ok {
			val, err = raw.LookupErr(alias)
		}
		if err != nil {
			return nil, fmt.Errorf("key %q not found", key)
		}
	}
	doc, ok := val.DocumentOK()
	if !ok {
		return nil, fmt.Errorf("key %q is not a document", key)
	}
	dec := NewDecoder(DefaultRegistry)
	return dec.Decode(bson.Raw(doc))
}

// DecodeChildren decodes an array of embedded document children from raw BSON.
// It uses DefaultRegistry to dispatch on $Type for each element.
func DecodeChildren(raw bson.Raw, key string) ([]element.Element, error) {
	val, err := raw.LookupErr(key)
	if err != nil {
		return nil, fmt.Errorf("key %q not found", key)
	}
	arr, ok := val.ArrayOK()
	if !ok {
		return nil, fmt.Errorf("key %q is not an array", key)
	}
	elems, err := arr.Elements()
	if err != nil {
		return nil, fmt.Errorf("key %q array elements: %w", key, err)
	}
	dec := NewDecoder(DefaultRegistry)
	result := make([]element.Element, 0, len(elems))
	for _, el := range elems {
		doc, ok := el.Value().DocumentOK()
		if !ok {
			continue // skip non-document elements
		}
		child, err := dec.Decode(bson.Raw(doc))
		if err != nil {
			continue // skip elements that fail to decode
		}
		result = append(result, child)
	}
	return result, nil
}

// decodeID extracts the $ID field, accepting either a UUID binary or a plain string.
func decodeID(raw bson.Raw) element.ID {
	val, err := raw.LookupErr("$ID")
	if err != nil {
		return ""
	}
	return decodeIDValue(val)
}

// decodeIDValue converts a BSON value (binary UUID or string) to an element.ID.
func decodeIDValue(val bson.RawValue) element.ID {
	switch val.Type {
	case bsontype.Binary:
		_, data, ok := val.BinaryOK()
		if ok && len(data) == 16 {
			return element.ID(BinaryToUUID(data))
		}
	case bsontype.String:
		s, _ := val.StringValueOK()
		return element.ID(s)
	}
	return ""
}

// BinaryToUUID converts a 16-byte binary to a UUID string using Microsoft GUID
// format (little-endian first 3 groups) to match Mendix standard representation.
func BinaryToUUID(data []byte) string {
	if len(data) != 16 {
		return ""
	}
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		data[3], data[2], data[1], data[0],
		data[5], data[4],
		data[7], data[6],
		data[8], data[9],
		data[10], data[11], data[12], data[13], data[14], data[15])
}
