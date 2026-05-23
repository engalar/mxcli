// SPDX-License-Identifier: Apache-2.0

package model

// DataTypeKind enumerates the canonical attribute / parameter type kinds.
//
// Lift collapses ast.TypeEnumeration and ast.TypeEntity (which the parser
// cannot disambiguate without project context) into KindUnresolvedRef; later
// resolution stages — or Persist — promote it to KindEnumRef or KindEntityRef
// once a registry / catalog is consulted.
type DataTypeKind int

const (
	KindUnknown DataTypeKind = iota
	KindString
	KindInteger
	KindLong
	KindDecimal
	KindBoolean
	KindDateTime
	KindBinary
	KindAutoNumber
	KindEnumRef
	KindEntityRef
	KindListOf
	KindUnresolvedRef
)

// DataType is the canonical representation of an attribute / parameter type.
type DataType struct {
	Kind      DataTypeKind
	Length    int    // For String(length); 0 means unbounded.
	Precision int    // For Decimal(p, s).
	Scale     int    // For Decimal(p, s).
	Ref       string // Qualified name for Enum/Entity/List-of refs.
}
