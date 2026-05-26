// SPDX-License-Identifier: Apache-2.0

// TDD test: property_key_overrides in supplements.json must permanently
// capture ALL direct edits made to modelsdk/gen/ files. Without these
// entries, codegen re-runs would silently undo the fixes.
//
// Each test here guards one property_key_override. When a direct gen-file
// edit is discovered, add a failing test here FIRST, then add the entry
// to supplements.json and run codegen.

package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// supplementsJSON reads and parses the codegen supplements.json file.
func supplementsJSON(t *testing.T) map[string]any {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	path := filepath.Join(root, "internal", "codegen", "supplements.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read supplements.json: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse supplements.json: %v", err)
	}
	return m
}

func propertyKeyOverrides(t *testing.T) map[string]any {
	t.Helper()
	m := supplementsJSON(t)
	overrides, ok := m["property_key_overrides"].(map[string]any)
	if !ok {
		t.Fatal("supplements.json missing property_key_overrides")
	}
	return overrides
}

// TestSupplements_AttributeType_NewType guards the Mendix 11+ attribute type
// BSON key change. In Mendix 11, DomainModels$Attribute stores its type in
// the "NewType" field (not the legacy "Type"). Without this supplement, the
// next codegen run regenerates initAttribute() with "Type", breaking the fix.
//
// Direct gen edit: modelsdk/gen/domainmodels/types.go initAttribute() +
//
//	Attribute.InitFromRaw()
//
// Commit that added direct edit: d6897939
func TestSupplements_AttributeType_NewType(t *testing.T) {
	overrides := propertyKeyOverrides(t)
	got, ok := overrides["Attribute.type"]
	if !ok {
		t.Fatal("supplements.json missing Attribute.type override; " +
			"without it the next codegen run will revert the NewType fix " +
			"and break CE0117 attribute-type resolution")
	}
	if got != "NewType" {
		t.Fatalf("Attribute.type override = %q, want NewType", got)
	}
}
