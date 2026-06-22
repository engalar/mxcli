package executor

import (
	"context"
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/diaglog"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/repos"
)

// HandlerDeps carries the execution dependencies that were previously
// scattered across ExecContext fields. Each handler captures only the
// deps it needs via closure at registration time.
//
// Transition: Handlers that use ExecContext fields are migrated to
// StmtHandlerFunc one at a time. Migrated handlers capture deps from
// a HandlerDeps that was populated from the Executor at startup.
// Once all handlers are migrated, HandlerDeps and ExecContext are removed.
type HandlerDeps struct {
	Output       io.Writer
	StatusOutput io.Writer
	Logger       *diaglog.Logger
	Quiet        bool

	// Deprecated: use specific role interfaces below.
	Backend              backend.FullBackend
	BackendFactory       BackendFactory
	ConnectionManager    backend.ConnectionManager
	ModuleLister         backend.ModuleLister
	FolderManager        backend.FolderManager
	ModuleSettingsReader backend.ModuleSettingsReader
	DomainModelReader    backend.DomainModelReader
	MicroflowReader      backend.MicroflowReader
	PageReader           backend.PageReader
	PageWriter           backend.PageWriter
	EnumerationReader    backend.EnumerationReader
	ConstantReader       backend.ConstantReader
	SettingsReader       backend.SettingsReader
	MapperReader         backend.MappingReader
	MapperWriter         backend.MappingWriter
	NavigationReader     backend.NavigationReader
	ScheduledEventReader backend.ScheduledEventReader
	MetadataReader       backend.MetadataReader

	// DomainModels repo for entity counting (Stage 3 repos).
	DomainModels repos.DomainModelRepository

	// Stage 3 flow/page/action repos for show handlers.
	MicroflowRepo        repos.MicroflowRepository
	NanoflowRepo         repos.NanoflowRepository
	PageRepo             repos.PageRepository
	LayoutRepo           repos.LayoutRepository
	SnippetRepo          repos.SnippetRepository
	JavaActionRepo       repos.JavaActionRepository
	JavaScriptActionRepo repos.JavaScriptActionRepository
	WorkflowRepo         repos.WorkflowRepository
	BusinessEventBackend backend.BusinessEventBackend

	// Security repo for project/module security reads (Phase 3d-1f).
	Security repos.SecurityRepository

	// Describe handler deps (Phase 3d-1h).
	ServiceLister backend.ServiceLister
	ImageBackend  backend.ImageBackend
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
		case ast.ShowStructure:
			return execShowStructureGenFuture(ctx, deps.Output, e.format, s, deps)
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
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeAssociationFuture(ctx, output, deps.ModuleLister, deps.DomainModelReader, s.Name)
			})
		case ast.DescribeMicroflow:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeMicroflowGenFuture(ctx, output, deps.MicroflowRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeNanoflow:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeNanoflowGenFuture(ctx, output, deps.NanoflowRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
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
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeLayoutFuture(ctx, output, deps.LayoutRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeWorkflow:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeWorkflowGenFuture(ctx, output, deps.WorkflowRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
			})
		case ast.DescribeJavaAction:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeJavaActionGenFuture(ctx, output, deps.JavaActionRepo, s.Name)
			})
		case ast.DescribeJavaScriptAction:
			ectx := ctx.(*ExecContext)
			return writeDescribeJSON(ectx, name, entry.label, func() error {
				return entry.handler(ectx, s)
			})
		case ast.DescribeModuleRole:
			return writeDescribeJSONFuture(deps.Output, e.format, name, entry.label, func(output io.Writer) error {
				return describeModuleRoleGenFuture(ctx, output, deps.Security, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, s.Name)
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
		default:
			// Not yet migrated — fall through to old handler.
			ectx := ctx.(*ExecContext)
			return writeDescribeJSON(ectx, name, entry.label, func() error {
				return entry.handler(ectx, s)
			})
		}
	})

	// ────────────────────────────────────────────────────
	// Phase 3d-2b: module/entity/association CRUD handlers
	// ────────────────────────────────────────────────────

	r.RegisterFuture("CreateModule", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateModule(ectx, stmt.(*ast.CreateModuleStmt))
	})
	r.RegisterFuture("DropModule", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropModule(ectx, stmt.(*ast.DropModuleStmt))
	})

	r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateEntity(ectx, stmt.(*ast.CreateEntityStmt))
	})
	r.RegisterFuture("AlterEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execAlterEntity(ectx, stmt.(*ast.AlterEntityStmt))
	})
	r.RegisterFuture("DropEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropEntity(ectx, stmt.(*ast.DropEntityStmt))
	})
	r.RegisterFuture("CreateViewEntity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateViewEntity(ectx, stmt.(*ast.CreateViewEntityStmt))
	})

	r.RegisterFuture("CreateAssociation", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateAssociation(ectx, stmt.(*ast.CreateAssociationStmt))
	})
	r.RegisterFuture("AlterAssociation", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execAlterAssociation(ectx, stmt.(*ast.AlterAssociationStmt))
	})
	r.RegisterFuture("DropAssociation", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropAssociation(ectx, stmt.(*ast.DropAssociationStmt))
	})

	// ────────────────────────────────────────────────────
	// Phase 3d-2c: microflow/page/workflow CRUD handlers
	// ────────────────────────────────────────────────────

	// Microflow handlers
	r.RegisterFuture("CreateMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateMicroflowGen(ectx, stmt.(*ast.CreateMicroflowStmt))
	})
	r.RegisterFuture("DropMicroflow", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropMicroflow(ectx, stmt.(*ast.DropMicroflowStmt))
	})

	// Nanoflow handlers
	r.RegisterFuture("CreateNanoflow", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateNanoflowGen(ectx, stmt.(*ast.CreateNanoflowStmt))
	})
	r.RegisterFuture("DropNanoflow", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropNanoflowGen(ectx, stmt.(*ast.DropNanoflowStmt))
	})

	// Page handlers
	r.RegisterFuture("CreatePageStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreatePageV3(ectx, stmt.(*ast.CreatePageStmtV3))
	})
	r.RegisterFuture("DropPage", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropPage(ectx, stmt.(*ast.DropPageStmt))
	})
	r.RegisterFuture("CreateSnippetStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateSnippetV3(ectx, stmt.(*ast.CreateSnippetStmtV3))
	})
	r.RegisterFuture("DropSnippet", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropSnippet(ectx, stmt.(*ast.DropSnippetStmt))
	})

	// Layout handler
	r.RegisterFuture("CreateLayout", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateOrModifyLayout(ectx, stmt.(*ast.CreateLayoutStmt))
	})

	// ALTER PAGE handler
	r.RegisterFuture("AlterPage", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execAlterPage(ectx, stmt.(*ast.AlterPageStmt))
	})

	// Workflow handlers
	r.RegisterFuture("CreateWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateWorkflowGen(ectx, stmt.(*ast.CreateWorkflowStmt))
	})
	r.RegisterFuture("DropWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropWorkflowGen(ectx, stmt.(*ast.DropWorkflowStmt))
	})
	r.RegisterFuture("AlterWorkflow", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execAlterWorkflow(ectx, stmt.(*ast.AlterWorkflowStmt))
	})

	// ────────────────────────────────────────────────────
	// Phase 3d-2d: security CRUD handlers migrated from *ExecContext
	// ────────────────────────────────────────────────────

	r.RegisterFuture("CreateModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateModuleRoleGen(ectx, stmt.(*ast.CreateModuleRoleStmt))
	})
	r.RegisterFuture("DropModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropModuleRoleGen(ectx, stmt.(*ast.DropModuleRoleStmt))
	})
	r.RegisterFuture("CreateUserRole", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateUserRoleGen(ectx, stmt.(*ast.CreateUserRoleStmt))
	})
	r.RegisterFuture("AlterUserRole", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execAlterUserRoleGen(ectx, stmt.(*ast.AlterUserRoleStmt))
	})
	r.RegisterFuture("DropUserRole", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropUserRoleGen(ectx, stmt.(*ast.DropUserRoleStmt))
	})
	r.RegisterFuture("GrantEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execGrantEntityAccessGen(ectx, stmt.(*ast.GrantEntityAccessStmt))
	})
	r.RegisterFuture("RevokeEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execRevokeEntityAccessGen(ectx, stmt.(*ast.RevokeEntityAccessStmt))
	})
	r.RegisterFuture("GrantPageAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execGrantPageAccessGen(ectx, stmt.(*ast.GrantPageAccessStmt))
	})
	r.RegisterFuture("RevokePageAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execRevokePageAccessGen(ectx, stmt.(*ast.RevokePageAccessStmt))
	})
	r.RegisterFuture("GrantMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execGrantMicroflowAccessGen(ectx, stmt.(*ast.GrantMicroflowAccessStmt))
	})
	r.RegisterFuture("RevokeMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execRevokeMicroflowAccessGen(ectx, stmt.(*ast.RevokeMicroflowAccessStmt))
	})
	r.RegisterFuture("GrantNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execGrantNanoflowAccessGen(ectx, stmt.(*ast.GrantNanoflowAccessStmt))
	})
	r.RegisterFuture("RevokeNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execRevokeNanoflowAccessGen(ectx, stmt.(*ast.RevokeNanoflowAccessStmt))
	})
	r.RegisterFuture("GrantWorkflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execGrantWorkflowAccess(ectx, stmt.(*ast.GrantWorkflowAccessStmt))
	})
	r.RegisterFuture("RevokeWorkflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execRevokeWorkflowAccess(ectx, stmt.(*ast.RevokeWorkflowAccessStmt))
	})
	r.RegisterFuture("GrantODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execGrantODataServiceAccessGen(ectx, stmt.(*ast.GrantODataServiceAccessStmt))
	})
	r.RegisterFuture("RevokeODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execRevokeODataServiceAccessGen(ectx, stmt.(*ast.RevokeODataServiceAccessStmt))
	})
	r.RegisterFuture("GrantPublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execGrantPublishedRestServiceAccessGen(ectx, stmt.(*ast.GrantPublishedRestServiceAccessStmt))
	})
	r.RegisterFuture("RevokePublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execRevokePublishedRestServiceAccessGen(ectx, stmt.(*ast.RevokePublishedRestServiceAccessStmt))
	})
	r.RegisterFuture("AlterProjectSecurity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execAlterProjectSecurityGen(ectx, stmt.(*ast.AlterProjectSecurityStmt))
	})
	r.RegisterFuture("UpdateSecurity", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execUpdateSecurityGen(ectx, stmt.(*ast.UpdateSecurityStmt))
	})
	r.RegisterFuture("CreateDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execCreateDemoUserGen(ectx, stmt.(*ast.CreateDemoUserStmt))
	})
	r.RegisterFuture("DropDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return execDropDemoUserGen(ectx, stmt.(*ast.DropDemoUserStmt))
	})
	r.RegisterFuture("AlterLanguage", func(ctx context.Context, stmt ast.Statement) error {
		ectx := phase3d2bNewExecContext(ctx, deps)
		return alterLanguage(ectx, stmt.(*ast.AlterLanguageStmt))
	})
}

