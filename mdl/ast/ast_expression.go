// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Microflow Expressions
// ============================================================================

// Expression represents any expression in a microflow.
type Expression interface {
	isExpression()
}

// LiteralKind represents the kind of literal value.
type LiteralKind int

const (
	LiteralString LiteralKind = iota
	LiteralInteger
	LiteralDecimal
	LiteralBoolean
	LiteralNull
	LiteralEmpty
)

// LiteralExpr represents a literal value: 'string', 123, true, empty
type LiteralExpr struct {
	Value any         // The literal value
	Kind  LiteralKind // The kind of literal
}

func (e *LiteralExpr) isExpression() {}

// VariableExpr represents a variable reference: $VariableName
type VariableExpr struct {
	Name string // Variable name (without $ prefix)
}

func (e *VariableExpr) isExpression() {}

// PathSegment represents a single segment in an attribute path with its separator.
type PathSegment struct {
	Name      string // Segment name (attribute name, association qualified name)
	Separator string // "/" for association traversal, "." for attribute access
}

// AttributePathExpr represents: $Var/Attr or $Var/Module.Association/Attr
type AttributePathExpr struct {
	Variable string        // Base variable name
	Path     []string      // Path segments (attribute names, associations) - legacy flat list
	Segments []PathSegment // Path segments with separator info (/ vs .)
}

func (e *AttributePathExpr) isExpression() {}

// BinaryExpr represents: left op right
type BinaryExpr struct {
	Left     Expression // Left operand
	Operator string     // Operator (+, -, *, div, AND, OR, =, <>, <, >, etc.)
	Right    Expression // Right operand
}

func (e *BinaryExpr) isExpression() {}

// UnaryExpr represents: op expr (NOT, -)
type UnaryExpr struct {
	Operator string     // Operator (NOT, -)
	Operand  Expression // Operand
}

func (e *UnaryExpr) isExpression() {}

// FunctionCallExpr represents: functionName(args)
type FunctionCallExpr struct {
	Name      string       // Function name
	Arguments []Expression // Arguments
}

func (e *FunctionCallExpr) isExpression() {}

// TokenExpr represents: [%TokenName%] (e.g., [%CurrentDateTime%])
type TokenExpr struct {
	Token string // Token name
}

func (e *TokenExpr) isExpression() {}

// ParenExpr represents: (expr)
type ParenExpr struct {
	Inner Expression // Inner expression
}

func (e *ParenExpr) isExpression() {}

// IdentifierExpr represents an unquoted identifier (e.g., attribute name in XPath: IsActive, Price)
// This is different from LiteralExpr which represents quoted strings.
type IdentifierExpr struct {
	Name string // Identifier name (unquoted)
}

func (e *IdentifierExpr) isExpression() {}

// QualifiedNameExpr represents a qualified name (e.g., Module.Entity, Module.Association)
// Used for association names in WHERE clauses which should not be quoted.
type QualifiedNameExpr struct {
	QualifiedName QualifiedName // The qualified name (Module.Name)
}

func (e *QualifiedNameExpr) isExpression() {}

// ConstantRefExpr represents a constant reference: @Module.ConstantName
type ConstantRefExpr struct {
	QualifiedName QualifiedName // The constant qualified name
}

func (e *ConstantRefExpr) isExpression() {}

// IfThenElseExpr represents an inline if-then-else expression:
// if condition then trueExpr else falseExpr
type IfThenElseExpr struct {
	Condition Expression // Condition expression
	ThenExpr  Expression // Expression when condition is true
	ElseExpr  Expression // Expression when condition is false
}

func (e *IfThenElseExpr) isExpression() {}

// SourceExpr preserves original expression source text while keeping the parsed
// expression tree available for callers that need semantic inspection.
type SourceExpr struct {
	Expression Expression
	Source     string
}

func (e *SourceExpr) isExpression() {}

// ============================================================================
// XPath-Specific Expression Types
// ============================================================================

// XPathPathExpr represents an XPath path with multiple steps and/or nested predicates.
// Examples: Module.Assoc/Entity/Attr, $var/Assoc[Active]/Attr, System.roles[reversed()]
// For single-step paths without predicates, the underlying expression type
// (IdentifierExpr, QualifiedNameExpr, VariableExpr, etc.) is used directly instead.
type XPathPathExpr struct {
	Steps []XPathStep
}

func (e *XPathPathExpr) isExpression() {}

// XPathStep represents a single step in an XPath path expression.
type XPathStep struct {
	Expr      Expression // The step expression (IdentifierExpr, QualifiedNameExpr, VariableExpr, LiteralExpr, TokenExpr)
	Predicate Expression // Optional nested predicate expression (the content inside [...])
}

// Unwrap returns the inner Expression if e is a *SourceExpr, otherwise e
// itself. Callers that do type assertions on Expression nodes should run the
// value through Unwrap first to remain robust against the visitor's source-
// preservation wrapper added in P1.11.
func Unwrap(e Expression) Expression {
	if se, ok := e.(*SourceExpr); ok {
		return se.Expression
	}
	return e
}
