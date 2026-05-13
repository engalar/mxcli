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

// TestQualifiedNameResolver_ResolveQualifiedName_PortPending guards the
// stubbed Stage 2 implementation. Once the body is ported, replace
// t.Skip with a real round-trip assertion against a known fixture name.
func TestQualifiedNameResolver_ResolveQualifiedName_PortPending(t *testing.T) {
	t.Skip("resolver port pending: ResolveQualifiedName body stubbed in Stage 2; finish before MicroflowRepo or executor consumes Names")
}
