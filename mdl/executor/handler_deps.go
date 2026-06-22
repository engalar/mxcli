package executor

import (
	"context"
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/diaglog"
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
	MetadataReader       backend.MetadataReader

	// DomainModels repo for entity counting (Stage 3 repos).
	DomainModels repos.DomainModelRepository
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
		case ast.ShowFragments:
			return listFragmentsFuture(ctx, deps.Output, e.fragments)
		default:
			return nil // fall through to old handler
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

		ConnectionManager: e.backend,
		ModuleLister:      e.backend,
		MetadataReader:    e.backend,
		FolderManager:     e.backend,
		DomainModels:      extractDomainModelsRepo(e.backend),
	}
}