// buildHandlerDeps populates a HandlerDeps from the current Executor state.
func (e *Executor) buildHandlerDeps() *HandlerDeps {
	if e.backend == nil {
		return nil
	}
	return &HandlerDeps{
		Output:       e.output,
		StatusOutput: e.statusOutput,
		Logger:       e.logger,
		Quiet:        e.quiet,
		Backend:      e.backend,

		ConnectionManager:    e.backend,
		ModuleLister:         e.backend,
		FolderManager:        e.backend,
		MetadataReader:       e.backend,
		EnumerationReader:    e.backend,
		ConstantReader:       e.backend,
		SettingsReader:       e.backend,
		NavigationReader:     e.backend,
		ScheduledEventReader: e.backend,
		DomainModelReader:    e.backend,
		DomainModels:         extractDomainModelsRepo(e.backend),
		MicroflowRepo:        extractMicroflowsRepo(e.backend),
		NanoflowRepo:         extractNanoflowsRepo(e.backend),
		PageRepo:             extractPagesRepo(e.backend),
		LayoutRepo:           extractLayoutsRepo(e.backend),
		SnippetRepo:          extractSnippetsRepo(e.backend),
		JavaActionRepo:       extractJavaActionsRepo(e.backend),
		JavaScriptActionRepo: extractJavaScriptActionsRepo(e.backend),
		WorkflowRepo:         extractWorkflowsRepo(e.backend),
		BusinessEventBackend: e.backend,
		Security:             extractSecurityRepo(e.backend),
		ServiceLister:        e.backend,
		ImageBackend:         e.backend,
	}
}
