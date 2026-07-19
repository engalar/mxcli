// SPDX-License-Identifier: Apache-2.0

// Package executor executes MDL AST statements against a Mendix project.
package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/diaglog"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	sqllib "github.com/mendixlabs/mxcli/sql"
)

// sessionTracker holds per-session mutable state (created/dropped tracking).
// Backend-level caching is handled by sub-backends (microflowCache, pageCache, etc.).
type sessionTracker struct {
	createdMicroflows map[string]*createdMicroflowInfo
	createdNanoflows  map[string]*createdNanoflowInfo
	createdPages      map[string]*createdPageInfo
	createdSnippets   map[string]*createdSnippetInfo

	droppedMicroflows map[string]*droppedUnitInfo
	droppedNanoflows  map[string]*droppedUnitInfo

	modifiedDomainModels map[model.ID]string
}

// executorCache — retained for backward compatibility with handler helper
// functions. New code should use sub-backend caches directly.
// Deprecated: Backend cache fields will be removed once all helpers
// route through sub-backend caches. Session tracking moves to sessionTracker.
type executorCache struct {
	sessionTracker
	metadataCache
	microflowCache
	pageCache
	domainModelCache
	securityCache
	workflowCache
	javaCache
}

// createdMicroflowInfo tracks a microflow created during this session.
type createdMicroflowInfo struct {
	ID               model.ID
	Name             string
	ModuleName       string
	ContainerID      model.ID
	ReturnEntityName string // Qualified entity name from return type (e.g., "Module.Entity")
}

// createdNanoflowInfo tracks a nanoflow created during this session.
type createdNanoflowInfo struct {
	ID               model.ID
	Name             string
	ModuleName       string
	ContainerID      model.ID
	ReturnEntityName string
}

// createdPageInfo tracks a page created during this session.
type createdPageInfo struct {
	ID          model.ID
	Name        string
	ModuleName  string
	ContainerID model.ID
}

// createdSnippetInfo tracks a snippet created during this session.
type createdSnippetInfo struct {
	ID          model.ID
	Name        string
	ModuleName  string
	ContainerID model.ID
}

// droppedUnitInfo remembers the original UnitID and ContainerID of a document
// dropped during this session so that a subsequent CREATE OR REPLACE/MODIFY
// with the same qualified name can reuse them instead of generating new UUIDs.
type droppedUnitInfo struct {
	ID           model.ID
	ContainerID  model.ID
	AllowedRoles []model.ID
}

// getEntityNames returns the entity name lookup map, using the pre-warmed cache if available.
func getEntityNames(ctx *ExecContext, h *ContainerHierarchy) map[model.ID]string {
	if ctx.Cache != nil && len(ctx.Cache.entityNames) > 0 {
		return ctx.Cache.entityNames
	}
	entityNames, err := buildAllEntityNamesGen(ctx)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warn("getEntityNames: buildAllEntityNamesGen failed", "error", err)
		}
		return map[model.ID]string{}
	}
	if ctx.Cache != nil {
		ctx.Cache.entityNames = entityNames
	}
	return entityNames
}

// invalidateMicroflowsCache clears the pre-warmed microflowNames map.
// Call from any write path that affects microflow or nanoflow units.
//
// Stage 3.2.6.5: the legacy `Cache.microflows` slice (sdk-typed) and
// the `getAllMicroflows` / `getMicroflowNames` helpers are gone — the
// only consumer (cmd_catalog.go's preWarmCache) now populates
// microflowNames directly from ctx.Microflows / ctx.Backend.
func invalidateMicroflowsCache(ctx *ExecContext) {
	if ctx.Cache != nil {
		ctx.Cache.microflowNames = nil
		ctx.Cache.microflowsWithContainerGen = nil
		ctx.Cache.nanoflowsWithContainerGen = nil
	}
}

// invalidateAllDocumentCaches clears all per-document-type caches in one call.
// Use after any rename/create/drop that might affect multiple listing caches,
// instead of calling individual invalidateXxxCache functions.
func invalidateAllDocumentCaches(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.microflowNames = nil
	ctx.Cache.microflowsWithContainerGen = nil
	ctx.Cache.nanoflowsWithContainerGen = nil
	ctx.Cache.pagesWithContainerGen = nil
	ctx.Cache.layoutsWithContainerGen = nil
	ctx.Cache.snippetsWithContainerGen = nil
	ctx.Cache.workflowsWithContainerGen = nil
	ctx.Cache.javaActionsWithContainerGen = nil
	ctx.Cache.javaScriptActionsWithContainerGen = nil
}

// getPageNames returns the page name lookup map, using the pre-warmed cache if available.
func getPageNames(ctx *ExecContext, h *ContainerHierarchy) map[model.ID]string {
	if ctx.Cache != nil && len(ctx.Cache.pageNames) > 0 {
		return ctx.Cache.pageNames
	}
	pageNames := make(map[model.ID]string)
	pgPairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warn("getPageNames: ListPagesGen failed", "error", err)
		}
		return pageNames
	}
	for _, pair := range pgPairs {
		pg := pair.Elem
		pageNames[model.ID(pg.ID())] = h.GetQualifiedName(model.ID(pair.ContainerID), pg.Name())
	}
	if ctx.Cache != nil {
		ctx.Cache.pageNames = pageNames
	}
	return pageNames
}

