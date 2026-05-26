// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// MappingCreateCall captures arguments to MappingWriter.Create.
type MappingCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Mapping         element.Element
}

// MappingMoveCall captures arguments to MappingWriter.Move.
type MappingMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingMappingRepository records every call to its methods.
type RecordingMappingRepository struct {
	GotIDs      []model.ID
	ListedTypes []string
	Created     []MappingCreateCall
	Updated     []element.Element
	Deleted     []model.ID
	Moved       []MappingMoveCall

	GetFunc        func(model.ID) (element.Element, error)
	ListByTypeFunc func(string) ([]element.Element, error)
	CreateFunc     func(MappingCreateCall) error
	UpdateFunc     func(element.Element) error
	DeleteFunc     func(model.ID) error
	MoveFunc       func(MappingMoveCall) error
}

var _ repos.MappingRepository = (*RecordingMappingRepository)(nil)

func (m *RecordingMappingRepository) Get(id model.ID) (element.Element, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingMappingRepository) ListByType(typeName string) ([]element.Element, error) {
	m.ListedTypes = append(m.ListedTypes, typeName)
	if m.ListByTypeFunc != nil {
		return m.ListByTypeFunc(typeName)
	}
	return nil, nil
}

func (m *RecordingMappingRepository) Create(parentUUID string, containmentName string, mapping element.Element) error {
	call := MappingCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Mapping: mapping}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingMappingRepository) Update(mapping element.Element) error {
	m.Updated = append(m.Updated, mapping)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(mapping)
	}
	return nil
}

func (m *RecordingMappingRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingMappingRepository) Move(id model.ID, newParentUUID string) error {
	call := MappingMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
