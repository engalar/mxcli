// SPDX-License-Identifier: Apache-2.0

package domainmodel

import (
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// DomainModelDeps is the domain-specific dependency container for entity,
// association, enumeration, and constant CRUD.
type DomainModelDeps struct {
	ConnectionManager          backend.ConnectionManager
	ModuleLister               backend.ModuleLister
	ModuleWriter               backend.ModuleWriter
	DomainModelReader          backend.DomainModelReader
	DomainModelWriter          backend.DomainModelWriter
	EnumerationReader          backend.EnumerationReader
	EnumerationWriter          backend.EnumerationWriter
	ConstantReader             backend.ConstantReader
	ConstantWriter             backend.ConstantWriter
	ServiceLister              backend.ServiceLister
	ServiceWriter              backend.ServiceWriter
	FolderManager              backend.FolderManager
	MetadataReader             backend.MetadataReader
	SecurityEntityAccessManager backend.SecurityEntityAccessManager
	DomainModels                repos.DomainModelRepository

	Output io.Writer

	FindOrCreateModule func(name string) (*model.Module, error)
	FindModule         func(name string) (*model.Module, error)

	GetDomainModelGenCached func(moduleID model.ID) (*genDm.DomainModel, error)
	SetDomainModelGenCached func(moduleID model.ID, dm *genDm.DomainModel)
	InvalidateDomainModelGenCache func(moduleID model.ID)
	InvalidateDomainModelsCache   func()
	InvalidateHierarchy    func()

	FindEntityGen func(qn ast.QualifiedName) (*genDm.Entity, string, error)
	FindEnumeration func(moduleName, enumName string) *model.Enumeration

	EntityPersistable func(entity *genDm.Entity) bool
	CheckFeature      func(area, name, statement, hint string) error

	WarnEntityReferences       func(entityQN string)
	WarnMicroflowEntityParamRefs func(entityQN string)
	TrackModifiedDomainModel   func(moduleID model.ID, moduleName string)
	ValidateModuleRole         func(role ast.QualifiedName) (bool, error)
	PruneInvalidUserRoles      func(exclude *model.ID) error
	CascadeRemoveRoleFromMicroflows func(moduleID model.ID, qualifiedRole string) error
	CascadeRemoveRoleFromNanoflows  func(moduleID model.ID, qualifiedRole string) error
	TrackCreatedEntity         func(moduleName, entityName string, entityID model.ID)

	FindModuleName func(containerID model.ID) string
	FindModuleID   func(containerID model.ID) model.ID
}
