// SPDX-License-Identifier: Apache-2.0

package typecheck_test

import (
	"testing"

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
