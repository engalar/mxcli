// SPDX-License-Identifier: Apache-2.0

package microflow

import (
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/version"
)

type MicroflowDeps struct {
	ConnectionManager backend.ConnectionManager
	ModuleLister      backend.ModuleLister
	ModuleWriter      backend.ModuleWriter
	FolderManager     backend.FolderManager
	DomainModelReader backend.DomainModelReader
	Output            io.Writer

	MicroflowsRepo interface {
		ListAll() ([]*genMf.Microflow, error)
		Create(containerID, documentType string, doc *genMf.Microflow) error
		Update(doc *genMf.Microflow) error
		Delete(id model.ID) error
		GetContainerUUID(id model.ID) (model.ID, error)
	}
	NanoflowsRepo interface {
		ListAll() ([]*genMf.Nanoflow, error)
		Create(parentUUID, containmentName string, nf *genMf.Nanoflow) error
		Update(nf *genMf.Nanoflow) error
		Delete(id model.ID) error
	}

	Version version.Version

	FindOrCreateModule         func(name string) (*model.Module, error)
	DefaultDocumentAccessRoles func(module *model.Module) []string
	InvalidateCache            func()

	BuildMicroflowFlowGraph func(body []ast.MicroflowStatement, returnType *ast.MicroflowReturnType, params []ast.MicroflowParam, isNanoflow bool) (element.Element, []element.Element, []element.Element, []string, error)

	ConvertASTToGenDataType  func(dt ast.DataType) element.Element
	ResolveAmbiguousDataType func(dt ast.DataType) ast.DataType
	TrackCreatedMicroflow    func(moduleName, mfName string, id, containerID model.ID, returnEntityName string)
	TrackCreatedNanoflow     func(moduleName, mfName string, id, containerID model.ID, returnEntityName string)
	WarnBrokenCallerRefs     func(qualifiedName string, removedParams []string)
	ConsumeDroppedMicroflow  func(qualifiedName string) (id, containerID model.ID)
	ConsumeDroppedNanoflow   func(qualifiedName string) (id, containerID model.ID)
}
