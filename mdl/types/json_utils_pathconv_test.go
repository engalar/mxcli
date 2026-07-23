// SPDX-License-Identifier: Apache-2.0

package types

import "testing"

// Regression guard for CE5015 (mx check failure on generated mappings).
//
// Root cause history: the committed path convention is "<parent>|<rawJsonKey>"
// (pipe separator + original lowercase JSON key). Both the JSON-structure side
// (BuildJsonElementsFromSnippet) and the import/export mapping side
// (buildImport/ExportMappingElementModel) must agree on this exact convention,
// because the IM/EM builder constructs a lookupPath and uses it to find the
// matching JSON-structure element and clone its JsonPath. A regression changed
// the JS side to "/"+capitalized-ExposedName while the lookup side used
// "/"+lowercase-JsonName, so the lookup missed, JsonPath was never cloned, and
// Mendix rejected the model with CE5015.
//
// This test pins the JS-structure path convention (pipe + original lowercase
// key). Do NOT relax it to "/" or capitalized names without also proving the
// IM/EM lookup path stays in lockstep (see TestPathConvention_JSAndImportMappingAgree).
func TestBuildJsonElementsFromSnippet_PathUsesPipeAndRawKey(t *testing.T) {
	elems, err := BuildJsonElementsFromSnippet(
		`{"priority_no":1,"process_variant_rule_id":"x"}`, nil)
	if err != nil {
		t.Fatalf("BuildJsonElementsFromSnippet returned error: %v", err)
	}
	if len(elems) != 1 {
		t.Fatalf("expected 1 root element, got %d", len(elems))
	}
	root := elems[0]
	if root.Path != "(Object)" {
		t.Fatalf("root Path = %q, want %q", root.Path, "(Object)")
	}

	want := map[string]bool{
		"(Object)|priority_no":             false,
		"(Object)|process_variant_rule_id": false,
	}
	for _, child := range root.Children {
		if _, ok := want[child.Path]; !ok {
			t.Errorf("unexpected child Path %q (separator must be '|' and key must stay original lowercase)", child.Path)
			continue
		}
		want[child.Path] = true
	}
	for path, seen := range want {
		if !seen {
			t.Errorf("missing expected child Path %q", path)
		}
	}
}
