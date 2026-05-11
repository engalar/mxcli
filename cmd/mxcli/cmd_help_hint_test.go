// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpHint_PrintsRegistryEntry(t *testing.T) {
	var buf bytes.Buffer
	if err := runHelpHint(&buf, "E001"); err != nil {
		t.Fatalf("run: %v", err)
	}
	out := buf.String()
	for _, w := range []string{"E001", "enum-string-mismatch", "WHEN THIS APPEARS", "HOW TO FIX", "EXAMPLES"} {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q:\n%s", w, out)
		}
	}
}

func TestHelpHint_UnknownCodeErrors(t *testing.T) {
	var buf bytes.Buffer
	err := runHelpHint(&buf, "E999")
	if err == nil {
		t.Fatal("expected error for unknown hint code")
	}
	if !strings.Contains(err.Error(), "E999") {
		t.Errorf("error %q should mention requested code", err)
	}
}
