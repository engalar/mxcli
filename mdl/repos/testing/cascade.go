// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
)

// RecordingCascadeService is a Recording mock for repos.CascadeService.
// Records every call's arguments; tests can inject Func overrides to
// simulate failures or specific behaviours.
type RecordingCascadeService struct {
	DeleteModuleCalls []model.ID
	DeleteFolderCalls []model.ID

	DeleteModuleFunc func(moduleID model.ID) error
	DeleteFolderFunc func(folderID model.ID) error
}

var _ repos.CascadeService = (*RecordingCascadeService)(nil)

func (r *RecordingCascadeService) DeleteModule(moduleID model.ID) error {
	r.DeleteModuleCalls = append(r.DeleteModuleCalls, moduleID)
	if r.DeleteModuleFunc != nil {
		return r.DeleteModuleFunc(moduleID)
	}
	return nil
}

func (r *RecordingCascadeService) DeleteFolder(folderID model.ID) error {
	r.DeleteFolderCalls = append(r.DeleteFolderCalls, folderID)
	if r.DeleteFolderFunc != nil {
		return r.DeleteFolderFunc(folderID)
	}
	return nil
}
