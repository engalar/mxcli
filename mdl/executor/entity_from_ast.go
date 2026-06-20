// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"crypto/rand"
	"fmt"
	"math"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// buildEntityFromAST directly constructs a *genDm.Entity from a parsed
// CREATE ENTITY statement, injecting the invariants the write path requires:
//   - Boolean attributes without an explicit default get DefaultValue="false"
//   - AutoNumber attributes without an explicit seed get DefaultValue="1"
//   - Enum defaults are truncated to the trailing segment (module prefix dropped)
//
// It replaces the former canonical entity Lift+Persist write pipeline; the
// logic is ported verbatim from mdl/canonical/entity/persist.go:buildGenEntity
// and lift.go:liftDataType, but operates on the AST directly with no
// intermediate EntityModel.
func buildEntityFromAST(moduleName string, s *ast.CreateEntityStmt) (*genDm.Entity, error) {
	if s == nil {
		return nil, fmt.Errorf("buildEntityFromAST: nil statement")
	}

	e := genDm.NewEntity()
	e.SetName(s.Name.Name)
	if s.Documentation != "" {
		e.SetDocumentation(s.Documentation)
	}
	if s.Position != nil {
		// Mendix MPR format uses semicolon-separated coordinates ("X;Y").
		// Space-separated ("X Y") is rejected by Studio Pro 11.6.6+.
		e.SetLocation(fmt.Sprintf("%d;%d", s.Position.X, s.Position.Y))
	}

	// Generalization vs NoGeneralization.
	if s.Generalization != nil {
		g := genDm.NewGeneralization()
		g.SetGeneralizationQualifiedName(s.Generalization.String())
		e.SetGeneralization(g)
	} else {
		ng := genDm.NewNoGeneralization()
		ng.SetPersistable(s.Kind != ast.EntityNonPersistent)
		applySystemMembersFromSlice(ng, s.SystemMembers)
		e.SetGeneralization(ng)
	}

	// Entity qualified name for validation-rule attribute refs.
	entityQN := s.Name.String()

	// Attributes — assign IDs eagerly so Indexes can reference them.
	attrIDs := make(map[string]element.ID, len(s.Attributes))
	for _, a := range s.Attributes {
		dt := astDataTypeToCanonical(a.Type)
		genAttr, err := buildGenAttrFromAST(a, dt)
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", a.Name, err)
		}
		genAttr.SetID(newEntityElementID())
		attrIDs[a.Name] = genAttr.ID()
		e.AddAttributes(genAttr)
	}

	// ValidationRules from NotNull / Unique flags, with optional custom
	// error messages (single en_US translation, matching Studio Pro's
	// default for unilingual projects).
	for _, a := range s.Attributes {
		if a.NotNull {
			vr := genDm.NewValidationRule()
			vr.SetAttributeQualifiedName(entityQN + "." + a.Name)
			vr.SetRuleInfo(genDm.NewRequiredRuleInfo())
			if a.NotNullError != "" {
				vr.SetErrorMessage(buildENUSText(a.NotNullError))
			}
			e.AddValidationRules(vr)
		}
		if a.Unique {
			vr := genDm.NewValidationRule()
			vr.SetAttributeQualifiedName(entityQN + "." + a.Name)
			vr.SetRuleInfo(genDm.NewUniqueRuleInfo())
			if a.UniqueError != "" {
				vr.SetErrorMessage(buildENUSText(a.UniqueError))
			}
			e.AddValidationRules(vr)
		}
	}

	// Indexes — reference attribute IDs assigned above.
	for _, idx := range s.Indexes {
		gi := genDm.NewIndex()
		for _, col := range idx.Columns {
			ia := genDm.NewIndexedAttribute()
			if id, ok := attrIDs[col.Name]; ok {
				ia.SetAttributeID(id)
			}
			ia.SetAscending(!col.Descending)
			gi.AddAttributes(ia)
		}
		e.AddIndexes(gi)
	}

	// Event handlers.
	for _, eh := range s.EventHandlers {
		h := genDm.NewEventHandler()
		h.SetMoment(eh.Moment)
		h.SetEvent(eh.Event)
		h.SetMicroflowQualifiedName(eh.Microflow.String())
		h.SetRaiseErrorOnFalse(eh.RaiseErrorOnFalse)
		h.SetPassEventObject(eh.PassEventObject)
		e.AddEventHandlers(h)
	}

	return e, nil
}

