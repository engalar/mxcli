// SPDX-License-Identifier: Apache-2.0

package security

import (
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// SecurityDeps is the domain-specific dependency container for security CRUD.
type SecurityDeps struct {
	ConnectionManager          backend.ConnectionManager
	ModuleLister               backend.ModuleLister
	ModuleWriter               backend.ModuleWriter
	FolderManager              backend.FolderManager
	DomainModelReader          backend.DomainModelReader
	MetadataReader             backend.MetadataReader
	SecurityProjectManager     backend.SecurityProjectManager
	SecurityModuleManager      backend.SecurityModuleManager
	SecurityEntityAccessManager backend.SecurityEntityAccessManager
	NavigationWriter           backend.NavigationWriter
	NavigationReader           backend.NavigationReader
	ImageBackend               backend.ImageBackend
	SettingsReader             backend.SettingsReader
	SettingsWriter             backend.SettingsWriter
	ServiceLister              backend.ServiceLister
	RenameManager              backend.RenameManager
	Backend                    backend.FullBackend

	Security      repos.SecurityRepository
	PagesRepo     repos.PageRepository
	SnippetsRepo  repos.SnippetRepository
	LayoutsRepo   repos.LayoutRepository
	MicroflowsRepo repos.MicroflowRepository
	NanoflowsRepo  repos.NanoflowRepository

	Output io.Writer
	Quiet  bool

	FindModule              func(name string) (*model.Module, error)
	FindOrCreateModule      func(name string) (*model.Module, error)
	FindModuleName          func(containerID model.ID) string
	FindModuleID            func(containerID model.ID) model.ID

	ValidateModuleRole      func(role ast.QualifiedName) (bool, error)
	CascadeRemoveRoleFromMicroflows func(moduleID model.ID, qualifiedRole string) error
	CascadeRemoveRoleFromNanoflows  func(moduleID model.ID, qualifiedRole string) error
	PruneInvalidUserRoles   func(exclude *model.ID) error

	LookupCreatedPageID     func(qualifiedName string) (model.ID, error)
	FilterAutoDocumentRoles func(roles []string) []string
	MergeAllowedRoles       func(existing []string, valid []ast.QualifiedName) ([]string, []string)
	FilterAllowedRoles      func(existing []string, roles []ast.QualifiedName) ([]string, []string)

	CheckFeature            func(area, name, statement, hint string) error

	GetProjectSecurityGen     func() (*genSec.ProjectSecurity, error)
	InvalidateProjectSecurityCache func()
	InvalidateModuleSecurityCache  func()
	GetDomainModelGenCached       func(moduleID model.ID) (*genDm.DomainModel, error)
	InvalidateDomainModelGenForModule func(moduleID model.ID)
	InvalidateDomainModelsCache   func()
	GetModulesFromCache           func() ([]*model.Module, error)
	FindEntityGen                 func(qn ast.QualifiedName) (*genDm.Entity, string, error)
	FormatAccessRuleResult        func(moduleName, entityName string, roleNames []string) string
	DetectUserEntity              func() (string, error)
	CachedDomainModels            func() ([]*genDm.DomainModel, error)
	EntityGeneralizationQN        func(entity *genDm.Entity) string
	TrackModifiedDomainModel      func(moduleID model.ID, moduleName string)
}
