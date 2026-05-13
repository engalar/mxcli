// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestRecordingDomainModelRepository_RecordsCreate(t *testing.T) {
	rec := &RecordingDomainModelRepository{}
	dm := genDm.NewDomainModel()
	if err := rec.Create("parent-uuid", "Documents", dm); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(rec.Created) != 1 || rec.Created[0].DomainModel != dm {
		t.Errorf("Created = %+v", rec.Created)
	}
}

func TestRecordingDomainModelRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingDomainModelRepository{
		CreateFunc: func(DomainModelCreateCall) error { return want },
	}
	err := rec.Create("p", "c", genDm.NewDomainModel())
	if !errors.Is(err, want) {
		t.Errorf("Create err = %v, want %v", err, want)
	}
}

func TestRecordingDomainModelRepository_RecordsAllCalls(t *testing.T) {
	rec := &RecordingDomainModelRepository{}
	_, _ = rec.Get("id-1")
	_, _ = rec.List("mod-1")
	_ = rec.Update(genDm.NewDomainModel())
	_ = rec.Delete("d-1")
	_ = rec.Move("m-1", "newp")

	if len(rec.GotIDs) != 1 || rec.GotIDs[0] != model.ID("id-1") {
		t.Errorf("GotIDs = %v", rec.GotIDs)
	}
	if len(rec.ListedModule) != 1 {
		t.Errorf("ListedModule = %v", rec.ListedModule)
	}
}
