//go:build poc
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"strings"
	"testing"
)

func TestCluster_GroupsBySlot(t *testing.T) {
	m := NewMiner()
	m.Records = []SlotRecord{
		{SlotPath: "IfStmt.Condition", SourceText: "$a = true"},
		{SlotPath: "IfStmt.Condition", SourceText: "$b = false"},
		{SlotPath: "IfStmt.Condition", SourceText: "$a = true"},
		{SlotPath: "ReturnStmt.Value", SourceText: "$x"},
	}
	sum := Cluster(m)
	if got := sum.SlotCount("IfStmt.Condition"); got != 3 {
		t.Errorf("IfStmt.Condition occurrences: got %d, want 3", got)
	}
	if got := sum.SlotCount("ReturnStmt.Value"); got != 1 {
		t.Errorf("ReturnStmt.Value occurrences: got %d, want 1", got)
	}
	samples := sum.SlotSamples("IfStmt.Condition", 5)
	if len(samples) != 2 {
		t.Errorf("unique samples: got %d, want 2", len(samples))
	}
}

func TestEmit_WritesValidGoFile(t *testing.T) {
	m := NewMiner()
	m.Records = []SlotRecord{
		{SlotPath: "IfStmt.Condition", SourceText: "$x = true"},
	}
	sum := Cluster(m)
	out := t.TempDir() + "/mined.go"
	if err := Emit(sum, out); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), `"IfStmt.Condition"`) {
		t.Errorf("expected IfStmt.Condition in emitted output; got %s", data)
	}
}
