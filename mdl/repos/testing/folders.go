// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
)

// FolderCreateCall captures arguments to FolderWriter.Create.
type FolderCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Folder          *genPr.Folder
}

// FolderMoveCall captures arguments to FolderWriter.Move.
type FolderMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingFolderRepository records every call to its methods.
type RecordingFolderRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	Created      []FolderCreateCall
	Updated      []*genPr.Folder
	Deleted      []model.ID
	Moved        []FolderMoveCall

	GetFunc    func(model.ID) (*genPr.Folder, error)
	ListFunc   func(model.ID) ([]*genPr.Folder, error)
	CreateFunc func(FolderCreateCall) error
	UpdateFunc func(*genPr.Folder) error
	DeleteFunc func(model.ID) error
	MoveFunc   func(FolderMoveCall) error
}

var _ repos.FolderRepository = (*RecordingFolderRepository)(nil)

func (m *RecordingFolderRepository) Get(id model.ID) (*genPr.Folder, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingFolderRepository) List(moduleID model.ID) ([]*genPr.Folder, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingFolderRepository) Create(parentUUID string, containmentName string, f *genPr.Folder) error {
	call := FolderCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Folder: f}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingFolderRepository) Update(f *genPr.Folder) error {
	m.Updated = append(m.Updated, f)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(f)
	}
	return nil
}

func (m *RecordingFolderRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingFolderRepository) Move(id model.ID, newParentUUID string) error {
	call := FolderMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
