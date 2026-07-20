// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// BuildPageV3WithDeps creates a pageBuilder from HandlerDeps and builds a page.
// Exported so the page subpackage can wire its BuildPage function field.
func BuildPageV3WithDeps(ctx context.Context, deps *HandlerDeps, s *ast.CreatePageStmtV3, moduleID model.ID, moduleName string) (*genPg.Page, model.ID, error) {
	pb := pageBuilderFromDeps(deps, moduleID, moduleName, false, nil)
	genPage, err := pb.buildPageV3(s)
	return genPage, pb.lastContainerID, err
}

// BuildSnippetV3WithDeps creates a pageBuilder from HandlerDeps and builds a snippet.
// Exported so the page subpackage can wire its BuildSnippet function field.
func BuildSnippetV3WithDeps(ctx context.Context, deps *HandlerDeps, s *ast.CreateSnippetStmtV3, moduleID model.ID, moduleName string) (*genPg.Snippet, model.ID, error) {
	pb := pageBuilderFromDeps(deps, moduleID, moduleName, true, nil)
	genSnippet, err := pb.buildSnippetV3(s)
	return genSnippet, pb.lastContainerID, err
}

// pageBuilderFromDeps creates a pageBuilder from HandlerDeps.
func pageBuilderFromDeps(deps *HandlerDeps, moduleID model.ID, moduleName string, isSnippet bool, fns map[string]*ast.DefineFragmentStmt) *pageBuilder {
	if fns == nil {
		fns = deps.Fragments
	}
	// Type-assert the backend interfaces from available deps. In production,
	// the concrete backend implements all role interfaces; assertions always
	// succeed. Uses ConnectionManager as the union backend.
	var serializationBackend backend.WidgetSerializationBackend
	if sb, ok := deps.ConnectionManager.(backend.WidgetSerializationBackend); ok {
		serializationBackend = sb
	}
	var widgetBackend backend.WidgetBuilderBackend
	if wb, ok := deps.ConnectionManager.(backend.WidgetBuilderBackend); ok {
		widgetBackend = wb
	}
	return &pageBuilder{
		moduleLister:         deps.ModuleLister,
		domainModelReader:    deps.DomainModelReader,
		pageReader:           deps.PageReader,
		metadataReader:       deps.MetadataReader,
		folderManager:        deps.FolderManager,
		connectionManager:    deps.ConnectionManager,
		serializationBackend: serializationBackend,
		moduleID:             moduleID,
		moduleName:           moduleName,
		widgetScope:          make(map[string]model.ID),
		paramScope:           make(map[string]model.ID),
		paramEntityNames:     make(map[string]string),
		execCache:            deps.Cache,
		isSnippet:            isSnippet,
		fragments:            fns,
		themeRegistry:        themeRegistryFromDeps(deps),
		widgetBackend:        widgetBackend,
		microflowsRepo:       deps.MicroflowRepo,
		nanoflowsRepo:        deps.NanoflowRepo,
		layoutsRepo:          deps.LayoutRepo,
		snippetsRepo:         deps.SnippetRepo,
		mxGraph:              mxGraphFromDeps(deps),
	}
}

// themeRegistryFromDeps safely extracts the ThemeRegistry from HandlerDeps.
func themeRegistryFromDeps(deps *HandlerDeps) *ThemeRegistry {
	return deps.ThemeRegistry
}

// mxGraphFromDeps safely extracts the mxGraph from HandlerDeps.
func mxGraphFromDeps(deps *HandlerDeps) *mxgraph.Graph {
	if deps.Graph != nil {
		return deps.Graph.MxGraph()
	}
	return nil
}
