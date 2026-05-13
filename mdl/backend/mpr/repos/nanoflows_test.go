// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureNanoflowCount = 13

func TestNanoflowRepo_List_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewNanoflowRepository(w)
	got, err := repo.List("")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(got) != fixtureNanoflowCount {
		t.Errorf("List: got %d nanoflows, want %d", len(got), fixtureNanoflowCount)
	}
}

func TestNanoflowRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, err := r.ListUnitsByType(nanoflowTypeName)
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	// Pick the first ref whose Type is exactly nanoflowTypeName.
	var firstID string
	for _, ref := range refs {
		if ref.Type == nanoflowTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no nanoflows")
	}

	repo := NewNanoflowRepository(w)
	first, err := repo.Get(model.ID(firstID))
	if err != nil {
		t.Fatalf("Get(%s): %v", firstID, err)
	}
	if first.ID() == "" {
		t.Error("decoded nanoflow has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded nanoflow has empty Name")
	}
	if first.TypeName() != nanoflowTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), nanoflowTypeName)
	}
}

// TestNanoflowRepo_CreateUpdateDeleteCycle exercises the full
// direct-mode write path against a known module's container UUID.
func TestNanoflowRepo_CreateUpdateDeleteCycle(t *testing.T) {
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

	repo := NewNanoflowRepository(w)
	baseline, err := repo.List("")
	if err != nil {
		t.Fatalf("List baseline: %v", err)
	}

	newNF := newEmptyNanoflow(t, "RepoNanoflowCycleTest")
	if err := repo.Create(parentID, "Documents", newNF); err != nil {
		t.Fatalf("Create: %v", err)
	}

	afterCreate, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.Get(model.ID(newNF.ID()))
	if err != nil {
		t.Fatalf("Get(newNF): %v", err)
	}
	if got.Name() != "RepoNanoflowCycleTest" {
		t.Errorf("post-Create Name = %q, want RepoNanoflowCycleTest", got.Name())
	}

	got.SetName("RepoNanoflowCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newNF.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoNanoflowCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoNanoflowCycleTestUpdated", got2.Name())
	}

	if err := repo.Delete(model.ID(newNF.ID())); err != nil {
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

// newEmptyNanoflow returns a freshly-minted *genMf.Nanoflow with a fresh
// ID, the canonical type name, and the given simple name set.
func newEmptyNanoflow(t *testing.T, name string) *genMf.Nanoflow {
	t.Helper()
	nf := genMf.NewNanoflow()
	nf.SetID(element.ID(mmpr.GenerateID()))
	nf.SetTypeName(nanoflowTypeName)
	nf.SetName(name)
	return nf
}
