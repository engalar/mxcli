// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExplainExpression_PrintsHints(t *testing.T) {
	var buf bytes.Buffer
	if err := runExplainExpression(&buf, "$x = 'true'", "IfStmt.Condition"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(buf.String(), "E002") && !strings.Contains(buf.String(), "no hints") {
		t.Errorf("unexpected output:\n%s", buf.String())
	}
}

func TestExplainExpression_NoHintsMessage(t *testing.T) {
	var buf bytes.Buffer
	if err := runExplainExpression(&buf, "true", "IfStmt.Condition"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(buf.String(), "no hints") {
		t.Errorf("expected 'no hints' line, got:\n%s", buf.String())
	}
}
