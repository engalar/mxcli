// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

func TestRecordingServiceRepository_RecordsAll(t *testing.T) {
	rec := &RecordingServiceRepository{}
	_, _ = rec.Get("a-1")
	_, _ = rec.ListByType("Services$ConsumedODataService")
	_ = rec.Create("p", "c", element.Element(nil))
	_ = rec.Update(element.Element(nil))
	_ = rec.Delete("d-1")
	_ = rec.Move("m-1", "newp")

	if len(rec.GotIDs) != 1 || rec.GotIDs[0] != model.ID("a-1") {
		t.Errorf("GotIDs = %v", rec.GotIDs)
	}
	if len(rec.ListedTypes) != 1 || rec.ListedTypes[0] != "Services$ConsumedODataService" {
		t.Errorf("ListedTypes = %v", rec.ListedTypes)
	}
	if len(rec.Created) != 1 || rec.Created[0].ParentUUID != "p" {
		t.Errorf("Created = %v", rec.Created)
	}
	if len(rec.Updated) != 1 {
		t.Errorf("Updated = %v", rec.Updated)
	}
	if len(rec.Deleted) != 1 {
		t.Errorf("Deleted = %v", rec.Deleted)
	}
	if len(rec.Moved) != 1 || rec.Moved[0].ID != model.ID("m-1") {
		t.Errorf("Moved = %v", rec.Moved)
	}
}

func TestRecordingServiceRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingServiceRepository{
		CreateFunc: func(ServiceCreateCall) error { return want },
	}
	err := rec.Create("p", "c", element.Element(nil))
	if !errors.Is(err, want) {
		t.Errorf("Create err = %v, want %v", err, want)
	}
}
