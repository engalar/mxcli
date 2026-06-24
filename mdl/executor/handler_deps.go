package executor

import (
	"context"
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/diaglog"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/repos"
	sqllib "github.com/mendixlabs/mxcli/sql"
)

// HandlerDeps carries all dependencies that were previously scattered across
// ExecContext fields. This is the single dependency container used by every
// statement handler and helper function.
type HandlerDeps struct {
	Output       io.Writer
	StatusOutput io.Writer
	Logger       *diaglog.Logger
	Quiet        bool

	Backend              backend.FullBackend
	BackendFactory       BackendFactory
	ConnectionManager    backend.ConnectionManager
	ModuleLister         backend.ModuleLister
	ModuleWriter         backend.ModuleWriter
	FolderManager        backend.FolderManager
	ModuleSettingsReader backend.ModuleSettingsReader
	ModuleSettingsWriter backend.ModuleSettingsWriter
	DomainModelReader    backend.DomainModelReader
	DomainModelWriter    backend.DomainModelWriter
	MicroflowReader      backend.MicroflowReader
	MicroflowWriter      backend.MicroflowWriter
	WorkflowReader       backend.WorkflowReader
	WorkflowWriter       backend.WorkflowWriter
	PageReader           backend.PageReader
	PageWriter           backend.PageWriter
	JavaActionReader     backend.JavaActionReader
	JavaActionWriter     backend.JavaActionWriter
	JavaScriptActionWriter backend.JavaScriptActionWriter
	EnumerationReader    backend.EnumerationReader
	EnumerationWriter    backend.EnumerationWriter
	ConstantReader       backend.ConstantReader
	ConstantWriter       backend.ConstantWriter
	SettingsReader       backend.SettingsReader
	SettingsWriter       backend.SettingsWriter
	MapperReader         backend.MappingReader
	MapperWriter         backend.MappingWriter
	UnitReader           backend.UnitReader
	UnitWriter           backend.UnitWriter
	NavigationReader     backend.NavigationReader
	NavigationWriter     backend.NavigationWriter
	ImageCollectionWriter backend.ImageCollectionWriter
	ScheduledEventReader   backend.ScheduledEventReader
	ServiceLister          backend.ServiceLister
	ServiceWriter          backend.ServiceWriter
	MetadataReader         backend.MetadataReader
	RenameManager          backend.RenameManager
	SecurityProjectManager      backend.SecurityProjectManager
	SecurityModuleManager       backend.SecurityModuleManager
	SecurityEntityAccessManager backend.SecurityEntityAccessManager
	PageModelAccess             backend.PageModelAccess
	PageMutationOperator        backend.PageMutationOperator
	WorkflowMutationOperator    backend.WorkflowMutationOperator
	WidgetBuilder               backend.WidgetBuilder
	ScriptTransactionManager    backend.ScriptTransactionManager
	AgentEditorOperator         backend.AgentEditorOperator
	ImageBackend                backend.ImageBackend
	BusinessEventBackend        backend.BusinessEventBackend

	DomainModels         repos.DomainModelRepository
	MicroflowRepo        repos.MicroflowRepository
	NanoflowRepo         repos.NanoflowRepository
	PageRepo             repos.PageRepository
	LayoutRepo           repos.LayoutRepository
	SnippetRepo          repos.SnippetRepository
	JavaActionRepo       repos.JavaActionRepository
	JavaScriptActionRepo repos.JavaScriptActionRepository
	WorkflowRepo         repos.WorkflowRepository
	Security             repos.SecurityRepository

	MprPath string
	Graph   *graphcatalog.ProjectGraph
	Perf    *PerfTimer
	Format  OutputFormat

	Settings  map[string]any
	Fragments map[string]*ast.DefineFragmentStmt
	SqlMgr    *sqllib.Manager
	Cache     *executorCache
	Session   *sessionTracker
	ThemeRegistry *ThemeRegistry

	ScriptDepth                      int
	DescribingMicroflowHasReturnValue bool

	ExecuteFn        func(ast.Statement) error
	ExecuteProgramFn func(*ast.Program) error
	FinalizeFn       func() error
	SyncGraph        func(*graphcatalog.ProjectGraph)
}

