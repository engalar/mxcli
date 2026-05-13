// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genCo "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
)

func TestRecordingConstantRepository_RecordsCreate(t *testing.T) {
	rec := &RecordingConstantRepository{}
	c := genCo.NewConstant()
	if err := rec.Create("parent-uuid", "Documents", c); err != nil {
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
	if got.Constant != c {
		t.Errorf("Constant != input pointer")
	}
}

func TestRecordingConstantRepository_FuncOverride(t *testing.T) {
	wantErr := errors.New("inject")
	rec := &RecordingConstantRepository{
		CreateFunc: func(ConstantCreateCall) error { return wantErr },
	}
	err := rec.Create("p", "c", genCo.NewConstant())
	if !errors.Is(err, wantErr) {
		t.Errorf("Create err = %v, want %v", err, wantErr)
	}
	if len(rec.Created) != 1 {
		t.Errorf("call still recorded? Created length = %d, want 1", len(rec.Created))
	}
}

func TestRecordingConstantRepository_RecordsAllCalls(t *testing.T) {
	rec := &RecordingConstantRepository{}
	_, _ = rec.Get("id-1")
	_, _ = rec.List("mod-1")
	_ = rec.Update(genCo.NewConstant())
	_ = rec.Delete("del-1")
	_ = rec.Move("mv-1", "newparent")

	if len(rec.GotIDs) != 1 || rec.GotIDs[0] != model.ID("id-1") {
		t.Errorf("GotIDs = %v", rec.GotIDs)
	}
	if len(rec.ListedModule) != 1 {
		t.Errorf("ListedModule = %v", rec.ListedModule)
	}
	if len(rec.Updated) != 1 {
		t.Errorf("Updated = %d, want 1", len(rec.Updated))
	}
	if len(rec.Deleted) != 1 {
		t.Errorf("Deleted = %v", rec.Deleted)
	}
	if len(rec.Moved) != 1 {
		t.Errorf("Moved = %+v", rec.Moved)
	}
}
