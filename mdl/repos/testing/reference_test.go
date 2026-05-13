// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

func TestRecordingReferenceService_RecordsAllCalls(t *testing.T) {
	r := &RecordingReferenceService{}
	if _, err := r.ScanRename("Old.Name", "New.Name"); err != nil {
		t.Fatalf("ScanRename default: %v", err)
	}
	if err := r.PatchNavigationProfile(model.ID("nav-1"), "Web", types.NavigationProfileSpec{}); err != nil {
		t.Fatalf("PatchNavigationProfile default: %v", err)
	}
	if err := r.UpdateEnumerationRefsInAllDomainModels("M.OldEnum", "M.NewEnum"); err != nil {
		t.Fatalf("UpdateEnumerationRefs default: %v", err)
	}
	if got, want := len(r.ScanRenameCalls), 1; got != want {
		t.Errorf("ScanRenameCalls len = %d, want %d", got, want)
	}
	if got, want := len(r.PatchNavigationCalls), 1; got != want {
		t.Errorf("PatchNavigationCalls len = %d, want %d", got, want)
	}
	if got, want := len(r.UpdateEnumerationRefsCalls), 1; got != want {
		t.Errorf("UpdateEnumerationRefsCalls len = %d, want %d", got, want)
	}
	if got, want := r.ScanRenameCalls[0].OldQN, "Old.Name"; got != want {
		t.Errorf("ScanRenameCalls[0].OldQN = %q, want %q", got, want)
	}
}

func TestRecordingReferenceService_FuncOverride_ScanRename(t *testing.T) {
	wantHits := []repos.RenameHit{{UnitID: "u1", UnitType: "Microflows$Microflow", Name: "MyFlow", Count: 3}}
	r := &RecordingReferenceService{
		ScanRenameFunc: func(o, n string) ([]repos.RenameHit, error) { return wantHits, nil },
	}
	hits, err := r.ScanRename("a", "b")
	if err != nil {
		t.Fatalf("ScanRename: %v", err)
	}
	if len(hits) != 1 || hits[0].Count != 3 {
		t.Errorf("hits = %+v, want %+v", hits, wantHits)
	}
}

func TestRecordingReferenceService_FuncOverride_PatchNavigationError(t *testing.T) {
	want := errors.New("nav patch failed")
	r := &RecordingReferenceService{
		PatchNavigationFunc: func(_ model.ID, _ string, _ types.NavigationProfileSpec) error { return want },
	}
	if err := r.PatchNavigationProfile(model.ID("nav-1"), "Web", types.NavigationProfileSpec{}); err != want {
		t.Errorf("err = %v, want %v", err, want)
	}
}
