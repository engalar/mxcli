// SPDX-License-Identifier: Apache-2.0

// Package executor executes MDL AST statements against a Mendix project.
package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/catalog"
	"github.com/mendixlabs/mxcli/mdl/diaglog"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	sqllib "github.com/mendixlabs/mxcli/sql"
)

// executorCache holds cached data for performance across multiple operations.
type executorCache struct {
	modules   []*model.Module
	units     []*types.UnitInfo
	folders   []*types.FolderInfo
	hierarchy *ContainerHierarchy
	// pages, layouts are cached separately as they may change during execution

	// Track items created during this session (not yet visible via reader)
	createdMicroflows map[string]*createdMicroflowInfo // qualifiedName -> info
	createdNanoflows  map[string]*createdNanoflowInfo  // qualifiedName -> info
	createdPages      map[string]*createdPageInfo      // qualifiedName -> info
	createdSnippets   map[string]*createdSnippetInfo   // qualifiedName -> info

	// Track items dropped during this session so that a subsequent
	// CREATE OR REPLACE/MODIFY with the same qualified name can reuse the
	// original UnitID and ContainerID. Studio Pro treats Unit rows with a
	// different UnitID (or the same UnitID under a different container) as
	// unrelated documents, producing broken projects on delete+insert
	// rewrites. Reusing both keeps the rewrite semantically equivalent to an
	// in-place update.
	droppedMicroflows map[string]*droppedUnitInfo // qualifiedName -> original IDs
	droppedNanoflows  map[string]*droppedUnitInfo // qualifiedName -> original IDs

	// Track domain models modified during this session for finalization
	modifiedDomainModels map[model.ID]string // domain model unit ID -> module name

	// Pre-warmed name lookup maps for parallel describe (goroutine-safe after init)
	entityNames    map[model.ID]string // entity ID -> "Module.EntityName"
	microflowNames map[model.ID]string // microflow ID -> "Module.MicroflowName"
	pageNames      map[model.ID]string // page ID -> "Module.PageName"

	// Cached gen-typed flow listings with container UUID resolved.
	// Populated lazily by listMicroflowsWithContainerGen /
	// listNanoflowsWithContainerGen so repeated callers in a single
	// session pay one ListAll + one batch container scan instead of
	// N per-element GetContainerUUID lookups (Followup E1).
	microflowsWithContainerGen []MicroflowGenWithContainer
	nanoflowsWithContainerGen  []NanoflowGenWithContainer

	// Cached gen-typed security listings. Populated lazily by
	// getProjectSecurityGen / listModuleSecurityWithContainerGen.
	projectSecurityGen             *genSec.ProjectSecurity
	moduleSecurityWithContainerGen []ModuleSecurityGenWithContainer

	// Cached gen-typed Java action / JavaScript action listings.
	// Populated lazily by listJavaActionsWithContainerGen /
	// listJavaScriptActionsWithContainerGen (Stage 3.3.2 A0).
	javaActionsWithContainerGen       []ContainerWithGen[*genJA.JavaAction]
	javaScriptActionsWithContainerGen []ContainerWithGen[*genJSA.JavaScriptAction]

	// Cached gen-typed DomainModel listing with container UUID resolved
	// (Stage 3.3.4 A0). Each module owns at most one DomainModel; the
	// container ID is the owning module's UUID.
	domainModelsWithContainerGen []DomainModelGenWithContainer

	// Legacy sdk-typed DomainModel listing retained until the final
	// backend-interface retirement removes the old surface.
	domainModels []*genDm.DomainModel

	// Flat gen-typed DomainModel list (Stage 3.3.4.C9). Populated lazily
	// by cachedDomainModelsGen via ctx.Backend.ListDomainModelsGen.
	// Used by cmd_pages_builder getDomainModels migration (C7).
	domainModelsGen []*genDm.DomainModel

	// Cached gen-typed Workflow listing with container UUID resolved
	// (Stage 3.3.3 A0).
	workflowsWithContainerGen []ContainerWithGen[*genWf.Workflow]

	// Cached gen-typed Page / Layout / Snippet listings with container
	// UUID resolved (Stage 3.3.5 A0).
	pagesWithContainerGen    []ContainerWithGen[*genPg.Page]
	layoutsWithContainerGen  []ContainerWithGen[*genPg.Layout]
	snippetsWithContainerGen []ContainerWithGen[*genPg.Snippet]

	// domainModelByModule caches moduleID → *genDm.DomainModel.
	// Populated lazily by getDomainModelGenCached; updated write-through
	// by setDomainModelGenCached after every UpdateDomainModelGen call.
	// Single-goroutine lifetime — no lock needed.
	domainModelByModule map[model.ID]*genDm.DomainModel
}

