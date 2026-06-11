// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// ---------------------------------------------------------------------------
// WorkflowBackend
// ---------------------------------------------------------------------------
//
// Stage 3.3.3.E1 retired the sdk-typed surface. The gen-typed methods +
// the pure-ID DeleteWorkflow are the only members left. All unset Funcs
// return descriptive errors per CLAUDE.md MockBackend audit rule.

func (m *MockBackend) DeleteWorkflow(id model.ID) error {
	if m.DeleteWorkflowFunc != nil {
		return m.DeleteWorkflowFunc(id)
	}
	return fmt.Errorf("MockBackend.DeleteWorkflow not configured")
}

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

func (m *MockBackend) ListTranslationNodes(docQN, docType string) ([]model.TranslationNode, error) {
	if m.ListTranslationNodesFunc != nil {
		return m.ListTranslationNodesFunc(docQN, docType)
	}
	return nil, fmt.Errorf("MockBackend.ListTranslationNodes not configured")
}

func (m *MockBackend) SetEnumerationTranslation(enumQN, valueName, langCode, text string) error {
	if m.SetEnumerationTranslationFunc != nil {
		return m.SetEnumerationTranslationFunc(enumQN, valueName, langCode, text)
	}
	return fmt.Errorf("MockBackend.SetEnumerationTranslation not configured")
}

func (m *MockBackend) SetMicroflowActionTranslation(docQN, actionType string, index int, property, langCode, text string) error {
	if m.SetMicroflowActionTranslationFunc != nil {
		return m.SetMicroflowActionTranslationFunc(docQN, actionType, index, property, langCode, text)
	}
	return fmt.Errorf("MockBackend.SetMicroflowActionTranslation not configured")
}

func (m *MockBackend) SetNavigationCaptionTranslation(profileName string, menuPath []string, langCode, text string) error {
	if m.SetNavigationCaptionTranslationFunc != nil {
		return m.SetNavigationCaptionTranslationFunc(profileName, menuPath, langCode, text)
	}
	return fmt.Errorf("MockBackend.SetNavigationCaptionTranslation not configured")
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

func (m *MockBackend) MoveImageCollection(ic *types.ImageCollection) error {
	if m.MoveImageCollectionFunc != nil {
		return m.MoveImageCollectionFunc(ic)
	}
	return fmt.Errorf("MockBackend.MoveImageCollection not configured")
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
