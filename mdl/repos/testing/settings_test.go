// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genSet "github.com/mendixlabs/mxcli/modelsdk/gen/settings"
)

func TestRecordingProjectSettingsRepository_RecordsAll(t *testing.T) {
	rec := &RecordingProjectSettingsRepository{}
	_, _ = rec.Get()
	_, _ = rec.Get()
	if err := rec.Update(genSet.NewProjectSettings()); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rec.GetCalls != 2 {
		t.Errorf("GetCalls = %d, want 2", rec.GetCalls)
	}
	if len(rec.Updated) != 1 {
		t.Errorf("Updated = %d, want 1", len(rec.Updated))
	}
}

func TestRecordingProjectSettingsRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingProjectSettingsRepository{
		UpdateFunc: func(*genSet.ProjectSettings) error { return want },
	}
	err := rec.Update(genSet.NewProjectSettings())
	if !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
}

func TestRecordingModuleSettingsRepository_RecordsAll(t *testing.T) {
	rec := &RecordingModuleSettingsRepository{}
	_, _ = rec.Get("mod-1")
	if err := rec.Update("mod-1", element.Element(nil)); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(rec.GotIDs) != 1 || rec.GotIDs[0] != model.ID("mod-1") {
		t.Errorf("GotIDs = %v", rec.GotIDs)
	}
	if len(rec.Updated) != 1 || rec.Updated[0].ModuleID != model.ID("mod-1") {
		t.Errorf("Updated = %v", rec.Updated)
	}
}
