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

// RobustExpr is the parser output type. Concrete AST nodes are added
// in P1.2; for the P1.1 skeleton this is just a marker interface so
// the Parser interface signature compiles.
type RobustExpr interface {
	robustExpr()
}

// Hint is a single diagnostic with optional fix suggestion. Full
// definition (Code, Slug, Severity, Where, ...) lands in P1.6; the
// stub here is enough for the Parser/HintSink interface signatures
// to compile.
type Hint struct{}