const (
	// maxOutputLines is the per-statement line limit. Statements that produce more
	// lines than this are aborted to prevent runaway output from infinite loops.
	maxOutputLines = 10_000
	// defaultExecuteTimeout is the maximum wall-clock time allowed for a single
	// statement when MXCLI_EXEC_TIMEOUT is not set.
	defaultExecuteTimeout = 5 * time.Minute
)

// configuredExecuteTimeout returns the per-statement wall-clock timeout. The
// value is read from the MXCLI_EXEC_TIMEOUT environment variable on every call
// so long-running audits can opt into a higher ceiling without recompiling.
//
// Accepts either a Go duration ("12m", "2h30m") or a bare number of seconds
// ("900"). Falls back to defaultExecuteTimeout when the variable is unset,
// empty, or fails to parse.
func configuredExecuteTimeout() time.Duration {
	raw := os.Getenv("MXCLI_EXEC_TIMEOUT")
	if raw == "" {
		return defaultExecuteTimeout
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return defaultExecuteTimeout
}

// BackendFactory creates a new backend instance for connecting to a project.
type BackendFactory func() backend.ConnectionBackend

// Executor executes MDL statements against a Mendix project.
type Executor struct {
	backend        backend.FullBackend // domain backend (populated on Connect)
	backendFactory BackendFactory      // factory for creating new backend instances
	output         io.Writer
	statusOutput   io.Writer    // writer for status/informational messages (stderr by default)
	guard          *outputGuard // line-limit wrapper around output
	mprPath        string
	cache          *executorCache
	graphCatalog   *graphcatalog.ProjectGraph         // in-memory project graph (lazy build) for code-search + lint
	quiet          bool                               // suppress connection and status messages
	format         OutputFormat                       // output format (table, json)
	logger         *diaglog.Logger                    // session diagnostics logger (nil = no logging)
	fragments      map[string]*ast.DefineFragmentStmt // script-scoped fragment definitions
	registry       *Registry                          // statement dispatch registry

	// perfStats accumulates per-statement execution timing so the
	// caller can print a summary when the script finishes.
	perfStats []perfStmt

	// reregisterHandlers holds closures that re-register subpackage handlers
	// (misc, microflow, page, etc.) with a fresh HandlerDeps. Called after
	// Connect sets the backend so handler closures capture a live backend
	// instead of the nil-backend snapshot taken at Executor construction time.
	reregisterHandlers []func(*HandlerDeps)
}

// perfStmt captures one statement's execution timing.
type perfStmt struct {
	Type     string
	Summary  string
	Duration time.Duration
	Err      bool
}

// New creates a new executor with the given output writer.
// Status messages (e.g. "Connected to:") go to os.Stderr by default so they
// do not pollute stdout when the caller redirects stdout to a file.
func New(output io.Writer) *Executor {
	guard := newOutputGuard(output, maxOutputLines)
	e := &Executor{
		output:       guard,
		statusOutput: os.Stderr,
		guard:        guard,
		fragments:    make(map[string]*ast.DefineFragmentStmt),
		registry:     NewRegistry(),
	}
	e.registerNonBackendHandlers()
	return e
}

// registerNonBackendHandlers registers handlers that work without a backend.
// Called from New(). The remaining handlers are registered by registerFutureOverlays
// (called from SetBackend), which overrides these minimal registrations.
func (e *Executor) registerNonBackendHandlers() {
	r := e.registry
	r.RegisterFuture("Connect", func(ctx context.Context, stmt ast.Statement) error {
		return execConnectFuture(ctx, stmt.(*ast.ConnectStmt), e)
	})
	r.RegisterFuture("Disconnect", func(ctx context.Context, stmt ast.Statement) error {
		return execDisconnectFuture(ctx, e)
	})
	r.RegisterFuture("Exit", func(ctx context.Context, stmt ast.Statement) error {
		return execExitFuture(ctx)
	})
	r.RegisterFuture("Help", func(ctx context.Context, stmt ast.Statement) error {
		return execHelpFuture(ctx, stmt.(*ast.HelpStmt), e.output, e.format)
	})
	// Fragment handlers — only need Output and e.fragments, no backend.
	r.RegisterFuture("DefineFragment", func(ctx context.Context, stmt ast.Statement) error {
		deps := &HandlerDeps{Output: e.output, Fragments: e.fragments}
		return ExecDefineFragmentFn(ctx, stmt.(*ast.DefineFragmentStmt), deps)
	})
	// Show and Describe need to be registered so non-backend subtypes
	// (e.g. ShowFragments, DescribeFragment) work before connect.
	// They delegate to the full ExecShow/ExecDescribe which handle
	// individual subtypes; backend-dependent subtypes will return
	// "not connected" errors at runtime.
	r.RegisterFuture("Show", func(ctx context.Context, stmt ast.Statement) error {
		s := stmt.(*ast.ShowStmt)
		if s.ObjectType == ast.ShowFragments {
			listFragmentsFuture(ctx, e.output, e.fragments)
			return nil
		}
		return mdlerrors.NewNotConnected()
	})
	r.RegisterFuture("Describe", func(ctx context.Context, stmt ast.Statement) error {
		s := stmt.(*ast.DescribeStmt)
		if entry, ok := describeHandlers[s.ObjectType]; ok {
			name := s.Name.String()
			deps := &HandlerDeps{Output: e.output, Format: e.format, Fragments: e.fragments}
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		}
		return mdlerrors.NewNotConnected()
	})
}

// SetBackendFactory sets the factory function used to create backend instances on Connect.
func (e *Executor) SetBackendFactory(f BackendFactory) {
	e.backendFactory = f
}

// SetBackend installs an already-connected backend on the executor and
// ensures the executor cache is initialized. Used by callers (e.g.
// `mxcli export`) that own the backend lifecycle outside the normal
// Connect/Disconnect MDL flow.
func (e *Executor) SetBackend(b backend.ConnectionBackend) {
	e.backend = b.(backend.FullBackend)
	if e.cache == nil {
		e.cache = &executorCache{}
	}
	e.registerFutureOverlays()
}

// SetQuiet enables or disables quiet mode (suppresses connection/status messages).
func (e *Executor) SetQuiet(quiet bool) {
	e.quiet = quiet
}

// SetFormat sets the output format (table or json).
func (e *Executor) SetFormat(f OutputFormat) {
	e.format = f
}

// Registry returns the statement registry for external handler registration.
func (e *Executor) Registry() *Registry {
	return e.registry
}

// BuildHandlerDeps constructs a HandlerDeps from the current executor state.
func (e *Executor) BuildHandlerDeps() *HandlerDeps {
	return e.buildHandlerDeps()
}

// AddReregister adds a closure that re-registers subpackage handlers
// with a fresh HandlerDeps. Called after Connect sets the backend.
func (e *Executor) AddReregister(fn func(*HandlerDeps)) {
	e.reregisterHandlers = append(e.reregisterHandlers, fn)
}

// reRegisterAll calls all stored re-registration closures with a fresh
// HandlerDeps, so handler closures capture the live backend.
func (e *Executor) reRegisterAll() {
	deps := e.buildHandlerDeps()
	for _, fn := range e.reregisterHandlers {
		fn(deps)
	}
}

// SetLogger sets the diagnostics logger for session logging.
func (e *Executor) SetLogger(l *diaglog.Logger) {
	e.logger = l
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		s := d.Seconds()
		return fmt.Sprintf("%.2fs", s)
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// FormatDuration is the exported equivalent of formatDuration.
func FormatDuration(d time.Duration) string {
	return formatDuration(d)
}

// PerfReport returns a human-readable performance report of all
// statements executed during this session. The report is written to w.
func (e *Executor) PerfReport(w io.Writer) {
	if len(e.perfStats) == 0 {
		return
	}
	var total time.Duration
	for _, ps := range e.perfStats {
		total += ps.Duration
	}

	// Count statement types.
	typeCount := make(map[string]int)
	for _, ps := range e.perfStats {
		typeCount[ps.Type]++
	}
	typeSummary := ""
	first := true
	for t, n := range typeCount {
		if !first {
			typeSummary += ", "
		}
		typeSummary += fmt.Sprintf("%d %s%s", n, t, pluralSuffix(n))
		first = false
	}
	if typeSummary == "" {
		typeSummary = "0 statements"
	}

	fmt.Fprintf(w, "\n── Performance ──────────────────────────────\n")
	fmt.Fprintf(w, "  Statements: %s\n", typeSummary)
	fmt.Fprintf(w, "  Total time: %s\n", formatDuration(total))
	fmt.Fprintf(w, "  ── Per statement ──\n")
	for _, ps := range e.perfStats {
		mark := " "
		if ps.Err {
			mark = "✗"
		}
		dur := formatDuration(ps.Duration)
		summary := ps.Summary
		if len(summary) > 50 {
			summary = summary[:47] + "..."
		}
		fmt.Fprintf(w, "  %s %s  %s  %s\n", mark, dur, ps.Type, summary)
	}
}

func pluralSuffix(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}

// SetProgressOut sets the writer for real-time progress messages.
// Defaults to os.Stderr. In daemon mode the caller wires this to a
// "progress" frame writer so the launcher can print progress immediately.
func (e *Executor) SetProgressOut(w io.Writer) {
	e.statusOutput = w
}

// Execute runs a single MDL statement with output-line and wall-clock guards.
func (e *Executor) Execute(stmt ast.Statement) error {
	start := time.Now()

	if e.guard != nil {
		e.guard.reset()
	}

	executeTimeout := configuredExecuteTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), executeTimeout)
	defer cancel()

	var err error
	done := make(chan struct{}, 1)
	go func() {
		err = e.executeInner(ctx, stmt)
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-ctx.Done():
		err = mdlerrors.NewValidationf("statement timed out after %v", executeTimeout)
	}

	elapsed := time.Since(start)

	if e.logger != nil {
		e.logger.Command(stmt.TypeName(), stmtSummary(stmt), elapsed, err)
	}

	e.perfStats = append(e.perfStats, perfStmt{
		Type:     stmt.TypeName(),
		Summary:  stmtSummary(stmt),
		Duration: elapsed,
		Err:      err != nil,
	})

	if err != nil {
		err = fmt.Errorf("%w (duration: %v)", err, elapsed)
	}
	return err
}

