// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func parseFor(t *testing.T, src string) (RobustExpr, []Hint) {
	t.Helper()
	p := NewParser()
	return p.Parse(src, Context{})
}

func TestParser_StringLit(t *testing.T) {
	expr, hints := parseFor(t, `'hello'`)
	if len(hints) != 0 {
		t.Fatalf("hints: %+v", hints)
	}
	s, ok := expr.(*StringLit)
	if !ok || s.Value != "hello" {
		t.Fatalf("got %T %+v", expr, expr)
	}
}

func TestParser_BoolEmptyVariable(t *testing.T) {
	e1, _ := parseFor(t, "true")
	if _, ok := e1.(*BoolLit); !ok {
		t.Fatalf("true → %T", e1)
	}
	e2, _ := parseFor(t, "empty")
	if _, ok := e2.(*EmptyExpr); !ok {
		t.Fatalf("empty → %T", e2)
	}
	e3, _ := parseFor(t, "$alert")
	if v, ok := e3.(*VariableExpr); !ok || v.Name != "alert" {
		t.Fatalf("$alert → %T %+v", e3, e3)
	}
}

func TestParser_AttributePath(t *testing.T) {
	e, _ := parseFor(t, "$alert/Status")
	p, ok := e.(*AttributePathExpr)
	if !ok {
		t.Fatalf("got %T", e)
	}
	if p.Variable != "alert" || len(p.Path) != 1 || p.Path[0] != "Status" {
		t.Fatalf("got %+v", p)
	}
}

func TestParser_QName3Part(t *testing.T) {
	e, _ := parseFor(t, "Module.Enum.Value")
	q, ok := e.(*QNameExpr)
	if !ok {
		t.Fatalf("got %T", e)
	}
	if q.Module != "Module" || q.Name != "Enum" || q.Sub != "Value" {
		t.Fatalf("got %+v", q)
	}
}
