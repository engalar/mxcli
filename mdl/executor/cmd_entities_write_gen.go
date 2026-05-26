// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D1: AST → gen-typed entity / attribute / association
// builders. These mirror helpers.go::convertDataType and the entity /
// association construction in cmd_entities.go::execCreateEntity but
// produce *genDm.* values for use by the gen-typed write path
// (D2-D7 executors landing on top of this).

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// astToAttributeGen builds a gen-typed *Attribute from an AST
// AttributeDef. The Type element is set via astToAttributeTypeGen.
// CalculatedValue / StoredValue are wired when the AST signals
// a.Calculated / a.HasDefault. Note that resolving CalculatedMicroflow
// to an ID is the caller's responsibility — the gen CalculatedValue
// stores the qualified name, not the ID, so this builder uses the
// AST string directly.
func astToAttributeGen(a *ast.Attribute) *genDm.Attribute {
	if a == nil {
		return nil
	}
	attr := genDm.NewAttribute()
	attr.SetName(a.Name)
	attr.SetDocumentation(a.Documentation)
	attr.SetType(astToAttributeTypeGen(a.Type))

	if a.Calculated {
		cv := genDm.NewCalculatedValue()
		if a.CalculatedMicroflow != nil {
			cv.SetMicroflowQualifiedName(a.CalculatedMicroflow.String())
		}
		attr.SetValue(cv)
	} else if a.HasDefault {
		sv := genDm.NewStoredValue()
		sv.SetDefaultValue(astAttributeDefaultStringGen(a))
		attr.SetValue(sv)
	}
	return attr
}

// astAttributeDefaultStringGen renders an AST default value as the
// string the gen StoredValue.DefaultValue stores. For enum attributes
// Mendix stores just the value name (the trailing segment of the
// qualified default), matching legacy execCreateEntity behavior.
func astAttributeDefaultStringGen(a *ast.Attribute) string {
	if !a.HasDefault {
		return ""
	}
	def := stringifyDefault(a.DefaultValue)
	if a.Type.Kind == ast.TypeEnumeration {
		// Strip leading "Module.Enum." prefix if present so the BSON
		// stores just the value name (e.g. "Open" not "MyMod.Status.Open").
		if i := lastDotIndex(def); i >= 0 && i < len(def)-1 {
			def = def[i+1:]
		}
	}
	return def
}

// stringifyDefault converts the AST DefaultValue (any) to its string
// form using the same fmt-based rendering as legacy execCreateEntity.
func stringifyDefault(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	}
	return fmt.Sprintf("%v", v)
}

func lastDotIndex(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

// astToAttributeTypeGen mirrors convertDataType for the gen-typed write
// path. Returns the concrete *genDm.*AttributeType element matching the
// AST kind. Falls back to a 200-char String for unknown kinds (matches
// legacy behavior).
//
// Schema gap: gen has no separate DateAttributeType — both Date and
// DateTime map to DateTimeAttributeType. The legacy convertDataType
// distinguished via DateAttributeType{} (no LocalizeDate flag). Until
// the gen schema gains a Date discriminator, the gen builder maps
// TypeDate → DateTimeAttributeType{} and TypeDateTime →
// DateTimeAttributeType{LocalizeDate=true}. Round-trip with Studio Pro
// for purely-date columns may need post-write fix-up via raw BSON.
func astToAttributeTypeGen(dt ast.DataType) element.Element {
	switch dt.Kind {
	case ast.TypeString:
		s := genDm.NewStringAttributeType()
		if dt.Length > 0 {
			s.SetLength(int32(dt.Length))
		}
		return s
	case ast.TypeInteger:
		return genDm.NewIntegerAttributeType()
	case ast.TypeLong:
		return genDm.NewLongAttributeType()
	case ast.TypeDecimal:
		return genDm.NewDecimalAttributeType()
	case ast.TypeBoolean:
		return genDm.NewBooleanAttributeType()
	case ast.TypeDateTime:
		dtt := genDm.NewDateTimeAttributeType()
		dtt.SetLocalizeDate(true)
		return dtt
	case ast.TypeDate:
		// Schema gap (R7) — gen has no Date type; render as DateTime
		// with LocalizeDate=false so the BSON round-trips deterministic.
		return genDm.NewDateTimeAttributeType()
	case ast.TypeAutoNumber:
		return genDm.NewAutoNumberAttributeType()
	case ast.TypeBinary:
		return genDm.NewBinaryAttributeType()
	case ast.TypeEnumeration:
		e := genDm.NewEnumerationAttributeType()
		if dt.EnumRef != nil {
			e.SetEnumerationQualifiedName(dt.EnumRef.String())
		}
		return e
	default:
		s := genDm.NewStringAttributeType()
		s.SetLength(200)
		return s
	}
}
