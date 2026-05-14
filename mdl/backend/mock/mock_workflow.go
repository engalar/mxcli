// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// ---------------------------------------------------------------------------
// WorkflowBackend
// ---------------------------------------------------------------------------
//
// Stage 3.3.3.C1+C5 — gen-typed siblings added; all unset Funcs return
// descriptive errors per CLAUDE.md MockBackend audit rule.

func (m *MockBackend) ListWorkflows() ([]*workflows.Workflow, error) {
	if m.ListWorkflowsFunc != nil {
		return m.ListWorkflowsFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListWorkflows not configured")
}

func (m *MockBackend) GetWorkflow(id model.ID) (*workflows.Workflow, error) {
	if m.GetWorkflowFunc != nil {
		return m.GetWorkflowFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetWorkflow not configured")
}

func (m *MockBackend) CreateWorkflow(wf *workflows.Workflow) error {
	if m.CreateWorkflowFunc != nil {
		return m.CreateWorkflowFunc(wf)
	}
	return fmt.Errorf("MockBackend.CreateWorkflow not configured")
}

func (m *MockBackend) UpdateWorkflow(wf *workflows.Workflow) error {
	if m.UpdateWorkflowFunc != nil {
		return m.UpdateWorkflowFunc(wf)
	}
	return fmt.Errorf("MockBackend.UpdateWorkflow not configured")
}

func (m *MockBackend) DeleteWorkflow(id model.ID) error {
	if m.DeleteWorkflowFunc != nil {
		return m.DeleteWorkflowFunc(id)
	}
	return fmt.Errorf("MockBackend.DeleteWorkflow not configured")
}

// ── Stage 3.3.3.C1 gen-typed siblings ───────────────────────────────────

func (m *MockBackend) ListWorkflowsGen() ([]*genWf.Workflow, error) {
	if m.ListWorkflowsGenFunc != nil {
		return m.ListWorkflowsGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListWorkflowsGen not configured")
}

func (m *MockBackend) GetWorkflowGen(id model.ID) (*genWf.Workflow, error) {
	if m.GetWorkflowGenFunc != nil {
		return m.GetWorkflowGenFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetWorkflowGen not configured")
}

func (m *MockBackend) CreateWorkflowGen(parentUUID, containmentName string, wf *genWf.Workflow) error {
	if m.CreateWorkflowGenFunc != nil {
		return m.CreateWorkflowGenFunc(parentUUID, containmentName, wf)
	}
	return fmt.Errorf("MockBackend.CreateWorkflowGen not configured")
}

func (m *MockBackend) UpdateWorkflowGen(wf *genWf.Workflow) error {
	if m.UpdateWorkflowGenFunc != nil {
		return m.UpdateWorkflowGenFunc(wf)
	}
	return fmt.Errorf("MockBackend.UpdateWorkflowGen not configured")
}

// ---------------------------------------------------------------------------
// SettingsBackend
// ---------------------------------------------------------------------------

func (m *MockBackend) GetProjectSettings() (*model.ProjectSettings, error) {
	if m.GetProjectSettingsFunc != nil {
		return m.GetProjectSettingsFunc()
	}
	return nil, nil
}

func (m *MockBackend) UpdateProjectSettings(ps *model.ProjectSettings) error {
	if m.UpdateProjectSettingsFunc != nil {
		return m.UpdateProjectSettingsFunc(ps)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ImageBackend
// ---------------------------------------------------------------------------

func (m *MockBackend) ListImageCollections() ([]*types.ImageCollection, error) {
	if m.ListImageCollectionsFunc != nil {
		return m.ListImageCollectionsFunc()
	}
	return nil, nil
}

func (m *MockBackend) CreateImageCollection(ic *types.ImageCollection) error {
	if m.CreateImageCollectionFunc != nil {
		return m.CreateImageCollectionFunc(ic)
	}
	return nil
}

func (m *MockBackend) UpdateImageCollection(ic *types.ImageCollection) error {
	if m.UpdateImageCollectionFunc != nil {
		return m.UpdateImageCollectionFunc(ic)
	}
	return fmt.Errorf("MockBackend.UpdateImageCollection not configured")
}

func (m *MockBackend) DeleteImageCollection(id string) error {
	if m.DeleteImageCollectionFunc != nil {
		return m.DeleteImageCollectionFunc(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ScheduledEventBackend
// ---------------------------------------------------------------------------

func (m *MockBackend) ListScheduledEvents() ([]*model.ScheduledEvent, error) {
	if m.ListScheduledEventsFunc != nil {
		return m.ListScheduledEventsFunc()
	}
	return nil, nil
}

func (m *MockBackend) GetScheduledEvent(id model.ID) (*model.ScheduledEvent, error) {
	if m.GetScheduledEventFunc != nil {
		return m.GetScheduledEventFunc(id)
	}
	return nil, nil
}
