// SPDX-License-Identifier: Apache-2.0

package exprcheck

import (
	"testing"
	"time"
)

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
		SlotPath:  "ChangeItem.Value:FraudDetection.Alert.Status",
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
func (f fakeCatalog) AttributeEnumQN(string, string) (string, bool)  { return f.enumQN, true }
func (f fakeCatalog) EnumCases(string) ([]string, bool)              { return f.values, true }
func (f fakeCatalog) MicroflowReturn(string) (TypeKind, bool)        { return KindUnknown, false }
func (f fakeCatalog) MicroflowParam(string, string) (TypeKind, bool) { return KindUnknown, false }

func TestParser_E001_ChangeItemEnumViaCatalog(t *testing.T) {
	p := NewParser()
	cat := lookupCatalog{
		kinds: map[string]TypeKind{"Sales.Customer|Status": KindEnumeration},
		enums: map[string]string{"Sales.Customer|Status": "Sales.CustomerStatus"},
		cases: map[string][]string{"Sales.CustomerStatus": {"Active", "Inactive"}},
	}
	_, hs := p.Parse("'Active'", Context{
		SlotPath: "ChangeItem.Value:Sales.Customer.Status",
		Slots:    DefaultSlotResolver(),
		Catalog:  cat,
	})
	if !hasCode(hs, "E001") {
		t.Fatalf("expected E001 hit, got %+v", hs)
	}
}

func TestParser_E001_CreateItemEnumViaCatalog(t *testing.T) {
	p := NewParser()
	cat := lookupCatalog{
		kinds: map[string]TypeKind{"M.Customer|Status": KindEnumeration},
		enums: map[string]string{"M.Customer|Status": "M.E"},
		cases: map[string][]string{"M.E": {"A"}},
	}
	_, hs := p.Parse("'A'", Context{
		SlotPath: "CreateItem.Value:M.Customer.Status",
		Slots:    DefaultSlotResolver(),
		Catalog:  cat,
	})
	if !hasCode(hs, "E001") {
		t.Fatalf("expected E001 hit, got %+v", hs)
	}
}

func TestParser_E001_NoFireForNonEnumAttr(t *testing.T) {
	p := NewParser()
	// Catalog returns String for this attribute → must NOT trigger E001.
	cat := lookupCatalog{
		kinds: map[string]TypeKind{"Sales.Customer|Name": KindString},
	}
	_, hs := p.Parse("'Acme'", Context{
		SlotPath: "ChangeItem.Value:Sales.Customer.Name",
		Slots:    DefaultSlotResolver(),
		Catalog:  cat,
	})
	if hasCode(hs, "E001") {
		t.Fatalf("E001 must not fire for String attribute; got %+v", hs)
	}
}

func TestParser_E001_NoFireWithoutEntityAttrSuffix(t *testing.T) {
	p := NewParser()
	// SlotPath has no ":entity.attr" suffix → no catalog lookup, no E001.
	cat := lookupCatalog{
		kinds: map[string]TypeKind{"Sales.Customer|Status": KindEnumeration},
	}
	_, hs := p.Parse("'Active'", Context{
		SlotPath: "ChangeItem.Value",
		Slots:    DefaultSlotResolver(),
		Catalog:  cat,
	})
	if hasCode(hs, "E001") {
		t.Fatalf("E001 must not fire when slot lacks entity.attr; got %+v", hs)
	}
}

// lookupCatalog is a precise CatalogReader keyed by "<entityQN>|<attr>".
type lookupCatalog struct {
	kinds map[string]TypeKind
	enums map[string]string
	cases map[string][]string
}

func (c lookupCatalog) AttributeKind(entity, attr string) (TypeKind, bool) {
	k, ok := c.kinds[entity+"|"+attr]
	return k, ok
}

func (c lookupCatalog) AttributeEnumQN(entity, attr string) (string, bool) {
	q, ok := c.enums[entity+"|"+attr]
	return q, ok
}

func (c lookupCatalog) EnumCases(qn string) ([]string, bool) {
	v, ok := c.cases[qn]
	return v, ok
}

