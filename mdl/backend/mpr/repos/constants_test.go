// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCo "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureConstantCount = 2

func TestConstantRepo_List_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewConstantRepository(w)
	got, err := repo.List("")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(got) != fixtureConstantCount {
		t.Errorf("List: got %d constants, want %d", len(got), fixtureConstantCount)
	}
}

func TestConstantRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, err := r.ListUnitsByType(constantTypeName)
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	var firstID string
	for _, ref := range refs {
		if ref.Type == constantTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no constants")
	}

	repo := NewConstantRepository(w)
	first, err := repo.Get(model.ID(firstID))
	if err != nil {
		t.Fatalf("Get(%s): %v", firstID, err)
	}
	if first.ID() == "" {
		t.Error("decoded constant has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded constant has empty Name")
	}
	if first.TypeName() != constantTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), constantTypeName)
	}
}

func TestConstantRepo_CreateUpdateDeleteCycle(t *testing.T) {
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

	repo := NewConstantRepository(w)
	baseline, err := repo.List("")
	if err != nil {
		t.Fatalf("List baseline: %v", err)
	}

	newC := newEmptyConstant(t, "RepoConstantCycleTest")
	if err := repo.Create(parentID, "Documents", newC); err != nil {
		t.Fatalf("Create: %v", err)
	}

	afterCreate, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.Get(model.ID(newC.ID()))
	if err != nil {
		t.Fatalf("Get(newC): %v", err)
	}
	if got.Name() != "RepoConstantCycleTest" {
		t.Errorf("post-Create Name = %q, want RepoConstantCycleTest", got.Name())
	}

	got.SetName("RepoConstantCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newC.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoConstantCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoConstantCycleTestUpdated", got2.Name())
	}

	if err := repo.Delete(model.ID(newC.ID())); err != nil {
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

func newEmptyConstant(t *testing.T, name string) *genCo.Constant {
	t.Helper()
	c := genCo.NewConstant()
	c.SetID(element.ID(mmpr.GenerateID()))
	c.SetTypeName(constantTypeName)
	c.SetName(name)
	return c
}
