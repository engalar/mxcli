// SPDX-License-Identifier: Apache-2.0

package typecheck

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

type defaultInferrer struct{}

// NewInferrer returns the default AST-walking TypeKind inferrer.
func NewInferrer() Inferrer { return &defaultInferrer{} }

func (d *defaultInferrer) Infer(
	node exprcheck.RobustExpr,
	scope VarScope,
	cat AttrCatalog,
	funcs FuncReg,
) exprcheck.TypeKind {
	if node == nil {
		return exprcheck.KindUnknown
	}
	switch n := node.(type) {
	case *exprcheck.StringLit:
		return exprcheck.KindString
	case *exprcheck.NumberLit:
		return n.Kind
	case *exprcheck.BoolLit:
		return exprcheck.KindBoolean
	case *exprcheck.EmptyExpr:
		return exprcheck.KindEmpty
	case *exprcheck.ConstantRef:
		return exprcheck.KindUnknown
	case *exprcheck.VariableExpr:
		return scope.TypeOf(n.Name)
	case *exprcheck.AttributePathExpr:
		return inferAttrPath(n, scope, cat)
	case *exprcheck.CallExpr:
		if k, ok := funcs.ReturnType(n.Name); ok {
			return k
		}
		return exprcheck.KindUnknown
	case *exprcheck.BinExpr:
		return d.inferBinExpr(n, scope, cat, funcs)
	case *exprcheck.UnaryExpr:
		if n.Op == "not" {
			return exprcheck.KindBoolean
		}
		return d.Infer(n.Operand, scope, cat, funcs)
	case *exprcheck.IfThenElseExpr:
		if t := d.Infer(n.Then, scope, cat, funcs); t != exprcheck.KindUnknown {
			return t
		}
		return d.Infer(n.Else, scope, cat, funcs)
	case *exprcheck.TokenExpr:
		return inferToken(n.Token)
	case *exprcheck.ParenExpr:
		return d.Infer(n.Inner, scope, cat, funcs)
	case *exprcheck.RecoveredExpr:
		return exprcheck.KindUnknown
	case *exprcheck.QNameExpr:
		return exprcheck.KindUnknown
	}
	return exprcheck.KindUnknown
}

func inferAttrPath(n *exprcheck.AttributePathExpr, scope VarScope, cat AttrCatalog) exprcheck.TypeKind {
	if len(n.Path) == 0 {
		k := scope.TypeOf(n.Variable)
		if k != exprcheck.KindUnknown {
			return k
		}
		if cat.EntityQNOf(n.Variable) != "" {
			return exprcheck.KindObject
		}
		return exprcheck.KindUnknown
	}
	if len(n.Path) == 1 {
		// Mendix system read-only primary key — every entity has it, type is always Long.
		if n.Path[0] == "id" {
			return exprcheck.KindLong
		}
		entityQN := cat.EntityQNOf(n.Variable)
		if entityQN == "" {
			return exprcheck.KindUnknown
		}
		k, ok := cat.AttributeKind(entityQN, n.Path[0])
		if !ok {
			return exprcheck.KindUnknown
		}
		return k
	}
	return exprcheck.KindUnknown
}

func (d *defaultInferrer) inferBinExpr(
	n *exprcheck.BinExpr,
	scope VarScope, cat AttrCatalog, funcs FuncReg,
) exprcheck.TypeKind {
	op := strings.ToLower(n.Op)

	switch op {
	case "=", "!=", "<", ">", "<=", ">=", "and", "or":
		return exprcheck.KindBoolean
	}

	lk := d.Infer(n.L, scope, cat, funcs)
	rk := d.Infer(n.R, scope, cat, funcs)

	switch op {
	case "+":
		if lk == exprcheck.KindString || rk == exprcheck.KindString {
			// Mendix auto-converts numeric types in string concatenation.
			return exprcheck.KindString
		}
		return widenNumeric(lk, rk)
	case "-", "*", "/", "div", "mod":
		return widenNumeric(lk, rk)
	}
	return exprcheck.KindUnknown
}

func widenNumeric(a, b exprcheck.TypeKind) exprcheck.TypeKind {
	rank := map[exprcheck.TypeKind]int{
		exprcheck.KindInteger: 1,
		exprcheck.KindLong:    2,
		exprcheck.KindDecimal: 3,
	}
	ra, oka := rank[a]
	rb, okb := rank[b]
	if !oka || !okb {
		return exprcheck.KindUnknown
	}
	if ra >= rb {
		return a
	}
	return b
}

func inferToken(token string) exprcheck.TypeKind {
	switch token {
	case "CurrentDateTime",
		"CurrentBeginOfDay", "CurrentBeginOfWeek",
		"CurrentBeginOfMonth", "CurrentBeginOfYear",
		"CurrentEndOfDay", "CurrentEndOfWeek",
		"CurrentEndOfMonth", "CurrentEndOfYear":
		return exprcheck.KindDateTime
	case "CurrentUser", "CurrentObject":
		return exprcheck.KindObject
	case "True", "False":
		return exprcheck.KindBoolean
	case "Null":
		return exprcheck.KindEmpty
	}
	return exprcheck.KindUnknown
}
