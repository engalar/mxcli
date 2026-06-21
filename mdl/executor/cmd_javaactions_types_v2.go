// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCA "github.com/mendixlabs/mxcli/modelsdk/gen/codeactions"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
)

// ─────────────────────────────────────────────────────────────────────
// A2 — formatJavaActionTypeGen + formatJavaActionReturnTypeGen
// ─────────────────────────────────────────────────────────────────────

// formatJavaActionTypeGen dispatches on elem.TypeName() to render the MDL
// type syntax for a gen-typed Java action parameter or return type.
//
// `typeParams` is the action's type-parameter def list (from
// ActionTypeParametersItems / TypeParametersItems) — needed to resolve
// EntityTypeParameterType.TypeParameterRefID and
// ParameterizedEntityType.TypeParameterRefID back to the displayed name
// (gen does not store the resolved name on the use site, only the BY_ID
// pointer).
//
// Accepts both "CodeActions$X" (Studio-Pro-emitted; primary) and
// "JavaActions$X" (gen-emitted; for elements created in this session)
// storage namespaces. See the schema-gap note at the top of this file.

// jaTypeNames maps SDK TypeName() values to their MDL string representations.
// Key is the TypeName() value; entries with multiple aliases share a handler.
type jaTypeHandler func(elem element.Element, typeParams []element.Element) string

var jaTypeDispatch map[string]jaTypeHandler

func init() {
	jaTypeDispatch = map[string]jaTypeHandler{
		"CodeActions$VoidType":         func(elem element.Element, typeParams []element.Element) string { return "Void" },
		"CodeActions$BooleanType":      func(elem element.Element, typeParams []element.Element) string { return "Boolean" },
		"JavaActions$BooleanType":      func(elem element.Element, typeParams []element.Element) string { return "Boolean" },
		"CodeActions$IntegerType":      func(elem element.Element, typeParams []element.Element) string { return "Integer" },
		"JavaActions$IntegerType":      func(elem element.Element, typeParams []element.Element) string { return "Integer" },
		"CodeActions$LongType":         func(elem element.Element, typeParams []element.Element) string { return "Long" },
		"CodeActions$DecimalType":      func(elem element.Element, typeParams []element.Element) string { return "Decimal" },
		"JavaActions$DecimalType":      func(elem element.Element, typeParams []element.Element) string { return "Decimal" },
		"CodeActions$StringType":       func(elem element.Element, typeParams []element.Element) string { return "String" },
		"JavaActions$StringType":       func(elem element.Element, typeParams []element.Element) string { return "String" },
		"CodeActions$DateTimeType":     func(elem element.Element, typeParams []element.Element) string { return "DateTime" },
		"JavaActions$DateTimeType":     func(elem element.Element, typeParams []element.Element) string { return "DateTime" },
		"CodeActions$FileDocumentType": func(elem element.Element, typeParams []element.Element) string { return "FileDocument" },

		"CodeActions$ConcreteEntityType": formatJAConcreteEntityType,
		"JavaActions$ConcreteEntityType": formatJAConcreteEntityType,
		"CodeActions$EntityType":         formatJAConcreteEntityType,

		"CodeActions$ListType": formatJAListType,
		"JavaActions$ListType": formatJAListType,

		"CodeActions$EnumerationType": formatJAEnumerationType,
		"JavaActions$EnumerationType": formatJAEnumerationType,
		"CodeActions$EntityTypeParameterType": func(elem element.Element, typeParams []element.Element) string {
			return formatJATypeParamEntity(elem, typeParams)
		},
		"JavaActions$EntityTypeParameterType": func(elem element.Element, typeParams []element.Element) string {
			return formatJATypeParamEntity(elem, typeParams)
		},
		"CodeActions$ParameterizedEntityType":                      formatJAParameterizedEntityType,
		"JavaActions$ParameterizedEntityType":                      formatJAParameterizedEntityType,
		"CodeActions$TypeParameter":                                formatJATypeParameter,
		"JavaActions$TypeParameter":                                formatJATypeParameter,
		"CodeActions$MicroflowType":                                func(elem element.Element, typeParams []element.Element) string { return "Microflow" },
		"JavaActions$MicroflowJavaActionParameterType":             func(elem element.Element, typeParams []element.Element) string { return "Microflow" },
		"JavaActions$MicroflowParameterType":                       func(elem element.Element, typeParams []element.Element) string { return "Microflow" },
		"CodeActions$NanoflowType":                                 func(elem element.Element, typeParams []element.Element) string { return "Nanoflow" },
		"JavaScriptActions$NanoflowJavaScriptActionParameterType":  func(elem element.Element, typeParams []element.Element) string { return "Nanoflow" },
		"JavaScriptActions$MicroflowJavaScriptActionParameterType": func(elem element.Element, typeParams []element.Element) string { return "Microflow" },
		"CodeActions$StringTemplateParameterType":                  formatJAStringTemplate,
		"CodeActions$BasicParameterType":                           formatJABasicParameterType,
		"JavaActions$BasicParameterType":                           formatJABasicParameterType,
	}
}

