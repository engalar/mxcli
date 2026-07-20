// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ModuleSecurityUpdateCall captures arguments to UpdateModuleSecurity.
type ModuleSecurityUpdateCall struct {
	ModuleID model.ID
	Security *genSec.ModuleSecurity
}

// RecordingSecurityRepository records every call to its methods.
type RecordingSecurityRepository struct {
	GetCalls            int
	GotModuleIDs        []model.ID
	Updated             []*genSec.ProjectSecurity
	UpdatedModule       []ModuleSecurityUpdateCall
	GetFunc             func() (*genSec.ProjectSecurity, error)
	GetModuleSecFunc    func(model.ID) (*genSec.ModuleSecurity, error)
	UpdateFunc          func(*genSec.ProjectSecurity) error
	UpdateModuleSecFunc func(ModuleSecurityUpdateCall) error
	EnsureFunc          func() (*genSec.ProjectSecurity, error)
	EnsureCalls         int
}

var _ repos.SecurityRepository = (*RecordingSecurityRepository)(nil)

func (m *RecordingSecurityRepository) Get() (*genSec.ProjectSecurity, error) {
	m.GetCalls++
	if m.GetFunc != nil {
		return m.GetFunc()
	}
	return nil, nil
}

func (m *RecordingSecurityRepository) GetModuleSecurity(id model.ID) (*genSec.ModuleSecurity, error) {
	m.GotModuleIDs = append(m.GotModuleIDs, id)
	if m.GetModuleSecFunc != nil {
		return m.GetModuleSecFunc(id)
	}
	return nil, nil
}

func (m *RecordingSecurityRepository) Update(s *genSec.ProjectSecurity) error {
	m.Updated = append(m.Updated, s)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(s)
	}
	return nil
}

func (m *RecordingSecurityRepository) UpdateModuleSecurity(id model.ID, s *genSec.ModuleSecurity) error {
	call := ModuleSecurityUpdateCall{ModuleID: id, Security: s}
	m.UpdatedModule = append(m.UpdatedModule, call)
	if m.UpdateModuleSecFunc != nil {
		return m.UpdateModuleSecFunc(call)
	}
	return nil
}

func (m *RecordingSecurityRepository) Ensure() (*genSec.ProjectSecurity, error) {
	m.EnsureCalls++
	if m.EnsureFunc != nil {
		return m.EnsureFunc()
	}
	return m.Get()
}
