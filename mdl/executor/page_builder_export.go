// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// BuildPageV3WithDeps creates a pageBuilder from HandlerDeps and builds a page.
// Exported so the page subpackage can wire its BuildPage function field.
func BuildPageV3WithDeps(ctx context.Context, deps *HandlerDeps, s *ast.CreatePageStmtV3, moduleID model.ID, moduleName string) (*genPg.Page, model.ID, error) {
	ectx := NewExecContext(ctx, deps)
	pb := pageBuilderFromExecCtx(ectx, moduleID, moduleName, false, nil)
	genPage, err := pb.buildPageV3(s)
	return genPage, pb.lastContainerID, err
}

// BuildSnippetV3WithDeps creates a pageBuilder from HandlerDeps and builds a snippet.
// Exported so the page subpackage can wire its BuildSnippet function field.
func BuildSnippetV3WithDeps(ctx context.Context, deps *HandlerDeps, s *ast.CreateSnippetStmtV3, moduleID model.ID, moduleName string) (*genPg.Snippet, model.ID, error) {
	ectx := NewExecContext(ctx, deps)
	pb := pageBuilderFromExecCtx(ectx, moduleID, moduleName, true, nil)
	genSnippet, err := pb.buildSnippetV3(s)
	return genSnippet, pb.lastContainerID, err
}

// pageBuilderFromExecCtx creates a pageBuilder from an ExecContext.
func pageBuilderFromExecCtx(ectx *ExecContext, moduleID model.ID, moduleName string, isSnippet bool, fns map[string]*ast.DefineFragmentStmt) *pageBuilder {
	if fns == nil {
		fns = ectx.Fragments
	}
	return &pageBuilder{
		moduleLister:         ectx.ModuleLister,
		domainModelReader:    ectx.DomainModelReader,
		pageReader:           ectx.PageReader,
		metadataReader:       ectx.MetadataReader,
		folderManager:        ectx.FolderManager,
		connectionManager:    ectx.ConnectionManager,
		serializationBackend: ectx.Backend,
		moduleID:             moduleID,
		moduleName:           moduleName,
		widgetScope:          make(map[string]model.ID),
		paramScope:           make(map[string]model.ID),
		paramEntityNames:     make(map[string]string),
		execCache:            ectx.Cache,
		isSnippet:            isSnippet,
		fragments:            fns,
		themeRegistry:        ectx.GetThemeRegistry(),
		widgetBackend:        ectx.Backend,
		microflowsRepo:       ectx.Microflows,
		nanoflowsRepo:        ectx.Nanoflows,
		layoutsRepo:          ectx.Layouts,
		snippetsRepo:         ectx.Snippets,
		mxGraph:              mxGraphFromExecCtx(ectx),
	}
}

// mxGraphFromExecCtx safely extracts the mxGraph from an ExecContext.
func mxGraphFromExecCtx(ectx *ExecContext) *mxgraph.Graph {
	if ectx.Graph != nil {
		return ectx.Graph.MxGraph()
	}
	return nil
}