// ── Typed accessors — prefer these over direct field access ────────

func (c *executorCache) EntityNames() map[model.ID]string        { return c.entityNames }
func (c *executorCache) SetEntityNames(v map[model.ID]string)    { c.entityNames = v }
func (c *executorCache) MicroflowNames() map[model.ID]string     { return c.microflowNames }
func (c *executorCache) SetMicroflowNames(v map[model.ID]string) { c.microflowNames = v }
func (c *executorCache) PageNames() map[model.ID]string          { return c.pageNames }
func (c *executorCache) SetPageNames(v map[model.ID]string)      { c.pageNames = v }

func (c *executorCache) MicroflowsWithContainer() []MicroflowGenWithContainer {
	return c.microflowsWithContainerGen
}
func (c *executorCache) SetMicroflowsWithContainer(v []MicroflowGenWithContainer) {
	c.microflowsWithContainerGen = v
}
func (c *executorCache) NanoflowsWithContainer() []NanoflowGenWithContainer {
	return c.nanoflowsWithContainerGen
}
func (c *executorCache) SetNanoflowsWithContainer(v []NanoflowGenWithContainer) {
	c.nanoflowsWithContainerGen = v
}
func (c *executorCache) PagesWithContainer() []ContainerWithGen[*genPg.Page] {
	return c.pagesWithContainerGen
}
func (c *executorCache) SetPagesWithContainer(v []ContainerWithGen[*genPg.Page]) {
	c.pagesWithContainerGen = v
}
func (c *executorCache) LayoutsWithContainer() []ContainerWithGen[*genPg.Layout] {
	return c.layoutsWithContainerGen
}
func (c *executorCache) SetLayoutsWithContainer(v []ContainerWithGen[*genPg.Layout]) {
	c.layoutsWithContainerGen = v
}
func (c *executorCache) SnippetsWithContainer() []ContainerWithGen[*genPg.Snippet] {
	return c.snippetsWithContainerGen
}
func (c *executorCache) SetSnippetsWithContainer(v []ContainerWithGen[*genPg.Snippet]) {
	c.snippetsWithContainerGen = v
}
func (c *executorCache) WorkflowsWithContainer() []ContainerWithGen[*genWf.Workflow] {
	return c.workflowsWithContainerGen
}
func (c *executorCache) SetWorkflowsWithContainer(v []ContainerWithGen[*genWf.Workflow]) {
	c.workflowsWithContainerGen = v
}
func (c *executorCache) JavaActionsWithContainer() []ContainerWithGen[*genJA.JavaAction] {
	return c.javaActionsWithContainerGen
}
func (c *executorCache) SetJavaActionsWithContainer(v []ContainerWithGen[*genJA.JavaAction]) {
	c.javaActionsWithContainerGen = v
}
func (c *executorCache) JavaScriptActionsWithContainer() []ContainerWithGen[*genJSA.JavaScriptAction] {
	return c.javaScriptActionsWithContainerGen
}
func (c *executorCache) SetJavaScriptActionsWithContainer(v []ContainerWithGen[*genJSA.JavaScriptAction]) {
	c.javaScriptActionsWithContainerGen = v
}
func (c *executorCache) DomainModelsWithContainer() []DomainModelGenWithContainer {
	return c.domainModelsWithContainerGen
}
func (c *executorCache) SetDomainModelsWithContainer(v []DomainModelGenWithContainer) {
	c.domainModelsWithContainerGen = v
}
func (c *executorCache) DomainModels() []*genDm.DomainModel        { return c.domainModels }
func (c *executorCache) SetDomainModels(v []*genDm.DomainModel)    { c.domainModels = v }
func (c *executorCache) DomainModelsGen() []*genDm.DomainModel     { return c.domainModelsGen }
func (c *executorCache) SetDomainModelsGen(v []*genDm.DomainModel) { c.domainModelsGen = v }

