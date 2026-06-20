// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
)

func (m *MockBackend) DeleteJavaAction(id model.ID) error {
	if m.DeleteJavaActionFunc != nil {
		return m.DeleteJavaActionFunc(id)
	}
	return fmt.Errorf("MockBackend.DeleteJavaAction not configured")
}

func (m *MockBackend) DeleteJavaSourceFile(moduleName, actionName string) error {
	if m.DeleteJavaSourceFileFunc != nil {
		return m.DeleteJavaSourceFileFunc(moduleName, actionName)
	}
	return fmt.Errorf("MockBackend.DeleteJavaSourceFile not configured")
}

func (m *MockBackend) RenameJavaSourceFile(moduleName, oldName, newName string) error {
	if m.RenameJavaSourceFileFunc != nil {
		return m.RenameJavaSourceFileFunc(moduleName, oldName, newName)
	}
	return fmt.Errorf("MockBackend.RenameJavaSourceFile not configured")
}

func (m *MockBackend) ReadJavaSourceFile(moduleName, actionName string) (string, error) {
	if m.ReadJavaSourceFileFunc != nil {
		return m.ReadJavaSourceFileFunc(moduleName, actionName)
	}
	return "", fmt.Errorf("MockBackend.ReadJavaSourceFile not configured")
}

// ── Stage 3.3.2.C3 gen-typed siblings ─────────────────────────────────
// Per CLAUDE.md MockBackend audit rule: stubs return descriptive errors
// rather than nil so accidental "happy path" coverage is impossible.

func (m *MockBackend) ListJavaActionsGen() ([]*genJA.JavaAction, error) {
	if m.ListJavaActionsGenFunc != nil {
		return m.ListJavaActionsGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListJavaActionsGen not configured")
}

func (m *MockBackend) ReadJavaActionByNameGen(qualifiedName string) (*genJA.JavaAction, error) {
	if m.ReadJavaActionByNameGenFunc != nil {
		return m.ReadJavaActionByNameGenFunc(qualifiedName)
	}
	return nil, fmt.Errorf("MockBackend.ReadJavaActionByNameGen not configured")
}

func (m *MockBackend) CreateJavaActionGen(parentUUID, containmentName string, ja *genJA.JavaAction) error {
	if m.CreateJavaActionGenFunc != nil {
		return m.CreateJavaActionGenFunc(parentUUID, containmentName, ja)
	}
	return fmt.Errorf("MockBackend.CreateJavaActionGen not configured")
}

func (m *MockBackend) UpdateJavaActionGen(ja *genJA.JavaAction) error {
	if m.UpdateJavaActionGenFunc != nil {
		return m.UpdateJavaActionGenFunc(ja)
	}
	return fmt.Errorf("MockBackend.UpdateJavaActionGen not configured")
}

func (m *MockBackend) WriteJavaSourceFileGen(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error {
	if m.WriteJavaSourceFileGenFunc != nil {
		return m.WriteJavaSourceFileGenFunc(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
	}
	return fmt.Errorf("MockBackend.WriteJavaSourceFileGen not configured")
}