// ExecuteProgram runs all statements in a program.
// When the backend supports ScriptTransactionBackend, all writes are
// batched into a single SQLite transaction — dramatically reducing I/O
// overhead for multi-statement files (e.g. 8-file capstone exec).
func (e *Executor) ExecuteProgram(prog *ast.Program) error {
	// Open a script transaction if the backend supports it.
	var stx backend.ScriptTransaction
	if sbe, ok := e.backend.(backend.ScriptTransactionBackend); ok {
		if tx, err := sbe.BeginScriptTransaction(); err == nil {
			stx = tx
		}
	}

	// Collect all names defined in the script for forward-reference hints.
	allDefined := newScriptContext()
	allDefined.collectDefinitions(prog)

	// Track which names have been created so far.
	created := newScriptContext()

	var execErr error
	for _, stmt := range prog.Statements {
		if err := e.Execute(stmt); err != nil {
			execErr = annotateForwardRef(err, stmt, created, allDefined)
			break
		}
		created.collectSingle(stmt)
	}

	// Run domain-model reconciliation while scriptBuf is still open so its
	// writes (writeUnitContents) are batched into the same atomic commit.
	if execErr == nil {
		execErr = e.finalizeProgramExecution()
	}

	if stx != nil {
		if execErr != nil {
			_ = stx.Rollback()
		} else if commitErr := stx.Commit(); commitErr != nil {
			return commitErr
		}
	}
	return execErr
}

