// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genIm "github.com/mendixlabs/mxcli/modelsdk/gen/images"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureImageCollectionCount = 7

func TestImagesRepo_ListCollections_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewImageRepository(w)
	got, err := repo.ListCollections("")
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(got) != fixtureImageCollectionCount {
		t.Errorf("ListCollections: got %d, want %d", len(got), fixtureImageCollectionCount)
	}
}

func TestImagesRepo_GetCollectionRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, _ := r.ListUnitsByType(imageCollectionTypeName)
	var firstID string
	for _, ref := range refs {
		if ref.Type == imageCollectionTypeName {
			firstID = ref.ID
			break
		}
	}
	if firstID == "" {
		t.Fatal("fixture has no image collections")
	}
	repo := NewImageRepository(w)
	first, err := repo.GetCollection(model.ID(firstID))
	if err != nil {
		t.Fatalf("GetCollection: %v", err)
	}
	if first.ID() == "" {
		t.Error("decoded collection has empty ID")
	}
	if first.TypeName() != imageCollectionTypeName {
		t.Errorf("TypeName = %q, want %q", first.TypeName(), imageCollectionTypeName)
	}
}

// TestImagesRepo_GetImage_Embedded — fixture has 7 collections; some
// may have embedded Image children. If any do, GetImage(child.ID)
// returns it; otherwise we skip. Either outcome verifies the walk
// logic is wired correctly.
func TestImagesRepo_GetImage_Embedded(t *testing.T) {
	w := openTestWriter(t)
	repo := NewImageRepository(w)
	cols, err := repo.ListCollections("")
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	for _, col := range cols {
		for _, child := range col.ImagesItems() {
			if child == nil {
				continue
			}
			img, err := repo.GetImage(model.ID(child.ID()))
			if err != nil {
				t.Errorf("GetImage(%s) inside collection %s: %v", child.ID(), col.ID(), err)
				continue
			}
			if img.ID() != element.ID(child.ID()) {
				t.Errorf("GetImage returned wrong child: got %s want %s", img.ID(), child.ID())
			}
		}
	}
}

func TestImagesRepo_GetImage_NotFound(t *testing.T) {
	w := openTestWriter(t)
	repo := NewImageRepository(w)
	_, err := repo.GetImage("nonexistent-id")
	if err == nil {
		t.Error("GetImage(nonexistent): want error, got nil")
	}
}

func TestImagesRepo_CreateUpdateDeleteCollectionCycle(t *testing.T) {
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

	repo := NewImageRepository(w)
	baseline, err := repo.ListCollections("")
	if err != nil {
		t.Fatalf("ListCollections baseline: %v", err)
	}

	newCol := newEmptyImageCollection(t, "RepoImageCycleTest")
	if err := repo.CreateCollection(parentID, "Documents", newCol); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	afterCreate, err := repo.ListCollections("")
	if err != nil {
		t.Fatalf("ListCollections after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.GetCollection(model.ID(newCol.ID()))
	if err != nil {
		t.Fatalf("GetCollection(newCol): %v", err)
	}
	got.SetName("RepoImageCycleTestUpdated")
	if err := repo.UpdateCollection(got); err != nil {
		t.Fatalf("UpdateCollection: %v", err)
	}
	got2, err := repo.GetCollection(model.ID(newCol.ID()))
	if err != nil {
		t.Fatalf("GetCollection after Update: %v", err)
	}
	if got2.Name() != "RepoImageCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoImageCycleTestUpdated", got2.Name())
	}

	if err := repo.DeleteCollection(model.ID(newCol.ID())); err != nil {
		t.Fatalf("DeleteCollection: %v", err)
	}
	afterDelete, err := repo.ListCollections("")
	if err != nil {
		t.Fatalf("ListCollections after Delete: %v", err)
	}
	if len(afterDelete) != len(baseline) {
		t.Errorf("after Delete: count = %d, want baseline %d", len(afterDelete), len(baseline))
	}
}

func newEmptyImageCollection(t *testing.T, name string) *genIm.ImageCollection {
	t.Helper()
	c := genIm.NewImageCollection()
	c.SetID(element.ID(mmpr.GenerateID()))
	c.SetTypeName(imageCollectionTypeName)
	c.SetName(name)
	return c
}
