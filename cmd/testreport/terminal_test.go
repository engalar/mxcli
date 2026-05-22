// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderTerminal_ShowsAllLayers(t *testing.T) {
	layers := map[string]*LayerSummary{
		"L1L2": {ID: "L1L2", Name: "L1/L2 Parser+Visitor", Priority: 1, Pass: 38, Fail: 0, Elapsed: 1.2},
		"L3": {ID: "L3", Name: "L3 Executor Mock", Priority: 2, Pass: 10, Fail: 2, Elapsed: 2.1,
			Failures: []FailureDetail{{Package: "pkg/executor", Test: "TestFoo", Output: "expected bar"}}},
	}
	benchDiff := ""
	var buf bytes.Buffer
	renderTerminal(&buf, layers, benchDiff, false)
	out := buf.String()

	if !strings.Contains(out, "L1/L2 Parser+Visitor") {
		t.Error("missing L1/L2 layer")
	}
	if !strings.Contains(out, "L3 Executor Mock") {
		t.Error("missing L3 layer")
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("missing FAIL indicator")
	}
	if !strings.Contains(out, "TestFoo") {
		t.Error("missing failure detail")
	}
}

func TestRenderTerminal_AllPass(t *testing.T) {
	layers := map[string]*LayerSummary{
		"L1L2": {ID: "L1L2", Name: "L1/L2 Parser+Visitor", Priority: 1, Pass: 142, Fail: 0, Elapsed: 1.2},
	}
	var buf bytes.Buffer
	renderTerminal(&buf, layers, "", false)
	out := buf.String()
	if strings.Contains(out, "FAIL") {
		t.Error("should not show FAIL when all pass")
	}
}
