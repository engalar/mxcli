// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerate_ContainsAllRegisteredCodes(t *testing.T) {
	var buf bytes.Buffer
	if err := Generate(&buf); err != nil {
		t.Fatalf("generate: %v", err)
	}
	out := buf.String()
	for _, code := range []string{"E001", "E002", "E003", "E004", "E005", "E006", "E007", "E008", "E009", "E010"} {
		if !strings.Contains(out, "## "+code) {
			t.Errorf("missing heading for %s", code)
		}
	}
	for _, marker := range []string{"# Expression Checker Hint Reference", "When this appears", "Why it's wrong", "How to fix"} {
		if !strings.Contains(out, marker) {
			t.Errorf("missing marker %q", marker)
		}
	}
}

func TestGenerate_CodesSorted(t *testing.T) {
	var buf bytes.Buffer
	if err := Generate(&buf); err != nil {
		t.Fatalf("generate: %v", err)
	}
	out := buf.String()
	prevPos := -1
	for _, code := range []string{"E001", "E002", "E003", "E004", "E005", "E006", "E007", "E008", "E009", "E010"} {
		pos := strings.Index(out, "## "+code+" ")
		if pos < 0 {
			continue
		}
		if pos < prevPos {
			t.Errorf("code %s appears before earlier code (pos=%d, prev=%d)", code, pos, prevPos)
		}
		prevPos = pos
	}
}