func formatJAConcreteEntityType(elem element.Element, typeParams []element.Element) string {
	if name := genJA.ReadBSONString(elem, "Entity"); name != "" {
		return name
	}
	if et, ok := elem.(*genJA.ConcreteEntityType); ok && et.EntityQualifiedName() != "" {
		return et.EntityQualifiedName()
	}
	return "Object"
}

func formatJAListType(elem element.Element, typeParams []element.Element) string {
	if lt, ok := elem.(*genJA.ListType); ok {
		if inner := lt.Parameter(); inner != nil {
			return "List of " + formatJavaActionTypeGen(inner, typeParams)
		}
	}
	if entity := genJA.ReadBSONString(elem, "Entity"); entity != "" {
		return "List of " + entity
	}
	if inner := genJA.DecodeChildElement(elem, "Parameter"); inner != nil {
		return "List of " + formatJavaActionTypeGen(inner, typeParams)
	}
	return "List"
}

func formatJAEnumerationType(elem element.Element, typeParams []element.Element) string {
	if et, ok := elem.(*genJA.EnumerationType); ok && et.EnumerationQualifiedName() != "" {
		return "Enum " + et.EnumerationQualifiedName()
	}
	if name := genJA.ReadBSONString(elem, "Enumeration"); name != "" {
		return "Enum " + name
	}
	return "Enumeration"
}

func formatJATypeParamEntity(elem element.Element, typeParams []element.Element) string {
	if name := resolveTypeParamNameFromEntityTypeParameterType(elem, typeParams); name != "" {
		return "entity <" + name + ">"
	}
	return "entity <>"
}

func formatJAParameterizedEntityType(elem element.Element, typeParams []element.Element) string {
	if name := resolveTypeParamNameFromParameterizedEntityType(elem, typeParams); name != "" {
		return name
	}
	return "T"
}

func formatJATypeParameter(elem element.Element, typeParams []element.Element) string {
	if tp, ok := elem.(*genJA.TypeParameter); ok && tp.Name() != "" {
		return tp.Name()
	}
	if name := genJA.ReadBSONString(elem, "TypeParameter"); name != "" {
		return name
	}
	return "T"
}

func formatJAStringTemplate(elem element.Element, typeParams []element.Element) string {
	if grammar := genJA.ReadBSONString(elem, "Grammar"); grammar != "" {
		return "StringTemplate(" + grammar + ")"
	}
	return "StringTemplate"
}

func formatJABasicParameterType(elem element.Element, typeParams []element.Element) string {
	if inner := genJA.DecodeChildElement(elem, "Type"); inner != nil {
		return formatJavaActionTypeGen(inner, typeParams)
	}
	return "Object"
}

func formatJavaActionTypeGen(elem element.Element, typeParams []element.Element) string {
	if elem == nil {
		return "Object"
	}
	if h, ok := jaTypeDispatch[elem.TypeName()]; ok {
		return h(elem, typeParams)
	}
	n := elem.TypeName()
	if i := strings.Index(n, "$"); i >= 0 {
		n = n[i+1:]
	}
	n = strings.TrimSuffix(n, "Type")
	if n == "" {
		return "Object"
	}
	return n
}