func (c lookupCatalog) MicroflowReturn(string) (TypeKind, bool)        { return KindUnknown, false }
func (c lookupCatalog) MicroflowParam(string, string) (TypeKind, bool) { return KindUnknown, false }

func TestParser_E004_ConcatLiteralIntWithString(t *testing.T) {
	// Mendix auto-converts Integer/Decimal in '+' string-concatenation context.
	// Verified: mx check shows 0 CE0117 for "'T14' + round(x)" and similar patterns.
	// E004 must NOT fire for numeric types (Integer, Decimal, Long).
	p := NewParser()
	_, hs := p.Parse("'count=' + 5", Context{Microflow: "M.F"})
	if hasCode(hs, "E004") {
		t.Fatalf("E004 must not fire for String+Integer (Mendix auto-converts numeric types): %+v", hs)
	}
}

func TestParser_E003_NullToEmpty(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("null", Context{Microflow: "M.F"})
	if !hasCode(hs, "E003") {
		t.Fatalf("expected E003, got %+v", hs)
	}
}

func TestParser_E002_BoolStringInBoolSlot(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("'true'", Context{
		SlotPath: "IfStmt.Condition",
		Slots:    DefaultSlotResolver(),
	})
	if !hasCode(hs, "E002") {
		t.Fatalf("expected E002, got %+v", hs)
	}
}

func TestParser_E007_DegenerateCallArgsTerminates(t *testing.T) {
	done := make(chan struct{})
	go func() {
		p := NewParser()
		_, _ = p.Parse("length(", Context{Microflow: "M.F"})
		_, _ = p.Parse("length(,)", Context{Microflow: "M.F"})
		_, _ = p.Parse("length()", Context{Microflow: "M.F"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("parser did not terminate on degenerate call args within 2s")
	}
}

func TestParser_E007_InsideCallArgs(t *testing.T) {
	p := NewParser()
	expr, hs := p.Parse("length(@@@bad@@@)", Context{Microflow: "M.F"})
	call, ok := expr.(*CallExpr)
	if !ok {
		t.Fatalf("got %T", expr)
	}
	if call.Name != "length" || len(call.Args) != 1 {
		t.Fatalf("call shape: %+v", call)
	}
	if _, ok := call.Args[0].(*RecoveredExpr); !ok {
		t.Fatalf("arg 0 = %T, want *RecoveredExpr", call.Args[0])
	}
	var sawE007 bool
	for _, h := range hs {
		if h.Code == "E007" {
			sawE007 = true
		}
	}
	if !sawE007 {
		t.Fatalf("expected E007 in %+v", hs)
	}
}

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

func TestInferKind_Coverage(t *testing.T) {
	ctx := Context{}
	cases := []struct {
		name string
		node RobustExpr
		want TypeKind
	}{
		{"StringLit", &StringLit{Value: "x"}, KindString},
		{"NumberLit int", &NumberLit{Value: "1", Kind: KindInteger}, KindInteger},
		{"NumberLit dec", &NumberLit{Value: "1.5", Kind: KindDecimal}, KindDecimal},
		{"BoolLit", &BoolLit{Value: true}, KindBoolean},
		{"EmptyExpr", &EmptyExpr{}, KindEmpty},
		{"VariableExpr no scope", &VariableExpr{Name: "x"}, KindUnknown},
		{"AttributePathExpr", &AttributePathExpr{Variable: "x", Path: []string{"Attr"}}, KindUnknown},
		{"QNameExpr", &QNameExpr{Module: "M", Name: "E", Sub: "V"}, KindUnknown},
		{"CallExpr known", &CallExpr{Name: "length", Args: []RobustExpr{&StringLit{Value: "x"}}}, KindInteger},
		{"CallExpr unknown func", &CallExpr{Name: "myCustomFunc"}, KindUnknown},
		{"BinExpr AND", &BinExpr{Op: "AND", L: &BoolLit{Value: true}, R: &BoolLit{Value: false}}, KindBoolean},
		{"BinExpr OR", &BinExpr{Op: "OR", L: &BoolLit{Value: true}, R: &BoolLit{Value: false}}, KindBoolean},
		{"BinExpr eq", &BinExpr{Op: "=", L: &StringLit{Value: "a"}, R: &StringLit{Value: "b"}}, KindBoolean},
		{"BinExpr + strings", &BinExpr{Op: "+", L: &StringLit{Value: "a"}, R: &StringLit{Value: "b"}}, KindString},
		{"UnaryExpr NOT", &UnaryExpr{Op: "NOT", Operand: &BoolLit{Value: true}}, KindBoolean},
		{"UnaryExpr minus int", &UnaryExpr{Op: "-", Operand: &NumberLit{Value: "1", Kind: KindInteger}}, KindInteger},
		{"ParenExpr bool", &ParenExpr{Inner: &BoolLit{Value: true}}, KindBoolean},
		{"IfThenElseExpr string branches", &IfThenElseExpr{
			Cond: &BoolLit{Value: true},
			Then: &StringLit{Value: "yes"},
			Else: &StringLit{Value: "no"},
		}, KindString},
		{"TokenExpr", &TokenExpr{Token: "Translation.text"}, KindString},
		{"ConstantRef", &ConstantRef{QName: "M.Const"}, KindUnknown},
		{"RecoveredExpr", &RecoveredExpr{SourceFragment: "???"}, KindUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inferKind(tc.node, ctx)
			if got != tc.want {
				t.Errorf("inferKind(%T) = %v, want %v", tc.node, got, tc.want)
			}
		})
	}
}

func TestParser_E009_NotNonBoolean(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"string operand", "not 'hello'"},
		{"integer operand", "not 5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if !hasCode(hs, "E009") {
				t.Fatalf("expected E009 for %q, got %+v", tc.src, hs)
			}
		})
	}
}