// finalizeProgramExecution runs post-execution reconciliation on modified domain models.
func (e *Executor) finalizeProgramExecution() error {
	if e.backend == nil || !e.backend.IsConnected() || e.cache == nil || len(e.cache.modifiedDomainModels) == 0 {
		return nil
	}

	for moduleID, moduleName := range e.cache.modifiedDomainModels {
		var dm *genDm.DomainModel
		if e.cache.domainModelByModule != nil {
			dm = e.cache.domainModelByModule[moduleID]
		}
		if dm == nil {
			var err error
			dm, err = e.backend.GetDomainModelGen(moduleID)
			if err != nil {
				continue // module may not have a domain model
			}
		}
		if dm == nil {
			continue
		}

		msgs, err := e.backend.ReconcileMemberAccesses(model.ID(dm.ID()), moduleName)
		if err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("reconcile security for module %s", moduleName), err)
		}
		if len(msgs) > 0 && !e.quiet {
			for _, msg := range msgs {
				fmt.Fprintf(e.output, "  [%s] reconciled: %s\n", moduleName, msg)
			}
		}
	}

	// Clear tracking
	e.cache.modifiedDomainModels = nil
	return nil
}

// Graph returns the in-memory project graph, or nil if not built.
func (e *Executor) Graph() *graphcatalog.ProjectGraph {
	return e.graphCatalog
}

// BuildGraph builds (or rebuilds) the in-memory project graph and returns it.
// Used by CLI lint/report commands that need a graphcatalog.LintReader.
func (e *Executor) BuildGraph() (*graphcatalog.ProjectGraph, error) {
	ctx := e.newExecContext(context.Background())
	if err := buildGraph(ctx); err != nil {
		return nil, err
	}
	// Inline syncBack: propagate graph state back to Executor
	if ctx.Graph != nil {
		e.graphCatalog = ctx.Graph
	}
	return ctx.Graph, nil
}

// ModuleOverview builds the module overview as JSON.
func (e *Executor) ModuleOverview() error {
	ctx := e.newExecContext(context.Background())
	return ModuleOverview(ctx)
}

// Search executes a SEARCH query with the given format.
// Any non-empty format enables JSON output; the SEARCH text is returned directly.
func (e *Executor) Search(query, format string) error {
	return fmt.Errorf("SEARCH command is not available in batch mode; use mxcli exec with SEARCH statement")
}

// IsConnected returns true if connected to a project.
func (e *Executor) IsConnected() bool {
	return e.backend != nil && e.backend.IsConnected()
}

// Backend returns the full backend, or nil if not connected.
//
// Deprecated: Callers should use role-specific accessors (LintReader(),
// CatalogReader(), etc.) or rely on ExecContext role fields instead.
func (e *Executor) Backend() backend.FullBackend {
	if e.backend == nil || !e.backend.IsConnected() {
		return nil
	}
	return e.backend
}

// LintReader returns a lint-compatible reader, or nil if not connected.
// Prefer this over Backend() when creating lint contexts.
func (e *Executor) LintReader() linter.LintReader {
	if e.backend == nil || !e.backend.IsConnected() {
		return nil
	}
	return e.backend
}

// Close closes the connection to the project and all SQL connections.
func (e *Executor) Close() error {
	var closeErr error
	if e.backend != nil && e.backend.IsConnected() {
		// Close SQL connections before the backend so the backend can cleanly
		// disconnect without pending SQL operations.
		ec := e.newExecContext(context.Background())
		if ec.SqlMgr != nil {
			ec.SqlMgr.CloseAll()
		}
		closeErr = e.backend.Disconnect()
		e.backend = nil
	}
	return closeErr
}

// ----------------------------------------------------------------------------
// Cache and Tracking
// ----------------------------------------------------------------------------

// rememberDroppedMicroflow records the UnitID and ContainerID of a microflow
// that is about to be deleted via DROP MICROFLOW. A follow-up CREATE OR
// REPLACE/MODIFY for the same qualified name will reuse both instead of
// generating a fresh UUID and defaulting to the module root, so Studio Pro
// continues to see the unit as "updated in place" rather than a delete+insert
// pair.
func rememberDroppedMicroflow(ctx *ExecContext, qualifiedName string, id, containerID model.ID, allowedRoles []model.ID) {
	if ctx == nil || qualifiedName == "" || id == "" {
		return
	}
	if ctx.Cache == nil {
		ctx.Cache = &executorCache{}
	}
	if ctx.Cache.droppedMicroflows == nil {
		ctx.Cache.droppedMicroflows = make(map[string]*droppedUnitInfo)
	}
	ctx.Cache.droppedMicroflows[qualifiedName] = &droppedUnitInfo{
		ID:           id,
		ContainerID:  containerID,
		AllowedRoles: cloneRoleIDs(allowedRoles),
	}
}

