// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// ServiceCreateCall captures arguments to ServiceWriter.Create.
type ServiceCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Service         element.Element
}

// ServiceMoveCall captures arguments to ServiceWriter.Move.
type ServiceMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingServiceRepository records every call to its methods.
type RecordingServiceRepository struct {
	GotIDs      []model.ID
	ListedTypes []string
	Created     []ServiceCreateCall
	Updated     []element.Element
	Deleted     []model.ID
	Moved       []ServiceMoveCall

	GetFunc        func(model.ID) (element.Element, error)
	ListByTypeFunc func(string) ([]element.Element, error)
	CreateFunc     func(ServiceCreateCall) error
	UpdateFunc     func(element.Element) error
	DeleteFunc     func(model.ID) error
	MoveFunc       func(ServiceMoveCall) error
}

var _ repos.ServiceRepository = (*RecordingServiceRepository)(nil)

func (m *RecordingServiceRepository) Get(id model.ID) (element.Element, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingServiceRepository) ListByType(typeName string) ([]element.Element, error) {
	m.ListedTypes = append(m.ListedTypes, typeName)
	if m.ListByTypeFunc != nil {
		return m.ListByTypeFunc(typeName)
	}
	return nil, nil
}

func (m *RecordingServiceRepository) Create(parentUUID string, containmentName string, svc element.Element) error {
	call := ServiceCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Service: svc}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingServiceRepository) Update(svc element.Element) error {
	m.Updated = append(m.Updated, svc)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(svc)
	}
	return nil
}

func (m *RecordingServiceRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingServiceRepository) Move(id model.ID, newParentUUID string) error {
	call := ServiceMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
