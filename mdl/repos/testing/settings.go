// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genSet "github.com/mendixlabs/mxcli/modelsdk/gen/settings"
)

// RecordingProjectSettingsRepository records every call to its methods.
type RecordingProjectSettingsRepository struct {
	GetCalls    int
	Updated     []*genSet.ProjectSettings
	GetFunc     func() (*genSet.ProjectSettings, error)
	UpdateFunc  func(*genSet.ProjectSettings) error
}

var _ repos.ProjectSettingsRepository = (*RecordingProjectSettingsRepository)(nil)

func (m *RecordingProjectSettingsRepository) Get() (*genSet.ProjectSettings, error) {
	m.GetCalls++
	if m.GetFunc != nil {
		return m.GetFunc()
	}
	return nil, nil
}

func (m *RecordingProjectSettingsRepository) Update(s *genSet.ProjectSettings) error {
	m.Updated = append(m.Updated, s)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(s)
	}
	return nil
}

// ModuleSettingsUpdateCall captures arguments to ModuleSettingsWriter.Update.
type ModuleSettingsUpdateCall struct {
	ModuleID model.ID
	Element  element.Element
}

// RecordingModuleSettingsRepository records every call to its methods.
type RecordingModuleSettingsRepository struct {
	GotIDs     []model.ID
	Updated    []ModuleSettingsUpdateCall
	GetFunc    func(model.ID) (element.Element, error)
	UpdateFunc func(ModuleSettingsUpdateCall) error
}

var _ repos.ModuleSettingsRepository = (*RecordingModuleSettingsRepository)(nil)

func (m *RecordingModuleSettingsRepository) Get(id model.ID) (element.Element, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingModuleSettingsRepository) Update(id model.ID, e element.Element) error {
	call := ModuleSettingsUpdateCall{ModuleID: id, Element: e}
	m.Updated = append(m.Updated, call)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(call)
	}
	return nil
}
