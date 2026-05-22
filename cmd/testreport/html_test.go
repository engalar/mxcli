// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderHTML_ContainsLayerNames(t *testing.T) {
	layerMap := map[string]*LayerSummary{
		"L1L2": {ID: "L1L2", Name: "L1/L2 Parser+Visitor", Priority: 1, Pass: 38, Fail: 0, Elapsed: 1.2},
		"L3": {ID: "L3", Name: "L3 Executor Mock", Priority: 2, Pass: 10, Fail: 1, Elapsed: 2.1,
			Failures: []FailureDetail{{Package: "pkg/executor", Test: "TestFoo", Output: "expected bar"}}},
	}
	var buf bytes.Buffer
	if err := renderHTML(&buf, layerMap, "", "abc123"); err != nil {
		t.Fatalf("renderHTML: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "L1/L2 Parser+Visitor") {
		t.Error("missing L1/L2 in HTML")
	}
	if !strings.Contains(out, "L3 Executor Mock") {
		t.Error("missing L3 in HTML")
	}
	if !strings.Contains(out, "TestFoo") {
		t.Error("missing failure in HTML")
	}
	if !strings.Contains(out, "abc123") {
		t.Error("missing git hash in HTML")
	}
}
