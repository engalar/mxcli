// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// Followup E6 retired Get / Create / Update / Move / Parse mock
// stubs; Followup F3 retired the sdk-typed ListMicroflows /
// GetMicroflow / ListNanoflows mock stubs. Production routes through
// ctx.Microflows / ctx.Nanoflows (modelsdk-native repos), and the
// matching FullBackend interface methods were removed. Tests that
// need to seed flow data should use the gen-typed
// repostesting.RecordingMicroflowRepository /
// RecordingNanoflowRepository via withMicroflowsRepo /
// withNanoflowsRepo, or configure the gen-typed *Func fields below.

func (m *MockBackend) DeleteMicroflow(id model.ID) error {
	if m.DeleteMicroflowFunc != nil {
		return m.DeleteMicroflowFunc(id)
	}
	return nil
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

func (m *MockBackend) GetMicroflowGen(id model.ID) (*genMf.Microflow, error) {
	if m.GetMicroflowGenFunc != nil {
		return m.GetMicroflowGenFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetMicroflowGen not configured")
}
