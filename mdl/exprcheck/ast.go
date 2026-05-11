// SPDX-License-Identifier: Apache-2.0

package exprcheck

type Position struct {
	Offset int
	Line   int
	Column int
}

type RobustExpr interface {
	isRobustExpr()
	Pos() Position
}

type baseNode struct{ P Position }

func (b baseNode) Pos() Position { return b.P }

type StringLit struct {
	baseNode
	Value string
}

type NumberLit struct {
	baseNode
	Value string
	Kind  TypeKind
}

type BoolLit struct {
	baseNode
	Value bool
}

type EmptyExpr struct{ baseNode }

type VariableExpr struct {
	baseNode
	Name string
}

type AttributePathExpr struct {
	baseNode
	Variable string
	Path     []string
}

type QNameExpr struct {
	baseNode
	Module string
	Name   string
	Sub    string
}

type CallExpr struct {
	baseNode
	Name string
	Args []RobustExpr
}

type BinExpr struct {
	baseNode
	Op string
	L  RobustExpr
	R  RobustExpr
}

type UnaryExpr struct {
	baseNode
	Op      string
	Operand RobustExpr
}

type ParenExpr struct {
	baseNode
	Inner RobustExpr
}

type IfThenElseExpr struct {
	baseNode
	Cond RobustExpr
	Then RobustExpr
	Else RobustExpr
}

type TokenExpr struct {
	baseNode
	Token string
	Arg   string
}

type ConstantRef struct {
	baseNode
	QName string
}

type RecoveredExpr struct {
	baseNode
	SourceFragment string
	Reason         string
}

func (*StringLit) isRobustExpr()         {}
func (*NumberLit) isRobustExpr()         {}
func (*BoolLit) isRobustExpr()           {}
func (*EmptyExpr) isRobustExpr()         {}
func (*VariableExpr) isRobustExpr()      {}
func (*AttributePathExpr) isRobustExpr() {}
func (*QNameExpr) isRobustExpr()         {}
func (*CallExpr) isRobustExpr()          {}
func (*BinExpr) isRobustExpr()           {}
func (*UnaryExpr) isRobustExpr()         {}
func (*ParenExpr) isRobustExpr()         {}
func (*IfThenElseExpr) isRobustExpr()    {}
func (*TokenExpr) isRobustExpr()         {}
func (*ConstantRef) isRobustExpr()       {}
func (*RecoveredExpr) isRobustExpr()     {}