// consumeDroppedMicroflow returns the original IDs of a microflow dropped
// earlier in this session (if any) and removes the entry so repeated CREATEs
// don't collide on the same ID. Returns nil when nothing was remembered.
func consumeDroppedMicroflow(ctx *ExecContext, qualifiedName string) *droppedUnitInfo {
	if ctx == nil || ctx.Cache == nil || ctx.Cache.droppedMicroflows == nil {
		return nil
	}
	info, ok := ctx.Cache.droppedMicroflows[qualifiedName]
	if !ok {
		return nil
	}
	delete(ctx.Cache.droppedMicroflows, qualifiedName)
	return info
}

// rememberDroppedNanoflow records the UnitID and ContainerID of a nanoflow
// that is about to be deleted so a subsequent CREATE OR REPLACE/MODIFY can reuse them.
func rememberDroppedNanoflow(ctx *ExecContext, qualifiedName string, id, containerID model.ID, allowedRoles []model.ID) {
	if ctx == nil || qualifiedName == "" || id == "" {
		return
	}
	if ctx.Cache == nil {
		ctx.Cache = &executorCache{}
	}
	if ctx.Cache.droppedNanoflows == nil {
		ctx.Cache.droppedNanoflows = make(map[string]*droppedUnitInfo)
	}
	ctx.Cache.droppedNanoflows[qualifiedName] = &droppedUnitInfo{
		ID:           id,
		ContainerID:  containerID,
		AllowedRoles: cloneRoleIDs(allowedRoles),
	}
}

// consumeDroppedNanoflow returns the original IDs of a nanoflow dropped
// earlier in this session (if any) and removes the entry.
func consumeDroppedNanoflow(ctx *ExecContext, qualifiedName string) *droppedUnitInfo {
	if ctx == nil || ctx.Cache == nil || ctx.Cache.droppedNanoflows == nil {
		return nil
	}
	info, ok := ctx.Cache.droppedNanoflows[qualifiedName]
	if !ok {
		return nil
	}
	delete(ctx.Cache.droppedNanoflows, qualifiedName)
	return info
}

// ExecRepos holds Stage 3 modelsdk-native repositories.
type ExecRepos struct {
	Microflows        repos.MicroflowRepository
	Nanoflows         repos.NanoflowRepository
	Security          repos.SecurityRepository
	JavaActions       repos.JavaActionRepository
	JavaScriptActions repos.JavaScriptActionRepository
	DomainModels      repos.DomainModelRepository
	Workflows         repos.WorkflowRepository
	Pages             repos.PageRepository
	Layouts           repos.LayoutRepository
	Snippets          repos.SnippetRepository
}

// ExecIO holds output writers and formatting settings.
type ExecIO struct {
	Output       io.Writer
	StatusOutput io.Writer
	Format       OutputFormat
	Quiet        bool
}

// ExecSession holds per-session mutable state.
type ExecSession struct {
	Cache                             *executorCache  // Deprecated: use Session for tracking, sub-backend caches for data
	Session                           *sessionTracker // Per-session create/drop tracking
	Fragments                         map[string]*ast.DefineFragmentStmt
	Settings                          map[string]any
	ScriptDepth                       int
	DescribingMicroflowHasReturnValue bool
}

// ExecConnection holds project connection state and external services.
type ExecConnection struct {
	MprPath        string
	BackendFactory BackendFactory
	SqlMgr         *sqllib.Manager
	ThemeRegistry  *ThemeRegistry
	// Graph is the in-memory project graph. *ProjectGraph implements both
	// graphcatalog.TraversalReader (executor code-search) and LintReader
	// (linter), so a single concrete-typed field serves both consumers.
	Graph *graphcatalog.ProjectGraph
	// Perf collects timing statistics when non-nil.
	Perf *PerfTimer
}

// ExecCallbacks holds function references for recursive execution.
type ExecCallbacks struct {
	ExecuteFn        func(ast.Statement) error
	ExecuteProgramFn func(*ast.Program) error
	FinalizeFn       func() error
	SyncGraph        func(*graphcatalog.ProjectGraph)
}