// formatJavaActionReturnTypeGen renders a return-type element. Identical
// dispatch as formatJavaActionTypeGen but treats nil as "Void" instead of
// "Object" because that's the convention for methods with no return.
func formatJavaActionReturnTypeGen(elem element.Element, typeParams []element.Element) string {
	if elem == nil {
		return "Void"
	}
	return formatJavaActionTypeGen(elem, typeParams)
}

// resolveTypeParamNameFromEntityTypeParameterType reads the
// TypeParameterPointer ID from elem and looks it up in the type-param
// def list. Works for both gen-typed and raw-fallback elements.
func resolveTypeParamNameFromEntityTypeParameterType(elem element.Element, typeParams []element.Element) string {
	if etp, ok := elem.(*genJA.EntityTypeParameterType); ok {
		if name := lookupTypeParamName(etp.TypeParameterRefID(), typeParams); name != "" {
			return name
		}
	}
	// Raw fallback: read the binary ID via the bsonutil helper would
	// require importing private decode logic; instead, fall back to
	// reading "TypeParameter" as a string (older Studio Pro variant)
	// or returning empty.
	if name := genJA.ReadBSONString(elem, "TypeParameter"); name != "" {
		return name
	}
	return ""
}

// resolveTypeParamNameFromParameterizedEntityType is the twin for
// ParameterizedEntityType (use-site referencing a type-param def by ID).
func resolveTypeParamNameFromParameterizedEntityType(elem element.Element, typeParams []element.Element) string {
	if pt, ok := elem.(*genJA.ParameterizedEntityType); ok {
		if name := lookupTypeParamName(pt.TypeParameterRefID(), typeParams); name != "" {
			return name
		}
	}
	if name := genJA.ReadBSONString(elem, "TypeParameter"); name != "" {
		return name
	}
	return ""
}

// lookupTypeParamName scans the def list for a TypeParameter whose ID
// matches `id`; returns its Name(). Empty string if not found or if id
// is empty.
func lookupTypeParamName(id element.ID, typeParams []element.Element) string {
	if id == "" {
		return ""
	}
	for _, tp := range typeParams {
		if tp == nil {
			continue
		}
		if typed, ok := tp.(*genJA.TypeParameter); ok && typed.ID() == id {
			return typed.Name()
		}
		// Unregistered raw element — match on raw $ID + read Name field.
		if tp.ID() == id {
			if name := genJA.ReadBSONString(tp, "Name"); name != "" {
				return name
			}
		}
	}
	return ""
}

// newGenElementByType allocates a bare *element.Base with the given
// storage TypeName. Used as a stub for gen schema gaps (LongType,
// VoidType, etc.) until codegen ships dedicated constructors. The
// resulting element has no properties wired so it can only carry its
// $Type marker on roundtrip — fine for JavaAction parameter / return
// types where the inner shape is empty (no nested fields).
//
// schema gap: Remove call sites once gen ships New<Name>Type for the
// missing CodeActions$ variants.
func newGenElementByType(name string) element.Element {
	b := &element.Base{}
	b.SetTypeName(name)
	return b
}

// ─────────────────────────────────────────────────────────────────────
// D1 — AST → gen JavaAction param/return type converters
// ─────────────────────────────────────────────────────────────────────

