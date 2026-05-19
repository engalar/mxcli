// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f0 — raw-BSON setter helper for unmodeled gen fields.
//
// `setRawBSONField` is the foundation helper for injecting a single
// BSON key/value pair onto a gen element whose schema doesn't expose
// a typed setter. The motivating example is `*genMf.CastAction.
// ObjectVariableName`: `cmd_microflows_format_data_gen.go` reads it
// via `codec.ReadBSONFieldString(o.Raw(), "ObjectVariableName")` —
// proof that the BSON field exists in real fixtures — but the gen
// codegen didn't emit a Go setter for it.
//
// How the injection works:
//
//   - element.Base.AddProperty (modelsdk/element/element.go:138) is
//     officially intended for this use case (its doc string reads
//     "Use this to inject inherited or ad-hoc properties that the
//     codegen doesn't produce, e.g. Document.Name on Microflow").
//   - We construct a fresh `*property.Primitive[T]` with the supplied
//     value pre-Set (so it reports Dirty=true).
//   - The encoder (`modelsdk/codec/encoder.go:48`) iterates
//     `elem.Properties()` and calls `setField(doc, key, val)` for
//     every dirty WritableProperty, which means our injected entry
//     lands in the output BSON document.
//
// Bit assignment for ad-hoc properties: bit 62 (codegen never uses
// this slot — it goes 0..N for typed properties and bit 63 is reserved
// for "is new").
//
// No-op cases:
//
//   - nil element: defensive; caller errors stay caught at the call
//     site rather than panicking deep in the codec.
//   - nil value: BSON null isn't equivalent to "absent field" in
//     Mendix BSON, so we skip rather than emit a stray null.
//   - empty string: Mendix BSON omits empty optional strings, so
//     emitting one would create a stray empty key.
//   - elements that don't expose AddProperty (gen elements all do,
//     but a defensive type-assert keeps custom Element implementations
//     from panicking).
//
// Schema-gap tracking: every call site that uses this helper should
// document the `<Action type>.<field name>` pair in a single TODO
// comment so the eventual codegen / supplements pass can find them.

package executor

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	msdkprop "github.com/mendixlabs/mxcli/modelsdk/property"
)

// rawSetterAdhocBit is the dirty-bit slot reserved for ad-hoc
// properties injected by setRawBSONField. Codegen never emits a
// Property bound to this bit (it goes 0..N for typed properties; bit
// 63 is reserved for the "is new element" marker), so collisions are
// impossible.
const rawSetterAdhocBit uint = 62

// setRawBSONField injects a BSON key/value pair onto a gen element
// for fields that the codegen didn't produce a typed setter for. See
// the package comment above for the motivation and behaviour rules.
//
// `value` is `any` so callers can hand it strings, ints, bools, etc.
// without per-type wrappers — the type is preserved through the
// encoder's `setField` (modelsdk/codec/encoder.go:152) which writes
// the value verbatim into the BSON document.
//
// The supported value types in this commit:
//
//   - string  — the common case (CastAction.ObjectVariableName, etc.)
//   - int / int32 / int64 — for numeric fields without typed setters
//   - bool — for boolean fields without typed setters
//
// Any other value type is silently dropped (no-op). The unsupported
// path is intentional: callers should know what type they're emitting,
// and a typed branch lets the test suite exercise each path in
// isolation.
func setRawBSONField(elem element.Element, key string, value any) {
	if elem == nil || value == nil {
		return
	}
	holder, ok := elem.(genBaseHolder)
	if !ok {
		return
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return
		}
		// String fields go through the typed property.Primitive path
		// so the schema-aware codec retains future-proof typing
		// (a later codegen pass can promote the ad-hoc property to
		// a generated one with no behaviour change).
		p := msdkprop.NewPrimitive[string](key, msdkprop.DecodeString)
		p.Set(v)
		holder.AddProperty(p, rawSetterAdhocBit)
	case int, int32, int64, bool:
		// Numeric and boolean fields use the simpler ad-hoc property
		// implementation; BSONValue() returns the value verbatim so
		// the codec writes whatever type the caller passed.
		holder.AddProperty(adhocAnyProperty{key: key, value: v}, rawSetterAdhocBit)
	}
}

// adhocAnyProperty is the minimal Property implementation used for
// non-string ad-hoc fields. Carries the value verbatim through to
// BSONValue() so the codec writes it unchanged. Always reports Dirty.
type adhocAnyProperty struct {
	key   string
	value any
}

func (p adhocAnyProperty) Name() string                     { return p.key }
func (p adhocAnyProperty) Init(b []byte)                    {}
func (p adhocAnyProperty) BSONValue() any                   { return p.value }
func (p adhocAnyProperty) Dirty() bool                      { return true }
func (p adhocAnyProperty) Bind(owner *element.Base, _ uint) {}

// setRawBSONChildElement encodes a child element and injects it under a custom
// BSON key on the parent element. This is needed when the gen-codegen writes a
// child under the wrong BSON key (e.g. "LayoutCall" instead of Mendix's historic
// "FormCall"). The encoded child BSON document is injected as an adhocAnyProperty
// so the codec writes it verbatim under the given key.
//
// Typical use: page.SetLayoutCall would write "LayoutCall"; use this instead to
// write under "FormCall" which is the key Mendix Studio Pro uses and expects.
func setRawBSONChildElement(parent element.Element, key string, child element.Element) {
	if parent == nil || child == nil {
		return
	}
	holder, ok := parent.(genBaseHolder)
	if !ok {
		return
	}

	enc := codec.Encoder{}
	childBytes, err := enc.Encode(child)
	if err != nil {
		return
	}

	var childDoc bson.D
	if err := bson.Unmarshal(childBytes, &childDoc); err != nil {
		return
	}

	holder.AddProperty(adhocAnyProperty{key: key, value: childDoc}, rawSetterAdhocBit)
}
