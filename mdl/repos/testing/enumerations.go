// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genEn "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
)

// EnumerationCreateCall captures arguments to EnumerationWriter.Create.
type EnumerationCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Enumeration     *genEn.Enumeration
}

// EnumerationMoveCall captures arguments to EnumerationWriter.Move.
type EnumerationMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingEnumerationRepository records every call to its methods.
type RecordingEnumerationRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	Created      []EnumerationCreateCall
	Updated      []*genEn.Enumeration
	Deleted      []model.ID
	Moved        []EnumerationMoveCall

	GetFunc    func(model.ID) (*genEn.Enumeration, error)
	ListFunc   func(model.ID) ([]*genEn.Enumeration, error)
	CreateFunc func(EnumerationCreateCall) error
	UpdateFunc func(*genEn.Enumeration) error
	DeleteFunc func(model.ID) error
	MoveFunc   func(EnumerationMoveCall) error
}

var _ repos.EnumerationRepository = (*RecordingEnumerationRepository)(nil)

func (m *RecordingEnumerationRepository) Get(id model.ID) (*genEn.Enumeration, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingEnumerationRepository) List(moduleID model.ID) ([]*genEn.Enumeration, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingEnumerationRepository) Create(parentUUID string, containmentName string, e *genEn.Enumeration) error {
	call := EnumerationCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Enumeration: e}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingEnumerationRepository) Update(e *genEn.Enumeration) error {
	m.Updated = append(m.Updated, e)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(e)
	}
	return nil
}

func (m *RecordingEnumerationRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingEnumerationRepository) Move(id model.ID, newParentUUID string) error {
	call := EnumerationMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