func TestParser_E009_NotBoolLit_NoHint(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"bool literal", "not true"},
		{"bool expression", "not (1 = 1)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if hasCode(hs, "E009") {
				t.Fatalf("E009 must not fire for bool operand %q; got %+v", tc.src, hs)
			}
		})
	}
}

func TestParser_E009_NotUnknownOperand_NoHint(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("not $Validation/IsValid", Context{Microflow: "M.F"})
	if hasCode(hs, "E009") {
		t.Fatalf("E009 must not fire for unknown-kind operand; got %+v", hs)
	}
}

func TestParser_E009_AndNonBoolean(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"string left operand", "'hello' and true"},
		{"integer right operand", "true and 5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if !hasCode(hs, "E009") {
				t.Fatalf("expected E009 for %q, got %+v", tc.src, hs)
			}
		})
	}
}

func TestParser_E009_OrNonBoolean(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("'hello' or true", Context{Microflow: "M.F"})
	if !hasCode(hs, "E009") {
		t.Fatalf("expected E009, got %+v", hs)
	}
}

func TestParser_E009_AndOrUnknown_NoHint(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"and with unknown right", "true and $x/Attr"},
		{"or with unknown left", "$x/Attr or true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if hasCode(hs, "E009") {
				t.Fatalf("E009 must not fire for unknown-kind operand %q; got %+v", tc.src, hs)
			}
		})
	}
}

func TestParser_E011_NotMissingParens(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"attribute path", "not $Validation/IsValid"},
		{"contains call", "not contains($s, '@')"},
		{"isMatch call", "not isMatch($v, '^[0-9]+$')"},
		{"compound and", "$x != empty and not isMatch($v, '^[0-9]+$')"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if !hasCode(hs, "E011") {
				t.Fatalf("expected E011 for %q, got %+v", tc.src, hs)
			}
		})
	}
}

func TestParser_E011_NotWithParens_NoHint(t *testing.T) {
	p := NewParser()
	cases := []struct {
		name string
		src  string
	}{
		{"attribute path with parens", "not($Validation/IsValid)"},
		{"isMatch with parens", "not(isMatch($v, '^[0-9]+$'))"},
		{"compound with parens", "$x != empty and not(contains($s, '@'))"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, hs := p.Parse(tc.src, Context{Microflow: "M.F"})
			if hasCode(hs, "E011") {
				t.Fatalf("E011 must not fire for %q; got %+v", tc.src, hs)
			}
		})
	}
}
