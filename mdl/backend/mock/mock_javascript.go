// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// ── Gen-typed JavaScriptBackend surface ──────────────────────────────

func (m *MockBackend) ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error) {
	if m.ListJavaScriptActionsGenFunc != nil {
		return m.ListJavaScriptActionsGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListJavaScriptActionsGen not configured")
}

func (m *MockBackend) ReadJavaScriptActionByNameGen(qualifiedName string) (*genJSA.JavaScriptAction, error) {
	if m.ReadJavaScriptActionByNameGenFunc != nil {
		return m.ReadJavaScriptActionByNameGenFunc(qualifiedName)
	}
	return nil, fmt.Errorf("MockBackend.ReadJavaScriptActionByNameGen not configured")
}

func (m *MockBackend) CreateJavaScriptActionGen(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error {
	if m.CreateJavaScriptActionGenFunc != nil {
		return m.CreateJavaScriptActionGenFunc(parentUUID, containmentName, jsa)
	}
	return fmt.Errorf("MockBackend.CreateJavaScriptActionGen not configured")
}

func (m *MockBackend) UpdateJavaScriptActionGen(jsa *genJSA.JavaScriptAction) error {
	if m.UpdateJavaScriptActionGenFunc != nil {
		return m.UpdateJavaScriptActionGenFunc(jsa)
	}
	return fmt.Errorf("MockBackend.UpdateJavaScriptActionGen not configured")
}
