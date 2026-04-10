// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"
)

func TestExtractPageParamNames(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single param",
			text:     "CREATE PAGE Mod.Page (Params: { $Order: Mod.Order })",
			expected: []string{"Order"},
		},
		{
			name:     "multiple params",
			text:     "CREATE PAGE Mod.Page (\n  Params: { $Customer: Mod.Customer, $Helper: Mod.Helper }\n)",
			expected: []string{"Customer", "Helper"},
		},
		{
			name:     "no params",
			text:     "CREATE PAGE Mod.Page (Title: 'Test')",
			expected: nil,
		},
		{
			name:     "skip DECLARE variables",
			text:     "DECLARE $Temp String = '';\n$Order: Mod.Order",
			expected: []string{"Order"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPageParamNames(tt.text)
			if len(got) != len(tt.expected) {
				t.Errorf("extractPageParamNames() got %v, want %v", got, tt.expected)
				return
			}
			for i, name := range got {
				if name != tt.expected[i] {
					t.Errorf("extractPageParamNames()[%d] = %q, want %q", i, name, tt.expected[i])
				}
			}
		})
	}
}

func TestVariableCompletionItems(t *testing.T) {
	s := &mdlServer{}
	docText := "CREATE PAGE Mod.Page (\n  Params: { $Customer: Mod.Customer }\n) {\n  DATAVIEW dv1 (DataSource: $Customer) {\n"

	items := s.variableCompletionItems(docText, "$")
	if len(items) == 0 {
		t.Fatal("expected completion items for $ prefix")
	}

	// Should contain $currentObject
	foundCurrentObj := false
	foundCustomer := false
	for _, item := range items {
		if item.Label == "$currentObject" {
			foundCurrentObj = true
		}
		if item.Label == "$Customer" {
			foundCustomer = true
		}
	}
	if !foundCurrentObj {
		t.Error("expected $currentObject in completion items")
	}
	if !foundCustomer {
		t.Error("expected $Customer in completion items")
	}
}
