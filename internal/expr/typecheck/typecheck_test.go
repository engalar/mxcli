// SPDX-License-Identifier: Apache-2.0

package typecheck_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/typecheck"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// ── compatible() tests ────────────────────────────────────────────────────────

func TestCompatible_SameKind(t *testing.T) {
	t.Parallel()
	if !typecheck.Compatible(exprcheck.KindString, exprcheck.KindString) {
		t.Error("same kind should be compatible")
	}
}

func TestCompatible_UnknownSkips(t *testing.T) {
	t.Parallel()
	if !typecheck.Compatible(exprcheck.KindUnknown, exprcheck.KindString) {
		t.Error("KindUnknown actual should skip (return true)")
	}
	if !typecheck.Compatible(exprcheck.KindString, exprcheck.KindUnknown) {
		t.Error("KindUnknown expected should skip (return true)")
	}
}

func TestCompatible_NumericCompat(t *testing.T) {
	t.Parallel()
	// All numeric types are mutually compatible — Mendix runtime handles conversions.
	if !typecheck.Compatible(exprcheck.KindInteger, exprcheck.KindLong) {
		t.Error("Integer should be compatible with Long")
	}
	if !typecheck.Compatible(exprcheck.KindInteger, exprcheck.KindDecimal) {
		t.Error("Integer should be compatible with Decimal")
	}
	if !typecheck.Compatible(exprcheck.KindDecimal, exprcheck.KindInteger) {
		t.Error("Decimal should be compatible with Integer (Mendix narrows at runtime)")
	}
}

func TestCompatible_EmptyIsNull(t *testing.T) {
	t.Parallel()
	// KindEmpty is Mendix null — compatible with any slot type.
	if !typecheck.Compatible(exprcheck.KindEmpty, exprcheck.KindObject) {
		t.Error("Empty should be compatible with Object slot")
	}
	if !typecheck.Compatible(exprcheck.KindEmpty, exprcheck.KindString) {
		t.Error("Empty (null) should be compatible with String slot")
	}
	if !typecheck.Compatible(exprcheck.KindEmpty, exprcheck.KindEnumeration) {
		t.Error("Empty (null) should be compatible with Enumeration slot")
	}
}

func TestCompatible_StringVsInteger(t *testing.T) {
	t.Parallel()
	if typecheck.Compatible(exprcheck.KindString, exprcheck.KindInteger) {
		t.Error("String should NOT be compatible with Integer slot")
	}
}

// ── FuncReg tests ─────────────────────────────────────────────────────────────

func TestFuncReg_DateTimeExtractionFunctions(t *testing.T) {
	t.Parallel()
	reg := typecheck.NewFuncReg()
	fns := []string{"year", "month", "dayOfYear", "dayOfMonth",
		"weekOfYear", "dayOfWeek", "hour", "minute", "second", "millisecond"}
	for _, name := range fns {
		k, ok := reg.ReturnType(name)
		if !ok {
			t.Errorf("ReturnType(%q): not found", name)
			continue
		}
		if k != exprcheck.KindInteger {
			t.Errorf("ReturnType(%q) = %v, want KindInteger", name, k)
		}
	}
}

