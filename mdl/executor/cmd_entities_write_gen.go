// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D1: AST → gen-typed entity / attribute / association
// builders. These mirror helpers.go::convertDataType and the entity /
// association construction in cmd_entities.go::execCreateEntity but
// produce *genDm.* values for use by the gen-typed write path
// (D2-D7 executors landing on top of this).

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

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
