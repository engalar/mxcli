// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
)

// JavaBackend provides Java and JavaScript action operations.
//
// Stage 3.3.2 C3 introduced *Gen siblings to every method that exposed
// sdk-typed values. Legacy methods stay until Phase E retires them.
type JavaBackend interface {
	ListJavaActions() ([]*types.JavaAction, error)
	ListJavaActionsFull() ([]*javaactions.JavaAction, error)
	ListJavaScriptActions() ([]*types.JavaScriptAction, error)
	ReadJavaActionByName(qualifiedName string) (*javaactions.JavaAction, error)
	ReadJavaScriptActionByName(qualifiedName string) (*types.JavaScriptAction, error)
	CreateJavaAction(ja *javaactions.JavaAction) error
	UpdateJavaAction(ja *javaactions.JavaAction) error
	DeleteJavaAction(id model.ID) error
	WriteJavaSourceFile(moduleName, actionName string, javaCode string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, extraImports []string, extraCode string) error
	DeleteJavaSourceFile(moduleName, actionName string) error
	RenameJavaSourceFile(moduleName, oldName, newName string) error
	ReadJavaSourceFile(moduleName, actionName string) (string, error)

	// ── Stage 3.3.2.C3 gen-typed siblings ─────────────────────────────
	// These return *genJA.JavaAction / *genJSA.JavaScriptAction sourced
	// from the modelsdk gen registry. Production wiring lives in
	// mdl/backend/mpr/backend.go and routes through the repo introduced
	// in Stage 3.3.2.A0. Mock backend provides descriptive-error stubs.
	ListJavaActionsGen() ([]*genJA.JavaAction, error)
	ReadJavaActionByNameGen(qualifiedName string) (*genJA.JavaAction, error)
	CreateJavaActionGen(parentUUID, containmentName string, ja *genJA.JavaAction) error
	UpdateJavaActionGen(ja *genJA.JavaAction) error
	WriteJavaSourceFileGen(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error
	ListJavaScriptActionsGen() ([]*genJSA.JavaScriptAction, error)
	ReadJavaScriptActionByNameGen(qualifiedName string) (*genJSA.JavaScriptAction, error)
}
