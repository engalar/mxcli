// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureFolderCount = 80

func TestFolderRepo_List_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewFolderRepository(w)
	got, err := repo.List("")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(got) != fixtureFolderCount {
		t.Errorf("List: got %d folders, want %d", len(got), fixtureFolderCount)
	}
}

func TestFolderRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, _ := r.ListUnitsByType(folderTypeName)
	var firstID string
	for _, ref := range refs {
		if ref.Type == folderTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no folders")
	}
	repo := NewFolderRepository(w)
	first, err := repo.Get(model.ID(firstID))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if first.ID() == "" {
		t.Error("decoded folder has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded folder has empty Name")
	}
	if first.TypeName() != folderTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), folderTypeName)
	}
}

func TestFolderRepo_CreateUpdateDeleteCycle(t *testing.T) {
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

	repo := NewFolderRepository(w)
	baseline, err := repo.List("")
	if err != nil {
		t.Fatalf("List baseline: %v", err)
	}

	newF := newEmptyFolder(t, "RepoFolderCycleTest")
	if err := repo.Create(parentID, "Folders", newF); err != nil {
		t.Fatalf("Create: %v", err)
	}
	afterCreate, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.Get(model.ID(newF.ID()))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got.SetName("RepoFolderCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newF.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoFolderCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoFolderCycleTestUpdated", got2.Name())
	}

	if err := repo.Delete(model.ID(newF.ID())); err != nil {
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

func newEmptyFolder(t *testing.T, name string) *genPr.Folder {
	t.Helper()
	f := genPr.NewFolder()
	f.SetID(element.ID(mmpr.GenerateID()))
	f.SetTypeName(folderTypeName)
	f.SetName(name)
	return f
}
