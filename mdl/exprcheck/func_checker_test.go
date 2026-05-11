// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestFuncChecker_ArityMismatch(t *testing.T) {
	p := NewParser()
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

func hasCode(hs []Hint, code string) bool {
	for _, h := range hs {
		if h.Code == code {
			return true
		}
	}
	return false
}
