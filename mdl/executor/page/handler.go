// SPDX-License-Identifier: Apache-2.0

package page

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	bgCtx := context.Background()
	ectx := executor.NewMinimalExecCtx(bgCtx, deps)

	d := PageDeps{
		ConnectionManager:    deps.ConnectionManager,
		ModuleLister:         deps.ModuleLister,
		ModuleWriter:         deps.ModuleWriter,
		PageWriter:           deps.PageWriter,
		PageReader:           deps.PageReader,
		FolderManager:        deps.FolderManager,
		MetadataReader:       deps.MetadataReader,
		DomainModelReader:    deps.DomainModelReader,
		WidgetBuilder:        deps.WidgetBuilder,
		PageModelAccess:      deps.PageModelAccess,
		PageMutationOperator: deps.PageMutationOperator,

		PagesRepo:        deps.PageRepo,
		SnippetsRepo:     deps.SnippetRepo,
		LayoutsRepo:      deps.LayoutRepo,
		MicroflowsRepo:   deps.MicroflowRepo,
		NanoflowsRepo:    deps.NanoflowRepo,

		Output: deps.Output,

		FindOrCreateModule: func(name string) (*model.Module, error) {
			return executor.FindOrCreateModuleWrap(ectx, name)
		},
		FindModule: func(name string) (*model.Module, error) {
			return executor.FindModuleWrap(ectx, name)
		},
		GetModuleName: func(moduleID model.ID) string {
			h, err := executor.GetHierarchyForMining(ectx)
			if err != nil {
				return ""
			}
			return h.GetModuleName(moduleID)
		},
		GetModuleID: func(containerID model.ID) model.ID {
			h, err := executor.GetHierarchyForMining(ectx)
			if err != nil {
				return ""
			}
			return h.FindModuleID(containerID)
		},

		CheckFeature: func(area, name, statement, hint string) error {
			return executor.CheckFeatureWrap(ectx, area, name, statement, hint)
		},
		InvalidateHierarchy: func() {
			executor.InvalidateHierarchyWrap(ectx)
		},
		InvalidatePagesGenCache: func() {
			executor.InvalidatePagesGenCacheWrap(ectx)
		},
		TrackCreatedPage: func(moduleName, pageName string, pageID model.ID, moduleID model.ID) {
			executor.TrackCreatedPageWrap(ectx, moduleName, pageName, pageID, moduleID)
		},
		TrackCreatedSnippet: func(moduleName, snippetName string, snippetID model.ID, moduleID model.ID) {
			executor.TrackCreatedSnippetWrap(ectx, moduleName, snippetName, snippetID, moduleID)
		},
		DefaultDocumentAccessRoleQNames: func(module *model.Module) []string {
			return executor.DefaultDocumentAccessRoleQNamesWrap(ectx, module)
		},

		BuildPage: func(s *ast.CreatePageStmtV3, moduleID model.ID, moduleName string) (*genPg.Page, model.ID, error) {
			return executor.BuildPageV3WithDeps(bgCtx, deps, s, moduleID, moduleName)
		},
		BuildSnippet: func(s *ast.CreateSnippetStmtV3, moduleID model.ID, moduleName string) (*genPg.Snippet, model.ID, error) {
			return executor.BuildSnippetV3WithDeps(bgCtx, deps, s, moduleID, moduleName)
		},

		BuildLayoutContent: func(s *ast.CreateLayoutStmt) element.Element {
			return executor.BuildLayoutContentWrap(ectx, s)
		},
		PageASTToModel: func(s *ast.CreatePageStmtV3, moduleName string) (*types.PageModel, error) {
			return executor.PageASTToModelWrap(ectx, s, moduleName)
		},
		PageModelHasLossyWidget: func(pm *types.PageModel) bool {
			return executor.PageModelHasLossyWidgetWrap(pm)
		},
	}

	r.RegisterFuture("CreatePageStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreatePageFn(ctx, stmt.(*ast.CreatePageStmtV3), d)
	})
	r.RegisterFuture("CreateSnippetStmtV3", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateSnippetFn(ctx, stmt.(*ast.CreateSnippetStmtV3), d)
	})
	r.RegisterFuture("DropPage", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropPageFn(ctx, stmt.(*ast.DropPageStmt), d)
	})
	r.RegisterFuture("DropSnippet", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropSnippetFn(ctx, stmt.(*ast.DropSnippetStmt), d)
	})
	r.RegisterFuture("CreateLayout", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateLayoutFn(ctx, stmt.(*ast.CreateLayoutStmt), d)
	})
}