// registerFutureOverlays registers new-style handlers (StmtHandlerFunc) for
// domains that have been migrated from *ExecContext. Called after backend is
// set so concrete dependencies are available for closure capture.
// Overrides old-style StmtHandler registrations from NewRegistry().
func (e *Executor) registerFutureOverlays() {
	r := e.registry

	// Connection handlers — registered even without backend because Connect
	// must work before any backend exists, and Disconnect must handle the
	// nil backend case gracefully.
	r.RegisterFuture("Connect", func(ctx context.Context, stmt ast.Statement) error {
		return execConnectFuture(ctx, stmt.(*ast.ConnectStmt), e)
	})
	r.RegisterFuture("Disconnect", func(ctx context.Context, stmt ast.Statement) error {
		return execDisconnectFuture(ctx, e)
	})

	deps := e.buildHandlerDeps()
	if deps == nil {
		return
	}

	// Connection status — uses ConnectionManager/ModuleLister (simple deps).
	r.RegisterFuture("Status", func(ctx context.Context, stmt ast.Statement) error {
		return execStatusFuture(ctx, deps.Output, deps.ConnectionManager, deps.ModuleLister, e.mprPath)
	})

	// Session handlers migrated from *ExecContext:
	// e.format is captured by reference (e is *Executor pointer) — reflects SET format changes.
	r.RegisterFuture("Set", func(ctx context.Context, stmt ast.Statement) error {
		return execSetFuture(ctx, stmt.(*ast.SetStmt), deps.Output, &e.format)
	})
	r.RegisterFuture("Update", func(ctx context.Context, stmt ast.Statement) error {
		return execUpdateFuture(ctx, deps, e)
	})
	r.RegisterFuture("Refresh", func(ctx context.Context, stmt ast.Statement) error {
		return execRefreshFuture(ctx, deps, e)
	})
	r.RegisterFuture("ExecuteScript", func(ctx context.Context, stmt ast.Statement) error {
		return execExecuteScriptFuture(ctx, stmt.(*ast.ExecuteScriptStmt), deps, e)
	})
	r.RegisterFuture("Help", func(ctx context.Context, stmt ast.Statement) error {
		return execHelpFuture(ctx, stmt.(*ast.HelpStmt), deps.Output, e.format)
	})
	r.RegisterFuture("Exit", func(ctx context.Context, stmt ast.Statement) error {
		return execExitFuture(ctx)
	})

	// SHOW handlers migrated from *ExecContext:
	r.RegisterFuture("Show", func(ctx context.Context, stmt ast.Statement) error {
		s := stmt.(*ast.ShowStmt)
		switch s.ObjectType {
		case ast.ShowModules:
			return listModulesFuture(ctx, deps.Output, e.format, deps.ConnectionManager, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.DomainModels)
		case ast.ShowVersion:
			return listVersionFuture(ctx, deps.Output, deps.ConnectionManager)
		case ast.ShowCatalogTables:
			return execShowCatalogTablesFuture(ctx, deps.Output)
		case ast.ShowCatalogStatus:
			return execShowCatalogStatusFuture(ctx, deps.Output)
		case ast.ShowEnumerations:
			return listEnumerationsFuture(ctx, deps.Output, e.format, deps.ConnectionManager, deps.EnumerationReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.InModule)
		case ast.ShowConstants:
			return listConstantsFuture(ctx, deps.Output, e.format, deps.ConnectionManager, deps.ConstantReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.InModule)
		case ast.ShowConstantValues:
			return listConstantValuesFuture(ctx, deps.Output, e.format, deps.ConnectionManager, deps.ConstantReader, deps.SettingsReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.InModule)
		case ast.ShowFragments:
			return listFragmentsFuture(ctx, deps.Output, e.fragments)
		case ast.ShowEntities:
			return listEntitiesGenFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.DomainModels, s.InModule)
		case ast.ShowEntity:
			return listEntityFuture(ctx, deps.Output, deps.ModuleLister, deps.DomainModels, s.Name)
		case ast.ShowAssociations:
			return listAssociationsFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.DomainModels, s.InModule)
		case ast.ShowAssociation:
			return listAssociationFuture(ctx, deps.Output, deps.ModuleLister, deps.DomainModelReader, s.Name)
		case ast.ShowMicroflows:
			return listMicroflowsFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.MicroflowRepo, s.InModule)
		case ast.ShowNanoflows:
			return listNanoflowsFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.NanoflowRepo, s.InModule)
		case ast.ShowPages:
			return listPagesFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.PageRepo, s.InModule)
		case ast.ShowSnippets:
			return listSnippetsFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.SnippetRepo, s.InModule)
		case ast.ShowLayouts:
			return listLayoutsFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.LayoutRepo, s.InModule)
		case ast.ShowJavaActions:
			return listJavaActionsFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.JavaActionRepo, s.InModule)
		case ast.ShowJavaScriptActions:
			return listJavaScriptActionsFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.JavaScriptActionRepo, s.InModule)
		case ast.ShowWorkflows:
			return listWorkflowsFuture(ctx, deps.Output, e.format, deps.ConnectionManager, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.WorkflowRepo, s.InModule)
		case ast.ShowBusinessEventServices:
			return listBusinessEventServicesFuture(ctx, deps.Output, e.format, deps.ConnectionManager, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.BusinessEventBackend, s.InModule)
		case ast.ShowBusinessEventClients:
			return listBusinessEventClientsFuture(ctx, deps.Output)
		case ast.ShowBusinessEvents:
			return listBusinessEventsFuture(ctx, deps.Output, e.format, deps.ConnectionManager, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.BusinessEventBackend, s.InModule)
		case ast.ShowProjectSecurity:
			return listProjectSecurityFuture(ctx, deps.Output, e.format, deps.Security)
		case ast.ShowModuleRoles:
			return listModuleRolesFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.Security, s.InModule)
		case ast.ShowUserRoles:
			return listUserRolesFuture(ctx, deps.Output, e.format, deps.Security)
		case ast.ShowDemoUsers:
			return listDemoUsersFuture(ctx, deps.Output, e.format, deps.Security)
		case ast.ShowAccessOn:
			return listAccessOnEntityFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.DomainModels, s.Name)
		case ast.ShowAccessOnMicroflow:
			return listAccessOnMicroflowFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.MicroflowRepo, s.Name)
		case ast.ShowAccessOnPage:
			return listAccessOnPageFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.PageRepo, s.Name)
		case ast.ShowAccessOnWorkflow:
			return listAccessOnWorkflowFuture(ctx, s.Name)
		case ast.ShowAccessOnNanoflow:
			return listAccessOnNanoflowFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.NanoflowRepo, s.Name)
		case ast.ShowSecurityMatrix:
			return listSecurityMatrixFuture(ctx, deps.Output, e.format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.Security, deps.DomainModels, deps.MicroflowRepo, deps.PageRepo, s.InModule)
		case ast.ShowNavigation:
			return listNavigationFuture(ctx, deps.Output, deps.NavigationReader)
		case ast.ShowNavigationMenu:
			return listNavigationMenuFuture(ctx, deps.Output, deps.NavigationReader, s.Name)
		case ast.ShowNavigationHomes:
			return listNavigationHomesFuture(ctx, deps.Output, deps.NavigationReader)
		case ast.ShowSettings:
			return listSettingsFuture(ctx, deps.Output, e.format, deps.SettingsReader)
		case ast.ShowLanguages:
			return listLanguagesFuture(ctx, deps.Output, e.format, deps.SettingsReader)
		case ast.ShowSupportedLanguages:
			return listSupportedLanguagesFuture(ctx, deps.Output, e.format)
		case ast.ShowODataClients:
			return listODataClientsFn(ctx, deps, e.format, s.InModule)
		case ast.ShowODataServices:
			return listODataServicesFn(ctx, deps, e.format, s.InModule)
		case ast.ShowExternalEntities:
			return listExternalEntitiesFn(ctx, deps, e.format, s.InModule)
		case ast.ShowExternalActions:
			return listExternalActionsFn(ctx, deps, e.format, s.InModule)
		case ast.ShowStructure:
			return execShowStructureGenFuture(ctx, deps.Output, e.format, s, deps)
		case ast.ShowCallers:
			return ExecShowCallersFn(ctx, s, deps)
		case ast.ShowCallees:
			return ExecShowCalleesFn(ctx, s, deps)
		case ast.ShowReferences:
			return ExecShowReferencesFn(ctx, s, deps)
		case ast.ShowImpact:
			return ExecShowImpactFn(ctx, s, deps)
		case ast.ShowExportMappings:
			return listExportMappingsFn(ctx, s.InModule, deps)
		case ast.ShowImportMappings:
			return listImportMappingsFn(ctx, s.InModule, deps)
		default:
			return nil // fall through to old handler
		}
	})

	// DESCRIBE handlers migrated from *ExecContext (Phase 3d-1h):
	r.RegisterFuture("Describe", func(ctx context.Context, stmt ast.Statement) error {
		s := stmt.(*ast.DescribeStmt)
		entry, ok := describeHandlers[s.ObjectType]
		if !ok {
			return mdlerrors.NewUnsupported("unknown describe object type")
		}
		name := s.Name.String()

		switch s.ObjectType {
		case ast.DescribeSettings:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeSettingsFuture(ctx, output, deps.ConnectionManager, deps.SettingsReader)
			})
		case ast.DescribeConstant:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeConstantFuture(ctx, output, deps.ConstantReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeEnumeration:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeEnumerationFuture(ctx, output, deps.EnumerationReader, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeDatabaseConnection:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeDatabaseConnectionFuture(ctx, output, deps.ServiceLister, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeImageCollection:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeImageCollectionFuture(ctx, output, deps.ImageBackend, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeNavigation:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeNavigationFuture(ctx, output, deps.NavigationReader, s.Name)
			})
		case ast.DescribeModule:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeModuleFuture(ctx, output, s.Name.Module, s.WithAll,
					deps.ModuleLister, deps.MetadataReader, deps.FolderManager,
					deps.EnumerationReader, deps.ConstantReader,
					deps.DomainModels, deps.Security,
					deps.MicroflowRepo, deps.NanoflowRepo,
					deps.PageRepo, deps.SnippetRepo, deps.LayoutRepo, deps.WorkflowRepo,
					deps.ImageBackend, deps.NavigationReader)
			})
		case ast.DescribeEntity:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeEntityGenFuture(ctx, output, deps.ModuleLister, deps.DomainModels, deps.Security, s.Name)
			})
		case ast.DescribeAssociation:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		case ast.DescribeMicroflow:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		case ast.DescribeNanoflow:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		case ast.DescribePage:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describePageFuture(ctx, output, deps.PageRepo, deps.ImageBackend, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeSnippet:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeSnippetFuture(ctx, output, deps.SnippetRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeLayout:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		case ast.DescribeWorkflow:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		case ast.DescribeJavaAction:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		case ast.DescribeJavaScriptAction:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		case ast.DescribeModuleRole:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		case ast.DescribeUserRole:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeUserRoleGenFuture(ctx, output, deps.Security, s.Name)
			})
		case ast.DescribeDemoUser:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeDemoUserGenFuture(ctx, output, deps.Security, s.Name.Name)
			})
		case ast.DescribeFragment:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeFragmentFuture(ctx, output, e.fragments, s.Name)
			})
		case ast.DescribeODataClient:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeODataClientFn(ctx, deps, s.Name)
			})
		case ast.DescribeODataService:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeODataServiceFn(ctx, deps, s.Name)
			})
		case ast.DescribeExternalEntity:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeExternalEntityFn(ctx, deps, s.Name)
			})
		case ast.DescribeExportMapping:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeExportMappingFn(ctx, s.Name, deps)
			})
		case ast.DescribeImportMapping:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeImportMappingFn(ctx, s.Name, deps)
			})
		default:
			return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
				return entry.handler(ctx, s, deps)
			})
		}
	})

	// ────────────────────────────────────────────────────
	// Phase 3d-2b: module/entity/association CRUD handlers
	// ────────────────────────────────────────────────────

	r.RegisterFuture("CreateModule", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateModuleFn(ctx, stmt.(*ast.CreateModuleStmt), deps)
	})
	r.RegisterFuture("DropModule", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropModuleFn(ctx, stmt.(*ast.DropModuleStmt), deps)
	})

	r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecCreateEntity(ectx, stmt.(*ast.CreateEntityStmt))
	})
	r.RegisterFuture("AlterEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterEntityGenFn(ctx, stmt.(*ast.AlterEntityStmt), deps)
	})
	r.RegisterFuture("DropEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecDropEntity(ectx, stmt.(*ast.DropEntityStmt))
	})
	r.RegisterFuture("CreateViewEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecCreateViewEntity(ectx, stmt.(*ast.CreateViewEntityStmt))
	})

	r.RegisterFuture("CreateAssociation", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecCreateAssociation(ectx, stmt.(*ast.CreateAssociationStmt))
	})
	r.RegisterFuture("AlterAssociation", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecAlterAssociationGen(ectx, stmt.(*ast.AlterAssociationStmt))
	})
	r.RegisterFuture("DropAssociation", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecDropAssociationGen(ectx, stmt.(*ast.DropAssociationStmt))
	})

	// ────────────────────────────────────────────────────
	// Phase 3d-2c: microflow/page/workflow CRUD handlers
	// ────────────────────────────────────────────────────

	// Microflow handlers
	r.RegisterFuture("CreateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateMicroflowGenFn(ctx, stmt.(*ast.CreateMicroflowStmt), deps)
	})
	r.RegisterFuture("DropMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropMicroflowFn(ctx, stmt.(*ast.DropMicroflowStmt), deps)
	})

	// Nanoflow handlers
	r.RegisterFuture("CreateNanoflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateNanoflowGenFn(ctx, stmt.(*ast.CreateNanoflowStmt), deps)
	})
	r.RegisterFuture("DropNanoflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropNanoflowGenFn(ctx, stmt.(*ast.DropNanoflowStmt), deps)
	})

	// Page handlers
	r.RegisterFuture("CreatePageStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreatePageV3Fn(ctx, stmt.(*ast.CreatePageStmtV3), deps)
	})
	r.RegisterFuture("DropPage", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecDropPage(ectx, stmt.(*ast.DropPageStmt))
	})
	r.RegisterFuture("CreateSnippetStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateSnippetV3Fn(ctx, stmt.(*ast.CreateSnippetStmtV3), deps)
	})
	r.RegisterFuture("DropSnippet", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecDropSnippet(ectx, stmt.(*ast.DropSnippetStmt))
	})

	// Layout handler
	r.RegisterFuture("CreateLayout", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateOrModifyLayoutFn(ctx, stmt.(*ast.CreateLayoutStmt), deps)
	})

	// ALTER PAGE handler
	r.RegisterFuture("AlterPage", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterPageFn(ctx, stmt.(*ast.AlterPageStmt), deps)
	})

	// Workflow handlers
	r.RegisterFuture("CreateWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateWorkflowGenFn(ctx, stmt.(*ast.CreateWorkflowStmt), deps)
	})
	r.RegisterFuture("DropWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropWorkflowGenFn(ctx, stmt.(*ast.DropWorkflowStmt), deps)
	})
	r.RegisterFuture("AlterWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecAlterWorkflow(ectx, stmt.(*ast.AlterWorkflowStmt))
	})

	// ────────────────────────────────────────────────────
	// Phase 3d-2d: security CRUD handlers migrated from *ExecContext
	// ────────────────────────────────────────────────────

	r.RegisterFuture("CreateModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateModuleRoleGenFn(ctx, stmt.(*ast.CreateModuleRoleStmt), deps)
	})
	r.RegisterFuture("DropModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropModuleRoleGenFn(ctx, stmt.(*ast.DropModuleRoleStmt), deps)
	})
	r.RegisterFuture("CreateUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateUserRoleGenFn(ctx, stmt.(*ast.CreateUserRoleStmt), deps)
	})
	r.RegisterFuture("AlterUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterUserRoleGenFn(ctx, stmt.(*ast.AlterUserRoleStmt), deps)
	})
	r.RegisterFuture("DropUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropUserRoleGenFn(ctx, stmt.(*ast.DropUserRoleStmt), deps)
	})
	r.RegisterFuture("GrantEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantEntityAccessGenFn(ctx, stmt.(*ast.GrantEntityAccessStmt), deps)
	})
	r.RegisterFuture("RevokeEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeEntityAccessGenFn(ctx, stmt.(*ast.RevokeEntityAccessStmt), deps)
	})
	r.RegisterFuture("GrantPageAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantPageAccessGenFn(ctx, stmt.(*ast.GrantPageAccessStmt), deps)
	})
	r.RegisterFuture("RevokePageAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokePageAccessGenFn(ctx, stmt.(*ast.RevokePageAccessStmt), deps)
	})
	r.RegisterFuture("GrantMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantMicroflowAccessGenFn(ctx, stmt.(*ast.GrantMicroflowAccessStmt), deps)
	})
	r.RegisterFuture("RevokeMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeMicroflowAccessGenFn(ctx, stmt.(*ast.RevokeMicroflowAccessStmt), deps)
	})
	r.RegisterFuture("GrantNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantNanoflowAccessGenFn(ctx, stmt.(*ast.GrantNanoflowAccessStmt), deps)
	})
	r.RegisterFuture("RevokeNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeNanoflowAccessGenFn(ctx, stmt.(*ast.RevokeNanoflowAccessStmt), deps)
	})
	r.RegisterFuture("GrantWorkflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecGrantWorkflowAccess(ectx, stmt.(*ast.GrantWorkflowAccessStmt))
	})
	r.RegisterFuture("RevokeWorkflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return ExecRevokeWorkflowAccess(ectx, stmt.(*ast.RevokeWorkflowAccessStmt))
	})
	r.RegisterFuture("GrantODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantODataServiceAccessGenFn(ctx, stmt.(*ast.GrantODataServiceAccessStmt), deps)
	})
	r.RegisterFuture("RevokeODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeODataServiceAccessGenFn(ctx, stmt.(*ast.RevokeODataServiceAccessStmt), deps)
	})
	r.RegisterFuture("GrantPublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantPublishedRestServiceAccessGenFn(ctx, stmt.(*ast.GrantPublishedRestServiceAccessStmt), deps)
	})
	r.RegisterFuture("RevokePublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokePublishedRestServiceAccessGenFn(ctx, stmt.(*ast.RevokePublishedRestServiceAccessStmt), deps)
	})
	r.RegisterFuture("AlterProjectSecurity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterProjectSecurityGenFn(ctx, stmt.(*ast.AlterProjectSecurityStmt), deps)
	})
	r.RegisterFuture("UpdateSecurity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecUpdateSecurityGenFn(ctx, stmt.(*ast.UpdateSecurityStmt), deps)
	})
	r.RegisterFuture("CreateDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateDemoUserGenFn(ctx, stmt.(*ast.CreateDemoUserStmt), deps)
	})
	r.RegisterFuture("DropDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropDemoUserGenFn(ctx, stmt.(*ast.DropDemoUserStmt), deps)
	})
	r.RegisterFuture("AlterLanguage", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(ctx, deps)
		return AlterLanguage(ectx, stmt.(*ast.AlterLanguageStmt))
	})

	// ────────────────────────────────────────────────────
	// Phase 3d-2e: remaining handler registrations
	// ────────────────────────────────────────────────────

	// Enumeration CRUD
	r.RegisterFuture("CreateEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(nil, deps)
		return ExecCreateEnumeration(ectx, stmt.(*ast.CreateEnumerationStmt))
	})
	r.RegisterFuture("AlterEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return fmt.Errorf("alter enumeration not yet implemented")
	})
	r.RegisterFuture("DropEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(nil, deps)
		return ExecDropEnumeration(ectx, stmt.(*ast.DropEnumerationStmt))
	})

	// Constant CRUD
	r.RegisterFuture("CreateConstant", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(nil, deps)
		return ExecCreateConstant(ectx, stmt.(*ast.CreateConstantStmt))
	})
	r.RegisterFuture("DropConstant", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(nil, deps)
		return ExecDropConstant(ectx, stmt.(*ast.DropConstantStmt))
	})

	// Module settings
	r.RegisterFuture("AlterModuleJarDep", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterModuleJarDepFuture(ctx, stmt, deps)
	})

	// Database connection
	r.RegisterFuture("CreateDatabaseConnection", func(ctx context.Context, stmt ast.Statement) error {
		ectx := NewExecContext(nil, deps)
		return ExecCreateDatabaseConnection(ectx, stmt.(*ast.CreateDatabaseConnectionStmt))
	})

	// Java/JavaScript action CRUD
	r.RegisterFuture("CreateJavaAction", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateJavaActionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropJavaAction", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropJavaActionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateJavaScriptAction", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateJavaScriptActionFuture(ctx, stmt, deps)
	})

	// Folder/rename/move
	r.RegisterFuture("DropFolder", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropFolderFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("MoveFolder", func(ctx context.Context, stmt ast.Statement) error {
		return ExecMoveFolderFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("Move", func(ctx context.Context, stmt ast.Statement) error {
		return ExecMoveFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("Rename", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRenameFuture(ctx, stmt, deps)
	})

	// Navigation
	r.RegisterFuture("AlterNavigation", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterNavigationFuture(ctx, stmt, deps)
	})

	// Image collection CRUD
	r.RegisterFuture("CreateImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateImageCollectionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropImageCollectionFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterImageCollection", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterImageCollectionFuture(ctx, stmt, deps)
	})

	// Settings
	r.RegisterFuture("AlterSettings", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterSettingsFuture(ctx, stmt, deps)
	})

	// Translate
	r.RegisterFuture("Translate", func(ctx context.Context, stmt ast.Statement) error {
		return ExecTranslateFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("TranslateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		return ExecTranslateMicroflowFuture(ctx, deps)
	})

	// Configuration CRUD
	r.RegisterFuture("CreateConfiguration", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateConfigurationFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropConfiguration", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropConfigurationFuture(ctx, stmt, deps)
	})

	// Business event service CRUD
	r.RegisterFuture("CreateBusinessEventService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateBusinessEventServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropBusinessEventService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropBusinessEventServiceFuture(ctx, stmt, deps)
	})

	// OData client CRUD
	r.RegisterFuture("CreateODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateODataClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterODataClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropODataClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropODataClientFuture(ctx, stmt, deps)
	})

	// OData service CRUD
	r.RegisterFuture("CreateODataService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateODataServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterODataService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterODataServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropODataService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropODataServiceFuture(ctx, stmt, deps)
	})

	// JSON structure CRUD
	r.RegisterFuture("CreateJsonStructure", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateJsonStructureFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropJsonStructure", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropJsonStructureFuture(ctx, stmt, deps)
	})

	// Import/Export mapping CRUD
	r.RegisterFuture("CreateImportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateImportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropImportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropImportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateExportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateExportMappingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropExportMapping", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropExportMappingFuture(ctx, stmt, deps)
	})

	// REST client CRUD
	r.RegisterFuture("CreateRestClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateRestClientFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropRestClient", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropRestClientFuture(ctx, stmt, deps)
	})

	// Contract from OpenAPI
	r.RegisterFuture("DescribeContractFromOpenAPI", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeContractFromOpenAPIFuture(ctx, stmt, deps)
	})

	// Published REST service CRUD
	r.RegisterFuture("CreatePublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreatePublishedRestServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropPublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropPublishedRestServiceFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterPublishedRestService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterPublishedRestServiceFuture(ctx, stmt, deps)
	})

	// External entities
	r.RegisterFuture("CreateExternalEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateExternalEntityFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateExternalEntities", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateExternalEntitiesFuture(ctx, stmt, deps)
	})

	// Data transformer CRUD
	r.RegisterFuture("CreateDataTransformer", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateDataTransformerFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropDataTransformer", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropDataTransformerFuture(ctx, stmt, deps)
	})

	// Widget commands
	r.RegisterFuture("ShowWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowWidgetsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("ShowInstalledWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowInstalledWidgetsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("UpdateWidgets", func(ctx context.Context, stmt ast.Statement) error {
		return ExecUpdateWidgetsFuture(ctx, stmt, deps)
	})

	// Catalog/query
	r.RegisterFuture("Select", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSelectFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeTranslations", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeTranslationsFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeCatalogTable", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeCatalogTableFuture(ctx, deps)
	})

	// Features
	r.RegisterFuture("ShowFeatures", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowFeaturesFuture(ctx, stmt, deps)
	})

	// Styling
	r.RegisterFuture("ShowDesignProperties", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowDesignPropertiesFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DescribeStyling", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeStylingFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("AlterStyling", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterStylingFuture(ctx, stmt, deps)
	})

	// Theme
	r.RegisterFuture("ShowThemeVariables", func(ctx context.Context, stmt ast.Statement) error {
		return ExecShowThemeVariablesFuture(ctx, stmt, deps)
	})

	// Search
	r.RegisterFuture("Search", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSearchFuture(ctx, stmt, deps)
	})

	// Refresh catalog
	r.RegisterFuture("RefreshCatalog", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRefreshCatalogFuture(ctx, stmt, deps)
	})

	// Lint
	r.RegisterFuture("Lint", func(ctx context.Context, stmt ast.Statement) error {
		return ExecLintFn(ctx, stmt.(*ast.LintStmt), deps)
	})

	// Fragment commands
	r.RegisterFuture("DefineFragment", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDefineFragmentFn(ctx, stmt.(*ast.DefineFragmentStmt), deps)
	})
	r.RegisterFuture("DescribeFragmentFrom", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDescribeFragmentFromFn(ctx, stmt.(*ast.DescribeFragmentFromStmt), deps)
	})

	// SQL commands
	r.RegisterFuture("SQLConnect", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLConnectFn(ctx, stmt.(*ast.SQLConnectStmt), deps)
	})
	r.RegisterFuture("SQLDisconnect", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLDisconnectFn(ctx, stmt.(*ast.SQLDisconnectStmt), deps)
	})
	r.RegisterFuture("SQLConnections", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLConnectionsFn(ctx, deps)
	})
	r.RegisterFuture("SQLQuery", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLQueryFn(ctx, stmt.(*ast.SQLQueryStmt), deps)
	})
	r.RegisterFuture("SQLShowTables", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLShowTablesFn(ctx, stmt.(*ast.SQLShowTablesStmt), deps)
	})
	r.RegisterFuture("SQLShowViews", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLShowViewsFn(ctx, stmt.(*ast.SQLShowViewsStmt), deps)
	})
	r.RegisterFuture("SQLShowFunctions", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLShowFunctionsFn(ctx, stmt.(*ast.SQLShowFunctionsStmt), deps)
	})
	r.RegisterFuture("SQLDescribeTable", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLDescribeTableFn(ctx, stmt.(*ast.SQLDescribeTableStmt), deps)
	})
	r.RegisterFuture("SQLGenerateConnector", func(ctx context.Context, stmt ast.Statement) error {
		return ExecSQLGenerateConnectorFn(ctx, stmt.(*ast.SQLGenerateConnectorStmt), deps)
	})

	// Import
	r.RegisterFuture("Import", func(ctx context.Context, stmt ast.Statement) error {
		return ExecImportFn(ctx, stmt.(*ast.ImportStmt), deps)
	})

	// Agent editor CRUD
	r.RegisterFuture("CreateModel", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateModelFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("DropModel", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropModelFuture(ctx, stmt, deps)
	})
	r.RegisterFuture("CreateConsumedMCPService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateConsumedMCPServiceFn(ctx, stmt.(*ast.CreateConsumedMCPServiceStmt), deps)
	})
	r.RegisterFuture("DropConsumedMCPService", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropConsumedMCPServiceFn(ctx, stmt.(*ast.DropConsumedMCPServiceStmt), deps)
	})
	r.RegisterFuture("CreateKnowledgeBase", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateKnowledgeBaseFn(ctx, stmt.(*ast.CreateKnowledgeBaseStmt), deps)
	})
	r.RegisterFuture("DropKnowledgeBase", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropKnowledgeBaseFn(ctx, stmt.(*ast.DropKnowledgeBaseStmt), deps)
	})
	r.RegisterFuture("CreateAgent", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateAgentFn(ctx, stmt.(*ast.CreateAgentStmt), deps)
	})
	r.RegisterFuture("DropAgent", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropAgentFn(ctx, stmt.(*ast.DropAgentStmt), deps)
	})
}

