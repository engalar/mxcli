// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureMicroflowCount = 17

func TestMicroflowRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMicroflowRepository(w)
	got, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != fixtureMicroflowCount {
		t.Errorf("ListAll: got %d microflows, want %d", len(got), fixtureMicroflowCount)
	}
}

func TestMicroflowRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, err := r.ListUnitsByType("Microflows$Microflow")
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("fixture has no microflows")
	}

	repo := NewMicroflowRepository(w)
	first, err := repo.Get(model.ID(refs[0].ID))
	if err != nil {
		t.Fatalf("Get(%s): %v", refs[0].ID, err)
	}
	if first.ID() == "" {
		t.Error("decoded microflow has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded microflow has empty Name")
	}
	if first.TypeName() != "Microflows$Microflow" {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), "Microflows$Microflow")
	}
}

// TestMicroflowRepo_CreateUpdateDeleteCycle exercises the full
// direct-mode write path against a known module's container UUID.
// Invariant: ListAll count returns to baseline after Delete.
func TestMicroflowRepo_CreateUpdateDeleteCycle(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, err := r.ListModules()
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}

	// Use a real module's UUID as parent so the new microflow chains up
	// correctly (FindByQualifiedName / List(moduleID) will see it).
	var parentID string
	for _, m := range mods {
		if m.Name == "MyFirstModule" {
			parentID = m.ID
			break
		}
	}
	if parentID == "" {
		t.Fatal("fixture missing MyFirstModule")
	}

	repo := NewMicroflowRepository(w)
	baseline, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll baseline: %v", err)
	}

	newMF := newEmptyMicroflow(t, "RepoCycleTest")
	if err := repo.Create(parentID, "Documents", newMF); err != nil {
		t.Fatalf("Create: %v", err)
	}

	afterCreate, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	// Verify Get sees it.
	got, err := repo.Get(model.ID(newMF.ID()))
	if err != nil {
		t.Fatalf("Get(newMF): %v", err)
	}
	if got.Name() != "RepoCycleTest" {
		t.Errorf("post-Create Name = %q, want RepoCycleTest", got.Name())
	}

	// Update.
	got.SetName("RepoCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newMF.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoCycleTestUpdated", got2.Name())
	}

	// Delete.
	if err := repo.Delete(model.ID(newMF.ID())); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	afterDelete, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll after Delete: %v", err)
	}
	if len(afterDelete) != len(baseline) {
		t.Errorf("after Delete: count = %d, want baseline %d", len(afterDelete), len(baseline))
	}
}

// TestMicroflowRepo_AutoCacheInvalidation guards against accidentally
// re-introducing manual ReaderCache.Invalidate calls — the underlying
// mmpr.Writer auto-invalidates on InsertUnit (addendum Blocker 4), so a
// Create followed by an immediate ListAll must see count + 1 with no
// explicit cache step in between.
func TestMicroflowRepo_AutoCacheInvalidation(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMicroflowRepository(w)
	baseline, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll baseline: %v", err)
	}

	mods, _ := w.ConcreteReader().ListModules()
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

	mf := newEmptyMicroflow(t, "AutoInvalidateProbe")
	if err := repo.Create(parentID, "Documents", mf); err != nil {
		t.Fatalf("Create: %v", err)
	}
	after, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll after Create: %v", err)
	}
	if len(after) != len(baseline)+1 {
		t.Errorf("auto-invalidation broken: post-Create count = %d, want %d", len(after), len(baseline)+1)
	}
}

// TestMicroflowRepo_FindByQualifiedName exercises the parser + walker
// against a freshly-created microflow whose QN we already know.
func TestMicroflowRepo_FindByQualifiedName(t *testing.T) {
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

	repo := NewMicroflowRepository(w)
	mf := newEmptyMicroflow(t, "QNLookupProbe")
	if err := repo.Create(parentID, "Documents", mf); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByQualifiedName("MyFirstModule.QNLookupProbe")
	if err != nil {
		t.Fatalf("FindByQualifiedName: %v", err)
	}
	if got == nil {
		t.Fatal("FindByQualifiedName returned nil for known microflow")
	}
	if got.ID() != mf.ID() {
		t.Errorf("FindByQualifiedName ID = %s, want %s", got.ID(), mf.ID())
	}
}

// TestMicroflowRepo_GetContainerUUID_RoundTrips proves the SQL-backed
// container-ID lookup returns the same parent that BuildContainerParent
// reports for every microflow in the fixture. This is the primitive
// genMicroflowQualifiedName uses to defeat BSON-roundtrip Container()
// loss.
func TestMicroflowRepo_GetContainerUUID_RoundTrips(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()

	parents, err := r.BuildContainerParent()
	if err != nil {
		t.Fatalf("BuildContainerParent: %v", err)
	}

	repo := NewMicroflowRepository(w)
	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("fixture has no microflows")
	}

	for _, mf := range all {
		got, err := repo.GetContainerUUID(model.ID(mf.ID()))
		if err != nil {
			t.Errorf("GetContainerUUID(%s) %s: %v", mf.Name(), mf.ID(), err)
			continue
		}
		want := parents[string(mf.ID())]
		if string(got) != want {
			t.Errorf("GetContainerUUID(%s): got %q, want %q", mf.Name(), got, want)
		}
	}
}

// TestMicroflowRepo_GetContainerUUID_NotFound — a UUID that doesn't
// match any unit must return an error (no silent empty value).
func TestMicroflowRepo_GetContainerUUID_NotFound(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMicroflowRepository(w)
	bogus := model.ID("00000000-0000-0000-0000-000000000000")
	if got, err := repo.GetContainerUUID(bogus); err == nil {
		t.Errorf("expected error for bogus UUID, got %q", got)
	}
}

// TestMicroflowRepo_IsRule_Negative — the Stage 2 fixture has no rules,
// so IsRule on any name must return false.
func TestMicroflowRepo_IsRule_Negative(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMicroflowRepository(w)
	got, err := repo.IsRule("MyFirstModule.SomeName")
	if err != nil {
		t.Fatalf("IsRule: %v", err)
	}
	if got {
		t.Error("IsRule returned true on a rule-less fixture")
	}
}

// newEmptyMicroflow returns a freshly-minted *genMf.Microflow with a
// fresh ID, the canonical type name, and the given simple name set.
// The container is positional: callers pass parentUUID into Create.
func newEmptyMicroflow(t *testing.T, name string) *genMf.Microflow {
	t.Helper()
	mf := genMf.NewMicroflow()
	mf.SetID(element.ID(mmpr.GenerateID()))
	mf.SetTypeName("Microflows$Microflow")
	mf.SetName(name)
	return mf
}
