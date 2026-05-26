// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// MicroflowCreateCall captures arguments to MicroflowWriter.Create.
type MicroflowCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Microflow       *genMf.Microflow
}

// MicroflowMoveCall captures arguments to MicroflowWriter.Move.
type MicroflowMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingMicroflowRepository records every call to its methods.
//
// Read methods return zero values unless the matching Func is set; write
// methods record their args and either invoke the matching Func or
// return nil (success). All recorded slices are appended in call order.
type RecordingMicroflowRepository struct {
	GotIDs          []model.ID
	ListedModule    []model.ID
	ListedAll       int
	FoundQNs        []string
	IsRuleQNs       []string
	GetContainerIDs []model.ID
	Created         []MicroflowCreateCall
	Updated         []*genMf.Microflow
	Deleted         []model.ID
	Moved           []MicroflowMoveCall

	GetFunc                 func(model.ID) (*genMf.Microflow, error)
	ListFunc                func(model.ID) ([]*genMf.Microflow, error)
	ListAllFunc             func() ([]*genMf.Microflow, error)
	FindByQualifiedNameFunc func(string) (*genMf.Microflow, error)
	IsRuleFunc              func(string) (bool, error)
	GetContainerUUIDFunc    func(model.ID) (model.ID, error)
	CreateFunc              func(MicroflowCreateCall) error
	UpdateFunc              func(*genMf.Microflow) error
	DeleteFunc              func(model.ID) error
	MoveFunc                func(MicroflowMoveCall) error
}

var _ repos.MicroflowRepository = (*RecordingMicroflowRepository)(nil)

func (m *RecordingMicroflowRepository) Get(id model.ID) (*genMf.Microflow, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingMicroflowRepository) List(moduleID model.ID) ([]*genMf.Microflow, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingMicroflowRepository) ListAll() ([]*genMf.Microflow, error) {
	m.ListedAll++
	if m.ListAllFunc != nil {
		return m.ListAllFunc()
	}
	return nil, nil
}

func (m *RecordingMicroflowRepository) FindByQualifiedName(qn string) (*genMf.Microflow, error) {
	m.FoundQNs = append(m.FoundQNs, qn)
	if m.FindByQualifiedNameFunc != nil {
		return m.FindByQualifiedNameFunc(qn)
	}
	return nil, nil
}

func (m *RecordingMicroflowRepository) IsRule(qn string) (bool, error) {
	m.IsRuleQNs = append(m.IsRuleQNs, qn)
	if m.IsRuleFunc != nil {
		return m.IsRuleFunc(qn)
	}
	return false, nil
}

func (m *RecordingMicroflowRepository) GetContainerUUID(id model.ID) (model.ID, error) {
	m.GetContainerIDs = append(m.GetContainerIDs, id)
	if m.GetContainerUUIDFunc != nil {
		return m.GetContainerUUIDFunc(id)
	}
	return "", nil
}

func (m *RecordingMicroflowRepository) Create(parentUUID string, containmentName string, mf *genMf.Microflow) error {
	call := MicroflowCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Microflow: mf}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingMicroflowRepository) Update(mf *genMf.Microflow) error {
	m.Updated = append(m.Updated, mf)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(mf)
	}
	return nil
}

func (m *RecordingMicroflowRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingMicroflowRepository) Move(id model.ID, newParentUUID string) error {
	call := MicroflowMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
