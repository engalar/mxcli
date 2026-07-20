// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// executeInner dispatches a statement to its registered handler.
func (e *Executor) executeInner(ctx context.Context, stmt ast.Statement) error {
	return e.registry.Dispatch(ctx, stmt)
}

// newExecContext builds an ExecContext from the current Executor state.
// Populates ALL fields, including role-specific backend interfaces.
func (e *Executor) newExecContext(ctx context.Context) *ExecContext {
	if e.cache == nil {
		e.cache = newExecutorCache()
	}
	if e.graphCatalog != nil {
		warmCacheFromGraph(e.cache, e.graphCatalog)
	}
	execCtx := &ExecContext{
		Context: ctx,
		Backend: e.backend,
		Logger:  e.logger,

		Output:       e.output,
		StatusOutput: e.statusOutput,
		Format:       e.format,
		Quiet:        e.quiet,

		Cache:       e.cache,
		Session:     &e.cache.sessionTracker,
		Fragments:   e.fragments,
		ScriptDepth: 0,

		MprPath:       e.mprPath,
		BackendFactory: e.backendFactory,
		Graph:         e.graphCatalog,

		Microflows:        extractMicroflowsRepo(e.backend),
		Nanoflows:         extractNanoflowsRepo(e.backend),
		Security:          extractSecurityRepo(e.backend),
		JavaActions:       extractJavaActionsRepo(e.backend),
		JavaScriptActions: extractJavaScriptActionsRepo(e.backend),
		DomainModels:      extractDomainModelsRepo(e.backend),
		Workflows:         extractWorkflowsRepo(e.backend),
		Pages:             extractPagesRepo(e.backend),
		Layouts:           extractLayoutsRepo(e.backend),
		Snippets:          extractSnippetsRepo(e.backend),

		ExecuteFn:        e.Execute,
		ExecuteProgramFn: e.ExecuteProgram,
		FinalizeFn:       e.finalizeProgramExecution,
	}
	execCtx.initRoles()
	execCtx.Deps = e.buildHandlerDeps()
	return execCtx
}

// Ensure ast import is used via executeInner's stmt parameter.
var _ ast.Statement = (*ast.HelpStmt)(nil)
