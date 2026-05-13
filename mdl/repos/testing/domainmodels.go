// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// DomainModelCreateCall captures arguments to DomainModelWriter.Create.
type DomainModelCreateCall struct {
	ParentUUID      string
	ContainmentName string
	DomainModel     *genDm.DomainModel
}

// DomainModelMoveCall captures arguments to DomainModelWriter.Move.
type DomainModelMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingDomainModelRepository records every call to its methods.
type RecordingDomainModelRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	Created      []DomainModelCreateCall
	Updated      []*genDm.DomainModel
	Deleted      []model.ID
	Moved        []DomainModelMoveCall

	GetFunc    func(model.ID) (*genDm.DomainModel, error)
	ListFunc   func(model.ID) ([]*genDm.DomainModel, error)
	CreateFunc func(DomainModelCreateCall) error
	UpdateFunc func(*genDm.DomainModel) error
	DeleteFunc func(model.ID) error
	MoveFunc   func(DomainModelMoveCall) error
}

var _ repos.DomainModelRepository = (*RecordingDomainModelRepository)(nil)

func (m *RecordingDomainModelRepository) Get(id model.ID) (*genDm.DomainModel, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingDomainModelRepository) List(moduleID model.ID) ([]*genDm.DomainModel, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingDomainModelRepository) Create(parentUUID string, containmentName string, dm *genDm.DomainModel) error {
	call := DomainModelCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, DomainModel: dm}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingDomainModelRepository) Update(dm *genDm.DomainModel) error {
	m.Updated = append(m.Updated, dm)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(dm)
	}
	return nil
}

func (m *RecordingDomainModelRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingDomainModelRepository) Move(id model.ID, newParentUUID string) error {
	call := DomainModelMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
