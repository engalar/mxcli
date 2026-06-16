// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
)

// JavaBackend provides Java action operations.
//
// Stage 3.3.2.E1 retired the sdk-typed surface entirely. All consumers
// use the gen-typed siblings. ID-/string-keyed file operations
// (DeleteJavaAction, *JavaSourceFile) carry no sdk types and remain.
type JavaBackend interface {
	DeleteJavaAction(id model.ID) error
	DeleteJavaSourceFile(moduleName, actionName string) error
	RenameJavaSourceFile(moduleName, oldName, newName string) error
	ReadJavaSourceFile(moduleName, actionName string) (string, error)

	// Gen-typed surface (production wiring in mdl/backend/mpr/backend.go;
	// mock stubs in mdl/backend/mock/mock_java.go return descriptive
	// errors per master plan §5 P3 / CLAUDE.md MockBackend audit rule).
	ListJavaActionsGen() ([]*genJA.JavaAction, error)
	ReadJavaActionByNameGen(qualifiedName string) (*genJA.JavaAction, error)
	CreateJavaActionGen(parentUUID, containmentName string, ja *genJA.JavaAction) error
	UpdateJavaActionGen(ja *genJA.JavaAction) error
	WriteJavaSourceFileGen(moduleName, actionName string, javaCode string, params []*genJA.JavaActionParameter, returnType element.Element, extraImports []string, extraCode string) error
}
