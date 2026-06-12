// SPDX-License-Identifier: Apache-2.0

// Stage 3.1 cutover plumbing: extract modelsdk-native repositories from
// a backend.FullBackend implementation that supports them. Defined as a
// standalone interface check so the executor doesn't import the concrete
// mprbackend package — keeps the dependency direction
// (executor → backend interface, never executor → concrete impl).

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
)

// microflowsRepoProvider is the duck-typed interface a backend must
// satisfy to expose the modelsdk-native MicroflowRepository. MprBackend
// implements this in repos_provider.go.
type microflowsRepoProvider interface {
	Microflows() repos.MicroflowRepository
}

// nanoflowsRepoProvider mirrors microflowsRepoProvider for nanoflows.
type nanoflowsRepoProvider interface {
	Nanoflows() repos.NanoflowRepository
}

// securityRepoProvider mirrors microflowsRepoProvider for the security domain.
type securityRepoProvider interface {
	Security() repos.SecurityRepository
}

// javaActionsRepoProvider mirrors microflowsRepoProvider for Java actions
// (Stage 3.3.2). MprBackend implements this in repos_provider.go.
type javaActionsRepoProvider interface {
	JavaActions() repos.JavaActionRepository
}

// javaScriptActionsRepoProvider mirrors microflowsRepoProvider for
// JavaScript actions (Stage 3.3.2).
type javaScriptActionsRepoProvider interface {
	JavaScriptActions() repos.JavaScriptActionRepository
}

// domainModelsRepoProvider mirrors microflowsRepoProvider for the
// domainmodel domain (Stage 3.3.4 A0). MprBackend implements this in
// repos_provider.go.
type domainModelsRepoProvider interface {
	DomainModels() repos.DomainModelRepository
}

// workflowsRepoProvider mirrors microflowsRepoProvider for the workflows
// domain (Stage 3.3.3 A0). MprBackend implements this in repos_provider.go.
type workflowsRepoProvider interface {
	Workflows() repos.WorkflowRepository
}

// pagesRepoProvider mirrors microflowsRepoProvider for the pages
// domain (Stage 3.3.5 A0). MprBackend implements this in
// repos_provider.go.
type pagesRepoProvider interface {
	Pages() repos.PageRepository
}

// layoutsRepoProvider mirrors pagesRepoProvider for layouts.
type layoutsRepoProvider interface {
	Layouts() repos.LayoutRepository
}

// snippetsRepoProvider mirrors pagesRepoProvider for snippets.
type snippetsRepoProvider interface {
	Snippets() repos.SnippetRepository
}

// extractMicroflowsRepo returns the modelsdk-native MicroflowRepository
// if the backend supports it, else nil. Callers (handlers) MUST check
// nil and fall back to ctx.Backend during the Stage 3.x incremental
// cutover — once every microflow handler migrates, this fallback can
// be removed and the field can be made non-nil-required.
func extractMicroflowsRepo(b backend.FullBackend) repos.MicroflowRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(microflowsRepoProvider); ok {
		return p.Microflows()
	}
	return nil
}

// extractNanoflowsRepo mirrors extractMicroflowsRepo for nanoflows.
func extractNanoflowsRepo(b backend.FullBackend) repos.NanoflowRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(nanoflowsRepoProvider); ok {
		return p.Nanoflows()
	}
	return nil
}

// extractSecurityRepo mirrors extractMicroflowsRepo for the security domain.
func extractSecurityRepo(b backend.FullBackend) repos.SecurityRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(securityRepoProvider); ok {
		return p.Security()
	}
	return nil
}

// extractJavaActionsRepo mirrors extractMicroflowsRepo for Java actions
// (Stage 3.3.2 A0).
func extractJavaActionsRepo(b backend.FullBackend) repos.JavaActionRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(javaActionsRepoProvider); ok {
		return p.JavaActions()
	}
	return nil
}

// extractJavaScriptActionsRepo mirrors extractJavaActionsRepo for JS actions.
func extractJavaScriptActionsRepo(b backend.FullBackend) repos.JavaScriptActionRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(javaScriptActionsRepoProvider); ok {
		return p.JavaScriptActions()
	}
	return nil
}

// extractDomainModelsRepo mirrors extractMicroflowsRepo for the
// domainmodel domain (Stage 3.3.4 A0).
func extractDomainModelsRepo(b backend.FullBackend) repos.DomainModelRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(domainModelsRepoProvider); ok {
		return p.DomainModels()
	}
	return nil
}

// extractWorkflowsRepo mirrors extractMicroflowsRepo for the workflows
// domain (Stage 3.3.3 A0).
func extractWorkflowsRepo(b backend.FullBackend) repos.WorkflowRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(workflowsRepoProvider); ok {
		return p.Workflows()
	}
	return nil
}

// extractPagesRepo mirrors extractMicroflowsRepo for the pages domain
// (Stage 3.3.5 A0).
func extractPagesRepo(b backend.FullBackend) repos.PageRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(pagesRepoProvider); ok {
		return p.Pages()
	}
	return nil
}

// extractLayoutsRepo mirrors extractPagesRepo for layouts.
func extractLayoutsRepo(b backend.FullBackend) repos.LayoutRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(layoutsRepoProvider); ok {
		return p.Layouts()
	}
	return nil
}

// extractSnippetsRepo mirrors extractPagesRepo for snippets.
func extractSnippetsRepo(b backend.FullBackend) repos.SnippetRepository {
	if b == nil {
		return nil
	}
	if p, ok := b.(snippetsRepoProvider); ok {
		return p.Snippets()
	}
	return nil
}

// deleteMicroflowViaRepoOrBackend is the Stage 3.1 cutover bridge for
// delete-by-ID. Uses the modelsdk-native MicroflowRepository when
// available (production paths through MprBackend), else falls back to
// the legacy ctx.Backend.DeleteMicroflow (test contexts using
// MockBackend or hand-built ExecContext literals).
//
// Only delete-by-ID migrates in Stage 3.1: it has no sdk-vs-gen type
// mismatch in its parameter or return surface. List/Get/Create/Update
// stay on ctx.Backend because handlers consume sdk/microflows.Microflow
// pervasively (DescribeMicroflowToString, renderMicroflowMDL, all 60+
// activity emitters). Migrating those is Stage 3.2+ work.
func (ctx *ExecContext) deleteMicroflowViaRepoOrBackend(id model.ID) error {
	if ctx.Microflows != nil {
		return ctx.Microflows.Delete(id)
	}
	return ctx.MicroflowWriter.DeleteMicroflow(id)
}

// deleteNanoflowViaRepoOrBackend mirrors deleteMicroflowViaRepoOrBackend.
func (ctx *ExecContext) deleteNanoflowViaRepoOrBackend(id model.ID) error {
	if ctx.Nanoflows != nil {
		return ctx.Nanoflows.Delete(id)
	}
	return ctx.MicroflowWriter.DeleteNanoflow(id)
}

// isRuleViaRepoOrBackend checks whether a microflow qualified name
// refers to a Rule. Routes through the Microflows repo when available
// (modelsdk-native scan) or falls back to backend.IsRule (legacy path).
// The free-function form takes the components explicitly so callers
// without an *ExecContext (e.g., flowBuilder methods) can use it too.
func isRuleViaRepoOrBackend(repo repos.MicroflowRepository, b backend.FullBackend, qn string) (bool, error) {
	if repo != nil {
		return repo.IsRule(qn)
	}
	if b == nil {
		return false, nil
	}
	return b.IsRule(qn)
}
