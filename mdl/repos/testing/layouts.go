// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// LayoutCreateCall captures arguments to LayoutWriter.Create.
type LayoutCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Layout          *genPg.Layout
}

// LayoutMoveCall captures arguments to LayoutWriter.Move.
type LayoutMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingLayoutRepository records every call to its methods.
type RecordingLayoutRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	Created      []LayoutCreateCall
	Updated      []*genPg.Layout
	Deleted      []model.ID
	Moved        []LayoutMoveCall

	GetFunc    func(model.ID) (*genPg.Layout, error)
	ListFunc   func(model.ID) ([]*genPg.Layout, error)
	CreateFunc func(LayoutCreateCall) error
	UpdateFunc func(*genPg.Layout) error
	DeleteFunc func(model.ID) error
	MoveFunc   func(LayoutMoveCall) error
}

var _ repos.LayoutRepository = (*RecordingLayoutRepository)(nil)

func (m *RecordingLayoutRepository) Get(id model.ID) (*genPg.Layout, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingLayoutRepository) List(moduleID model.ID) ([]*genPg.Layout, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingLayoutRepository) Create(parentUUID string, containmentName string, layout *genPg.Layout) error {
	call := LayoutCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Layout: layout}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingLayoutRepository) Update(layout *genPg.Layout) error {
	m.Updated = append(m.Updated, layout)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(layout)
	}
	return nil
}

func (m *RecordingLayoutRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingLayoutRepository) Move(id model.ID, newParentUUID string) error {
	call := LayoutMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
