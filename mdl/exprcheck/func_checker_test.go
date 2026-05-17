// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestFuncChecker_ArityMismatch(t *testing.T) {
	p := NewParser()
	// substring requires min 2 args — 1 arg should still fail
	_, hs := p.Parse("substring('hi')", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Fatalf("expected E006, got %+v", hs)
	}
}

func TestFuncChecker_KnownArityOK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("length('hi')", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("unexpected E006: %+v", hs)
	}
}

// Optional argument tests — verify that Mendix functions with optional args
// do not fire E006 when called with the minimum required argument count.

func TestFuncChecker_Round_OneArg_OK(t *testing.T) {
	p := NewParser()
	// round(x) with 1 arg is valid in Mendix (rounds to 0 decimal places)
	_, hs := p.Parse("round(random())", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("round(x) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_Round_TwoArgs_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("round(1.567, 2)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("round(x, d) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_Round_ThreeArgs_Fail(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("round(1.0, 2, 3)", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Fatalf("round() with 3 args must fire E006, got %+v", hs)
	}
}

func TestFuncChecker_Substring_TwoArgs_OK(t *testing.T) {
	p := NewParser()
	// substring(string, startPos) is valid in Mendix (extracts to end of string)
	_, hs := p.Parse("substring('hello', 2)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("substring(s, pos) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_Substring_ThreeArgs_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("substring('hello', 0, 3)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("substring(s, pos, len) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_Substring_OneArg_Fail(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("substring('hello')", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Fatalf("substring with 1 arg must fire E006, got %+v", hs)
	}
}

func hasCode(hs []Hint, code string) bool {
	for _, h := range hs {
		if h.Code == code {
			return true
		}
	}
	return false
}
