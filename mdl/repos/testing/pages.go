// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// PageCreateCall captures arguments to PageWriter.Create.
type PageCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Page            *genPg.Page
}

// PageMoveCall captures arguments to PageWriter.Move.
type PageMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// SetWidgetPropertyCall captures arguments to PageMutator.SetWidgetProperty.
type SetWidgetPropertyCall struct {
	WidgetID model.ID
	Prop     string
	Value    any
}

// InsertWidgetCall captures arguments to PageMutator.InsertWidget.
type InsertWidgetCall struct {
	ParentID model.ID
	Slot     string
	Widget   element.Element
}

// ReplaceWidgetCall captures arguments to PageMutator.ReplaceWidget.
type ReplaceWidgetCall struct {
	WidgetID    model.ID
	Replacement element.Element
}

// RecordingPageMutator records every PageMutator method call.
type RecordingPageMutator struct {
	SetWidgetPropertyCalls []SetWidgetPropertyCall
	InsertWidgetCalls      []InsertWidgetCall
	DeleteWidgetCalls      []model.ID
	ReplaceWidgetCalls     []ReplaceWidgetCall
	SetLayoutCalls         []string
	CommitCalls            int

	SetWidgetPropertyFunc func(SetWidgetPropertyCall) error
	InsertWidgetFunc      func(InsertWidgetCall) error
	DeleteWidgetFunc      func(model.ID) error
	ReplaceWidgetFunc     func(ReplaceWidgetCall) error
	SetLayoutFunc         func(string) error
	CommitFunc            func() error
}

var _ repos.PageMutator = (*RecordingPageMutator)(nil)

func (m *RecordingPageMutator) SetWidgetProperty(widgetID model.ID, prop string, value any) error {
	call := SetWidgetPropertyCall{WidgetID: widgetID, Prop: prop, Value: value}
	m.SetWidgetPropertyCalls = append(m.SetWidgetPropertyCalls, call)
	if m.SetWidgetPropertyFunc != nil {
		return m.SetWidgetPropertyFunc(call)
	}
	return nil
}

func (m *RecordingPageMutator) InsertWidget(parentID model.ID, slot string, widget element.Element) error {
	call := InsertWidgetCall{ParentID: parentID, Slot: slot, Widget: widget}
	m.InsertWidgetCalls = append(m.InsertWidgetCalls, call)
	if m.InsertWidgetFunc != nil {
		return m.InsertWidgetFunc(call)
	}
	return nil
}

func (m *RecordingPageMutator) DeleteWidget(widgetID model.ID) error {
	m.DeleteWidgetCalls = append(m.DeleteWidgetCalls, widgetID)
	if m.DeleteWidgetFunc != nil {
		return m.DeleteWidgetFunc(widgetID)
	}
	return nil
}

func (m *RecordingPageMutator) ReplaceWidget(widgetID model.ID, replacement element.Element) error {
	call := ReplaceWidgetCall{WidgetID: widgetID, Replacement: replacement}
	m.ReplaceWidgetCalls = append(m.ReplaceWidgetCalls, call)
	if m.ReplaceWidgetFunc != nil {
		return m.ReplaceWidgetFunc(call)
	}
	return nil
}

func (m *RecordingPageMutator) SetLayout(layoutQN string) error {
	m.SetLayoutCalls = append(m.SetLayoutCalls, layoutQN)
	if m.SetLayoutFunc != nil {
		return m.SetLayoutFunc(layoutQN)
	}
	return nil
}

func (m *RecordingPageMutator) Commit() error {
	m.CommitCalls++
	if m.CommitFunc != nil {
		return m.CommitFunc()
	}
	return nil
}

// RecordingPageRepository records every PageRepository method call. Read
// methods return zero values unless the matching Func is set; writes
// always succeed unless their Func returns an error.
type RecordingPageRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	ListedAll    int
	FoundQNs     []string
	OpenedIDs    []model.ID
	Created      []PageCreateCall
	Updated      []*genPg.Page
	Deleted      []model.ID
	Moved        []PageMoveCall

	GetFunc                 func(model.ID) (*genPg.Page, error)
	ListFunc                func(model.ID) ([]*genPg.Page, error)
	ListAllFunc             func() ([]*genPg.Page, error)
	FindByQualifiedNameFunc func(string) (*genPg.Page, error)
	GetContainerUUIDFunc    func(model.ID) (model.ID, error)
	OpenForMutationFunc     func(model.ID) (repos.PageMutator, error)
	CreateFunc              func(PageCreateCall) error
	UpdateFunc              func(*genPg.Page) error
	DeleteFunc              func(model.ID) error
	MoveFunc                func(PageMoveCall) error
}

var _ repos.PageRepository = (*RecordingPageRepository)(nil)

func (m *RecordingPageRepository) Get(id model.ID) (*genPg.Page, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingPageRepository) List(moduleID model.ID) ([]*genPg.Page, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingPageRepository) ListAll() ([]*genPg.Page, error) {
	m.ListedAll++
	if m.ListAllFunc != nil {
		return m.ListAllFunc()
	}
	return nil, nil
}

func (m *RecordingPageRepository) FindByQualifiedName(qn string) (*genPg.Page, error) {
	m.FoundQNs = append(m.FoundQNs, qn)
	if m.FindByQualifiedNameFunc != nil {
		return m.FindByQualifiedNameFunc(qn)
	}
	return nil, nil
}

func (m *RecordingPageRepository) GetContainerUUID(id model.ID) (model.ID, error) {
	if m.GetContainerUUIDFunc != nil {
		return m.GetContainerUUIDFunc(id)
	}
	return "", nil
}

func (m *RecordingPageRepository) OpenForMutation(pageID model.ID) (repos.PageMutator, error) {
	m.OpenedIDs = append(m.OpenedIDs, pageID)
	if m.OpenForMutationFunc != nil {
		return m.OpenForMutationFunc(pageID)
	}
	return &RecordingPageMutator{}, nil
}

func (m *RecordingPageRepository) Create(parentUUID string, containmentName string, page *genPg.Page) error {
	call := PageCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Page: page}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingPageRepository) Update(page *genPg.Page) error {
	m.Updated = append(m.Updated, page)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(page)
	}
	return nil
}

func (m *RecordingPageRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingPageRepository) Move(id model.ID, newParentUUID string) error {
	call := PageMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}
