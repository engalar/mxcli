// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
)

// Lift converts an ast.CreateEntityStmt into the canonical EntityModel.
func Lift(s *ast.CreateEntityStmt) (*EntityModel, error) {
	if s == nil {
		return nil, fmt.Errorf("entity.Lift: nil statement")
	}
	m := &EntityModel{
		Name:          QualifiedName{Module: s.Name.Module, Name: s.Name.Name},
		Kind:          liftKind(s.Kind),
		Documentation: s.Documentation,
		SystemMembers: append([]string(nil), s.SystemMembers...),
	}
	if s.Position != nil {
		m.Position = &Position{X: s.Position.X, Y: s.Position.Y}
	}
	if s.Generalization != nil {
		qn := QualifiedName{Module: s.Generalization.Module, Name: s.Generalization.Name}
		m.Extends = &qn
	}
	for _, a := range s.Attributes {
		m.Attributes = append(m.Attributes, liftAttribute(a))
	}
	for _, idx := range s.Indexes {
		m.Indexes = append(m.Indexes, liftIndex(idx))
	}
	for _, eh := range s.EventHandlers {
		m.EventHandlers = append(m.EventHandlers, EventHandlerModel{
			Moment:            eh.Moment,
			Event:             eh.Event,
			Microflow:         QualifiedName{Module: eh.Microflow.Module, Name: eh.Microflow.Name},
			RaiseErrorOnFalse: eh.RaiseErrorOnFalse,
			PassEventObject:   eh.PassEventObject,
		})
	}
	return m, nil
}

func liftKind(k ast.EntityKind) EntityKind {
	switch k {
	case ast.EntityNonPersistent:
		return EntityNonPersistent
	case ast.EntityView:
		return EntityView
	case ast.EntityExternal:
		return EntityExternal
	default:
		return EntityPersistent
	}
}

func liftAttribute(a ast.Attribute) AttributeModel {
	attr := AttributeModel{
		Name:          a.Name,
		Type:          liftDataType(a.Type),
		Documentation: a.Documentation,
		NotNull:       a.NotNull,
		NotNullError:  a.NotNullError,
		Unique:        a.Unique,
		UniqueError:   a.UniqueError,
		HasDefault:    a.HasDefault,
		Calculated:    a.Calculated,
	}
	if a.HasDefault {
		attr.DefaultValue = formatDefault(a.DefaultValue)
	}
	if a.CalculatedMicroflow != nil {
		qn := QualifiedName{Module: a.CalculatedMicroflow.Module, Name: a.CalculatedMicroflow.Name}
		attr.CalculatedMicroflow = &qn
	}
	return attr
}

func liftIndex(idx ast.Index) IndexModel {
	im := IndexModel{}
	for _, col := range idx.Columns {
		im.Columns = append(im.Columns, IndexColumn{Name: col.Name, Ascending: !col.Descending})
	}
	return im
}

// liftDataType collapses TypeEnumeration and TypeEntity into KindUnresolvedRef
// when only a bare qualified name is available — the parser cannot tell them
// apart without project context. Callers that already know the kind (e.g.,
// ast.TypeEntity for microflow parameters) get the strongly-typed kind.
func liftDataType(dt ast.DataType) canonical.DataType {
	switch dt.Kind {
	case ast.TypeString:
		return canonical.DataType{Kind: canonical.KindString, Length: dt.Length}
	case ast.TypeInteger:
		return canonical.DataType{Kind: canonical.KindInteger}
	case ast.TypeLong:
		return canonical.DataType{Kind: canonical.KindLong}
	case ast.TypeDecimal:
		return canonical.DataType{Kind: canonical.KindDecimal, Precision: dt.Precision, Scale: dt.Scale}
	case ast.TypeBoolean:
		return canonical.DataType{Kind: canonical.KindBoolean}
	case ast.TypeDateTime, ast.TypeDate:
		return canonical.DataType{Kind: canonical.KindDateTime}
	case ast.TypeBinary:
		return canonical.DataType{Kind: canonical.KindBinary}
	case ast.TypeAutoNumber:
		return canonical.DataType{Kind: canonical.KindAutoNumber}
	case ast.TypeEnumeration:
		ref := ""
		if dt.EnumRef != nil {
			ref = dt.EnumRef.String()
		}
		return canonical.DataType{Kind: canonical.KindUnresolvedRef, Ref: ref}
	case ast.TypeEntity:
		ref := ""
		if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return canonical.DataType{Kind: canonical.KindEntityRef, Ref: ref}
	case ast.TypeListOf:
		ref := ""
		if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return canonical.DataType{Kind: canonical.KindListOf, Ref: ref}
	default:
		return canonical.DataType{Kind: canonical.KindUnknown}
	}
}

// formatDefault renders an attribute default value for MDL serialisation.
// Strings are emitted with single-quote escaping; everything else gets the
// default %v representation.
func formatDefault(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	default:
		return fmt.Sprintf("%v", x)
	}
}
