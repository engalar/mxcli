// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const (
	// fixtureJavaActionCount counts MPR-stored Java actions only. The legacy
	// `show java actions` command additionally synthesizes System.VerifyPassword
	// from sdk/mpr.BuildSystemJavaActions, so user-visible counts are off by
	// one — that synthesis is wired in the Phase A read formatter, not at
	// the repo layer.
	fixtureJavaActionCount       = 2
	fixtureJavaScriptActionCount = 16
)

func TestJavaActionRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaActionRepository(w)
	got, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != fixtureJavaActionCount {
		t.Errorf("ListAll: got %d java actions, want %d", len(got), fixtureJavaActionCount)
	}
}

func TestJavaActionRepo_GetRoundTrip_DecodesGenType(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, err := r.ListUnitsByType(javaActionTypeName)
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("fixture has no java actions")
	}

	repo := NewJavaActionRepository(w)
	first, err := repo.Get(model.ID(refs[0].ID))
	if err != nil {
		t.Fatalf("Get(%s): %v", refs[0].ID, err)
	}
	if first.ID() == "" {
		t.Error("decoded java action has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded java action has empty Name")
	}
	if first.TypeName() != javaActionTypeName {
		t.Errorf("decoded java action TypeName = %q, want %q", first.TypeName(), javaActionTypeName)
	}
}

func TestJavaActionRepo_FindByQualifiedName(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaActionRepository(w)
	ja, err := repo.FindByQualifiedName("FeedbackModule.ValidateEmail")
	if err != nil {
		t.Fatalf("FindByQualifiedName: %v", err)
	}
	if ja == nil {
		t.Fatal("FindByQualifiedName returned nil for known fixture entry")
	}
	if ja.Name() != "ValidateEmail" {
		t.Errorf("Name = %q, want ValidateEmail", ja.Name())
	}
}

func TestJavaActionRepo_GetContainerUUID_NonEmpty(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaActionRepository(w)
	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("fixture has no java actions")
	}
	cid, err := repo.GetContainerUUID(model.ID(all[0].ID()))
	if err != nil {
		t.Fatalf("GetContainerUUID: %v", err)
	}
	if cid == "" {
		t.Error("GetContainerUUID returned empty container UUID")
	}
}

func TestJavaActionRepo_Create_NilElement_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaActionRepository(w)
	if err := repo.Create("", "", nil); err == nil {
		t.Error("Create(nil): expected error, got nil")
	}
	if err := repo.Update(nil); err == nil {
		t.Error("Update(nil): expected error, got nil")
	}
}

// TestJavaActionRepo_CreateUpdateDeleteCycle exercises the full
// direct-mode write path against a known module's container UUID.
// Mirrors the nanoflow cycle test (Phase D template).
func TestJavaActionRepo_CreateUpdateDeleteCycle(t *testing.T) {
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

	repo := NewJavaActionRepository(w)
	baseline, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll baseline: %v", err)
	}

	newJA := newEmptyJavaAction(t, "RepoJavaActionCycleTest")
	if err := repo.Create(parentID, "Documents", newJA); err != nil {
		t.Fatalf("Create: %v", err)
	}

	afterCreate, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.Get(model.ID(newJA.ID()))
	if err != nil {
		t.Fatalf("Get(newJA): %v", err)
	}
	if got.Name() != "RepoJavaActionCycleTest" {
		t.Errorf("post-Create Name = %q, want RepoJavaActionCycleTest", got.Name())
	}

	got.SetName("RepoJavaActionCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newJA.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoJavaActionCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoJavaActionCycleTestUpdated", got2.Name())
	}

	if err := repo.Delete(model.ID(newJA.ID())); err != nil {
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

// newEmptyJavaAction returns a freshly-minted *genJA.JavaAction with a
// fresh ID, the canonical type name, and the given simple name set.
func newEmptyJavaAction(t *testing.T, name string) *genJA.JavaAction {
	t.Helper()
	ja := genJA.NewJavaAction()
	ja.SetID(element.ID(mmpr.GenerateID()))
	ja.SetTypeName(javaActionTypeName)
	ja.SetName(name)
	return ja
}

func TestJavaScriptActionRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaScriptActionRepository(w)
	got, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) < fixtureJavaScriptActionCount {
		t.Errorf("ListAll: got %d javascript actions, want >= %d", len(got), fixtureJavaScriptActionCount)
	}
}

func TestJavaScriptActionRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, err := r.ListUnitsByType(javaScriptActionTypeName)
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("fixture has no javascript actions")
	}
	repo := NewJavaScriptActionRepository(w)
	first, err := repo.Get(model.ID(refs[0].ID))
	if err != nil {
		t.Fatalf("Get(%s): %v", refs[0].ID, err)
	}
	if first.ID() == "" {
		t.Error("decoded javascript action has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded javascript action has empty Name")
	}
}
