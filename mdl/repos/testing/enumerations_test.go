// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genEn "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
)

func TestRecordingEnumerationRepository_RecordsCreate(t *testing.T) {
	rec := &RecordingEnumerationRepository{}
	e := genEn.NewEnumeration()
	if err := rec.Create("parent-uuid", "Documents", e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(rec.Created) != 1 {
		t.Fatalf("Created length = %d, want 1", len(rec.Created))
	}
	if rec.Created[0].Enumeration != e {
		t.Errorf("Enumeration != input pointer")
	}
}

func TestRecordingEnumerationRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingEnumerationRepository{
		CreateFunc: func(EnumerationCreateCall) error { return want },
	}
	err := rec.Create("p", "c", genEn.NewEnumeration())
	if !errors.Is(err, want) {
		t.Errorf("Create err = %v, want %v", err, want)
	}
}

func TestRecordingEnumerationRepository_RecordsAllCalls(t *testing.T) {
	rec := &RecordingEnumerationRepository{}
	_, _ = rec.Get("id-1")
	_, _ = rec.List("mod-1")
	_ = rec.Update(genEn.NewEnumeration())
	_ = rec.Delete("d-1")
	_ = rec.Move("m-1", "newp")

	if len(rec.GotIDs) != 1 || rec.GotIDs[0] != model.ID("id-1") {
		t.Errorf("GotIDs = %v", rec.GotIDs)
	}
	if len(rec.ListedModule) != 1 {
		t.Errorf("ListedModule = %v", rec.ListedModule)
	}
}
