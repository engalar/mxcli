// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genEn "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureEnumerationCount = 7

func TestEnumerationRepo_List_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewEnumerationRepository(w)
	got, err := repo.List("")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(got) != fixtureEnumerationCount {
		t.Errorf("List: got %d enumerations, want %d", len(got), fixtureEnumerationCount)
	}
}

func TestEnumerationRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, _ := r.ListUnitsByType(enumerationTypeName)
	var firstID string
	for _, ref := range refs {
		if ref.Type == enumerationTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no enumerations")
	}
	repo := NewEnumerationRepository(w)
	first, err := repo.Get(model.ID(firstID))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if first.ID() == "" {
		t.Error("decoded enumeration has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded enumeration has empty Name")
	}
	if first.TypeName() != enumerationTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), enumerationTypeName)
	}
}

func TestEnumerationRepo_CreateUpdateDeleteCycle(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, _ := r.ListModules()
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

	repo := NewEnumerationRepository(w)
	baseline, err := repo.List("")
	if err != nil {
		t.Fatalf("List baseline: %v", err)
	}

	newE := newEmptyEnumeration(t, "RepoEnumCycleTest")
	if err := repo.Create(parentID, "Documents", newE); err != nil {
		t.Fatalf("Create: %v", err)
	}
	afterCreate, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.Get(model.ID(newE.ID()))
	if err != nil {
		t.Fatalf("Get(newE): %v", err)
	}
	got.SetName("RepoEnumCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newE.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoEnumCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoEnumCycleTestUpdated", got2.Name())
	}

	if err := repo.Delete(model.ID(newE.ID())); err != nil {
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

func newEmptyEnumeration(t *testing.T, name string) *genEn.Enumeration {
	t.Helper()
	e := genEn.NewEnumeration()
	e.SetID(element.ID(mmpr.GenerateID()))
	e.SetTypeName(enumerationTypeName)
	e.SetName(name)
	return e
}
