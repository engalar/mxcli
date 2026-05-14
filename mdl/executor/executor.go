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
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	sqllib "github.com/mendixlabs/mxcli/sql"
)

// executorCache holds cached data for performance across multiple operations.
type executorCache struct {
	modules      []*model.Module
	units        []*types.UnitInfo
	folders      []*types.FolderInfo
	domainModels []*domainmodel.DomainModel
	hierarchy    *ContainerHierarchy
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
	entityNames := make(map[model.ID]string)
	dms, err := ctx.Backend.ListDomainModels()
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warn("getEntityNames: ListDomainModels failed", "error", err)
		}
		return entityNames
	}
	for _, dm := range dms {
		modName := h.GetModuleName(dm.ContainerID)
		for _, ent := range dm.Entities {
			entityNames[ent.ID] = modName + "." + ent.Name
		}
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
	}
}

// getPageNames returns the page name lookup map, using the pre-warmed cache if available.
func getPageNames(ctx *ExecContext, h *ContainerHierarchy) map[model.ID]string {
	if ctx.Cache != nil && len(ctx.Cache.pageNames) > 0 {
		return ctx.Cache.pageNames
	}
	pageNames := make(map[model.ID]string)
	pgs, err := ctx.Backend.ListPages()
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warn("getPageNames: ListPages failed", "error", err)
		}
		return pageNames
	}
	for _, pg := range pgs {
		pageNames[pg.ID] = h.GetQualifiedName(pg.ContainerID, pg.Name)
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
	guard          *outputGuard // line-limit wrapper around output
	mprPath        string
	settings       map[string]any
	cache          *executorCache
	catalog        *catalog.Catalog
	quiet          bool                               // suppress connection and status messages
	format         OutputFormat                       // output format (table, json)
	logger         *diaglog.Logger                    // session diagnostics logger (nil = no logging)
	fragments      map[string]*ast.DefineFragmentStmt // script-scoped fragment definitions
	sqlMgr         *sqllib.Manager                    // external SQL connection manager (lazy init)
	themeRegistry  *ThemeRegistry                     // cached theme design property definitions (lazy init)
	registry       *Registry                          // statement dispatch registry
	catalogMu      sync.RWMutex                       // protects catalog field from background goroutine writes
	catalogGen     uint64                             // monotonic generation counter for catalog swaps
}

// New creates a new executor with the given output writer.
func New(output io.Writer) *Executor {
	guard := newOutputGuard(output, maxOutputLines)
	return &Executor{
		output:   guard,
		guard:    guard,
		settings: make(map[string]any),
		registry: NewRegistry(),
	}
}

// SetBackendFactory sets the factory function used to create backend instances on Connect.
func (e *Executor) SetBackendFactory(f BackendFactory) {
	e.backendFactory = f
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

	if e.logger != nil {
		e.logger.Command(stmtTypeName(stmt), stmtSummary(stmt), time.Since(start), err)
	}
	return err
}

// ExecuteProgram runs all statements in a program.
func (e *Executor) ExecuteProgram(prog *ast.Program) error {
	// Collect all names defined in the script for forward-reference hints.
	allDefined := newScriptContext()
	allDefined.collectDefinitions(prog)

	// Track which names have been created so far.
	created := newScriptContext()

	for _, stmt := range prog.Statements {
		if err := e.Execute(stmt); err != nil {
			return annotateForwardRef(err, stmt, created, allDefined)
		}
		created.collectSingle(stmt)
	}
	return e.finalizeProgramExecution()
}

// finalizeProgramExecution runs post-execution reconciliation on modified domain models.
func (e *Executor) finalizeProgramExecution() error {
	if e.backend == nil || !e.backend.IsConnected() || e.cache == nil || len(e.cache.modifiedDomainModels) == 0 {
		return nil
	}

	for moduleID, moduleName := range e.cache.modifiedDomainModels {
		dm, err := e.backend.GetDomainModel(moduleID)
		if err != nil {
			continue // module may not have a domain model
		}

		count, err := e.backend.ReconcileMemberAccesses(dm.ID, moduleName)
		if err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("reconcile security for module %s", moduleName), err)
		}
		if count > 0 && !e.quiet {
			fmt.Fprintf(e.output, "Reconciled %d access rule(s) in module %s\n", count, moduleName)
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
