// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
)

// ModuleCreateCall captures arguments to ModuleWriter.Create.
type ModuleCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Module          *genPr.Module
}

// RecordingModuleRepository records every call to its methods.
type RecordingModuleRepository struct {
	GotIDs       []model.ID
	ListAllCalls int
	FoundNames   []string
	Created      []ModuleCreateCall
	Updated      []*genPr.Module
	Deleted      []model.ID

	GetFunc        func(model.ID) (*genPr.Module, error)
	ListAllFunc    func() ([]*genPr.Module, error)
	FindByNameFunc func(string) (*genPr.Module, error)
	CreateFunc     func(ModuleCreateCall) error
	UpdateFunc     func(*genPr.Module) error
	DeleteFunc     func(model.ID) error
}

var _ repos.ModuleRepository = (*RecordingModuleRepository)(nil)

func (m *RecordingModuleRepository) Get(id model.ID) (*genPr.Module, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingModuleRepository) ListAll() ([]*genPr.Module, error) {
	m.ListAllCalls++
	if m.ListAllFunc != nil {
		return m.ListAllFunc()
	}
	return nil, nil
}

func (m *RecordingModuleRepository) FindByName(name string) (*genPr.Module, error) {
	m.FoundNames = append(m.FoundNames, name)
	if m.FindByNameFunc != nil {
		return m.FindByNameFunc(name)
	}
	return nil, nil
}

func (m *RecordingModuleRepository) Create(parentUUID string, containmentName string, mod *genPr.Module) error {
	call := ModuleCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Module: mod}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingModuleRepository) Update(mod *genPr.Module) error {
	m.Updated = append(m.Updated, mod)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(mod)
	}
	return nil
}

func (m *RecordingModuleRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}
