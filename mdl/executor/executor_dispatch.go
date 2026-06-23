// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// executeInner dispatches a statement to its registered handler.
// Prefers RegisterFuture (StmtHandlerFunc, no ExecContext). Falls back
// to deprecated StmtHandler Dispatch (with ExecContext) when no future
// handler is registered (tests that don't call SetBackend).
func (e *Executor) executeInner(ctx context.Context, stmt ast.Statement) error {
	if e.registry.HasFutureHandler(stmt) {
		return e.registry.DispatchFuture(ctx, stmt)
	}
	ectx := e.newExecContext(ctx)
	return e.registry.Dispatch(ectx, stmt)
}

// newExecContext builds an ExecContext from the current Executor state.
// Deprecated: only used by autocomplete.go, BuildGraph(), and CLI callers.
func (e *Executor) newExecContext(ctx context.Context) *ExecContext {
	if e.cache == nil {
		e.cache = &executorCache{}
	}
	if e.graphCatalog != nil {
		warmCacheFromGraph(e.cache, e.graphCatalog)
	}
	execCtx := &ExecContext{
		Context: ctx,
		Backend: e.backend,
		Logger:  e.logger,
		ExecIO: ExecIO{Output: e.output, StatusOutput: e.statusOutput, Format: e.format, Quiet: e.quiet},
		ExecSession: ExecSession{Fragments: e.fragments, Cache: e.cache},
		ExecConnection: ExecConnection{MprPath: e.mprPath, Graph: e.graphCatalog, BackendFactory: e.backendFactory},
		ExecRepos: ExecRepos{
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
		},
	}
	execCtx.initRoles()
	return execCtx
}

// Ensure ast import is used via executeInner's stmt parameter.
var _ ast.Statement = (*ast.HelpStmt)(nil)