func TestFuncReg_KnownStringFunctions(t *testing.T) {
	t.Parallel()
	reg := typecheck.NewFuncReg()
	cases := map[string]exprcheck.TypeKind{
		"toString":    exprcheck.KindString,
		"toLowerCase": exprcheck.KindString,
		"length":      exprcheck.KindInteger,
		"contains":    exprcheck.KindBoolean,
		"round":       exprcheck.KindDecimal,
		"addDays":     exprcheck.KindDateTime,
	}
	for name, want := range cases {
		got, ok := reg.ReturnType(name)
		if !ok {
			t.Errorf("ReturnType(%q): not found", name)
			continue
		}
		if got != want {
			t.Errorf("ReturnType(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestFuncReg_Unknown(t *testing.T) {
	t.Parallel()
	reg := typecheck.NewFuncReg()
	_, ok := reg.ReturnType("notAFunction")
	if ok {
		t.Error("expected false for unknown function")
	}
}

// ── SlotResolver tests ────────────────────────────────────────────────────────

type mockCat struct {
	attrs     map[string]exprcheck.TypeKind
	entityQNs map[string]string // varName → entityQN; nil means always ""
}

func (m *mockCat) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	key := entityQN + "." + attrName
	if k, ok := m.attrs[key]; ok {
		return k, true
	}
	return exprcheck.KindUnknown, false
}

func (m *mockCat) EntityQNOf(varName string) string {
	if m.entityQNs == nil {
		return ""
	}
	return m.entityQNs[varName]
}

type mockIdx struct {
	paramKinds map[string]map[string]exprcheck.TypeKind
}

func (m *mockIdx) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}
func (m *mockIdx) VarTypeKind(_, _ string) exprcheck.TypeKind { return exprcheck.KindUnknown }
func (m *mockIdx) VarEntityQN(_, _ string) string             { return "" }
func (m *mockIdx) MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool) {
	name := calleeQN
	if i := strings.LastIndex(calleeQN, "."); i >= 0 {
		name = calleeQN[i+1:]
	}
	if params, ok := m.paramKinds[name]; ok {
		if k, ok2 := params[paramName]; ok2 {
			return k, true
		}
	}
	return exprcheck.KindUnknown, false
}
func (m *mockIdx) MicroflowReturnKind(_ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

func TestSlotResolver_StaticSlots(t *testing.T) {
	t.Parallel()
	idx := &mockIdx{}
	cat := &mockCat{}
	resolver := typecheck.NewSlotResolver(idx)

	tests := []struct {
		unitType string
		field    string
		want     exprcheck.TypeKind
	}{
		// ExpressionSplitCondition intentionally removed: can be Boolean or Enumeration.
		{"Microflows$WhileLoopCondition", "WhileExpression", exprcheck.KindBoolean},
		{"Microflows$TemplateParameter", "Expression", exprcheck.KindString},
		{"Microflows$CustomRange", "LimitExpression", exprcheck.KindInteger},
	}

	for _, tt := range tests {
		rec := scan.ExprRecord{UnitType: tt.unitType, Field: tt.field}
		got, ok := resolver.Expect(rec, cat)
		if !ok {
			t.Errorf("Expect(%s/%s): got !ok", tt.unitType, tt.field)
			continue
		}
		if got != tt.want {
			t.Errorf("Expect(%s/%s) = %v, want %v", tt.unitType, tt.field, got, tt.want)
		}
	}
}

func TestSlotResolver_AttrTarget(t *testing.T) {
	t.Parallel()
	idx := &mockIdx{}
	cat := &mockCat{
		attrs: map[string]exprcheck.TypeKind{
			"WF_Engine.WFInstance.Wf_No": exprcheck.KindString,
		},
	}
	resolver := typecheck.NewSlotResolver(idx)

	rec := scan.ExprRecord{
		UnitType:     "Microflows$ChangeActionItem",
		Field:        "Value",
		TargetAttrQN: "WF_Engine.WFInstance.Wf_No",
	}
	got, ok := resolver.Expect(rec, cat)
	if !ok {
		t.Fatal("expected ok=true for ChangeActionItem with TargetAttrQN")
	}
	if got != exprcheck.KindString {
		t.Errorf("got %v, want KindString", got)
	}
}

func TestSlotResolver_CallArgTarget(t *testing.T) {
	t.Parallel()
	idx := &mockIdx{
		paramKinds: map[string]map[string]exprcheck.TypeKind{
			"SUB_AdvanceStepStatus": {
				"WFInstanceId": exprcheck.KindLong,
			},
		},
	}
	cat := &mockCat{}
	resolver := typecheck.NewSlotResolver(idx)

	rec := scan.ExprRecord{
		UnitType:  "Microflows$MicroflowCallParameterMapping",
		Field:     "Argument",
		CalleeQN:  "WF_Engine.SUB_AdvanceStepStatus",
		ParamName: "WFInstanceId",
	}
	got, ok := resolver.Expect(rec, cat)
	if !ok {
		t.Fatal("expected ok=true for call arg target")
	}
	if got != exprcheck.KindLong {
		t.Errorf("got %v, want KindLong", got)
	}
}

func TestSlotResolver_UnknownSlot(t *testing.T) {
	t.Parallel()
	idx := &mockIdx{}
	cat := &mockCat{}
	resolver := typecheck.NewSlotResolver(idx)

	rec := scan.ExprRecord{UnitType: "SomeUnknown$Type", Field: "Value"}
	_, ok := resolver.Expect(rec, cat)
	if ok {
		t.Error("expected ok=false for unknown slot")
	}
}

// ── Inferrer tests ────────────────────────────────────────────────────────────

type mockScope struct{ kinds map[string]exprcheck.TypeKind }

func (m *mockScope) TypeOf(name string) exprcheck.TypeKind {
	if k, ok := m.kinds[name]; ok {
		return k
	}
	return exprcheck.KindUnknown
}

func TestInferrer_Literals(t *testing.T) {
	t.Parallel()
	inf := typecheck.NewInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	tests := []struct {
		raw  string
		want exprcheck.TypeKind
	}{
		{"'hello'", exprcheck.KindString},
		{"42", exprcheck.KindInteger},
		{"3.14", exprcheck.KindDecimal},
		{"true", exprcheck.KindBoolean},
		{"false", exprcheck.KindBoolean},
		{"empty", exprcheck.KindEmpty},
	}

	for _, tt := range tests {
		ast, _ := parser.Parse(tt.raw, exprcheck.Context{})
		got := inf.Infer(ast, scope, cat, funcs)
		if got != tt.want {
			t.Errorf("Infer(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestInferrer_VariableExpr(t *testing.T) {
	t.Parallel()
	inf := typecheck.NewInferrer()
	scope := &mockScope{kinds: map[string]exprcheck.TypeKind{"MyVar": exprcheck.KindString}}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	ast, _ := parser.Parse("$MyVar", exprcheck.Context{})
	got := inf.Infer(ast, scope, cat, funcs)
	if got != exprcheck.KindString {
		t.Errorf("Infer($MyVar) = %v, want KindString", got)
	}
}

func TestInferrer_FunctionCall(t *testing.T) {
	t.Parallel()
	inf := typecheck.NewInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	tests := []struct {
		raw  string
		want exprcheck.TypeKind
	}{
		{"year($d)", exprcheck.KindInteger},
		{"toString(42)", exprcheck.KindString},
		{"length('abc')", exprcheck.KindInteger},
		{"contains('abc', 'a')", exprcheck.KindBoolean},
	}
	for _, tt := range tests {
		ast, _ := parser.Parse(tt.raw, exprcheck.Context{})
		got := inf.Infer(ast, scope, cat, funcs)
		if got != tt.want {
			t.Errorf("Infer(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestInferrer_AttrPath_SingleHop(t *testing.T) {
	t.Parallel()
	inf := typecheck.NewInferrer()
	scope := &mockScope{}
	cat := &mockCat{
		entityQNs: map[string]string{"Entity": "Mod.Entity"},
		attrs:     map[string]exprcheck.TypeKind{"Mod.Entity.Name": exprcheck.KindString},
	}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	ast, _ := parser.Parse("$Entity/Name", exprcheck.Context{})
	got := inf.Infer(ast, scope, cat, funcs)
	if got != exprcheck.KindString {
		t.Errorf("Infer($Entity/Name) = %v, want KindString", got)
	}
}

func TestInferrer_BooleanComparison(t *testing.T) {
	t.Parallel()
	inf := typecheck.NewInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	parser := exprcheck.NewParser()
	ast, _ := parser.Parse("1 = 2", exprcheck.Context{})
	got := inf.Infer(ast, scope, cat, funcs)
	if got != exprcheck.KindBoolean {
		t.Errorf("Infer(1 = 2) = %v, want KindBoolean", got)
	}
}

func TestInferrer_RecoveredExprIsUnknown(t *testing.T) {
	t.Parallel()
	inf := typecheck.NewInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	recovered := &exprcheck.RecoveredExpr{}
	got := inf.Infer(recovered, scope, cat, funcs)
	if got != exprcheck.KindUnknown {
		t.Errorf("Infer(RecoveredExpr) = %v, want KindUnknown", got)
	}
}

func TestInferToken_NewTokens(t *testing.T) {
	t.Parallel()
	inf := typecheck.NewInferrer()
	scope := &mockScope{}
	cat := &mockCat{}
	funcs := typecheck.NewFuncReg()

	// helper: build TokenExpr AST node
	tokenNode := func(name string) exprcheck.RobustExpr {
		return &exprcheck.TokenExpr{Token: name}
	}

	cases := []struct {
		token string
		want  exprcheck.TypeKind
	}{
		// Time-point tokens that were missing before
		{"BeginOfCurrentMinute", exprcheck.KindDateTime},
		{"EndOfCurrentMinute", exprcheck.KindDateTime},
		{"BeginOfCurrentHour", exprcheck.KindDateTime},
		{"EndOfCurrentHour", exprcheck.KindDateTime},
		{"BeginOfYesterday", exprcheck.KindDateTime},
		{"EndOfYesterday", exprcheck.KindDateTime},
		{"BeginOfTomorrow", exprcheck.KindDateTime},
		{"EndOfTomorrow", exprcheck.KindDateTime},
		// UTC variants
		{"BeginOfCurrentDayUTC", exprcheck.KindDateTime},
		{"BeginOfCurrentWeekUTC", exprcheck.KindDateTime},
		{"BeginOfCurrentMinuteUTC", exprcheck.KindDateTime},
		// Duration tokens (value = integer milliseconds)
		{"SecondLength", exprcheck.KindInteger},
		{"MinuteLength", exprcheck.KindInteger},
		{"HourLength", exprcheck.KindInteger},
		{"DayLength", exprcheck.KindInteger},
		{"WeekLength", exprcheck.KindInteger},
		{"MonthLength", exprcheck.KindInteger},
		{"YearLength", exprcheck.KindInteger},
		// UserRole_* → KindObject (type always correct; name validated by SEM-08)
		{"UserRole_Administrator", exprcheck.KindObject},
		{"UserRole_AnyName", exprcheck.KindObject},
	}

	for _, tc := range cases {
		got := inf.Infer(tokenNode(tc.token), scope, cat, funcs)
		if got != tc.want {
			t.Errorf("Infer(TokenExpr{%q}) = %v, want %v", tc.token, got, tc.want)
		}
	}
}
