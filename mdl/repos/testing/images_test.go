// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genIm "github.com/mendixlabs/mxcli/modelsdk/gen/images"
)

func TestRecordingImageRepository_RecordsAll(t *testing.T) {
	rec := &RecordingImageRepository{}
	_, _ = rec.GetImage("img-1")
	_, _ = rec.GetCollection("col-1")
	_, _ = rec.ListCollections("mod-1")
	_ = rec.CreateCollection("p", "c", genIm.NewImageCollection())
	_ = rec.UpdateCollection(genIm.NewImageCollection())
	_ = rec.DeleteCollection("del-1")

	if len(rec.GotImageIDs) != 1 || rec.GotImageIDs[0] != model.ID("img-1") {
		t.Errorf("GotImageIDs = %v", rec.GotImageIDs)
	}
	if len(rec.GotCollectionIDs) != 1 {
		t.Errorf("GotCollectionIDs = %v", rec.GotCollectionIDs)
	}
	if len(rec.ListedModule) != 1 {
		t.Errorf("ListedModule = %v", rec.ListedModule)
	}
	if len(rec.CreatedCollections) != 1 || rec.CreatedCollections[0].ParentUUID != "p" {
		t.Errorf("CreatedCollections = %v", rec.CreatedCollections)
	}
	if len(rec.UpdatedCollections) != 1 {
		t.Errorf("UpdatedCollections = %d", len(rec.UpdatedCollections))
	}
	if len(rec.DeletedCollections) != 1 || rec.DeletedCollections[0] != model.ID("del-1") {
		t.Errorf("DeletedCollections = %v", rec.DeletedCollections)
	}
}

func TestRecordingImageRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingImageRepository{
		CreateCollectionFunc: func(ImageCollectionCreateCall) error { return want },
	}
	err := rec.CreateCollection("p", "c", genIm.NewImageCollection())
	if !errors.Is(err, want) {
		t.Errorf("CreateCollection err = %v, want %v", err, want)
	}
}