// astDataTypeToJavaActionParamTypeGen mirrors astDataTypeToJavaActionParamType
// (cmd_javaactions.go:455) but returns a gen element.Element instead of an
// sdk-typed CodeActionParameterType. typeParamIDs resolves type-parameter
// names (e.g., "pEntity") to the TypeParameter unit IDs created in the
// same execCreateJavaActionGen call so EntityTypeParameterType can carry
// the BY_ID pointer.
//
// schema gap: gen has no constructor for several CodeActions$ variants
// (LongType, FileDocumentType). Phase D code emits a plain *element.Base
// for those by allocating via element.New for the CodeActions$ namespace
// (Studio Pro's storage convention). The format helpers already dispatch
// on TypeName() for both namespaces (see schema-gap note (2) at top of
// file). Remove these element.New fallbacks when codegen ships them.
func astDataTypeToJavaActionParamTypeGen(dt ast.DataType, typeParamIDs map[string]element.ID) element.Element {
	switch dt.Kind {
	case ast.TypeBoolean:
		return genCA.NewBooleanType()
	case ast.TypeInteger:
		return genCA.NewIntegerType()
	case ast.TypeLong:
		return newGenElementByType("CodeActions$LongType")
	case ast.TypeDecimal:
		return genCA.NewDecimalType()
	case ast.TypeString:
		return genCA.NewStringType()
	case ast.TypeDateTime, ast.TypeDate:
		return genCA.NewDateTimeType()
	case ast.TypeEntityTypeParam:
		etp := genCA.NewEntityTypeParameterType()
		if id, ok := typeParamIDs[dt.TypeParamName]; ok {
			etp.SetTypeParameterID(id)
		}
		return etp
	case ast.TypeEntity, ast.TypeEnumeration:
		// Check first if this is a bare unqualified name that matches a type parameter
		// (e.g. InputObject: T where T is a declared type parameter).
		if dt.EnumRef != nil && dt.EnumRef.Module == "" {
			if id, ok := typeParamIDs[dt.EnumRef.Name]; ok {
				pe := genCA.NewParameterizedEntityType()
				pe.SetTypeParameterID(id)
				return pe
			}
		}
		// TypeEnumeration with a qualified name is treated as entity
		// type here; the visitor cannot distinguish entity from
		// enumeration types for bare qualified names like
		// Module.EntityName (see CLAUDE.md).
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		} else if dt.EnumRef != nil {
			entityName = dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
		et := genCA.NewConcreteEntityType()
		et.SetEntityQualifiedName(entityName)
		return et
	case ast.TypeListOf:
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		}
		inner := genCA.NewConcreteEntityType()
		inner.SetEntityQualifiedName(entityName)
		lt := genCA.NewListType()
		lt.SetParameter(inner)
		return lt
	default:
		return genJA.NewStringType()
	}
}

// astDataTypeToJavaActionReturnTypeGen converts an AST DataType to a gen
// element.Element for use as a Java action return type. Twin of
// astDataTypeToJavaActionParamTypeGen with TypeVoid → CodeActions$VoidType
// fallback (gen has no NewVoidType()).
//
// "void" sometimes parses as a bare TypeEnumeration with a single-part
// EnumRef name (e.g., {Module: "", Name: "void"}); we detect that
// pattern and emit VoidType to match the legacy converter behaviour.
func astDataTypeToJavaActionReturnTypeGen(dt ast.DataType, typeParamIDs map[string]element.ID) element.Element {
	if dt.Kind == ast.TypeVoid {
		// schema gap: no NewVoidType() in gen. Allocate a bare
		// *element.Base with the CodeActions$VoidType storage
		// namespace. Remove when gen ships NewVoidType().
		return newGenElementByType("CodeActions$VoidType")
	}
	// Detect "void" parsed as a bare TypeEnumeration / TypeEntity name.
	if dt.Kind == ast.TypeEntity || dt.Kind == ast.TypeEnumeration {
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		} else if dt.EnumRef != nil {
			entityName = dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
		if strings.EqualFold(strings.TrimPrefix(entityName, "."), "void") {
			return newGenElementByType("CodeActions$VoidType")
		}
	}
	return astDataTypeToJavaActionParamTypeGen(dt, typeParamIDs)
}

// isTypeParamRef checks whether an ast.DataType refers to a type
// parameter by name. Used by the AST→gen converters when resolving
// EntityTypeParameterType references against an action's type parameter list.
func isTypeParamRef(dt ast.DataType, typeParamNames map[string]bool) bool {
	name := getTypeParamRefName(dt)
	return name != "" && typeParamNames[name]
}

// getTypeParamRefName extracts the name from a DataType that could be a
// type parameter reference. Returns empty string if not a type-param ref.
func getTypeParamRefName(dt ast.DataType) string {
	switch dt.Kind {
	case ast.TypeEnumeration:
		if dt.EnumRef != nil && dt.EnumRef.Module == "" {
			return dt.EnumRef.Name
		}
		if dt.EnumRef != nil {
			return dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
	case ast.TypeEntity:
		if dt.EntityRef != nil && dt.EntityRef.Module == "" {
			return dt.EntityRef.Name
		}
		if dt.EntityRef != nil {
			return dt.EntityRef.Module + "." + dt.EntityRef.Name
		}
	}
	return ""
}
