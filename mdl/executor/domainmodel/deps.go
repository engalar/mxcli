// SPDX-License-Identifier: Apache-2.0

package domainmodel

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
)

// DomainModelDeps is the domain-specific dependency container for entity,
// association, enumeration, constant, and database connection CRUD.
// Defined and available for migration but currently unused — the real
// implementations still live in the executor package and use HandlerDeps.
type DomainModelDeps struct {
	ConnectionManager backend.ConnectionManager
	ModuleLister      backend.ModuleLister
	ModuleWriter      backend.ModuleWriter
	DomainModelReader backend.DomainModelReader
	DomainModelWriter backend.DomainModelWriter
	EnumerationReader backend.EnumerationReader
	EnumerationWriter backend.EnumerationWriter
	ConstantReader    backend.ConstantReader
	ConstantWriter    backend.ConstantWriter
	ServiceLister     backend.ServiceLister
	ServiceWriter     backend.ServiceWriter
	FolderManager     backend.FolderManager
	MetadataReader    backend.MetadataReader

	DomainModels repos.DomainModelRepository
}
