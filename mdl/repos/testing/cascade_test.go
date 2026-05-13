// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/model"
)

func TestRecordingCascadeService_RecordsCalls(t *testing.T) {
	r := &RecordingCascadeService{}
	if err := r.DeleteModule(model.ID("mod-1")); err != nil {
		t.Fatalf("DeleteModule default should not error: %v", err)
	}
	if err := r.DeleteFolder(model.ID("folder-1")); err != nil {
		t.Fatalf("DeleteFolder default should not error: %v", err)
	}
	if got, want := len(r.DeleteModuleCalls), 1; got != want {
		t.Errorf("DeleteModuleCalls len = %d, want %d", got, want)
	}
	if got, want := len(r.DeleteFolderCalls), 1; got != want {
		t.Errorf("DeleteFolderCalls len = %d, want %d", got, want)
	}
	if got, want := r.DeleteModuleCalls[0], model.ID("mod-1"); got != want {
		t.Errorf("DeleteModuleCalls[0] = %q, want %q", got, want)
	}
}

func TestRecordingCascadeService_FuncOverrides(t *testing.T) {
	want := errors.New("simulated failure")
	r := &RecordingCascadeService{
		DeleteModuleFunc: func(id model.ID) error { return want },
	}
	if err := r.DeleteModule(model.ID("mod-1")); err != want {
		t.Errorf("DeleteModule err = %v, want %v", err, want)
	}
	if len(r.DeleteModuleCalls) != 1 {
		t.Errorf("call still recorded even when Func overrides")
	}
}
