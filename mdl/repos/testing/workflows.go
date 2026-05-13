// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// WorkflowCreateCall captures arguments to WorkflowWriter.Create.
type WorkflowCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Workflow        *genWf.Workflow
}

// WorkflowMoveCall captures arguments to WorkflowWriter.Move.
type WorkflowMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingWorkflowRepository records every call to its methods.
type RecordingWorkflowRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	OpenedIDs    []model.ID
	Created      []WorkflowCreateCall
	Updated      []*genWf.Workflow
	Deleted      []model.ID
	Moved        []WorkflowMoveCall

	GetFunc             func(model.ID) (*genWf.Workflow, error)
	ListFunc            func(model.ID) ([]*genWf.Workflow, error)
	CreateFunc          func(WorkflowCreateCall) error
	UpdateFunc          func(*genWf.Workflow) error
	DeleteFunc          func(model.ID) error
	MoveFunc            func(WorkflowMoveCall) error
	OpenForMutationFunc func(model.ID) (repos.WorkflowMutator, error)
}

var _ repos.WorkflowRepository = (*RecordingWorkflowRepository)(nil)

func (m *RecordingWorkflowRepository) Get(id model.ID) (*genWf.Workflow, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *RecordingWorkflowRepository) List(moduleID model.ID) ([]*genWf.Workflow, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListFunc != nil {
		return m.ListFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingWorkflowRepository) Create(parentUUID string, containmentName string, wf *genWf.Workflow) error {
	call := WorkflowCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Workflow: wf}
	m.Created = append(m.Created, call)
	if m.CreateFunc != nil {
		return m.CreateFunc(call)
	}
	return nil
}

func (m *RecordingWorkflowRepository) Update(wf *genWf.Workflow) error {
	m.Updated = append(m.Updated, wf)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(wf)
	}
	return nil
}

func (m *RecordingWorkflowRepository) Delete(id model.ID) error {
	m.Deleted = append(m.Deleted, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *RecordingWorkflowRepository) Move(id model.ID, newParentUUID string) error {
	call := WorkflowMoveCall{ID: id, NewParentUUID: newParentUUID}
	m.Moved = append(m.Moved, call)
	if m.MoveFunc != nil {
		return m.MoveFunc(call)
	}
	return nil
}

func (m *RecordingWorkflowRepository) OpenForMutation(id model.ID) (repos.WorkflowMutator, error) {
	m.OpenedIDs = append(m.OpenedIDs, id)
	if m.OpenForMutationFunc != nil {
		return m.OpenForMutationFunc(id)
	}
	return &RecordingWorkflowMutator{}, nil
}

// SetActivityPropertyCall captures one SetActivityProperty call.
type SetActivityPropertyCall struct {
	ActivityID model.ID
	Prop       string
	Value      any
}

// InsertActivityCall captures one InsertActivity call.
type InsertActivityCall struct {
	ParentID model.ID
	Slot     string
	Activity element.Element
}

// ReplaceActivityCall captures one ReplaceActivity call.
type ReplaceActivityCall struct {
	ActivityID  model.ID
	Replacement element.Element
}

// RecordingWorkflowMutator records every call to its methods.
type RecordingWorkflowMutator struct {
	SetActivityPropertyCalls []SetActivityPropertyCall
	InsertActivityCalls      []InsertActivityCall
	DeleteActivityCalls      []model.ID
	ReplaceActivityCalls     []ReplaceActivityCall
	CommitCalls              int

	SetActivityPropertyFunc func(SetActivityPropertyCall) error
	InsertActivityFunc      func(InsertActivityCall) error
	DeleteActivityFunc      func(model.ID) error
	ReplaceActivityFunc     func(ReplaceActivityCall) error
	CommitFunc              func() error
}

var _ repos.WorkflowMutator = (*RecordingWorkflowMutator)(nil)

func (m *RecordingWorkflowMutator) SetActivityProperty(activityID model.ID, prop string, value any) error {
	call := SetActivityPropertyCall{ActivityID: activityID, Prop: prop, Value: value}
	m.SetActivityPropertyCalls = append(m.SetActivityPropertyCalls, call)
	if m.SetActivityPropertyFunc != nil {
		return m.SetActivityPropertyFunc(call)
	}
	return nil
}

func (m *RecordingWorkflowMutator) InsertActivity(parentID model.ID, slot string, activity element.Element) error {
	call := InsertActivityCall{ParentID: parentID, Slot: slot, Activity: activity}
	m.InsertActivityCalls = append(m.InsertActivityCalls, call)
	if m.InsertActivityFunc != nil {
		return m.InsertActivityFunc(call)
	}
	return nil
}

func (m *RecordingWorkflowMutator) DeleteActivity(activityID model.ID) error {
	m.DeleteActivityCalls = append(m.DeleteActivityCalls, activityID)
	if m.DeleteActivityFunc != nil {
		return m.DeleteActivityFunc(activityID)
	}
	return nil
}

func (m *RecordingWorkflowMutator) ReplaceActivity(activityID model.ID, replacement element.Element) error {
	call := ReplaceActivityCall{ActivityID: activityID, Replacement: replacement}
	m.ReplaceActivityCalls = append(m.ReplaceActivityCalls, call)
	if m.ReplaceActivityFunc != nil {
		return m.ReplaceActivityFunc(call)
	}
	return nil
}

func (m *RecordingWorkflowMutator) Commit() error {
	m.CommitCalls++
	if m.CommitFunc != nil {
		return m.CommitFunc()
	}
	return nil
}
