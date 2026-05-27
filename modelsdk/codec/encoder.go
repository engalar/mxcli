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
// Dirty elements rebuild using buildDoc.
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

// rebuildEntry records which properties of an element need re-encoding.
type rebuildEntry struct {
	wp  element.WritableProperty
	cp  element.ChildProperty
	clp element.ChildListProperty
}

// buildDoc constructs a bson.D for a dirty element.
//
// Key optimization over the previous implementation: instead of calling
// bson.Unmarshal(raw, &bson.D) — which recursively decodes the entire BSON
// tree into Go objects (3 591 allocs for a 25 KB microflow) — we iterate the
// raw bytes with bson.Raw.Elements() and pass clean fields through as
// bson.RawValue (zero-alloc). Only fields that are actually dirty are decoded
// and re-encoded.
func (e *Encoder) buildDoc(elem element.Element) (bson.D, error) {
	// Build an index of properties that need rebuilding: dirty scalars,
	// dirty children, or child lists with at least one dirty member.
	rebuild := make(map[string]*rebuildEntry, len(elem.Properties()))
	for _, prop := range elem.Properties() {
		wp, ok := prop.(element.WritableProperty)
		if !ok {
			continue
		}
		cp, _ := prop.(element.ChildProperty)
		clp, _ := prop.(element.ChildListProperty)

		needsRebuild := wp.Dirty()
		if !needsRebuild {
			if cp != nil {
				ch := cp.ChildElement()
				needsRebuild = ch != nil && ch.IsDirty()
			} else if clp != nil {
				needsRebuild = anyChildDirty(clp)
			}
		}
		if needsRebuild {
			rebuild[prop.Name()] = &rebuildEntry{wp, cp, clp}
		}
	}

	raw := elem.Raw()

	// === New element (no raw bytes) ===
	// Iterate elem.Properties() — not the rebuild map — to preserve stable
	// field ordering. Go map iteration is non-deterministic; using the map
	// as the source of ordering would produce different BSON byte sequences
	// across runs (TestSerializeWorkflowActivityGen_RoundTripIsStable).
	if raw == nil {
		doc := bson.D{
			{Key: "$ID", Value: idToBinarySubtype0(elem.ID())},
			{Key: "$Type", Value: elem.TypeName()},
		}
		for _, prop := range elem.Properties() {
			rb, needsRebuild := rebuild[prop.Name()]
			if !needsRebuild {
				continue
			}
			val, err := e.encodeEntry(rb)
			if err != nil {
				return nil, err
			}
			if val != nil {
				doc = append(doc, bson.E{Key: prop.Name(), Value: val})
			}
		}
		return doc, nil
	}

	// === Existing element — iterate raw bytes, pass clean fields through ===
	rawElems, err := bson.Raw(raw).Elements()
	if err != nil {
		return nil, fmt.Errorf("read raw elements for %s: %w", elem.TypeName(), err)
	}

	doc := make(bson.D, 0, len(rawElems))
	for _, re := range rawElems {
		key := re.Key()
		rb, dirty := rebuild[key]
		if !dirty {
			// Clean field: pass through raw bytes without allocating any Go objects.
			doc = append(doc, bson.E{Key: key, Value: re.Value()})
			continue
		}
		// Dirty field: encode new value.
		val, err := e.encodeEntry(rb)
		if err != nil {
			return nil, err
		}
		if val != nil {
			doc = append(doc, bson.E{Key: key, Value: val})
		}
		// Mark handled so we don't append it again below.
		delete(rebuild, key)
	}

	// Append dirty fields that didn't exist in the raw bytes (new properties).
	// Iterate Properties() for stable ordering — not the rebuild map.
	for _, prop := range elem.Properties() {
		rb, ok := rebuild[prop.Name()]
		if !ok {
			continue
		}
		val, err := e.encodeEntry(rb)
		if err != nil {
			return nil, err
		}
		if val != nil {
			doc = append(doc, bson.E{Key: prop.Name(), Value: val})
		}
	}

	return doc, nil
}

// encodeEntry produces the BSON value for a single dirty property.
func (e *Encoder) encodeEntry(rb *rebuildEntry) (any, error) {
	wp := rb.wp

	// Child (Part) property.
	if rb.cp != nil {
		child := rb.cp.ChildElement()
		if wp.Dirty() {
			if child == nil {
				return nil, nil // deleted
			}
			return e.buildDoc(child)
		}
		// Child itself is dirty (parent pointer unchanged).
		if child != nil && child.IsDirty() {
			return e.buildDoc(child)
		}
		return nil, nil
	}

	// Child list (PartList) property.
	if rb.clp != nil {
		children := rb.clp.ChildElements()
		if wp.Dirty() {
			// Full rebuild: all children re-encoded.
			arr := make(bson.A, 0, 1+len(children))
			arr = append(arr, int32(3))
			for _, child := range children {
				childDoc, err := e.buildDoc(child)
				if err != nil {
					return nil, err
				}
				arr = append(arr, childDoc)
			}
			return arr, nil
		}
		// Selective rebuild: dirty children re-encoded, clean ones pass through raw bytes.
		arr := make(bson.A, 0, 1+len(children))
		arr = append(arr, int32(3))
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
		return arr, nil
	}

	// Scalar / Enum / Ref property.
	val := wp.BSONValue()
	if id, ok := val.(element.ID); ok && id != "" {
		return idToBinary(id), nil
	}
	return val, nil
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

// idToBinarySubtype0 converts a UUID string to BSON Binary subtype 0.
// When id is empty a fresh UUID is generated.
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
