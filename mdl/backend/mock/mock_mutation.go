// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// ---------------------------------------------------------------------------
// PageMutationBackend
// ---------------------------------------------------------------------------

func (m *MockBackend) OpenPageForMutation(unitID model.ID) (backend.PageMutator, error) {
	if m.OpenPageForMutationFunc != nil {
		return m.OpenPageForMutationFunc(unitID)
	}
	return nil, fmt.Errorf("MockBackend.OpenPageForMutation not configured")
}

// ---------------------------------------------------------------------------
// WorkflowMutationBackend
// ---------------------------------------------------------------------------

func (m *MockBackend) OpenWorkflowForMutation(unitID model.ID) (backend.WorkflowMutator, error) {
	if m.OpenWorkflowForMutationFunc != nil {
		return m.OpenWorkflowForMutationFunc(unitID)
	}
	return nil, fmt.Errorf("MockBackend.OpenWorkflowForMutation not configured")
}

// ---------------------------------------------------------------------------
// WidgetSerializationBackend
// ---------------------------------------------------------------------------

func (m *MockBackend) SerializeWidget(w backend.Widget) (any, error) {
	if m.SerializeWidgetFunc != nil {
		return m.SerializeWidgetFunc(w)
	}
	return nil, nil
}

func (m *MockBackend) SerializeClientAction(a backend.ClientAction) (any, error) {
	if m.SerializeClientActionFunc != nil {
		return m.SerializeClientActionFunc(a)
	}
	return nil, nil
}

func (m *MockBackend) SerializeDataSource(ds backend.DataSource) (any, error) {
	if m.SerializeDataSourceFunc != nil {
		return m.SerializeDataSourceFunc(ds)
	}
	return nil, nil
}

func (m *MockBackend) SerializeWorkflowActivityGen(a element.Element) (any, error) {
	if m.SerializeWorkflowActivityGenFunc != nil {
		return m.SerializeWorkflowActivityGenFunc(a)
	}
	return nil, fmt.Errorf("MockBackend.SerializeWorkflowActivityGen not configured")
}

// ---------------------------------------------------------------------------
// WidgetBuilderBackend
// ---------------------------------------------------------------------------

func (m *MockBackend) LoadWidgetTemplate(widgetID string, projectPath string) (backend.WidgetObjectBuilder, error) {
	if m.LoadWidgetTemplateFunc != nil {
		return m.LoadWidgetTemplateFunc(widgetID, projectPath)
	}
	return nil, nil
}

func (m *MockBackend) SerializeWidgetToOpaque(w backend.Widget) backend.OpaqueWidget {
	if m.SerializeWidgetToOpaqueFunc != nil {
		return m.SerializeWidgetToOpaqueFunc(w)
	}
	return nil
}

func (m *MockBackend) SerializeDataSourceToOpaque(ds backend.DataSource) backend.OpaqueDataSource {
	if m.SerializeDataSourceToOpaqueFunc != nil {
		return m.SerializeDataSourceToOpaqueFunc(ds)
	}
	return nil
}

func (m *MockBackend) SerializeClientActionToOpaque(a backend.ClientAction) backend.OpaqueAction {
	if m.SerializeClientActionToOpaqueFunc != nil {
		return m.SerializeClientActionToOpaqueFunc(a)
	}
	return nil
}

func (m *MockBackend) BuildCreateAttributeObject(attributePath string, objectTypeID, propertyTypeID, valueTypeID string) (any, error) {
	if m.BuildCreateAttributeObjectFunc != nil {
		return m.BuildCreateAttributeObjectFunc(attributePath, objectTypeID, propertyTypeID, valueTypeID)
	}
	return nil, nil
}

func (m *MockBackend) BuildDataGrid2Widget(id model.ID, name string, spec backend.DataGridSpec, projectPath string) (*backend.CustomWidget, error) {
	if m.BuildDataGrid2WidgetFunc != nil {
		return m.BuildDataGrid2WidgetFunc(id, name, spec, projectPath)
	}
	return nil, fmt.Errorf("MockBackend.BuildDataGrid2Widget not configured")
}

func (m *MockBackend) BuildFilterWidget(spec backend.FilterWidgetSpec, projectPath string) (backend.Widget, error) {
	if m.BuildFilterWidgetFunc != nil {
		return m.BuildFilterWidgetFunc(spec, projectPath)
	}
	return nil, fmt.Errorf("MockBackend.BuildFilterWidget not configured")
}
