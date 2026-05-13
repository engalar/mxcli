// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
)

func TestRecordingModuleRepository_RecordsAll(t *testing.T) {
	rec := &RecordingModuleRepository{}
	_, _ = rec.Get("id-1")
	_, _ = rec.ListAll()
	_, _ = rec.ListAll()
	_, _ = rec.FindByName("Foo")
	_ = rec.Create("p", "c", genPr.NewModule())
	_ = rec.Update(genPr.NewModule())
	_ = rec.Delete("d-1")

	if len(rec.GotIDs) != 1 || rec.GotIDs[0] != model.ID("id-1") {
		t.Errorf("GotIDs = %v", rec.GotIDs)
	}
	if rec.ListAllCalls != 2 {
		t.Errorf("ListAllCalls = %d, want 2", rec.ListAllCalls)
	}
	if len(rec.FoundNames) != 1 || rec.FoundNames[0] != "Foo" {
		t.Errorf("FoundNames = %v", rec.FoundNames)
	}
	if len(rec.Created) != 1 {
		t.Errorf("Created = %v", rec.Created)
	}
	if len(rec.Updated) != 1 {
		t.Errorf("Updated = %d, want 1", len(rec.Updated))
	}
	if len(rec.Deleted) != 1 {
		t.Errorf("Deleted = %v", rec.Deleted)
	}
}

func TestRecordingModuleRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingModuleRepository{
		CreateFunc: func(ModuleCreateCall) error { return want },
	}
	err := rec.Create("p", "c", genPr.NewModule())
	if !errors.Is(err, want) {
		t.Errorf("Create err = %v, want %v", err, want)
	}
}
