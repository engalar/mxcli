// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// NanoflowCreateCall captures arguments to NanoflowWriter.Create.
type NanoflowCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Nanoflow        *genMf.Nanoflow
}

// NanoflowMoveCall captures arguments to NanoflowWriter.Move.
type NanoflowMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingNanoflowRepository records every call to its methods.
//
// Read methods return zero values unless the matching Func is set; write
// methods record their args and either invoke the matching Func or
// return nil (success). All recorded slices are appended in call order.
type RecordingNanoflowRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	Created      []NanoflowCreateCall
	Updated      []*genMf.Nanoflow
	Deleted      []model.ID
	Moved        []NanoflowMoveCall

	GetFunc    func(model.ID) (*genMf.Nanoflow, error)
	ListFunc   func(model.ID) ([]*genMf.Nanoflow, error)
	CreateFunc func(NanoflowCreateCall) error
	UpdateFunc func(*genMf.Nanoflow) error
	DeleteFunc func(model.ID) error
	MoveFunc   func(NanoflowMoveCall) error
}

var _ repos.NanoflowRepository = (*RecordingNanoflowRepository)(nil)

func (m *RecordingNanoflowRepository) Get(id model.ID) (*genMf.Nanoflow, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingNanoflowRepository) List(moduleID model.ID) ([]*genMf.Nanoflow, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingNanoflowRepository) Create(parentUUID string, containmentName string, nf *genMf.Nanoflow) error {
	call := NanoflowCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Nanoflow: nf}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingNanoflowRepository) Update(nf *genMf.Nanoflow) error {
	m.Updated = append(m.Updated, nf)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(nf)
	}
	return nil
}

func (m *RecordingNanoflowRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingNanoflowRepository) Move(id model.ID, newParentUUID string) error {
	call := NanoflowMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
