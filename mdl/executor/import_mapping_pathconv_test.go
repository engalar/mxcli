// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// Regression guard for CE5015: the JSON-structure element Path and the import
// mapping's lookup/clone must use the SAME path convention (pipe separator +
// original lowercase JSON key). If they diverge, buildImportMappingElementModel
// fails to find the matching JS element, never clones its JsonPath, and Mendix
// mx check rejects the model with CE5015.
//
// This test builds JS elements from a snippet (the real producer), builds the
// path->element map exactly as the executor does, then runs the IM element
// builder and asserts the resulting JsonPath equals the canonical JS element
// Path it must have matched. b is nil (safe: object-kind elements don't touch
// the domain model, and resolveAttributeType guards nil). Pure unit test, no DB.
func TestPathConvention_JSAndImportMappingAgree(t *testing.T) {
	const jsonKey = "priority_no"
	// Canonical convention shared by both sides: "<parent>|<rawJsonKey>".
	const canonicalPath = "(Object)|" + jsonKey

	jsElems, err := types.BuildJsonElementsFromSnippet(
		`{"priority_no":1,"process_variant_rule_id":"x"}`, nil)
	if err != nil {
		t.Fatalf("BuildJsonElementsFromSnippet: %v", err)
	}

	jsMap := make(map[string]*types.JsonElement)
	buildJsonElementPathMap(jsElems, jsMap)

	if _, ok := jsMap[canonicalPath]; !ok {
		t.Fatalf("JS structure has no element at canonical path %q; keys=%v — "+
			"JS side violates the pipe+rawKey convention (CE5015 regression)",
			canonicalPath, keysOf(jsMap))
	}

	// Build the IM element for the same key as a child of the root object.
	def := &ast.ImportMappingElementDef{
		Attribute: "PriorityNo",
		JsonName:  jsonKey,
	}
	im := buildImportMappingElementModel("Sales", def, "Sales.Order", "(Object)", nil, jsMap, false)

	if im.JsonPath != canonicalPath {
		t.Fatalf("import mapping JsonPath = %q, want %q; "+
			"lookup/clone failed — path conventions have diverged (CE5015 regression)",
			im.JsonPath, canonicalPath)
	}
	if im.ExposedName == "" {
		t.Fatalf("import mapping ExposedName is empty — properties were not cloned from JS element (CE5015 regression)")
	}
}

func keysOf(m map[string]*types.JsonElement) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
