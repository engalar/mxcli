// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/sdk/microflows"
)

// Followup E6 retired Get / Create / Update / Move / Parse mock
// stubs — production routes through ctx.Microflows / ctx.Nanoflows
// (modelsdk-native repos), and the matching FullBackend interface
// methods were removed. Tests that need to seed flow data should use
// repostesting.RecordingMicroflowRepository / RecordingNanoflowRepository
// via withMicroflowsRepo / withNanoflowsRepo.

func (m *MockBackend) ListMicroflows() ([]*microflows.Microflow, error) {
	if m.ListMicroflowsFunc != nil {
		return m.ListMicroflowsFunc()
	}
	return nil, nil
}

func (m *MockBackend) GetMicroflow(id model.ID) (*microflows.Microflow, error) {
	if m.GetMicroflowFunc != nil {
		return m.GetMicroflowFunc(id)
	}
	return nil, nil
}

func (m *MockBackend) DeleteMicroflow(id model.ID) error {
	if m.DeleteMicroflowFunc != nil {
		return m.DeleteMicroflowFunc(id)
	}
	return nil
}

func (m *MockBackend) ListNanoflows() ([]*microflows.Nanoflow, error) {
	if m.ListNanoflowsFunc != nil {
		return m.ListNanoflowsFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListNanoflows not configured")
}

func (m *MockBackend) DeleteNanoflow(id model.ID) error {
	if m.DeleteNanoflowFunc != nil {
		return m.DeleteNanoflowFunc(id)
	}
	return fmt.Errorf("MockBackend.DeleteNanoflow not configured")
}

func (m *MockBackend) IsRule(qualifiedName string) (bool, error) {
	if m.IsRuleFunc != nil {
		return m.IsRuleFunc(qualifiedName)
	}
	return false, nil
}

func (m *MockBackend) ListMicroflowsGen() ([]*genMf.Microflow, error) {
	if m.ListMicroflowsGenFunc != nil {
		return m.ListMicroflowsGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListMicroflowsGen not configured")
}

func (m *MockBackend) ListNanoflowsGen() ([]*genMf.Nanoflow, error) {
	if m.ListNanoflowsGenFunc != nil {
		return m.ListNanoflowsGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListNanoflowsGen not configured")
}
