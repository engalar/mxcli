// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestShowFunctions_All(t *testing.T) {
	var buf bytes.Buffer
	if err := runShowFunctions(&buf, ""); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(buf.String(), "length") {
		t.Errorf("missing length in:\n%s", buf.String())
	}
}

func TestShowFunctions_Single(t *testing.T) {
	var buf bytes.Buffer
	if err := runShowFunctions(&buf, "length"); err != nil {
		t.Fatalf("run: %v", err)
	}
	out := buf.String()
	for _, w := range []string{"length", "String", "Integer"} {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}
