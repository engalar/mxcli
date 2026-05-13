// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"strings"
	"testing"
)

func TestIDGeneratorNewID_FormatRoundTrip(t *testing.T) {
	g := NewIDGenerator()
	for i := 0; i < 8; i++ {
		id := g.NewID()
		s := string(id)
		if len(s) != 36 {
			t.Fatalf("expected 36-char UUID, got %d chars: %q", len(s), s)
		}
		if strings.Count(s, "-") != 4 {
			t.Fatalf("expected 4 dashes in UUID, got %d: %q", strings.Count(s, "-"), s)
		}
	}
	// Two consecutive IDs must differ.
	a, b := g.NewID(), g.NewID()
	if a == b {
		t.Fatalf("IDGenerator returned identical IDs back-to-back: %s", a)
	}
}

func TestReaderCacheInvalidate_DoesNotPanic(t *testing.T) {
	w := openTestWriter(t)
	c := NewReaderCache(w)
	c.Invalidate()
	c.InvalidateUnit("any-id")

	// Re-reading after invalidation must still succeed (the underlying
	// concrete reader rebuilds its cache lazily).
	r := w.ConcreteReader()
	if _, err := r.ListUnitsByType("Microflows$Microflow"); err != nil {
		t.Fatalf("ListUnitsByType after Invalidate: %v", err)
	}
}

func TestQualifiedNameResolver_ModuleNameByID_FromFixture(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, err := r.ListModules()
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(mods) == 0 {
		t.Fatal("fixture has no modules")
	}
	res := NewQualifiedNameResolver(w)
	for _, m := range mods {
		if m.ID == "" || m.Name == "" {
			continue // System module has placeholder data; skip
		}
		got, err := res.ModuleNameByID(typedID(m.ID))
		if err != nil {
			t.Fatalf("ModuleNameByID(%s): %v", m.ID, err)
		}
		if got != m.Name {
			t.Fatalf("ModuleNameByID(%s) = %q, want %q", m.ID, got, m.Name)
		}
	}
}

// TestResolveQualifiedName_FromFixture covers the four most common
// kinds (microflow, page, entity, layout) plus an enumeration if the
// fixture has any. Replaces the prior Stage-2-port-pending t.Skip.
func TestResolveQualifiedName_FromFixture(t *testing.T) {
	w := openTestWriter(t)
	res := NewQualifiedNameResolver(w)

	cases := []struct {
		qn       string
		wantKind string
	}{
		{"Administration.ChangeMyPassword", "microflow"},
		{"Atlas_Web_Content.ACT_Login", "nanoflow"},
		{"Administration.MyAccount", "page"},
		{"Administration.Account", "entity"},
		{"Atlas_Core.Atlas_Default", "layout"},
		{"Administration.ReadMe", "snippet"},
	}
	for _, tc := range cases {
		id, kind, err := res.ResolveQualifiedName(tc.qn)
		if err != nil {
			t.Errorf("ResolveQualifiedName(%q): unexpected error: %v", tc.qn, err)
			continue
		}
		if kind != tc.wantKind {
			t.Errorf("ResolveQualifiedName(%q) kind = %q, want %q", tc.qn, kind, tc.wantKind)
		}
		if id == "" {
			t.Errorf("ResolveQualifiedName(%q) id = empty", tc.qn)
		}
	}
}

// TestResolveQualifiedName_NotFound asserts a fabricated QN whose
// module exists but whose simple name does not yields an explicit
// not-found error rather than silently returning empty.
func TestResolveQualifiedName_NotFound(t *testing.T) {
	w := openTestWriter(t)
	res := NewQualifiedNameResolver(w)

	id, kind, err := res.ResolveQualifiedName("Administration.NoSuchThing_zzzz_42")
	if err == nil {
		t.Fatalf("expected not-found error, got id=%q kind=%q", id, kind)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want substring \"not found\"", err)
	}
	if id != "" || kind != "" {
		t.Errorf("on not-found want empty id/kind, got id=%q kind=%q", id, kind)
	}
}

// TestResolveQualifiedName_InvalidQN asserts that a QN missing the
// "Module.Simple" structure surfaces an explicit malformed error.
func TestResolveQualifiedName_InvalidQN(t *testing.T) {
	w := openTestWriter(t)
	res := NewQualifiedNameResolver(w)

	for _, bad := range []string{"BadFormat", "", ".LeadingDot", "TrailingDot."} {
		id, kind, err := res.ResolveQualifiedName(bad)
		if err == nil {
			t.Errorf("ResolveQualifiedName(%q) want error, got id=%q kind=%q", bad, id, kind)
			continue
		}
		if !strings.Contains(err.Error(), "invalid qualified name") {
			t.Errorf("ResolveQualifiedName(%q) error = %v, want substring \"invalid qualified name\"", bad, err)
		}
	}
}
