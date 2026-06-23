// SPDX-License-Identifier: Apache-2.0

package page

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
)

// PageDeps is the domain-specific dependency container for page/snippet/layout
// CRUD. Defined and available for migration but currently unused — the real
// implementations still live in the executor package and use HandlerDeps.
// See microflow.MicroflowDeps for the established pattern.
type PageDeps struct {
	ConnectionManager backend.ConnectionManager
	ModuleLister      backend.ModuleLister
	ModuleWriter      backend.ModuleWriter
	PageWriter        backend.PageWriter
	PageReader        backend.PageReader
	FolderManager     backend.FolderManager
	MetadataReader    backend.MetadataReader
	DomainModelReader backend.DomainModelReader
	WidgetBuilder     backend.WidgetBuilder
	PageModelAccess   backend.PageModelAccess
	PageMutationOperator backend.PageMutationOperator

	PagesRepo   repos.PageRepository
	SnippetsRepo repos.SnippetRepository
	LayoutsRepo repos.LayoutRepository

	MicroflowsRepo  repos.MicroflowRepository
	NanoflowsRepo   repos.NanoflowRepository

	WidgetBackend          backend.FullBackend
	SerializationBackend   backend.FullBackend
}