// ExecContext carries all dependencies a statement handler needs.
//
// Fields are grouped into embedded sub-structs by responsibility.
// All ctx.Xxx field accesses continue to work via Go field promotion;
// only struct literal initializers need to use the sub-struct names.
type ExecContext struct {
	context.Context

	// Backend is retained for backward compat (initRoles fallback,
	// ImportBuffer type assertion). New code should use role fields.
	// TODO: remove once mock backend implements BackendFactory.
	Backend backend.FullBackend
	// backendFactory holds the backend as BackendFactory for initRoles().
	// Set alongside Backend in executor_connect.go.
	backendFactory backend.BackendFactory

	// Logger is the session diagnostics logger (nil = no logging).
	Logger *diaglog.Logger

	// Role-specific backend interfaces. Populated lazily from Backend.
	// Handler code should use these instead of ctx.Backend when only
	// one domain is needed.
	ModuleLister                backend.ModuleLister
	ModuleWriter                backend.ModuleWriter
	DomainModelReader           backend.DomainModelReader
	DomainModelWriter           backend.DomainModelWriter
	MicroflowReader             backend.MicroflowReader
	MicroflowWriter             backend.MicroflowWriter
	WorkflowReader              backend.WorkflowReader
	WorkflowWriter              backend.WorkflowWriter
	PageReader                  backend.PageReader
	PageWriter                  backend.PageWriter
	JavaActionReader            backend.JavaActionReader
	JavaActionWriter            backend.JavaActionWriter
	JavaScriptActionWriter      backend.JavaScriptActionWriter
	EnumerationReader           backend.EnumerationReader
	EnumerationWriter           backend.EnumerationWriter
	ConstantReader              backend.ConstantReader
	ConstantWriter              backend.ConstantWriter
	SettingsReader              backend.SettingsReader
	SettingsWriter              backend.SettingsWriter
	MappingReader               backend.MappingReader
	MappingWriter               backend.MappingWriter
	UnitReader                  backend.UnitReader
	UnitWriter                  backend.UnitWriter
	NavigationReader            backend.NavigationReader
	NavigationWriter            backend.NavigationWriter
	ImageCollectionWriter       backend.ImageCollectionWriter
	ScheduledEventReader        backend.ScheduledEventReader
	ServiceLister               backend.ServiceLister
	ServiceWriter               backend.ServiceWriter
	MetadataReader              backend.MetadataReader
	CacheInvalidator            backend.CacheInvalidator
	ConnectionManager           backend.ConnectionManager
	FolderManager               backend.FolderManager
	ModuleSettingsReader        backend.ModuleSettingsReader
	ModuleSettingsWriter        backend.ModuleSettingsWriter
	RenameManager               backend.RenameManager
	SecurityProjectManager      backend.SecurityProjectManager
	SecurityModuleManager       backend.SecurityModuleManager
	SecurityEntityAccessManager backend.SecurityEntityAccessManager
	PageModelAccess             backend.PageModelAccess
	PageMutationOperator        backend.PageMutationOperator
	WorkflowMutationOperator    backend.WorkflowMutationOperator
	WidgetBuilder               backend.WidgetBuilder
	ScriptTransactionManager    backend.ScriptTransactionManager
	AgentEditorOperator         backend.AgentEditorOperator

	ExecRepos
	ExecIO
	ExecSession
	ExecConnection
	ExecCallbacks
}

// initRoles populates the role-specific backend fields. Prefers
// backendFactory (set directly in executor_connect.go) over
// ctx.Backend (deprecated FullBackend) for backward compat.
func (ctx *ExecContext) initRoles() {
	if ctx == nil {
		return
	}
	bf := ctx.backendFactory
	if bf == nil && ctx.Backend != nil {
		// Keep fallback for mock backends that don't implement BackendFactory.
		bf, _ = ctx.Backend.(backend.BackendFactory)
	}
	if bf != nil {
		ctx.ModuleLister = bf.ModuleLister()
		ctx.ModuleWriter = bf.ModuleWriter()
		ctx.DomainModelReader = bf.DomainModelReader()
		ctx.DomainModelWriter = bf.DomainModelWriter()
		ctx.MicroflowReader = bf.MicroflowReader()
		ctx.MicroflowWriter = bf.MicroflowWriter()
		ctx.WorkflowReader = bf.WorkflowReader()
		ctx.WorkflowWriter = bf.WorkflowWriter()
		ctx.PageReader = bf.PageReader()
		ctx.PageWriter = bf.PageWriter()
		ctx.JavaActionReader = bf.JavaActionReader()
		ctx.JavaActionWriter = bf.JavaActionWriter()
		ctx.JavaScriptActionWriter = bf.JavaScriptActionWriter()
		ctx.EnumerationReader = bf.EnumerationReader()
		ctx.EnumerationWriter = bf.EnumerationWriter()
		ctx.ConstantReader = bf.ConstantReader()
		ctx.ConstantWriter = bf.ConstantWriter()
		ctx.SettingsReader = bf.SettingsReader()
		ctx.SettingsWriter = bf.SettingsWriter()
		ctx.MappingReader = bf.MappingReader()
		ctx.MappingWriter = bf.MappingWriter()
		ctx.UnitReader = bf.UnitReader()
		ctx.UnitWriter = bf.UnitWriter()
		ctx.NavigationReader = bf.NavigationReader()
		ctx.NavigationWriter = bf.NavigationWriter()
		ctx.ImageCollectionWriter = bf.ImageCollectionWriter()
		ctx.ScheduledEventReader = bf.ScheduledEventReader()
		ctx.ServiceLister = bf.ServiceLister()
		ctx.ServiceWriter = bf.ServiceWriter()
		ctx.MetadataReader = bf.MetadataReader()
		ctx.ConnectionManager = bf
		ctx.FolderManager = bf.FolderManager()
		ctx.ModuleSettingsReader = bf.ModuleSettingsReader()
		ctx.ModuleSettingsWriter = bf.ModuleSettingsWriter()
		ctx.RenameManager = bf.RenameManager()
		ctx.SecurityProjectManager = bf.SecurityProjectManager()
		ctx.SecurityModuleManager = bf.SecurityModuleManager()
		ctx.SecurityEntityAccessManager = bf.SecurityEntityAccessManager()
		ctx.PageModelAccess = bf.PageModelAccess()
		ctx.PageMutationOperator = bf.PageMutationOperator()
		ctx.WorkflowMutationOperator = bf.WorkflowMutationOperator()
		ctx.WidgetBuilder = bf.WidgetBuilder()
		ctx.ScriptTransactionManager = bf.ScriptTransactionManager()
		ctx.AgentEditorOperator = bf.AgentEditorOperator()
		return
	}
	// Fallback: ctx.Backend (deprecated FullBackend).
	// Kept for mock backends that don't implement BackendFactory.
	if ctx.Backend == nil {
		return
	}
	ctx.ModuleLister = ctx.Backend
	ctx.ModuleWriter = ctx.Backend
	ctx.DomainModelReader = ctx.Backend
	ctx.DomainModelWriter = ctx.Backend
	ctx.MicroflowReader = ctx.Backend
	ctx.MicroflowWriter = ctx.Backend
	ctx.WorkflowReader = ctx.Backend
	ctx.WorkflowWriter = ctx.Backend
	ctx.PageReader = ctx.Backend
	ctx.PageWriter = ctx.Backend
	ctx.JavaActionReader = ctx.Backend
	ctx.JavaActionWriter = ctx.Backend
	ctx.JavaScriptActionWriter = ctx.Backend
	ctx.EnumerationReader = ctx.Backend
	ctx.EnumerationWriter = ctx.Backend
	ctx.ConstantReader = ctx.Backend
	ctx.ConstantWriter = ctx.Backend
	ctx.SettingsReader = ctx.Backend
	ctx.SettingsWriter = ctx.Backend
	ctx.MappingReader = ctx.Backend
	ctx.MappingWriter = ctx.Backend
	ctx.UnitReader = ctx.Backend
	ctx.UnitWriter = ctx.Backend
	ctx.NavigationReader = ctx.Backend
	ctx.NavigationWriter = ctx.Backend
	ctx.ImageCollectionWriter = ctx.Backend
	ctx.ScheduledEventReader = ctx.Backend
	ctx.ServiceLister = ctx.Backend
	ctx.ServiceWriter = ctx.Backend
	ctx.MetadataReader = ctx.Backend
	ctx.ConnectionManager = ctx.Backend
	ctx.FolderManager = ctx.Backend
	ctx.ModuleSettingsReader = ctx.Backend
	ctx.ModuleSettingsWriter = ctx.Backend
	ctx.RenameManager = ctx.Backend
	ctx.SecurityProjectManager = ctx.Backend
	ctx.SecurityModuleManager = ctx.Backend
	ctx.SecurityEntityAccessManager = ctx.Backend
	ctx.PageModelAccess = ctx.Backend
	ctx.PageMutationOperator = ctx.Backend
	ctx.WorkflowMutationOperator = ctx.Backend
	ctx.WidgetBuilder = ctx.Backend
	ctx.ScriptTransactionManager = ctx.Backend
	ctx.AgentEditorOperator = ctx.Backend
}

