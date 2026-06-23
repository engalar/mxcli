// SPDX-License-Identifier: Apache-2.0

package security

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
)

// SecurityDeps is the domain-specific dependency container for security CRUD.
// Defined and available for migration but currently unused — the real
// implementations still live in the executor package and use HandlerDeps.
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

	Security    repos.SecurityRepository
	PagesRepo   repos.PageRepository
	SnippetsRepo repos.SnippetRepository
	LayoutsRepo repos.LayoutRepository
	MicroflowsRepo repos.MicroflowRepository
	NanoflowsRepo  repos.NanoflowRepository
}
