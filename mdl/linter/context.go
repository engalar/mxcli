// SPDX-License-Identifier: Apache-2.0

package linter

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// LintReader provides deep, gen-typed read access to MPR document bodies that
// the in-memory graph (graphcatalog) does not surface as node properties.
// Implemented by MprBackend.
//
// The graph catalog exposes lightweight node listings (Name / Module /
// QualifiedName); rules that need richer document data — entity access rules,
// entity persistability, microflow activity bodies — reach for these gen-typed
// accessors instead.
type LintReader interface {
	GetMicroflowGen(id model.ID) (*genMf.Microflow, error)
	GetProjectSecurityGen() (*genSec.ProjectSecurity, error)
	GetNavigation() (*types.NavigationDocument, error)
	// ListDomainModelsGen returns every domain model as a gen-typed object,
	// so rules can inspect entities' AccessRules and Persistable state that
	// graphcatalog's EntityNode does not carry.
	ListDomainModelsGen() ([]*genDm.DomainModel, error)
	// PageReader supplies the gen-typed Page/Layout/Snippet read surface
	// (ListPagesGen, GetPageGen, GetPageContainerUUID, etc.). Embedded to
	// avoid re-declaring page read methods already defined on the backend;
	// lint rules pair ListPagesGen with GetPageContainerUUID to build
	// qualified names from gen-typed Page listings.
	backend.PageReader
	ListModules() ([]*model.Module, error)
	ListFolders() ([]*types.FolderInfo, error)
	GetRawUnit(id model.ID) (map[string]any, error)
}

// LintContext wraps the project graph and a deep reader, exposing rule-friendly
// APIs. The graph supplies node listings; the reader supplies gen-typed bodies.
type LintContext struct {
	graph    graphcatalog.LintReader
	reader   LintReader
	excluded map[string]bool
}

// NewLintContext creates a LintContext from a graph catalog and a deep reader.
// Either may be nil; rules must guard with Graph() != nil / Reader() != nil.
func NewLintContext(graph graphcatalog.LintReader, reader LintReader) *LintContext {
	return &LintContext{
		graph:    graph,
		reader:   reader,
		excluded: make(map[string]bool),
	}
}

// Graph returns the graph catalog, or nil if not set.
func (ctx *LintContext) Graph() graphcatalog.LintReader {
	return ctx.graph
}

// Reader returns the deep reader, or nil if not set.
func (ctx *LintContext) Reader() LintReader {
	return ctx.reader
}

// SetExcludedModules sets the list of modules to exclude from linting.
func (ctx *LintContext) SetExcludedModules(modules []string) {
	ctx.excluded = make(map[string]bool)
	for _, m := range modules {
		ctx.excluded[m] = true
	}
}

// IsExcluded returns true if the module should be excluded from linting.
func (ctx *LintContext) IsExcluded(moduleName string) bool {
	return ctx.excluded[moduleName]
}

// Entities returns all entities across non-excluded modules.
func (ctx *LintContext) Entities() []graphcatalog.EntityNode {
	if ctx.graph == nil {
		return nil
	}
	all := ctx.graph.Entities("")
	result := make([]graphcatalog.EntityNode, 0, len(all))
	for _, e := range all {
		if ctx.excluded[e.Module] {
			continue
		}
		result = append(result, e)
	}
	return result
}

// Microflows returns all microflows and nanoflows across non-excluded modules.
func (ctx *LintContext) Microflows() []graphcatalog.MicroflowNode {
	if ctx.graph == nil {
		return nil
	}
	all := ctx.graph.Microflows("")
	result := make([]graphcatalog.MicroflowNode, 0, len(all))
	for _, mf := range all {
		if ctx.excluded[mf.Module] {
			continue
		}
		result = append(result, mf)
	}
	return result
}

// Pages returns all pages across non-excluded modules.
func (ctx *LintContext) Pages() []graphcatalog.PageNode {
	if ctx.graph == nil {
		return nil
	}
	all := ctx.graph.Pages("")
	result := make([]graphcatalog.PageNode, 0, len(all))
	for _, p := range all {
		if ctx.excluded[p.Module] {
			continue
		}
		result = append(result, p)
	}
	return result
}

// Enumerations returns all enumerations across non-excluded modules.
func (ctx *LintContext) Enumerations() []graphcatalog.EnumerationNode {
	if ctx.graph == nil {
		return nil
	}
	all := ctx.graph.Enumerations("")
	result := make([]graphcatalog.EnumerationNode, 0, len(all))
	for _, e := range all {
		if ctx.excluded[e.Module] {
			continue
		}
		result = append(result, e)
	}
	return result
}

// Snippets returns all snippets across non-excluded modules.
func (ctx *LintContext) Snippets() []graphcatalog.SnippetNode {
	if ctx.graph == nil {
		return nil
	}
	all := ctx.graph.Snippets("")
	result := make([]graphcatalog.SnippetNode, 0, len(all))
	for _, s := range all {
		if ctx.excluded[s.Module] {
			continue
		}
		result = append(result, s)
	}
	return result
}

// DatabaseConnections returns all database connections.
func (ctx *LintContext) DatabaseConnections() []graphcatalog.DatabaseConnectionNode {
	if ctx.graph == nil {
		return nil
	}
	return ctx.graph.DatabaseConnections()
}

// Attributes returns the attributes of the given entity.
func (ctx *LintContext) Attributes(entityQualifiedName string) []graphcatalog.AttributeNode {
	if ctx.graph == nil {
		return nil
	}
	return ctx.graph.Attributes(entityQualifiedName)
}

// Widgets returns the widgets contained in the given page.
func (ctx *LintContext) Widgets(pageQualifiedName string) []graphcatalog.WidgetNode {
	if ctx.graph == nil {
		return nil
	}
	return ctx.graph.Widgets(pageQualifiedName)
}

// Permissions returns all entity-access permission nodes.
func (ctx *LintContext) Permissions() []graphcatalog.PermissionNode {
	if ctx.graph == nil {
		return nil
	}
	return ctx.graph.Permissions()
}

// RoleMappings returns all user-role to module-role mappings.
func (ctx *LintContext) RoleMappings() []graphcatalog.RoleMappingNode {
	if ctx.graph == nil {
		return nil
	}
	return ctx.graph.RoleMappings()
}

// UserRoleInfo represents a user role from project security.
type UserRoleInfo struct {
	Name        string
	IsAnonymous bool
	ModuleRoles []string
}

// UserRoles returns the user roles from project security via the deep reader.
func (ctx *LintContext) UserRoles() []UserRoleInfo {
	if ctx.reader == nil {
		return nil
	}
	ps, err := ctx.reader.GetProjectSecurityGen()
	if err != nil || ps == nil {
		return nil
	}

	guestRole := ps.GuestUserRoleName()
	var roles []UserRoleInfo
	for _, item := range ps.UserRolesItems() {
		ur, ok := item.(*genSec.UserRole)
		if !ok {
			continue
		}
		roles = append(roles, UserRoleInfo{
			Name:        ur.Name(),
			IsAnonymous: ur.Name() == guestRole,
			ModuleRoles: ur.ModuleRolesQualifiedNames(),
		})
	}
	return roles
}
