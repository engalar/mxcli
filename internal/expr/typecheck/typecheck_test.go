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
	if !typecheck.Compatible(exprcheck.KindString, exprcheck.KindString) {
		t.Error("same kind should be compatible")
	}
}

func TestCompatible_UnknownSkips(t *testing.T) {
	if !typecheck.Compatible(exprcheck.KindUnknown, exprcheck.KindString) {
		t.Error("KindUnknown actual should skip (return true)")
	}
	if !typecheck.Compatible(exprcheck.KindString, exprcheck.KindUnknown) {
		t.Error("KindUnknown expected should skip (return true)")
	}
}

func TestCompatible_NumericWidening(t *testing.T) {
	if !typecheck.Compatible(exprcheck.KindInteger, exprcheck.KindLong) {
		t.Error("Integer should widen to Long")
	}
	if !typecheck.Compatible(exprcheck.KindInteger, exprcheck.KindDecimal) {
		t.Error("Integer should widen to Decimal")
	}
	if typecheck.Compatible(exprcheck.KindDecimal, exprcheck.KindInteger) {
		t.Error("Decimal should NOT narrow to Integer")
	}
}

func TestCompatible_EmptyToObject(t *testing.T) {
	if !typecheck.Compatible(exprcheck.KindEmpty, exprcheck.KindObject) {
		t.Error("Empty should be compatible with Object slot")
	}
	if typecheck.Compatible(exprcheck.KindEmpty, exprcheck.KindString) {
		t.Error("Empty should NOT be compatible with String slot")
	}
}

func TestCompatible_StringVsInteger(t *testing.T) {
	if typecheck.Compatible(exprcheck.KindString, exprcheck.KindInteger) {
		t.Error("String should NOT be compatible with Integer slot")
	}
}

// ── FuncReg tests ─────────────────────────────────────────────────────────────

func TestFuncReg_DateTimeExtractionFunctions(t *testing.T) {
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
	reg := typecheck.NewFuncReg()
	_, ok := reg.ReturnType("notAFunction")
	if ok {
		t.Error("expected false for unknown function")
	}
}

// ── SlotResolver tests ────────────────────────────────────────────────────────

type mockCat struct {
	attrs map[string]exprcheck.TypeKind
}

func (m *mockCat) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	key := entityQN + "." + attrName
	if k, ok := m.attrs[key]; ok {
		return k, true
	}
	return exprcheck.KindUnknown, false
}

func (m *mockCat) EntityQNOf(_ string) string { return "" }

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
	idx := &mockIdx{}
	cat := &mockCat{}
	resolver := typecheck.NewSlotResolver(idx)

	tests := []struct {
		unitType string
		field    string
		want     exprcheck.TypeKind
	}{
		{"Microflows$ExpressionSplitCondition", "Expression", exprcheck.KindBoolean},
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
	idx := &mockIdx{}
	cat := &mockCat{}
	resolver := typecheck.NewSlotResolver(idx)

	rec := scan.ExprRecord{UnitType: "SomeUnknown$Type", Field: "Value"}
	_, ok := resolver.Expect(rec, cat)
	if ok {
		t.Error("expected ok=false for unknown slot")
	}
}
