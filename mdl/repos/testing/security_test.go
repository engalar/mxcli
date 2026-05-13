// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

func TestRecordingSecurityRepository_RecordsAll(t *testing.T) {
	rec := &RecordingSecurityRepository{}
	_, _ = rec.Get()
	_, _ = rec.GetModuleSecurity("mod-1")
	_ = rec.Update(genSec.NewProjectSecurity())
	_ = rec.UpdateModuleSecurity("mod-1", genSec.NewModuleSecurity())

	if rec.GetCalls != 1 {
		t.Errorf("GetCalls = %d, want 1", rec.GetCalls)
	}
	if len(rec.GotModuleIDs) != 1 || rec.GotModuleIDs[0] != model.ID("mod-1") {
		t.Errorf("GotModuleIDs = %v", rec.GotModuleIDs)
	}
	if len(rec.Updated) != 1 {
		t.Errorf("Updated = %d, want 1", len(rec.Updated))
	}
	if len(rec.UpdatedModule) != 1 || rec.UpdatedModule[0].ModuleID != model.ID("mod-1") {
		t.Errorf("UpdatedModule = %v", rec.UpdatedModule)
	}
}

func TestRecordingSecurityRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingSecurityRepository{
		UpdateFunc: func(*genSec.ProjectSecurity) error { return want },
	}
	err := rec.Update(genSec.NewProjectSecurity())
	if !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
}
