// SPDX-License-Identifier: Apache-2.0

package page

import (
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// PageDeps is the domain-specific dependency container for page/snippet/layout
// CRUD.
type PageDeps struct {
	ConnectionManager          backend.ConnectionManager
	ModuleLister               backend.ModuleLister
	ModuleWriter               backend.ModuleWriter
	PageWriter                 backend.PageWriter
	PageReader                 backend.PageReader
	FolderManager              backend.FolderManager
	MetadataReader             backend.MetadataReader
	DomainModelReader          backend.DomainModelReader
	WidgetBuilder              backend.WidgetBuilder
	PageModelAccess            backend.PageModelAccess
	PageMutationOperator       backend.PageMutationOperator

	PagesRepo   repos.PageRepository
	SnippetsRepo repos.SnippetRepository
	LayoutsRepo repos.LayoutRepository
	MicroflowsRepo  repos.MicroflowRepository
	NanoflowsRepo   repos.NanoflowRepository

	Output io.Writer

	FindOrCreateModule func(name string) (*model.Module, error)
	FindModule         func(name string) (*model.Module, error)
	GetModuleName      func(moduleID model.ID) string
	GetModuleID        func(containerID model.ID) model.ID

	CheckFeature      func(area, name, statement, hint string) error
	InvalidateHierarchy       func()
	InvalidatePagesGenCache   func()
	TrackCreatedPage          func(moduleName, pageName string, pageID model.ID, moduleID model.ID)
	TrackCreatedSnippet       func(moduleName, snippetName string, snippetID model.ID, moduleID model.ID)
	DefaultDocumentAccessRoleQNames func(module *model.Module) []string

	BuildPage   func(s *ast.CreatePageStmtV3, moduleID model.ID, moduleName string) (*genPg.Page, model.ID, error)
	BuildSnippet func(s *ast.CreateSnippetStmtV3, moduleID model.ID, moduleName string) (*genPg.Snippet, model.ID, error)

	BuildLayoutContent  func(s *ast.CreateLayoutStmt) element.Element
	PageASTToModel      func(s *ast.CreatePageStmtV3, moduleName string) (*types.PageModel, error)
	PageModelHasLossyWidget func(pm *types.PageModel) bool
}