// buildGenAttrFromAST converts an ast.Attribute (plus its already-lifted
// canonical DataType) into a fully wired *genDm.Attribute.
func buildGenAttrFromAST(a ast.Attribute, dt canonical.DataType) (*genDm.Attribute, error) {
	attr := genDm.NewAttribute()
	attr.SetName(a.Name)
	if a.Documentation != "" {
		attr.SetDocumentation(a.Documentation)
	}
	at, err := canonicalDataTypeToGenAttrType(dt)
	if err != nil {
		return nil, err
	}
	attr.SetType(at)

	if a.Calculated && a.CalculatedMicroflow != nil {
		cv := genDm.NewCalculatedValue()
		cv.SetMicroflowQualifiedName(a.CalculatedMicroflow.String())
		attr.SetValue(cv)
		return attr, nil
	}

	if a.HasDefault || dt.Kind == canonical.KindAutoNumber {
		sv := genDm.NewStoredValue()
		raw := ""
		if a.HasDefault {
			raw = stripAttrDefaultQuotes(formatEntityDefault(a.DefaultValue))
		}
		// AutoNumber attributes require a StoredValue with seed "1" when no
		// explicit seed is provided — Mendix enforces CE7247 "Value cannot be
		// empty" if the StoredValue is absent or has an empty DefaultValue.
		if dt.Kind == canonical.KindAutoNumber && raw == "" {
			raw = "1"
		}
		// Enum attributes: Mendix stores only the trailing value name in
		// StoredValue.DefaultValue (e.g. "Draft", not "Module.Enum.Draft").
		// The runtime constructs the full reference as EnumerationQN + "." +
		// DefaultValue; storing the full path causes CE1613 (double prefix).
		if dt.Kind == canonical.KindUnresolvedRef || dt.Kind == canonical.KindEnumRef {
			if i := strings.LastIndex(raw, "."); i >= 0 {
				raw = raw[i+1:]
			}
		}
		sv.SetDefaultValue(raw)
		attr.SetValue(sv)
	}

	return attr, nil
}

// canonicalDataTypeToGenAttrType maps a canonical DataType to its gen
// attribute-type constructor. KindUnresolvedRef is treated as enumeration —
// by the time the AST reaches the write path, bare references are expected to
// be enum types (the common case in domain-model statements).
func canonicalDataTypeToGenAttrType(dt canonical.DataType) (element.Element, error) {
	switch dt.Kind {
	case canonical.KindString:
		st := genDm.NewStringAttributeType()
		if dt.Length > 0 {
			if dt.Length > math.MaxInt32 {
				return nil, fmt.Errorf("String length %d exceeds int32 max", dt.Length)
			}
			st.SetLength(int32(dt.Length))
		} else {
			// Unlimited string: Mendix stores length=0, not -1.
			// CE0151 "Length should be >= 0" fires when length is negative.
			st.SetLength(0)
		}
		return st, nil
	case canonical.KindInteger:
		return genDm.NewIntegerAttributeType(), nil
	case canonical.KindLong:
		return genDm.NewLongAttributeType(), nil
	case canonical.KindDecimal:
		return genDm.NewDecimalAttributeType(), nil
	case canonical.KindBoolean:
		return genDm.NewBooleanAttributeType(), nil
	case canonical.KindDateTime:
		return genDm.NewDateTimeAttributeType(), nil
	case canonical.KindBinary:
		return genDm.NewBinaryAttributeType(), nil
	case canonical.KindAutoNumber:
		return genDm.NewAutoNumberAttributeType(), nil
	case canonical.KindEnumRef, canonical.KindUnresolvedRef:
		ea := genDm.NewEnumerationAttributeType()
		ea.SetEnumerationQualifiedName(dt.Ref)
		return ea, nil
	case canonical.KindEntityRef, canonical.KindListOf:
		return nil, fmt.Errorf("entity/list-of types not allowed as entity attributes")
	default:
		return nil, fmt.Errorf("unknown DataTypeKind %d", int(dt.Kind))
	}
}

// applySystemMembersFromSlice wires the four System.* presence bits on a
// NoGeneralization. All four flags are always explicitly Set (true or false) —
// Mendix expects every flag present in BSON; absent fields trigger CE0161 for
// XPath constraints on the entity. Names are matched case-insensitively against
// the standard owner/createdDate/changedDate/changedBy vocabulary; unknown
// names are silently ignored.
func applySystemMembersFromSlice(ng *genDm.NoGeneralization, members []string) {
	enabled := make(map[string]bool, len(members))
	for _, name := range members {
		enabled[strings.ToLower(strings.TrimSpace(name))] = true
	}
	ng.SetHasOwner(enabled["owner"])
	ng.SetHasChangedBy(enabled["changedby"])
	ng.SetHasCreatedDate(enabled["createddate"])
	ng.SetHasChangedDate(enabled["changeddate"])
}

// buildENUSText builds a single-translation Text (en_US) for entity validation
// error messages, matching Studio Pro's default for unilingual projects.
func buildENUSText(msg string) *genTexts.Text {
	t := genTexts.NewText()
	tr := genTexts.NewTranslation()
	tr.SetLanguageCode("en_US")
	tr.SetText(msg)
	t.AddTranslations(tr)
	return t
}

// stripAttrDefaultQuotes removes surrounding single quotes from string default
// values and un-doubles internal escapes ("it”s" -> "it's"). Other scalar
// default forms pass through unchanged.
func stripAttrDefaultQuotes(v string) string {
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		inner := v[1 : len(v)-1]
		return strings.ReplaceAll(inner, "''", "'")
	}
	return v
}

// newEntityElementID returns a random RFC 4122 v4 UUID as an element.ID,
// matching mdl/canonical/entity/persist.go:newElementID exactly.
func newEntityElementID() element.ID {
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
