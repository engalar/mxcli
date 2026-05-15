// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

var _ backend.PageMutator = (*MockPageMutator)(nil)

// MockPageMutator implements backend.PageMutator. Every interface method is
// backed by a public function field. If the field is nil the method returns
// nil error (never panics). ContainerType defaults to ContainerPage when unset;
// all other methods return zero values.
type MockPageMutator struct {
	ContainerTypeFunc        func() backend.ContainerKind
	SetWidgetPropertyFunc    func(widgetRef string, prop string, value any) error
	SetWidgetDataSourceFunc  func(widgetRef string, ds pages.DataSource) error
	SetColumnPropertyFunc    func(gridRef string, columnRef string, prop string, value any) error
	InsertWidgetFunc         func(widgetRef string, columnRef string, position backend.InsertPosition, widgets []pages.Widget) error
	DropWidgetFunc           func(refs []backend.WidgetRef) error
	ReplaceWidgetFunc        func(widgetRef string, columnRef string, widgets []pages.Widget) error
	FindWidgetFunc           func(name string) bool
	AddVariableFunc          func(name, dataType, defaultValue string) error
	DropVariableFunc         func(name string) error
	SetLayoutFunc            func(newLayout string, paramMappings map[string]string) error
	SetPluggablePropertyFunc func(widgetRef string, propKey string, op backend.PluggablePropertyOp, ctx backend.PluggablePropertyContext) error
	EnclosingEntityFunc      func(widgetRef string) string
	WidgetScopeFunc          func() map[string]model.ID
	ParamScopeFunc           func() (map[string]model.ID, map[string]string)
	SaveFunc                 func() error

	// Stage 3.3.5.D0 gen-typed siblings.
	SetWidgetDataSourceGenFunc func(widgetRef string, ds element.Element) error
	InsertWidgetGenFunc        func(widgetRef string, columnRef string, position backend.InsertPosition, widgets []element.Element) error
	ReplaceWidgetGenFunc       func(widgetRef string, columnRef string, widgets []element.Element) error
}

func (m *MockPageMutator) ContainerType() backend.ContainerKind {
	if m.ContainerTypeFunc != nil {
		return m.ContainerTypeFunc()
	}
	return backend.ContainerPage
}

func (m *MockPageMutator) SetWidgetProperty(widgetRef string, prop string, value any) error {
	if m.SetWidgetPropertyFunc != nil {
		return m.SetWidgetPropertyFunc(widgetRef, prop, value)
	}
	return nil
}

func (m *MockPageMutator) SetWidgetDataSource(widgetRef string, ds pages.DataSource) error {
	if m.SetWidgetDataSourceFunc != nil {
		return m.SetWidgetDataSourceFunc(widgetRef, ds)
	}
	return nil
}

func (m *MockPageMutator) SetColumnProperty(gridRef string, columnRef string, prop string, value any) error {
	if m.SetColumnPropertyFunc != nil {
		return m.SetColumnPropertyFunc(gridRef, columnRef, prop, value)
	}
	return nil
}

func (m *MockPageMutator) InsertWidget(widgetRef string, columnRef string, position backend.InsertPosition, widgets []pages.Widget) error {
	if m.InsertWidgetFunc != nil {
		return m.InsertWidgetFunc(widgetRef, columnRef, position, widgets)
	}
	return nil
}

func (m *MockPageMutator) DropWidget(refs []backend.WidgetRef) error {
	if m.DropWidgetFunc != nil {
		return m.DropWidgetFunc(refs)
	}
	return nil
}

func (m *MockPageMutator) ReplaceWidget(widgetRef string, columnRef string, widgets []pages.Widget) error {
	if m.ReplaceWidgetFunc != nil {
		return m.ReplaceWidgetFunc(widgetRef, columnRef, widgets)
	}
	return nil
}

func (m *MockPageMutator) FindWidget(name string) bool {
	if m.FindWidgetFunc != nil {
		return m.FindWidgetFunc(name)
	}
	return false
}

func (m *MockPageMutator) AddVariable(name, dataType, defaultValue string) error {
	if m.AddVariableFunc != nil {
		return m.AddVariableFunc(name, dataType, defaultValue)
	}
	return nil
}

func (m *MockPageMutator) DropVariable(name string) error {
	if m.DropVariableFunc != nil {
		return m.DropVariableFunc(name)
	}
	return nil
}

func (m *MockPageMutator) SetLayout(newLayout string, paramMappings map[string]string) error {
	if m.SetLayoutFunc != nil {
		return m.SetLayoutFunc(newLayout, paramMappings)
	}
	return nil
}

func (m *MockPageMutator) SetPluggableProperty(widgetRef string, propKey string, op backend.PluggablePropertyOp, ctx backend.PluggablePropertyContext) error {
	if m.SetPluggablePropertyFunc != nil {
		return m.SetPluggablePropertyFunc(widgetRef, propKey, op, ctx)
	}
	return nil
}

func (m *MockPageMutator) EnclosingEntity(widgetRef string) string {
	if m.EnclosingEntityFunc != nil {
		return m.EnclosingEntityFunc(widgetRef)
	}
	return ""
}

func (m *MockPageMutator) WidgetScope() map[string]model.ID {
	if m.WidgetScopeFunc != nil {
		return m.WidgetScopeFunc()
	}
	return nil
}

func (m *MockPageMutator) ParamScope() (map[string]model.ID, map[string]string) {
	if m.ParamScopeFunc != nil {
		return m.ParamScopeFunc()
	}
	return nil, nil
}

func (m *MockPageMutator) Save() error {
	if m.SaveFunc != nil {
		return m.SaveFunc()
	}
	return nil
}

func (m *MockPageMutator) SetWidgetDataSourceGen(widgetRef string, ds element.Element) error {
	if m.SetWidgetDataSourceGenFunc != nil {
		return m.SetWidgetDataSourceGenFunc(widgetRef, ds)
	}
	return fmt.Errorf("MockPageMutator.SetWidgetDataSourceGen not configured")
}

func (m *MockPageMutator) InsertWidgetGen(widgetRef string, columnRef string, position backend.InsertPosition, widgets []element.Element) error {
	if m.InsertWidgetGenFunc != nil {
		return m.InsertWidgetGenFunc(widgetRef, columnRef, position, widgets)
	}
	return fmt.Errorf("MockPageMutator.InsertWidgetGen not configured")
}

func (m *MockPageMutator) ReplaceWidgetGen(widgetRef string, columnRef string, widgets []element.Element) error {
	if m.ReplaceWidgetGenFunc != nil {
		return m.ReplaceWidgetGenFunc(widgetRef, columnRef, widgets)
	}
	return fmt.Errorf("MockPageMutator.ReplaceWidgetGen not configured")
}
