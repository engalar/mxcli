// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// AgentCreateCall captures arguments to AgentWriter.Create.
type AgentCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Agent           element.Element
}

// AgentMoveCall captures arguments to AgentWriter.Move.
type AgentMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingAgentRepository records every call to its methods.
type RecordingAgentRepository struct {
	GotIDs      []model.ID
	ListedTypes []string
	Created     []AgentCreateCall
	Updated     []element.Element
	Deleted     []model.ID
	Moved       []AgentMoveCall

	GetFunc        func(model.ID) (element.Element, error)
	ListByTypeFunc func(string) ([]element.Element, error)
	CreateFunc     func(AgentCreateCall) error
	UpdateFunc     func(element.Element) error
	DeleteFunc     func(model.ID) error
	MoveFunc       func(AgentMoveCall) error
}

var _ repos.AgentRepository = (*RecordingAgentRepository)(nil)

func (m *RecordingAgentRepository) Get(id model.ID) (element.Element, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingAgentRepository) ListByType(typeName string) ([]element.Element, error) {
	m.ListedTypes = append(m.ListedTypes, typeName)
	if m.ListByTypeFunc != nil {
		return m.ListByTypeFunc(typeName)
	}
	return nil, nil
}

func (m *RecordingAgentRepository) Create(parentUUID string, containmentName string, agent element.Element) error {
	call := AgentCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Agent: agent}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingAgentRepository) Update(agent element.Element) error {
	m.Updated = append(m.Updated, agent)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(agent)
	}
	return nil
}

func (m *RecordingAgentRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingAgentRepository) Move(id model.ID, newParentUUID string) error {
	call := AgentMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
