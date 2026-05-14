// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureLayoutCount = 22

func TestLayoutRepo_List_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewLayoutRepository(w)
	got, err := repo.List("")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(got) != fixtureLayoutCount {
		t.Errorf("List: got %d layouts, want %d", len(got), fixtureLayoutCount)
	}
}

func TestLayoutRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, err := r.ListUnitsByType(layoutTypeName)
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	var firstID string
	for _, ref := range refs {
		if ref.Type == layoutTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no layouts")
	}

	repo := NewLayoutRepository(w)
	first, err := repo.Get(model.ID(firstID))
	if err != nil {
		t.Fatalf("Get(%s): %v", firstID, err)
	}
	if first.ID() == "" {
		t.Error("decoded layout has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded layout has empty Name")
	}
	if first.TypeName() != layoutTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), layoutTypeName)
	}
}

func TestLayoutRepo_CreateUpdateDeleteCycle(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, err := r.ListModules()
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}

	var parentID string
	for _, m := range mods {
		if m.Name == "MyFirstModule" {
			parentID = m.ID
			break
		}
	}
	if parentID == "" {
		t.Skip("fixture missing MyFirstModule")
	}

	repo := NewLayoutRepository(w)
	baseline, err := repo.List("")
	if err != nil {
		t.Fatalf("List baseline: %v", err)
	}

	newL := newEmptyLayout(t, "RepoLayoutCycleTest")
	if err := repo.Create(parentID, "Documents", newL); err != nil {
		t.Fatalf("Create: %v", err)
	}

	afterCreate, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.Get(model.ID(newL.ID()))
	if err != nil {
		t.Fatalf("Get(newL): %v", err)
	}
	if got.Name() != "RepoLayoutCycleTest" {
		t.Errorf("post-Create Name = %q, want RepoLayoutCycleTest", got.Name())
	}

	got.SetName("RepoLayoutCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newL.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoLayoutCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoLayoutCycleTestUpdated", got2.Name())
	}

	if err := repo.Delete(model.ID(newL.ID())); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	afterDelete, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	if len(afterDelete) != len(baseline) {
		t.Errorf("after Delete: count = %d, want baseline %d", len(afterDelete), len(baseline))
	}
}

// TestLayoutRepo_ListAll_FixtureCount asserts the strict Forms$Layout
// filtering — without it ListAll could return additional Forms$* prefix
// matches.
func TestLayoutRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewLayoutRepository(w)
	got, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != fixtureLayoutCount {
		t.Errorf("ListAll: got %d layouts, want %d", len(got), fixtureLayoutCount)
	}
}

func TestLayoutRepo_FindByQualifiedName_FreshLayout(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, err := r.ListModules()
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	var parentID string
	for _, m := range mods {
		if m.Name == "MyFirstModule" {
			parentID = m.ID
			break
		}
	}
	if parentID == "" {
		t.Skip("fixture missing MyFirstModule")
	}

	repo := NewLayoutRepository(w)
	probe := newEmptyLayout(t, "QNLayoutProbe")
	if err := repo.Create(parentID, "Documents", probe); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByQualifiedName("MyFirstModule.QNLayoutProbe")
	if err != nil {
		t.Fatalf("FindByQualifiedName: %v", err)
	}
	if got == nil {
		t.Fatal("FindByQualifiedName returned nil for known layout")
	}
	if got.ID() != probe.ID() {
		t.Errorf("FindByQualifiedName ID = %s, want %s", got.ID(), probe.ID())
	}
}

func TestLayoutRepo_GetContainerUUID_NonEmpty(t *testing.T) {
	w := openTestWriter(t)
	repo := NewLayoutRepository(w)
	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("fixture has no layouts")
	}
	cid, err := repo.GetContainerUUID(model.ID(all[0].ID()))
	if err != nil {
		t.Fatalf("GetContainerUUID: %v", err)
	}
	if cid == "" {
		t.Error("GetContainerUUID returned empty container UUID")
	}
}

func newEmptyLayout(t *testing.T, name string) *genPg.Layout {
	t.Helper()
	l := genPg.NewLayout()
	l.SetID(element.ID(mmpr.GenerateID()))
	l.SetTypeName(layoutTypeName)
	l.SetName(name)
	return l
}
