// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

func singleENUSText(msg string) *genTexts.Text {
	t := genTexts.NewText()
	tr := genTexts.NewTranslation()
	tr.SetLanguageCode("en_US")
	tr.SetText(msg)
	t.AddTranslations(tr)
	return t
}

// Persist serialises the canonical EntityModel to gen-typed BSON structures
// and writes them to the project via ctx.Backend. A zero ExistingEntityID
// triggers CreateEntityGen; a non-zero one triggers UpdateEntityGen.
//
// The gen entity is built from scratch on every call — Persist does not
// merge with whatever the backend currently holds. Callers that need
// merge-semantics should Hydrate, mutate the EntityModel, then Persist.
func (m *EntityModel) Persist(ctx model.PersistContext) error {
	if m == nil {
		return fmt.Errorf("entity.Persist: nil model")
	}
	if ctx.Backend == nil {
		return fmt.Errorf("entity.Persist: nil backend")
	}
	if ctx.DomainModelID == "" {
		return fmt.Errorf("entity.Persist: missing DomainModelID")
	}
	gen, err := buildGenEntity(m)
	if err != nil {
		return fmt.Errorf("entity.Persist: %w", err)
	}
	if ctx.ExistingEntityID != "" {
		gen.SetID(element.ID(ctx.ExistingEntityID))
		return ctx.Backend.UpdateEntityGen(ctx.DomainModelID, gen)
	}
	return ctx.Backend.CreateEntityGen(ctx.DomainModelID, gen)
}

// buildGenEntity converts an EntityModel to a fully wired *genDm.Entity.
// Attribute IDs are pre-assigned so that IndexedAttribute refs can resolve.
func buildGenEntity(m *EntityModel) (*genDm.Entity, error) {
	e := genDm.NewEntity()
	e.SetName(m.Name.Name)
	if m.Documentation != "" {
		e.SetDocumentation(m.Documentation)
	}
	if m.Position != nil {
		e.SetLocation(fmt.Sprintf("%d %d", m.Position.X, m.Position.Y))
	}

	// Generalization vs NoGeneralization.
	if m.Extends != nil {
		g := genDm.NewGeneralization()
		g.SetGeneralizationQualifiedName(m.Extends.String())
		e.SetGeneralization(g)
	} else {
		ng := genDm.NewNoGeneralization()
		ng.SetPersistable(m.Kind != EntityNonPersistent)
		applySystemMembers(ng, m.SystemMembers)
		e.SetGeneralization(ng)
	}

	// Attributes — assign IDs eagerly so Indexes can reference them.
	attrIDs := make(map[string]element.ID, len(m.Attributes))
	for _, am := range m.Attributes {
		genAttr, err := buildGenAttribute(am)
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", am.Name, err)
		}
		genAttr.SetID(newElementID())
		attrIDs[am.Name] = genAttr.ID()
		e.AddAttributes(genAttr)
	}

	// ValidationRules from NotNull / Unique flags, with optional custom
	// error messages (single en_US translation, matching Studio Pro's
	// default for unilingual projects).
	entityQN := m.Name.String()
	for _, am := range m.Attributes {
		if am.NotNull {
			vr := genDm.NewValidationRule()
			vr.SetAttributeQualifiedName(entityQN + "." + am.Name)
			vr.SetRuleInfo(genDm.NewRequiredRuleInfo())
			if am.NotNullError != "" {
				vr.SetErrorMessage(singleENUSText(am.NotNullError))
			}
			e.AddValidationRules(vr)
		}
		if am.Unique {
			vr := genDm.NewValidationRule()
			vr.SetAttributeQualifiedName(entityQN + "." + am.Name)
			vr.SetRuleInfo(genDm.NewUniqueRuleInfo())
			if am.UniqueError != "" {
				vr.SetErrorMessage(singleENUSText(am.UniqueError))
			}
			e.AddValidationRules(vr)
		}
	}

	// Indexes — reference attribute IDs assigned above.
	for _, im := range m.Indexes {
		idx := genDm.NewIndex()
		for _, col := range im.Columns {
			ia := genDm.NewIndexedAttribute()
			if id, ok := attrIDs[col.Name]; ok {
				ia.SetAttributeID(id)
			}
			ia.SetAscending(col.Ascending)
			idx.AddAttributes(ia)
		}
		e.AddIndexes(idx)
	}

	return e, nil
}

