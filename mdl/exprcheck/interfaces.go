// SPDX-License-Identifier: Apache-2.0

package exprcheck

type Context struct {
	SlotPath  string
	Microflow string
	File      string
	Line      int
	Column    int

	Scope   Scope
	Catalog CatalogReader
	Slots   SlotResolver
}

type Parser interface {
	Parse(source string, ctx Context) (RobustExpr, []Hint)
}

type SlotResolver interface {
	Expect(slotPath string) (SlotConstraint, bool)
}

type CatalogReader interface {
	AttributeKind(entityQN, attrName string) (TypeKind, bool)
	EnumCases(enumQN string) ([]string, bool)
	MicroflowReturn(qn string) (TypeKind, bool)
	MicroflowParam(qn, paramName string) (TypeKind, bool)
}

type HintSink interface {
	Emit(hints ...Hint)
}

type Scope interface {
	Lookup(name string) (TypeKind, bool)
}

type SlotConstraint struct {
	Kind      TypeKind
	ResolveBy string
	Frequency int
	Samples   []string
}

type TypeKind int

const (
	KindUnknown TypeKind = iota
	KindAny
	KindBoolean
	KindString
	KindInteger
	KindLong
	KindDecimal
	KindDateTime
	KindBinary
	KindObject
	KindList
	KindEnumeration
	KindEmpty
)

// Hint is now defined as a type alias to hints.Hint in hint.go (P1.6).
// RobustExpr lives in ast.go (P1.2).
