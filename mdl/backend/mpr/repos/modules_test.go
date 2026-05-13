// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
)

const fixtureModuleCount = 8

func TestModuleRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewModuleRepository(w)
	got, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != fixtureModuleCount {
		t.Errorf("ListAll: got %d modules, want %d", len(got), fixtureModuleCount)
	}
}

func TestModuleRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, _ := r.ListUnitsByType(moduleTypeName)
	var firstID string
	for _, ref := range refs {
		if ref.Type == moduleTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no modules")
	}
	repo := NewModuleRepository(w)
	first, err := repo.Get(model.ID(firstID))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if first.ID() == "" {
		t.Error("decoded module has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded module has empty Name")
	}
	if first.TypeName() != moduleTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), moduleTypeName)
	}
}

func TestModuleRepo_FindByName(t *testing.T) {
	w := openTestWriter(t)
	repo := NewModuleRepository(w)
	got, err := repo.FindByName("MyFirstModule")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if got == nil {
		t.Fatal("FindByName(MyFirstModule): want match, got nil")
	}
	if got.Name() != "MyFirstModule" {
		t.Errorf("FindByName returned %q", got.Name())
	}

	miss, err := repo.FindByName("DoesNotExist_xyz")
	if err != nil {
		t.Fatalf("FindByName(miss): %v", err)
	}
	if miss != nil {
		t.Error("FindByName(miss): want nil, got value")
	}
}

// Note: Module units sit at the project root; they are not container
// children of any other module. The Update path is exercised via the
// fixture-backed roundtrip rather than a Create cycle (creating a
// fresh Module would also need a Project unit to point to it, which
// is out of scope for Stage 2.6).
func TestModuleRepo_UpdateRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	repo := NewModuleRepository(w)
	first, err := repo.FindByName("MyFirstModule")
	if err != nil || first == nil {
		t.Skip("fixture missing MyFirstModule")
	}

	if err := repo.Update(first); err != nil {
		t.Fatalf("Update: %v", err)
	}
	again, err := repo.Get(model.ID(first.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if again.Name() != "MyFirstModule" {
		t.Errorf("post-Update Name = %q, want MyFirstModule", again.Name())
	}
}
