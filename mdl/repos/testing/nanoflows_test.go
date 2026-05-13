// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestRecordingNanoflowRepository_RecordsCreate(t *testing.T) {
	rec := &RecordingNanoflowRepository{}
	nf := genMf.NewNanoflow()
	if err := rec.Create("parent-uuid", "Documents", nf); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(rec.Created) != 1 {
		t.Fatalf("Created length = %d, want 1", len(rec.Created))
	}
	got := rec.Created[0]
	if got.ParentUUID != "parent-uuid" {
		t.Errorf("ParentUUID = %q, want parent-uuid", got.ParentUUID)
	}
	if got.ContainmentName != "Documents" {
		t.Errorf("ContainmentName = %q, want Documents", got.ContainmentName)
	}
	if got.Nanoflow != nf {
		t.Errorf("Nanoflow != input pointer")
	}
}

func TestRecordingNanoflowRepository_FuncOverride(t *testing.T) {
	wantErr := errors.New("inject")
	rec := &RecordingNanoflowRepository{
		CreateFunc: func(NanoflowCreateCall) error { return wantErr },
	}
	err := rec.Create("p", "c", genMf.NewNanoflow())
	if !errors.Is(err, wantErr) {
		t.Errorf("Create err = %v, want %v", err, wantErr)
	}
	if len(rec.Created) != 1 {
		t.Errorf("call still recorded? Created length = %d, want 1", len(rec.Created))
	}
}

func TestRecordingNanoflowRepository_RecordsAllCalls(t *testing.T) {
	rec := &RecordingNanoflowRepository{}
	_, _ = rec.Get("id-1")
	_, _ = rec.Get("id-2")
	_, _ = rec.List("mod-1")
	_ = rec.Update(genMf.NewNanoflow())
	_ = rec.Delete("del-1")
	_ = rec.Move("mv-1", "newparent")

	if len(rec.GotIDs) != 2 {
		t.Errorf("GotIDs = %d, want 2", len(rec.GotIDs))
	}
	if len(rec.ListedModule) != 1 || rec.ListedModule[0] != model.ID("mod-1") {
		t.Errorf("ListedModule = %v", rec.ListedModule)
	}
	if len(rec.Updated) != 1 {
		t.Errorf("Updated = %d, want 1", len(rec.Updated))
	}
	if len(rec.Deleted) != 1 || rec.Deleted[0] != model.ID("del-1") {
		t.Errorf("Deleted = %v", rec.Deleted)
	}
	if len(rec.Moved) != 1 || rec.Moved[0].ID != model.ID("mv-1") || rec.Moved[0].NewParentUUID != "newparent" {
		t.Errorf("Moved = %+v", rec.Moved)
	}
}
