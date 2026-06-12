// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"io"
	"path/filepath"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/catalog"
	"github.com/mendixlabs/mxcli/mdl/diaglog"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	sqllib "github.com/mendixlabs/mxcli/sql"
)

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
	Cache                             *executorCache
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
	Catalog        *catalog.Catalog
}

// ExecCallbacks holds function references for recursive execution.
type ExecCallbacks struct {
	ExecuteFn        func(ast.Statement) error
	ExecuteProgramFn func(*ast.Program) error
	FinalizeFn       func() error
	SyncCatalog      func(*catalog.Catalog)
}

// ExecContext carries all dependencies a statement handler needs.
//
// Fields are grouped into embedded sub-structs by responsibility.
// All ctx.Xxx field accesses continue to work via Go field promotion;
// only struct literal initializers need to use the sub-struct names.
type ExecContext struct {
	context.Context

	// Backend provides all domain operations. Nil when not connected.
	// Deprecated: All production code has been migrated to role-specific
	// fields (DomainModelReader, PageWriter, etc.). Only the ImportBuffer
	// type assertion and some test fixtures still reference this field.
	// Remove once those are migrated.
	Backend backend.FullBackend

	// Logger is the session diagnostics logger (nil = no logging).
	Logger *diaglog.Logger

	// Role-specific backend interfaces. Populated lazily from Backend.
	// Handler code should use these instead of ctx.Backend when only
	// one domain is needed.
	ModuleLister      backend.ModuleLister
	ModuleWriter      backend.ModuleWriter
	DomainModelReader backend.DomainModelReader
	DomainModelWriter backend.DomainModelWriter
	MicroflowReader   backend.MicroflowReader
	MicroflowWriter   backend.MicroflowWriter
	WorkflowReader    backend.WorkflowReader
	WorkflowWriter    backend.WorkflowWriter
	PageReader        backend.PageReader
	PageWriter        backend.PageWriter
	JavaActionReader  backend.JavaActionReader
	JavaActionWriter  backend.JavaActionWriter
	EnumerationReader backend.EnumerationReader
	EnumerationWriter backend.EnumerationWriter
	ConstantReader    backend.ConstantReader
	ConstantWriter    backend.ConstantWriter
	SettingsReader    backend.SettingsReader
	SettingsWriter    backend.SettingsWriter
	MappingReader     backend.MappingReader
	MappingWriter     backend.MappingWriter
	UnitReader        backend.UnitReader
	UnitWriter        backend.UnitWriter
	NavigationReader  backend.NavigationReader
	NavigationWriter  backend.NavigationWriter
	ImageCollectionWriter     backend.ImageCollectionWriter
	ScheduledEventReader      backend.ScheduledEventReader
	ServiceLister             backend.ServiceLister
	ServiceWriter             backend.ServiceWriter
	MetadataReader            backend.MetadataReader
	ConnectionManager         backend.ConnectionManager
	FolderManager             backend.FolderManager
	ModuleSettingsReader      backend.ModuleSettingsReader
	ModuleSettingsWriter      backend.ModuleSettingsWriter
	RenameManager             backend.RenameManager
	SecurityProjectManager    backend.SecurityProjectManager
	SecurityModuleManager     backend.SecurityModuleManager
	SecurityEntityAccessManager backend.SecurityEntityAccessManager
	PageModelAccess                backend.PageModelAccess
	PageMutationOperator           backend.PageMutationOperator
	WorkflowMutationOperator       backend.WorkflowMutationOperator
	WidgetBuilder                  backend.WidgetBuilder
	ScriptTransactionManager       backend.ScriptTransactionManager
	AgentEditorOperator            backend.AgentEditorOperator

	ExecRepos
	ExecIO
	ExecSession
	ExecConnection
	ExecCallbacks
}

// initRoles populates the role-specific backend fields from Backend. Safe to
// call multiple times. Repopulates every call so reconnecting to a new project
// picks up the new Backend's role interface implementations.
func (ctx *ExecContext) initRoles() {
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

// Connected returns true if a project is connected via the Backend.
func (ctx *ExecContext) Connected() bool {
	return ctx.Backend != nil && ctx.ConnectionManager.IsConnected()
}

// ConnectedForWrite returns true if a project is connected and the backend
// supports write operations. Currently equivalent to Connected() since
// MprBackend always supports writes.
func (ctx *ExecContext) ConnectedForWrite() bool {
	return ctx.Connected()
}

// InvalidateCache clears the hierarchy/entity/microflow cache so that
// subsequent statements see fresh data.
func (ctx *ExecContext) InvalidateCache() {
	ctx.Cache = nil
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
	if ctx.Backend == nil || !ctx.ConnectionManager.IsConnected() {
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

// statusWriter returns the StatusOutput writer if set, otherwise io.Discard.
// Use this for informational messages that should not pollute stdout when
// the caller redirects stdout to a file.
func (ctx *ExecContext) statusWriter() io.Writer {
	if ctx.StatusOutput != nil {
		return ctx.StatusOutput
	}
	return io.Discard
}
