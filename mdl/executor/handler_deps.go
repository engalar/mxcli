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
	deps := e.buildHandlerDeps()
	if deps == nil {
		return
	}
	r := e.registry

	// Session handlers migrated from *ExecContext:
	// e.format is captured by reference (e is *Executor pointer) — reflects SET format changes.
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
