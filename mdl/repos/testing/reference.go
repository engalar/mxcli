// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// RenameCall captures the args of a ScanRename invocation.
type RenameCall struct{ OldQN, NewQN string }

// PatchNavigationCall captures the args of a PatchNavigationProfile invocation.
type PatchNavigationCall struct {
	NavDocID    model.ID
	ProfileName string
	Spec        types.NavigationProfileSpec
}

// EnumRefsUpdateCall captures the args of an UpdateEnumerationRefsInAllDomainModels invocation.
type EnumRefsUpdateCall struct{ OldQN, NewQN string }

// RecordingReferenceService is a Recording mock for repos.ReferenceService.
// Every call's arguments are captured; tests can inject Func overrides
// for deterministic responses or failure simulation.
type RecordingReferenceService struct {
	ScanRenameCalls            []RenameCall
	PatchNavigationCalls       []PatchNavigationCall
	UpdateEnumerationRefsCalls []EnumRefsUpdateCall

	ScanRenameFunc            func(oldQN, newQN string) ([]repos.RenameHit, error)
	PatchNavigationFunc       func(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error
	UpdateEnumerationRefsFunc func(oldQN, newQN string) error
}

var _ repos.ReferenceService = (*RecordingReferenceService)(nil)

func (r *RecordingReferenceService) ScanRename(oldQN, newQN string) ([]repos.RenameHit, error) {
	r.ScanRenameCalls = append(r.ScanRenameCalls, RenameCall{OldQN: oldQN, NewQN: newQN})
	if r.ScanRenameFunc != nil {
		return r.ScanRenameFunc(oldQN, newQN)
	}
	return nil, nil
}

func (r *RecordingReferenceService) PatchNavigationProfile(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error {
	r.PatchNavigationCalls = append(r.PatchNavigationCalls, PatchNavigationCall{
		NavDocID: navDocID, ProfileName: profileName, Spec: spec,
	})
	if r.PatchNavigationFunc != nil {
		return r.PatchNavigationFunc(navDocID, profileName, spec)
	}
	return nil
}

func (r *RecordingReferenceService) UpdateEnumerationRefsInAllDomainModels(oldQN, newQN string) error {
	r.UpdateEnumerationRefsCalls = append(r.UpdateEnumerationRefsCalls, EnumRefsUpdateCall{OldQN: oldQN, NewQN: newQN})
	if r.UpdateEnumerationRefsFunc != nil {
		return r.UpdateEnumerationRefsFunc(oldQN, newQN)
	}
	return nil
}
