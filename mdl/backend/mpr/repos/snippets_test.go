// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureSnippetCount = 4

func TestSnippetRepo_List_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewSnippetRepository(w)
	got, err := repo.List("")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(got) != fixtureSnippetCount {
		t.Errorf("List: got %d snippets, want %d", len(got), fixtureSnippetCount)
	}
}

func TestSnippetRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, _ := r.ListUnitsByType(snippetTypeName)
	var firstID string
	for _, ref := range refs {
		if ref.Type == snippetTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no snippets")
	}
	repo := NewSnippetRepository(w)
	first, err := repo.Get(model.ID(firstID))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if first.ID() == "" {
		t.Error("decoded snippet has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded snippet has empty Name")
	}
	if first.TypeName() != snippetTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), snippetTypeName)
	}
}

func TestSnippetRepo_CreateUpdateDeleteCycle(t *testing.T) {
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

	repo := NewSnippetRepository(w)
	baseline, err := repo.List("")
	if err != nil {
		t.Fatalf("List baseline: %v", err)
	}

	newS := newEmptySnippet(t, "RepoSnippetCycleTest")
	if err := repo.Create(parentID, "Documents", newS); err != nil {
		t.Fatalf("Create: %v", err)
	}
	afterCreate, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.Get(model.ID(newS.ID()))
	if err != nil {
		t.Fatalf("Get(newS): %v", err)
	}
	got.SetName("RepoSnippetCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newS.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoSnippetCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoSnippetCycleTestUpdated", got2.Name())
	}

	if err := repo.Delete(model.ID(newS.ID())); err != nil {
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

func TestSnippetRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewSnippetRepository(w)
	got, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != fixtureSnippetCount {
		t.Errorf("ListAll: got %d snippets, want %d", len(got), fixtureSnippetCount)
	}
}

func TestSnippetRepo_FindByQualifiedName_FreshSnippet(t *testing.T) {
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

	repo := NewSnippetRepository(w)
	probe := newEmptySnippet(t, "QNSnippetProbe")
	if err := repo.Create(parentID, "Documents", probe); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByQualifiedName("MyFirstModule.QNSnippetProbe")
	if err != nil {
		t.Fatalf("FindByQualifiedName: %v", err)
	}
	if got == nil {
		t.Fatal("FindByQualifiedName returned nil for known snippet")
	}
	if got.ID() != probe.ID() {
		t.Errorf("FindByQualifiedName ID = %s, want %s", got.ID(), probe.ID())
	}
}

func TestSnippetRepo_GetContainerUUID_NonEmpty(t *testing.T) {
	w := openTestWriter(t)
	repo := NewSnippetRepository(w)
	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("fixture has no snippets")
	}
	cid, err := repo.GetContainerUUID(model.ID(all[0].ID()))
	if err != nil {
		t.Fatalf("GetContainerUUID: %v", err)
	}
	if cid == "" {
		t.Error("GetContainerUUID returned empty container UUID")
	}
}

func newEmptySnippet(t *testing.T, name string) *genPg.Snippet {
	t.Helper()
	s := genPg.NewSnippet()
	s.SetID(element.ID(mmpr.GenerateID()))
	s.SetTypeName(snippetTypeName)
	s.SetName(name)
	return s
}
