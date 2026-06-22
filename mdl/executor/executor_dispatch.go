// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
)

// executeInner dispatches a statement to its registered handler.
func (e *Executor) executeInner(ctx context.Context, stmt ast.Statement) error {
	ectx := e.newExecContext(ctx)
	err := e.registry.Dispatch(ectx, stmt)
	// Only sync back when the context has not been cancelled. Execute() runs
	// executeInner in a goroutine with a wall-clock timeout; if the timeout
	// fires, the goroutine keeps running but Execute() has already returned.
	// Syncing stale state back at that point would race with subsequent calls.
	// Any handler-side state changes made after cancellation are intentionally
	// lost — this is expected behavior, not a regression.
	if ctx.Err() == nil {
		e.syncBack(ectx)
	}
	return err
}

// syncBack copies mutated ExecContext fields back to the Executor so that
// the next newExecContext call picks up handler-side state changes.
//
// Fields intentionally NOT synced back (read-only from handler perspective):
//   - Output, Quiet, Logger — set once at Executor construction
//   - BackendFactory — set once at Executor construction
//   - OutputGuard — removed; writeDescribeJSON captures via Output swap only
//   - ExecuteFn, ExecuteProgramFn, FinalizeFn — bound to Executor methods, immutable
//
// Format IS synced back so that `SET format = json` takes effect for all
// subsequent statements in the same session.
func (e *Executor) syncBack(ctx *ExecContext) {
	e.backend = ctx.Backend
	e.mprPath = ctx.MprPath
	e.cache = ctx.Cache
	e.format = ctx.Format
	if ctx.Graph != nil {
		e.graphCatalog = ctx.Graph
	}
	e.sqlMgr = ctx.SqlMgr
	e.themeRegistry = ctx.ThemeRegistry
}

// newExecContext builds an ExecContext from the current Executor state.
func (e *Executor) newExecContext(ctx context.Context) *ExecContext {
	// Ensure cache exists (Connect sets it; direct Executor construction may leave it nil).
	if e.cache == nil {
		e.cache = &executorCache{}
	}
	// Warm name caches from graph if available — O(nodes) vs O(N²) backend scan.
	if e.graphCatalog != nil {
		warmCacheFromGraph(e.cache, e.graphCatalog)
	}

	execCtx := &ExecContext{
		Context: ctx,
		Backend: e.backend,
		Logger:  e.logger,
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
		ExecIO: ExecIO{
			Output:       e.output,
			StatusOutput: e.statusOutput,
			Format:       e.format,
			Quiet:        e.quiet,
		},
		ExecSession: ExecSession{
			Fragments: e.fragments,
			Cache:     e.cache,
		},
		ExecConnection: ExecConnection{
			MprPath:        e.mprPath,
			SqlMgr:         e.sqlMgr,
			ThemeRegistry:  e.themeRegistry,
			Graph:          e.graphCatalog,
			BackendFactory: e.backendFactory,
		},
		ExecCallbacks: ExecCallbacks{
			ExecuteFn:        e.Execute,
			ExecuteProgramFn: e.ExecuteProgram,
			FinalizeFn:       e.finalizeProgramExecution,
			SyncGraph: func(pg *graphcatalog.ProjectGraph) {
				e.graphCatalog = pg
			},
		},
	}
	execCtx.initRoles()
	return execCtx
}

// Ensure ast import is used via executeInner's stmt parameter.
var _ ast.Statement = (*ast.HelpStmt)(nil)
