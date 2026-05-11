// SPDX-License-Identifier: Apache-2.0

package hints

import "testing"

func TestRegistry_AllCodesPresent(t *testing.T) {
	want := []string{"E001", "E002", "E003", "E004", "E005", "E006", "E007", "E008", "E009", "E010"}
	for _, c := range want {
		e, ok := Registry.Lookup(c)
		if !ok {
			t.Errorf("%s missing", c)
			continue
		}
		if e.Slug == "" || e.Trigger == "" || e.WhyWrong == "" || e.HowToFix == "" || len(e.Examples) == 0 {
			t.Errorf("%s incomplete: %+v", c, e)
		}
	}
}

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