// execContextToDeps bridges old-style *ExecContext to *HandlerDeps.
func execContextToDeps(ectx *ExecContext) *HandlerDeps {
	return &HandlerDeps{
		Output:       ectx.Output,
		StatusOutput: ectx.StatusOutput,
		Logger:       ectx.Logger,
		Quiet:        ectx.Quiet,
		Backend:      ectx.Backend,

		ConnectionManager:    ectx.ConnectionManager,
		ModuleLister:         ectx.ModuleLister,
		ModuleWriter:         ectx.ModuleWriter,
		DomainModelReader:    ectx.DomainModelReader,
		DomainModelWriter:    ectx.DomainModelWriter,
		MicroflowReader:      ectx.MicroflowReader,
		MicroflowWriter:      ectx.MicroflowWriter,
		WorkflowReader:       ectx.WorkflowReader,
		WorkflowWriter:       ectx.WorkflowWriter,
		PageReader:           ectx.PageReader,
		PageWriter:           ectx.PageWriter,
		JavaActionReader:     ectx.JavaActionReader,
		JavaActionWriter:     ectx.JavaActionWriter,
		JavaScriptActionWriter: ectx.JavaScriptActionWriter,
		EnumerationReader:    ectx.EnumerationReader,
		EnumerationWriter:    ectx.EnumerationWriter,
		ConstantReader:       ectx.ConstantReader,
		ConstantWriter:       ectx.ConstantWriter,
		SettingsReader:       ectx.SettingsReader,
		SettingsWriter:       ectx.SettingsWriter,
		MapperReader:         ectx.MappingReader,
		MapperWriter:         ectx.MappingWriter,
		UnitReader:           ectx.UnitReader,
		UnitWriter:           ectx.UnitWriter,
		NavigationReader:     ectx.NavigationReader,
		NavigationWriter:     ectx.NavigationWriter,
		ImageCollectionWriter: ectx.ImageCollectionWriter,
		ScheduledEventReader: ectx.ScheduledEventReader,
		ServiceLister:        ectx.ServiceLister,
		ServiceWriter:        ectx.ServiceWriter,
		MetadataReader:       ectx.MetadataReader,
		FolderManager:        ectx.FolderManager,
		ModuleSettingsReader: ectx.ModuleSettingsReader,
		ModuleSettingsWriter: ectx.ModuleSettingsWriter,
		RenameManager:        ectx.RenameManager,
		SecurityProjectManager:      ectx.SecurityProjectManager,
		SecurityModuleManager:       ectx.SecurityModuleManager,
		SecurityEntityAccessManager: ectx.SecurityEntityAccessManager,
		PageModelAccess:             ectx.PageModelAccess,
		PageMutationOperator:        ectx.PageMutationOperator,
		WorkflowMutationOperator:    ectx.WorkflowMutationOperator,
		WidgetBuilder:               ectx.WidgetBuilder,
		ScriptTransactionManager:    ectx.ScriptTransactionManager,
		AgentEditorOperator:         ectx.AgentEditorOperator,
		ImageBackend:                ectx.Backend,
		BusinessEventBackend:        ectx.Backend,

		DomainModels:      ectx.DomainModels,
		MicroflowRepo:     ectx.Microflows,
		NanoflowRepo:      ectx.Nanoflows,
		PageRepo:          ectx.Pages,
		LayoutRepo:        ectx.Layouts,
		SnippetRepo:       ectx.Snippets,
		JavaActionRepo:    ectx.JavaActions,
		JavaScriptActionRepo: ectx.JavaScriptActions,
		WorkflowRepo:      ectx.Workflows,
		Security:          ectx.Security,

		SqlMgr:    ectx.SqlMgr,
		Cache:     ectx.Cache,
		Session:   ectx.Session,
		Fragments: ectx.Fragments,
		Settings:  ectx.Settings,
		Format:    ectx.Format,
		MprPath:   ectx.MprPath,
		Graph:     ectx.Graph,
		Perf:      ectx.Perf,

		ScriptDepth:                      ectx.ScriptDepth,
		DescribingMicroflowHasReturnValue: ectx.DescribingMicroflowHasReturnValue,
		ThemeRegistry:  ectx.ThemeRegistry,
		ExecuteFn:      ectx.ExecuteFn,
		ExecuteProgramFn: ectx.ExecuteProgramFn,
		FinalizeFn:     ectx.FinalizeFn,
		SyncGraph:      ectx.SyncGraph,
	}
}

