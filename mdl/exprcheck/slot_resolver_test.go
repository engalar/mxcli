// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestSlotResolver_KnownSlots(t *testing.T) {
	r := DefaultSlotResolver()
	cases := []struct {
		path string
		kind TypeKind
	}{
		{"IfStmt.Condition", KindBoolean},
		{"WhileStmt.Condition", KindBoolean},
		{"RetrieveStmt.LimitExpr", KindInteger},
		{"RetrieveStmt.OffsetExpr", KindInteger},
	}
	for _, c := range cases {
		sc, ok := r.Expect(c.path)
		if !ok {
			t.Errorf("%s: not registered", c.path)
			continue
		}
		if sc.Kind != c.kind {
			t.Errorf("%s: kind = %v, want %v", c.path, sc.Kind, c.kind)
		}
	}
}

func TestSlotToContext_HumanWords(t *testing.T) {
	cases := map[string]string{
		"IfStmt.Condition":         "IF condition",
		"WhileStmt.Condition":      "WHILE condition",
		"ChangeItem.Value":         "field of CHANGE",
		"CreateItem.Value":         "field of CREATE",
		"ReturnStmt.Value":         "RETURN value",
		"RetrieveStmt.LimitExpr":   "LIMIT clause",
		"RetrieveStmt.OffsetExpr":  "OFFSET clause",
		"LogStmt.Message":          "LOG message",
		"MfSetStmt.Value":          "right-hand side of SET",
		"DeclareStmt.InitialValue": "initial value of DECLARE",
		"CallArgument.Value":       "argument of CALL",
	}
	for path, want := range cases {
		if got := SlotToContext(path); got != want {
			t.Errorf("SlotToContext(%q) = %q, want %q", path, got, want)
		}
	}
}
