// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestShowExprSlot_PrintsExpectation(t *testing.T) {
	var buf bytes.Buffer
	if err := runShowExprSlot(&buf, "IfStmt.Condition"); err != nil {
		t.Fatalf("run: %v", err)
	}
	out := buf.String()
	for _, w := range []string{"IF condition", "Boolean", "Sample expressions"} {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in output:\n%s", w, out)
		}
	}
}
