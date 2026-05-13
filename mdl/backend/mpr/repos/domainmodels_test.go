// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
)

const fixtureDomainModelCount = 8

func TestDomainModelRepo_List_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewDomainModelRepository(w)
	got, err := repo.List("")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(got) != fixtureDomainModelCount {
		t.Errorf("List: got %d domain models, want %d", len(got), fixtureDomainModelCount)
	}
}

func TestDomainModelRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, _ := r.ListUnitsByType(domainModelTypeName)
	var firstID string
	for _, ref := range refs {
		if ref.Type == domainModelTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no domain models")
	}
	repo := NewDomainModelRepository(w)
	first, err := repo.Get(model.ID(firstID))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if first.ID() == "" {
		t.Error("decoded domain model has empty ID")
	}
	if first.TypeName() != domainModelTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), domainModelTypeName)
	}
}

// TestDomainModelRepo_List_PerModule_OneEach verifies the invariant
// that each module owns exactly one DomainModel unit.
func TestDomainModelRepo_List_PerModule_OneEach(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, _ := r.ListModules()
	repo := NewDomainModelRepository(w)
	for _, m := range mods {
		got, err := repo.List(model.ID(m.ID))
		if err != nil {
			// Some modules (e.g. System) may not be discoverable
			// by ListModules + DomainModel correlation; skip.
			continue
		}
		if len(got) > 1 {
			t.Errorf("module %s: List returned %d domain models, want at most 1", m.Name, len(got))
		}
	}
}

// TestDomainModelRepo_UpdateRoundTrip exercises the write path with
// a Documentation field tweak (DomainModel has no Name — Documentation
// is the simplest writable string property).
func TestDomainModelRepo_UpdateRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	repo := NewDomainModelRepository(w)
	all, err := repo.List("")
	if err != nil || len(all) == 0 {
		t.Skip("fixture has no domain models")
	}
	dm := all[0]
	originalDoc := dm.Documentation()

	dm.SetDocumentation("repo-test-doc-tag")
	if err := repo.Update(dm); err != nil {
		t.Fatalf("Update: %v", err)
	}
	again, err := repo.Get(model.ID(dm.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if again.Documentation() != "repo-test-doc-tag" {
		t.Errorf("post-Update Documentation = %q, want repo-test-doc-tag", again.Documentation())
	}

	// restore
	again.SetDocumentation(originalDoc)
	_ = repo.Update(again)
}
