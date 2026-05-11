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

func TestParser_E001_StringInEnumSlot(t *testing.T) {
	p := NewParser()
	expr, hints := p.Parse(`'NewAlert'`, Context{
		SlotPath:  "ChangeItem.Value",
		Microflow: "FraudDetection.SUB_CreateAlert",
		File:      "fraud.mdl",
		Line:      43,
		Column:    20,
		Slots:     DefaultSlotResolver(),
	})
	if _, ok := expr.(*StringLit); !ok {
		t.Fatalf("expected StringLit, got %T", expr)
	}
	var sawE001 bool
	for _, h := range hints {
		if h.Code == "E001" {
			sawE001 = true
		}
	}
	if sawE001 {
		t.Errorf("with no catalog, E001 should not fire; got %+v", hints)
	}
}

func TestParser_E001_HitsForKnownEnumKind(t *testing.T) {
	p := NewParser()
	cat := fakeCatalog{kind: KindEnumeration, enumQN: "FraudDetection.AlertStatus", values: []string{"NewAlert", "Validated"}}
	expr, hints := p.Parse(`'NewAlert'`, Context{
		SlotPath:  "ChangeItem.Value",
		Microflow: "FraudDetection.SUB_CreateAlert",
		Slots:     DefaultSlotResolver(),
		Catalog:   cat,
	})
	if _, ok := expr.(*StringLit); !ok {
		t.Fatalf("got %T", expr)
	}
	if len(hints) == 0 || hints[0].Code != "E001" {
		t.Fatalf("expected E001, got %+v", hints)
	}
	if got := hints[0].Reference.Enum; got != "FraudDetection.AlertStatus" {
		t.Errorf("enum ref = %q", got)
	}
}

type fakeCatalog struct {
	kind   TypeKind
	enumQN string
	values []string
}

func (f fakeCatalog) AttributeKind(string, string) (TypeKind, bool)  { return f.kind, true }
func (f fakeCatalog) EnumCases(string) ([]string, bool)              { return f.values, true }
func (f fakeCatalog) MicroflowReturn(string) (TypeKind, bool)        { return KindUnknown, false }
func (f fakeCatalog) MicroflowParam(string, string) (TypeKind, bool) { return KindUnknown, false }

func TestParser_E007_RecoveryAtPrimary(t *testing.T) {
	p := NewParser()
	expr, hs := p.Parse("'count=' + length(@@@broken@@@) + ' items'", Context{
		SlotPath:  "MfSetStmt.Value",
		Microflow: "M.F",
	})
	if _, ok := expr.(*BinExpr); !ok {
		t.Fatalf("outer = %T, want *BinExpr", expr)
	}
	var sawE007 bool
	for _, h := range hs {
		if h.Code == "E007" {
			sawE007 = true
			if h.Fix == "" || h.Problem == "" {
				t.Errorf("E007 missing prose: %+v", h)
			}
		}
	}
	if !sawE007 {
		t.Fatalf("expected E007 in hints %+v", hs)
	}
}