func buildGenAttribute(am AttributeModel) (*genDm.Attribute, error) {
	attr := genDm.NewAttribute()
	attr.SetName(am.Name)
	if am.Documentation != "" {
		attr.SetDocumentation(am.Documentation)
	}
	at, err := buildGenAttributeType(am.Type)
	if err != nil {
		return nil, err
	}
	attr.SetType(at)
	if am.HasDefault {
		sv := genDm.NewStoredValue()
		sv.SetDefaultValue(stripStringLiteralQuotes(am.DefaultValue))
		attr.SetValue(sv)
	}
	return attr, nil
}

// buildGenAttributeType maps a canonical DataType to its gen attribute-type
// constructor. KindUnresolvedRef is treated as enumeration — by the time the
// canonical model reaches Persist, the executor is expected to have resolved
// bare references against the catalog; if not, we fall back to enum (the
// common case in domain-model statements).
func buildGenAttributeType(dt model.DataType) (element.Element, error) {
	switch dt.Kind {
	case model.KindString:
		st := genDm.NewStringAttributeType()
		if dt.Length > 0 {
			st.SetLength(int32(dt.Length))
		}
		return st, nil
	case model.KindInteger:
		return genDm.NewIntegerAttributeType(), nil
	case model.KindLong:
		return genDm.NewLongAttributeType(), nil
	case model.KindDecimal:
		return genDm.NewDecimalAttributeType(), nil
	case model.KindBoolean:
		return genDm.NewBooleanAttributeType(), nil
	case model.KindDateTime:
		return genDm.NewDateTimeAttributeType(), nil
	case model.KindBinary:
		return genDm.NewBinaryAttributeType(), nil
	case model.KindAutoNumber:
		return genDm.NewAutoNumberAttributeType(), nil
	case model.KindEnumRef, model.KindUnresolvedRef:
		ea := genDm.NewEnumerationAttributeType()
		ea.SetEnumerationQualifiedName(dt.Ref)
		return ea, nil
	case model.KindEntityRef, model.KindListOf:
		return nil, fmt.Errorf("entity/list-of types not allowed as entity attributes")
	default:
		return nil, fmt.Errorf("unknown DataTypeKind %d", int(dt.Kind))
	}
}

// applySystemMembers wires the four System.* presence bits on a
// NoGeneralization. All four flags are always explicitly Set (true or
// false) — Mendix expects every flag present in BSON; absent fields
// trigger CE0161 for XPath constraints on the entity. Names are matched
// case-insensitively against the standard owner/createdDate/changedDate/
// changedBy vocabulary; unknown names are silently ignored.
func applySystemMembers(ng *genDm.NoGeneralization, members []string) {
	enabled := make(map[string]bool, len(members))
	for _, name := range members {
		enabled[strings.ToLower(strings.TrimSpace(name))] = true
	}
	ng.SetHasOwner(enabled["owner"])
	ng.SetHasChangedBy(enabled["changedby"])
	ng.SetHasCreatedDate(enabled["createddate"])
	ng.SetHasChangedDate(enabled["changeddate"])
}

// stripStringLiteralQuotes removes surrounding single quotes from string
// default values and un-doubles internal escapes ("it''s" -> "it's"). Other
// scalar default forms pass through unchanged.
func stripStringLiteralQuotes(v string) string {
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		inner := v[1 : len(v)-1]
		return strings.ReplaceAll(inner, "''", "'")
	}
	return v
}

// newElementID returns a random RFC 4122 v4 UUID as an element.ID.
func newElementID() element.ID {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure on Linux is effectively impossible; if it does
		// happen, return a zero ID — the backend will surface the error when
		// it tries to assign / look up the entity.
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return element.ID(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}