// Connected returns true if a project is connected.
func (ctx *ExecContext) Connected() bool {
	if ctx.ConnectionManager != nil {
		return ctx.ConnectionManager.IsConnected()
	}
	return false
}

// ConnectedForWrite returns true if a project is connected and the backend
// supports write operations. Currently equivalent to Connected() since
// MprBackend always supports writes.
func (ctx *ExecContext) ConnectedForWrite() bool {
	return ctx.Connected()
}

// InvalidateCache clears the executor cache and invalidates backend
// caches so subsequent statements see fresh data.
func (ctx *ExecContext) InvalidateCache() {
	ctx.Cache = nil
	if ctx.MetadataReader != nil {
		ctx.MetadataReader.InvalidateCache()
	}
	if ctx.CacheInvalidator != nil {
		ctx.CacheInvalidator.InvalidateCache()
	}
}

// GetThemeRegistry returns the cached theme registry, loading it lazily
// from the project's theme sources on first access.
func (ctx *ExecContext) GetThemeRegistry() *ThemeRegistry {
	if ctx.ThemeRegistry != nil {
		return ctx.ThemeRegistry
	}
	if ctx.MprPath == "" {
		return nil
	}
	projectDir := filepath.Dir(ctx.MprPath)
	registry, err := loadThemeRegistry(projectDir)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warn("failed to load theme registry", "error", err)
		}
		return nil
	}
	ctx.ThemeRegistry = registry
	return ctx.ThemeRegistry
}

// ensureCache initializes the ExecContext cache if nil.
func (ctx *ExecContext) ensureCache() {
	if ctx.Cache == nil {
		ctx.Cache = &executorCache{}
	}
}

// trackModifiedDomainModel records a domain model that was modified during
// execution, so it can be reconciled at the end of the program.
func (ctx *ExecContext) trackModifiedDomainModel(moduleID model.ID, moduleName string) {
	if !ctx.Connected() {
		return
	}
	ctx.ensureCache()
	if ctx.Cache.modifiedDomainModels == nil {
		ctx.Cache.modifiedDomainModels = make(map[model.ID]string)
	}
	ctx.Cache.modifiedDomainModels[moduleID] = moduleName
}

// trackCreatedMicroflow registers a microflow created during this session.
func (ctx *ExecContext) trackCreatedMicroflow(moduleName, mfName string, id, containerID model.ID, returnEntityName string) {
	ctx.ensureCache()
	if ctx.Cache.createdMicroflows == nil {
		ctx.Cache.createdMicroflows = make(map[string]*createdMicroflowInfo)
	}
	qualifiedName := moduleName + "." + mfName
	ctx.Cache.createdMicroflows[qualifiedName] = &createdMicroflowInfo{
		ID:               id,
		Name:             mfName,
		ModuleName:       moduleName,
		ContainerID:      containerID,
		ReturnEntityName: returnEntityName,
	}
}

