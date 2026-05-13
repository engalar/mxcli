// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
)

func TestRecordingFolderRepository_RecordsCreate(t *testing.T) {
	rec := &RecordingFolderRepository{}
	f := genPr.NewFolder()
	if err := rec.Create("parent-uuid", "Folders", f); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(rec.Created) != 1 || rec.Created[0].Folder != f {
		t.Errorf("Created = %+v", rec.Created)
	}
}

func TestRecordingFolderRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingFolderRepository{
		CreateFunc: func(FolderCreateCall) error { return want },
	}
	err := rec.Create("p", "c", genPr.NewFolder())
	if !errors.Is(err, want) {
		t.Errorf("Create err = %v, want %v", err, want)
	}
}

func TestRecordingFolderRepository_RecordsAllCalls(t *testing.T) {
	rec := &RecordingFolderRepository{}
	_, _ = rec.Get("id-1")
	_, _ = rec.List("mod-1")
	_ = rec.Update(genPr.NewFolder())
	_ = rec.Delete("d-1")
	_ = rec.Move("m-1", "newp")

	if len(rec.GotIDs) != 1 || rec.GotIDs[0] != model.ID("id-1") {
		t.Errorf("GotIDs = %v", rec.GotIDs)
	}
	if len(rec.ListedModule) != 1 {
		t.Errorf("ListedModule = %v", rec.ListedModule)
	}
}
