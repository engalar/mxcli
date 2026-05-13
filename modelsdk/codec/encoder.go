package codec

import (
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/mpr"
	"go.mongodb.org/mongo-driver/bson"
)

// Encoder serializes Element trees back to BSON bytes.
type Encoder struct{}

// Encode serializes an element to []byte.
// Clean elements passthrough raw bytes unchanged.
// Dirty elements rebuild: start from raw fields, overlay dirty property values,
// recursively encode child elements.
func (e *Encoder) Encode(elem element.Element) ([]byte, error) {
	raw := elem.Raw()
	if raw != nil && !elem.IsDirty() {
		return []byte(raw), nil
	}

	doc, err := e.buildDoc(elem)
	if err != nil {
		return nil, err
	}
	return bson.Marshal(doc)
}

// buildDoc constructs a bson.D for an element, merging raw fields with dirty overrides.
func (e *Encoder) buildDoc(elem element.Element) (bson.D, error) {
	// Start from raw if available (preserves unknown fields).
	var doc bson.D
	if raw := elem.Raw(); raw != nil {
		if err := bson.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal raw for %s: %w", elem.TypeName(), err)
		}
	} else {
		// New element — start with identity fields.
		doc = bson.D{
			{Key: "$ID", Value: idToBinarySubtype0(elem.ID())},
			{Key: "$Type", Value: elem.TypeName()},
		}
	}

	// Overlay dirty properties.
	for _, prop := range elem.Properties() {
		wp, ok := prop.(element.WritableProperty)
		if !ok {
			continue
		}

		key := prop.Name()

		// Handle child properties (Part) — recursive encode.
		if cp, ok := prop.(element.ChildProperty); ok {
			if !wp.Dirty() {
				continue
			}
			child := cp.ChildElement()
			if child != nil {
				childDoc, err := e.buildDoc(child)
				if err != nil {
					return nil, err
				}
				doc = setField(doc, key, childDoc)
			} else {
				doc = setField(doc, key, nil)
			}
			continue
		}

		// Handle child list properties (PartList) — three branches:
		//   1. Self-dirty: list was modified (Append/Remove) → full rebuild.
		//   2. Child-dirty: list unchanged but a child was modified → selective rebuild.
		//   3. Completely clean → skip (don't touch the raw field).
		if clp, ok := prop.(element.ChildListProperty); ok {
			if wp.Dirty() {
				// Branch 1: full rebuild — all children re-encoded.
				children := clp.ChildElements()
				arr := bson.A{int32(3)}
				for _, child := range children {
					childDoc, err := e.buildDoc(child)
					if err != nil {
						return nil, err
					}
					arr = append(arr, childDoc)
				}
				doc = setField(doc, key, arr)
			} else if anyChildDirty(clp) {
				// Branch 2: selective rebuild — dirty children re-encoded, clean ones pass through raw bytes.
				children := clp.ChildElements()
				arr := bson.A{int32(3)}
				for _, child := range children {
					if child.IsDirty() {
						childDoc, err := e.buildDoc(child)
						if err != nil {
							return nil, err
						}
						arr = append(arr, childDoc)
					} else {
						arr = append(arr, bson.Raw(child.Raw()))
					}
				}
				doc = setField(doc, key, arr)
			}
			// Branch 3: completely clean — leave raw field untouched.
			continue
		}

		// Scalar/Enum/Ref properties — only process if dirty.
		if !wp.Dirty() {
			continue
		}

		// Scalar/Enum/Ref properties — use BSONValue directly.
		val := wp.BSONValue()
		// Convert element.ID to binary UUID for BSON compatibility.
		if id, ok := val.(element.ID); ok && id != "" {
			doc = setField(doc, key, idToBinary(id))
		} else {
			doc = setField(doc, key, val)
		}
	}

	return doc, nil
}

// anyChildDirty reports whether any element in the ChildListProperty is dirty.
func anyChildDirty(clp element.ChildListProperty) bool {
	for _, child := range clp.ChildElements() {
		if child.IsDirty() {
			return true
		}
	}
	return false
}

// setField replaces or appends a field in a bson.D.
func setField(doc bson.D, key string, val any) bson.D {
	for i, e := range doc {
		if e.Key == key {
			doc[i].Value = val
			return doc
		}
	}
	return append(doc, bson.E{Key: key, Value: val})
}

// idToBinarySubtype0 converts a UUID string to BSON Binary subtype 0
// using the Mendix byte-swap convention (via sdk/mpr.IDToBsonBinary).
// When id is empty (new element, no ID assigned), a fresh UUID is generated
// so the BSON always carries a valid Binary $ID.
func idToBinarySubtype0(id element.ID) any {
	if id == "" {
		return mpr.IDToBsonBinary(mpr.GenerateID())
	}
	return mpr.IDToBsonBinary(string(id))
}

// idToBinary converts a UUID string to Mendix BSON Binary format.
func idToBinary(id element.ID) any {
	return idToBinarySubtype0(id)
}