// trackCreatedNanoflow registers a nanoflow created during this session.
// The cache is consumed by execDropNanoflow (cleanup on DROP) and will be
// used by future resolvers for session-local nanoflow lookups (matching
// the createdMicroflows pattern).
func (ctx *ExecContext) trackCreatedNanoflow(moduleName, nfName string, id, containerID model.ID, returnEntityName string) {
	ctx.ensureCache()
	if ctx.Cache.createdNanoflows == nil {
		ctx.Cache.createdNanoflows = make(map[string]*createdNanoflowInfo)
	}
	qualifiedName := moduleName + "." + nfName
	ctx.Cache.createdNanoflows[qualifiedName] = &createdNanoflowInfo{
		ID:               id,
		Name:             nfName,
		ModuleName:       moduleName,
		ContainerID:      containerID,
		ReturnEntityName: returnEntityName,
	}
}

// trackCreatedPage registers a page created during this session.
func (ctx *ExecContext) trackCreatedPage(moduleName, pageName string, id, containerID model.ID) {
	ctx.ensureCache()
	if ctx.Cache.createdPages == nil {
		ctx.Cache.createdPages = make(map[string]*createdPageInfo)
	}
	qualifiedName := moduleName + "." + pageName
	ctx.Cache.createdPages[qualifiedName] = &createdPageInfo{
		ID:          id,
		Name:        pageName,
		ModuleName:  moduleName,
		ContainerID: containerID,
	}
}

// trackCreatedSnippet registers a snippet created during this session.
func (ctx *ExecContext) trackCreatedSnippet(moduleName, snippetName string, id, containerID model.ID) {
	ctx.ensureCache()
	if ctx.Cache.createdSnippets == nil {
		ctx.Cache.createdSnippets = make(map[string]*createdSnippetInfo)
	}
	qualifiedName := moduleName + "." + snippetName
	ctx.Cache.createdSnippets[qualifiedName] = &createdSnippetInfo{
		ID:          id,
		Name:        snippetName,
		ModuleName:  moduleName,
		ContainerID: containerID,
	}
}

// getCreatedMicroflow returns info about a microflow created during this
// session, or nil if not found.
func (ctx *ExecContext) getCreatedMicroflow(qualifiedName string) *createdMicroflowInfo {
	if ctx.Cache == nil || ctx.Cache.createdMicroflows == nil {
		return nil
	}
	return ctx.Cache.createdMicroflows[qualifiedName]
}

// getCreatedPage returns info about a page created during this session,
// or nil if not found.
func (ctx *ExecContext) getCreatedPage(qualifiedName string) *createdPageInfo {
	if ctx.Cache == nil || ctx.Cache.createdPages == nil {
		return nil
	}
	return ctx.Cache.createdPages[qualifiedName]
}

// ensureSqlMgr lazily initializes and returns the SQL connection manager.
func (ctx *ExecContext) ensureSqlMgr() *sqllib.Manager {
	if ctx.SqlMgr == nil {
		ctx.SqlMgr = sqllib.NewManager()
	}
	return ctx.SqlMgr
}

func GetHierarchyForMining(ctx *ExecContext) (*ContainerHierarchy, error) {
	return getHierarchy(ctx)
}

// getDomainModelGenCached returns the DomainModel for moduleID using
// executorCache.domainModelByModule as a write-through cache.
// On miss it calls ctx.DomainModelReader.GetDomainModelGen and stores the result.
func getDomainModelGenCached(ctx *ExecContext, moduleID model.ID) (*genDm.DomainModel, error) {
	ctx.initRoles()
	ctx.ensureCache()
	if ctx.Cache.domainModelByModule != nil {
		if dm, ok := ctx.Cache.domainModelByModule[moduleID]; ok {
			return dm, nil
		}
	}
	dm, err := ctx.DomainModelReader.GetDomainModelGen(moduleID)
	if err != nil {
		return nil, err
	}
	if ctx.Cache.domainModelByModule == nil {
		ctx.Cache.domainModelByModule = make(map[model.ID]*genDm.DomainModel)
	}
	ctx.Cache.domainModelByModule[moduleID] = dm
	return dm, nil
}

// setDomainModelGenCached updates the cached DomainModel for moduleID.
// Call immediately after ctx.DomainModelWriter.UpdateDomainModelGen(dm) for write-through.
func setDomainModelGenCached(ctx *ExecContext, moduleID model.ID, dm *genDm.DomainModel) {
	ctx.ensureCache()
	if ctx.Cache.domainModelByModule == nil {
		ctx.Cache.domainModelByModule = make(map[model.ID]*genDm.DomainModel)
	}
	ctx.Cache.domainModelByModule[moduleID] = dm
}

// LintReader returns ctx.Backend as a lint reader.
// FullBackend implements LintReader via BackendFactory embedding.
func (ctx *ExecContext) LintReader() linter.LintReader {
	if ctx.Backend == nil {
		return nil
	}
	return ctx.Backend
}

// statusWriter returns the StatusOutput writer if set, otherwise io.Discard.
// Use this for informational messages that should not pollute stdout when
// the caller redirects stdout to a file.
func (ctx *ExecContext) statusWriter() io.Writer {
	if ctx.StatusOutput != nil {
		return ctx.StatusOutput
	}
	return io.Discard
}
