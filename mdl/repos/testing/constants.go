// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genCo "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
)

// ConstantCreateCall captures arguments to ConstantWriter.Create.
type ConstantCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Constant        *genCo.Constant
}

// ConstantMoveCall captures arguments to ConstantWriter.Move.
type ConstantMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingConstantRepository records every call to its methods.
type RecordingConstantRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	Created      []ConstantCreateCall
	Updated      []*genCo.Constant
	Deleted      []model.ID
	Moved        []ConstantMoveCall

	GetFunc    func(model.ID) (*genCo.Constant, error)
	ListFunc   func(model.ID) ([]*genCo.Constant, error)
	CreateFunc func(ConstantCreateCall) error
	UpdateFunc func(*genCo.Constant) error
	DeleteFunc func(model.ID) error
	MoveFunc   func(ConstantMoveCall) error
}

var _ repos.ConstantRepository = (*RecordingConstantRepository)(nil)

func (m *RecordingConstantRepository) Get(id model.ID) (*genCo.Constant, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingConstantRepository) List(moduleID model.ID) ([]*genCo.Constant, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingConstantRepository) Create(parentUUID string, containmentName string, c *genCo.Constant) error {
	call := ConstantCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Constant: c}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingConstantRepository) Update(c *genCo.Constant) error {
	m.Updated = append(m.Updated, c)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(c)
	}
	return nil
}

func (m *RecordingConstantRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingConstantRepository) Move(id model.ID, newParentUUID string) error {
	call := ConstantMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
