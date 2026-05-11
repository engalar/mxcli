// SPDX-License-Identifier: Apache-2.0

package hints

import "testing"

func TestRegistry_HasE001(t *testing.T) {
	e, ok := Registry.Lookup("E001")
	if !ok {
		t.Fatal("E001 not registered")
	}
	if e.Slug != "enum-string-mismatch" {
		t.Fatalf("E001 slug = %q, want enum-string-mismatch", e.Slug)
	}
	if e.HowToFix == "" || e.WhyWrong == "" {
		t.Fatal("E001 missing prose fields")
	}
	if len(e.Examples) == 0 {
		t.Fatal("E001 missing examples")
	}
}
