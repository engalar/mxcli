// SPDX-License-Identifier: Apache-2.0
//
// BSON field-order regression tests.
//
// Studio Pro's StreamingBsonUnitReader parses fields sequentially.
// If BSON keys appear in the wrong order, ResolvePostponedProperties
// fails with KeyNotFoundException because child-element $IDs aren't
// registered before they're referenced.
//
// These tests guard the property_order_overrides in
// internal/codegen/supplements.json — DO NOT change descriptor order
// without updating the override AND verifying against mx check.

package microflows

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
)

func TestDatabaseRetrieveSourceBSONFieldOrder(t *testing.T) {
	td, ok := codec.DefaultDescRegistry.Lookup("Microflows$DatabaseRetrieveSource")
	if !ok {
		t.Fatal("Microflows$DatabaseRetrieveSource not registered in DefaultDescRegistry")
	}

	// Studio Pro expects: Entity → NewSortings → Range → XpathConstraint.
	// Current (wrong):   Entity → Range → XpathConstraint → NewSortings.
	want := []string{"Entity", "NewSortings", "Range", "XpathConstraint"}

	got := make([]string, len(td.Properties))
	for i, p := range td.Properties {
		got[i] = p.BSONKey
	}

	if len(got) != len(want) {
		t.Fatalf("property count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("BSON field[%d] = %q, want %q\n  full order got:  %v\n  full order want: %v",
				i, got[i], want[i], got, want)
		}
	}
}