func (c *executorCache) ProjectSecurityGen() *genSec.ProjectSecurity     { return c.projectSecurityGen }
func (c *executorCache) SetProjectSecurityGen(v *genSec.ProjectSecurity) { c.projectSecurityGen = v }
func (c *executorCache) ModuleSecurityWithContainer() []ModuleSecurityGenWithContainer {
	return c.moduleSecurityWithContainerGen
}
func (c *executorCache) SetModuleSecurityWithContainer(v []ModuleSecurityGenWithContainer) {
	c.moduleSecurityWithContainerGen = v
}
func (c *executorCache) DomainModelByModule() map[model.ID]*genDm.DomainModel {
	return c.domainModelByModule
}
func (c *executorCache) SetDomainModelByModule(v map[model.ID]*genDm.DomainModel) {
	c.domainModelByModule = v
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
type BackendFactory func() backend.FullBackend

// Executor executes MDL statements against a Mendix project.
type Executor struct {
	backend        backend.FullBackend // domain backend (populated on Connect)
	backendFactory BackendFactory      // factory for creating new backend instances
	output         io.Writer
	statusOutput   io.Writer    // writer for status/informational messages (stderr by default)
	guard          *outputGuard // line-limit wrapper around output
	mprPath        string
	settings       map[string]any
	cache          *executorCache
	catalog        *catalog.Catalog
	graphCatalog   *graphcatalog.ProjectGraph         // in-memory project graph (lazy build) for code-search + lint
	quiet          bool                               // suppress connection and status messages
	format         OutputFormat                       // output format (table, json)
	logger         *diaglog.Logger                    // session diagnostics logger (nil = no logging)
	fragments      map[string]*ast.DefineFragmentStmt // script-scoped fragment definitions
	sqlMgr         *sqllib.Manager                    // external SQL connection manager (lazy init)
	themeRegistry  *ThemeRegistry                     // cached theme design property definitions (lazy init)
	registry       *Registry                          // statement dispatch registry
	catalogMu      sync.RWMutex                       // protects catalog field from background goroutine writes
	catalogGen     uint64                             // monotonic generation counter for catalog swaps

	// perfStats accumulates per-statement execution timing so the
	// caller can print a summary when the script finishes.
	perfStats []perfStmt
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
	return &Executor{
		output:       guard,
		statusOutput: os.Stderr,
		guard:        guard,
		settings:     make(map[string]any),
		registry:     NewRegistry(),
	}
}

// SetBackendFactory sets the factory function used to create backend instances on Connect.
func (e *Executor) SetBackendFactory(f BackendFactory) {
	e.backendFactory = f
}

// SetBackend installs an already-connected backend on the executor and
// ensures the executor cache is initialized. Used by callers (e.g.
// `mxcli export`) that own the backend lifecycle outside the normal
// Connect/Disconnect MDL flow.
func (e *Executor) SetBackend(b backend.FullBackend) {
	e.backend = b
	if e.cache == nil {
		e.cache = &executorCache{}
	}
}

// SetQuiet enables or disables quiet mode (suppresses connection/status messages).
func (e *Executor) SetQuiet(quiet bool) {
	e.quiet = quiet
}

// SetFormat sets the output format (table or json).
func (e *Executor) SetFormat(f OutputFormat) {
	e.format = f
}

// SetLogger sets the diagnostics logger for session logging.
func (e *Executor) SetLogger(l *diaglog.Logger) {
	e.logger = l
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
// Each statement gets a fresh line budget. If the statement exceeds maxOutputLines
// lines of output or runs longer than the configured timeout, it is aborted with an error.
func (e *Executor) Execute(stmt ast.Statement) error {
	start := time.Now()

	// Reset per-statement line counter.
	if e.guard != nil {
		e.guard.reset()
	}

	// Enforce wall-clock timeout via context.WithTimeout.
	// The goroutine pattern is retained because handlers are not yet
	// context-aware; threading context through handlers is a follow-up.
	executeTimeout := configuredExecuteTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), executeTimeout)
	defer cancel()

	type result struct{ err error }
	ch := make(chan result, 1)
	go func() {
		ch <- result{e.executeInner(ctx, stmt)}
	}()

	var err error
	select {
	case r := <-ch:
		err = r.err
	case <-ctx.Done():
		err = mdlerrors.NewValidationf("statement timed out after %v", executeTimeout)
	}

	elapsed := time.Since(start)

	if e.logger != nil {
		e.logger.Command(stmtTypeName(stmt), stmtSummary(stmt), elapsed, err)
	}

	e.perfStats = append(e.perfStats, perfStmt{
		Type:     stmtTypeName(stmt),
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

// Catalog returns the catalog, or nil if not built.
func (e *Executor) Catalog() *catalog.Catalog {
	e.catalogMu.RLock()
	c := e.catalog
	e.catalogMu.RUnlock()
	return c
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
	e.syncBack(ctx)
	return e.graphCatalog, nil
}

// IsConnected returns true if connected to a project.
func (e *Executor) IsConnected() bool {
	return e.backend != nil && e.backend.IsConnected()
}

// Backend returns the full backend, or nil if not connected.
func (e *Executor) Backend() backend.FullBackend {
	if e.backend == nil || !e.backend.IsConnected() {
		return nil
	}
	return e.backend
}

// Close closes the connection to the project and all SQL connections.
func (e *Executor) Close() error {
	var closeErr error
	if e.backend != nil && e.backend.IsConnected() {
		closeErr = e.backend.Disconnect()
		e.backend = nil
	}
	if e.sqlMgr != nil {
		e.sqlMgr.CloseAll()
		e.sqlMgr = nil
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
