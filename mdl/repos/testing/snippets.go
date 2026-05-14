// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// SnippetCreateCall captures arguments to SnippetWriter.Create.
type SnippetCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Snippet         *genPg.Snippet
}

// SnippetMoveCall captures arguments to SnippetWriter.Move.
type SnippetMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingSnippetRepository records every call to its methods.
type RecordingSnippetRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	ListedAll    int
	FoundQNs     []string
	Created      []SnippetCreateCall
	Updated      []*genPg.Snippet
	Deleted      []model.ID
	Moved        []SnippetMoveCall

	GetFunc                 func(model.ID) (*genPg.Snippet, error)
	ListFunc                func(model.ID) ([]*genPg.Snippet, error)
	ListAllFunc             func() ([]*genPg.Snippet, error)
	FindByQualifiedNameFunc func(string) (*genPg.Snippet, error)
	GetContainerUUIDFunc    func(model.ID) (model.ID, error)
	CreateFunc              func(SnippetCreateCall) error
	UpdateFunc              func(*genPg.Snippet) error
	DeleteFunc              func(model.ID) error
	MoveFunc                func(SnippetMoveCall) error
}

var _ repos.SnippetRepository = (*RecordingSnippetRepository)(nil)

func (m *RecordingSnippetRepository) Get(id model.ID) (*genPg.Snippet, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingSnippetRepository) List(moduleID model.ID) ([]*genPg.Snippet, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingSnippetRepository) ListAll() ([]*genPg.Snippet, error) {
	m.ListedAll++
	if m.ListAllFunc != nil {
		return m.ListAllFunc()
	}
	return nil, nil
}

func (m *RecordingSnippetRepository) FindByQualifiedName(qn string) (*genPg.Snippet, error) {
	m.FoundQNs = append(m.FoundQNs, qn)
	if m.FindByQualifiedNameFunc != nil {
		return m.FindByQualifiedNameFunc(qn)
	}
	return nil, nil
}

func (m *RecordingSnippetRepository) GetContainerUUID(id model.ID) (model.ID, error) {
	if m.GetContainerUUIDFunc != nil {
		return m.GetContainerUUIDFunc(id)
	}
	return "", nil
}

func (m *RecordingSnippetRepository) Create(parentUUID string, containmentName string, s *genPg.Snippet) error {
	call := SnippetCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Snippet: s}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingSnippetRepository) Update(s *genPg.Snippet) error {
	m.Updated = append(m.Updated, s)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(s)
	}
	return nil
}

func (m *RecordingSnippetRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingSnippetRepository) Move(id model.ID, newParentUUID string) error {
	call := SnippetMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
