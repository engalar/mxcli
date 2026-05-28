// SPDX-License-Identifier: Apache-2.0

package exprcheck

// Context carries per-call metadata into the parser.
//
// Two usage modes:
//   - Syntax-only: Catalog and Slots are nil; semantic hint rules (E001, E002,
//     E009, E011, …) are silently skipped. Use NewSyntaxContext.
//   - Semantic: Catalog and Slots are non-nil; all hint rules are active.
//     Use NewSemanticContext.
//
// Scope is always optional: when nil, variable type inference falls back to KindUnknown.
type Context struct {
	SlotPath  string
	Microflow string
	File      string
	Line      int
	Column    int

	Scope   Scope
	Catalog CatalogReader // nil → semantic checks disabled
	Slots   SlotResolver  // nil → slot-kind checks disabled
}

// IsSemanticEnabled reports whether Catalog and Slots are both wired so that
// semantic hint rules can run. Callers can gate expensive lookups behind this
// check instead of repeating nil guards.
func (c Context) IsSemanticEnabled() bool {
	return c.Catalog != nil && c.Slots != nil
}

// NewSyntaxContext creates a Context for syntax-only parsing (E003, E007, E011
// structurally wired; semantic rules that need Catalog/Slots are inactive).
func NewSyntaxContext(slotPath, microflow string) Context {
	return Context{SlotPath: slotPath, Microflow: microflow}
}

// NewSemanticContext creates a Context with full semantic checking enabled.
// Both slots and catalog must be non-nil.
func NewSemanticContext(slotPath, microflow string, slots SlotResolver, catalog CatalogReader) Context {
	return Context{
		SlotPath:  slotPath,
		Microflow: microflow,
		Slots:     slots,
		Catalog:   catalog,
	}
}

type Parser interface {
	Parse(source string, ctx Context) (RobustExpr, []Hint)
}

type SlotResolver interface {
	Expect(slotPath string) (SlotConstraint, bool)
}

type CatalogReader interface {
	AttributeKind(entityQN, attrName string) (TypeKind, bool)
	AttributeEnumQN(entityQN, attrName string) (string, bool)
	EnumCases(enumQN string) ([]string, bool)
	MicroflowReturn(qn string) (TypeKind, bool)
	MicroflowParam(qn, paramName string) (TypeKind, bool)
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