// buildHandlerDeps populates a HandlerDeps from the current Executor state.
func (e *Executor) buildHandlerDeps() *HandlerDeps {
	if e.backend == nil {
		return &HandlerDeps{
			Output:       e.output,
			StatusOutput: e.statusOutput,
			Logger:       e.logger,
			Quiet:        e.quiet,
			Fragments:    e.fragments,
			Format:       e.format,
			MprPath:      e.mprPath,
			Settings:     nil,
		}
	}
	return &HandlerDeps{
		Output:       e.output,
		StatusOutput: e.statusOutput,
		Logger:       e.logger,
		Quiet:        e.quiet,
		Backend:      e.backend,

		ConnectionManager:    e.backend,
		ModuleLister:         e.backend,
		ModuleWriter:         e.backend,
		DomainModelReader:    e.backend,
		DomainModelWriter:    e.backend,
		MicroflowReader:      e.backend,
		MicroflowWriter:      e.backend,
		WorkflowReader:       e.backend,
		WorkflowWriter:       e.backend,
		PageReader:           e.backend,
		PageWriter:           e.backend,
		JavaActionReader:     e.backend,
		JavaActionWriter:     e.backend,
		JavaScriptActionWriter: e.backend,
		EnumerationReader:    e.backend,
		EnumerationWriter:    e.backend,
		ConstantReader:       e.backend,
		ConstantWriter:       e.backend,
		SettingsReader:       e.backend,
		SettingsWriter:       e.backend,
		MapperReader:         e.backend,
		MapperWriter:         e.backend,
		UnitReader:           e.backend,
		UnitWriter:           e.backend,
		NavigationReader:     e.backend,
		NavigationWriter:     e.backend,
		ImageCollectionWriter: e.backend,
		ScheduledEventReader: e.backend,
		ServiceLister:        e.backend,
		ServiceWriter:        e.backend,
		MetadataReader:       e.backend,
		FolderManager:        e.backend,
		ModuleSettingsReader: e.backend,
		ModuleSettingsWriter: e.backend,
		RenameManager:        e.backend,
		SecurityProjectManager:      e.backend,
		SecurityModuleManager:       e.backend,
		SecurityEntityAccessManager: e.backend,
		PageModelAccess:             e.backend,
		PageMutationOperator:        e.backend,
		WorkflowMutationOperator:    e.backend,
		WidgetBuilder:               e.backend,
		ScriptTransactionManager:    e.backend,
		AgentEditorOperator:         e.backend,
		ImageBackend:                e.backend,
		BusinessEventBackend:        e.backend,

		DomainModels:         extractDomainModelsRepo(e.backend),
		MicroflowRepo:        extractMicroflowsRepo(e.backend),
		NanoflowRepo:         extractNanoflowsRepo(e.backend),
		PageRepo:             extractPagesRepo(e.backend),
		LayoutRepo:           extractLayoutsRepo(e.backend),
		SnippetRepo:          extractSnippetsRepo(e.backend),
		JavaActionRepo:       extractJavaActionsRepo(e.backend),
		JavaScriptActionRepo: extractJavaScriptActionsRepo(e.backend),
		WorkflowRepo:         extractWorkflowsRepo(e.backend),
		Security:             extractSecurityRepo(e.backend),

		Fragments:                  e.fragments,
		Format:                     e.format,
		Settings:                   nil,
		MprPath:                    e.mprPath,
		Graph:                      nil,
		Perf:                       nil,
		Cache: e.cache,
		Session: func() *sessionTracker {
			if e.cache != nil {
				return &e.cache.sessionTracker
			}
			return &sessionTracker{}
		}(),
		ExecuteFn:                  e.Execute,
		ExecuteProgramFn:           e.ExecuteProgram,
		FinalizeFn:                 e.finalizeProgramExecution,
	}
}
