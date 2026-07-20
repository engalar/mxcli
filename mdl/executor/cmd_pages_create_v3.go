// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"log"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// ============================================================================
// V3 Page Creation
// ============================================================================

// ExecCreatePageV3Deps is the HandlerDeps version of ExecCreatePageV3.
func ExecCreatePageV3Deps(ctx context.Context, s *ast.CreatePageStmtV3, deps *HandlerDeps) error {
	tmpCtx := newMinimalExecCtx(ctx, deps)
	return ExecCreatePageV3(tmpCtx, s)
}

// ExecCreatePageV3 handles CREATE PAGE statement with V3 syntax.
func ExecCreatePageV3(ctx *ExecContext, s *ast.CreatePageStmtV3) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	// Version pre-check: page parameters require 11.0+
	if len(s.Parameters) > 0 {
		if err := checkFeature(ctx, "pages", "page_parameters",
			"create page with parameters",
			"pass data via a non-persistent entity or microflow parameter instead"); err != nil {
			return err
		}
	}

	// Find or auto-create module
	module, err := findOrCreateModule(ctx, s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("find module %s", s.Name.Module), err)
	}
	moduleID := module.ID

	// Check if page already exists - collect ALL duplicates
	existingPgPairs, _ := listPagesWithContainerGen(ctx)
	var pagesToDelete []model.ID
	var existingAllowedRoles []string
	preserveAllowedRoles := false
	for _, pair := range existingPgPairs {
		p := pair.Elem
		modID := getModuleID(ctx, model.ID(pair.ContainerID))
		modName := getModuleName(ctx, modID)
		if modName == s.Name.Module && p.Name() == s.Name.Name {
			if !s.IsReplace && !s.IsModify && len(pagesToDelete) == 0 {
				return mdlerrors.NewAlreadyExists("page", s.Name.String())
			}
			if len(pagesToDelete) == 0 {
				// Preserve existing allowed roles (as qualified name strings)
				existingAllowedRoles = append(existingAllowedRoles, p.AllowedRolesQualifiedNames()...)
				preserveAllowedRoles = true
			}
			pagesToDelete = append(pagesToDelete, model.ID(p.ID()))
		}
	}

	// Build the page BEFORE deleting the old one (atomic: if build fails, old page is preserved)
	ctx.WidgetBuilder.BeginPageBuild()
	pb := &pageBuilder{
		moduleLister:         ctx.ModuleLister,
		domainModelReader:    ctx.DomainModelReader,
		pageReader:           ctx.PageReader,
		metadataReader:       ctx.MetadataReader,
		folderManager:        ctx.FolderManager,
		connectionManager:    ctx.ConnectionManager,
		serializationBackend: ctx.Backend,
		moduleID:             moduleID,
		moduleName:           s.Name.Module,
		widgetScope:          make(map[string]model.ID),
		paramScope:           make(map[string]model.ID),
		paramEntityNames:     make(map[string]string),
		execCache:            ctx.Cache,
		fragments:            ctx.Fragments,
		themeRegistry:        ctx.GetThemeRegistry(),
		widgetBackend:        ctx.Backend,
		microflowsRepo:       ctx.Microflows,
		nanoflowsRepo:        ctx.Nanoflows,
		snippetsRepo:         ctx.Snippets,
		mxGraph:              ctx.Graph.MxGraph(),
	}

	// buildPageV3 now returns *genPg.Page directly (Stage 3.3.5.Cat-B).
	// The builder sets pb.lastContainerID to the resolved folder/module ID.
	genPage, err := pb.buildPageV3(s)
	ctx.WidgetBuilder.EndPageBuild()
	if err != nil {
		return mdlerrors.NewBackend("build page", err)
	}
	containerID := pb.lastContainerID

	// Set allowed roles on the gen page
	if preserveAllowedRoles {
		genPage.SetAllowedRolesQualifiedNames(existingAllowedRoles)
	} else {
		defaultRoles := defaultDocumentAccessRoleQNames(ctx, module)
		if len(defaultRoles) > 0 {
			genPage.SetAllowedRolesQualifiedNames(defaultRoles)
		}
	}

	// Replace or create the page in the MPR
	if len(pagesToDelete) > 0 {
		// Reuse first existing page's UUID to avoid git delete+add (which crashes Studio Pro RevStatusCache)
		genPage.SetID(element.ID(pagesToDelete[0]))
		if err := ctx.PageWriter.UpdatePageGen(genPage); err != nil {
			return mdlerrors.NewBackend("update page", err)
		}
		// Delete any additional duplicates
		for _, id := range pagesToDelete[1:] {
			if err := ctx.PageWriter.DeletePageGen(id); err != nil {
				return mdlerrors.NewBackend("delete duplicate page", err)
			}
		}
	} else {
		if err := ctx.PageWriter.CreatePageGen(string(containerID), "Documents", genPage); err != nil {
			return mdlerrors.NewBackend("create page", err)
		}
	}

	// Task 8 wire-up: after the gen-typed page is in the MPR, overlay the
	// widget tree from the PageModel IR so subsequent describe roundtrips
	// stay stable. The gen builder seeds the unit with rich BSON; the IR
	// rewrites only the widget array inside LayoutCall.Arguments[].Widget.
	//
	// IMPORTANT: only overlay when the IR can FULLY reproduce the widget
	// tree. Pluggable widgets (DataGrid/Gallery/ComboBox/Image) and any
	// unknown widget have a lossy BSON encoding today (widgetToBSON in
	// mpr/page_model.go writes a minimal CustomWidget without the Object
	// property tree). Overlaying those would erase the builder's rich
	// columns/filters/datasource fields — see TestRoundtripPage_V3DataGrid*
	// regressions when this gate is removed.
	//
	// Failure is non-fatal — log and continue so the gen builder remains
	// source of truth when overlay isn't safe.
	if pm, pmErr := pageASTToModel(s, s.Name.Module); pmErr == nil && pm != nil {
		if pageModelHasLossyWidget(pm) {
			// Skip overlay; preserve builder's rich BSON.
		} else if werr := ctx.PageModelAccess.WritePageModel(model.ID(genPage.ID()), pm); werr != nil {
			log.Printf("warning: WritePageModel overlay failed for %s.%s: %v",
				s.Name.Module, s.Name.Name, werr)
		}
	} else if pmErr != nil {
		log.Printf("warning: pageASTToModel for %s.%s: %v",
			s.Name.Module, s.Name.Name, pmErr)
	}

	// Track the created page so it can be resolved by subsequent page references
	ctx.trackCreatedPage(s.Name.Module, s.Name.Name, model.ID(genPage.ID()), moduleID)

	// Invalidate cached page listing so subsequent findPageIDGen / grant page
	// calls see the newly created page (Bug A — stale pagesWithContainerGen).
	invalidatePagesGenCache(ctx)

	fmt.Fprintf(ctx.Output, "Created page %s\n", s.Name.String())
	return nil
}

// ExecCreateSnippetV3Deps is the HandlerDeps version of ExecCreateSnippetV3.
func ExecCreateSnippetV3Deps(ctx context.Context, s *ast.CreateSnippetStmtV3, deps *HandlerDeps) error {
	tmpCtx := newMinimalExecCtx(ctx, deps)
	return ExecCreateSnippetV3(tmpCtx, s)
}

// ExecCreateSnippetV3 handles CREATE SNIPPET statement with V3 syntax.
func ExecCreateSnippetV3(ctx *ExecContext, s *ast.CreateSnippetStmtV3) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	// Find or auto-create module
	module, err := findOrCreateModule(ctx, s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("find module %s", s.Name.Module), err)
	}
	moduleID := module.ID

	// Check if snippet already exists - collect ALL duplicates
	existingSnippetPairs, _ := listSnippetsWithContainerGen(ctx)
	var snippetsToDelete []model.ID
	for _, pair := range existingSnippetPairs {
		snip := pair.Elem
		modID := getModuleID(ctx, model.ID(pair.ContainerID))
		modName := getModuleName(ctx, modID)
		if modName == s.Name.Module && snip.Name() == s.Name.Name {
			if !s.IsReplace && !s.IsModify && len(snippetsToDelete) == 0 {
				return mdlerrors.NewAlreadyExists("snippet", s.Name.String())
			}
			snippetsToDelete = append(snippetsToDelete, model.ID(snip.ID()))
		}
	}

	// Build the snippet BEFORE deleting the old one (atomic: if build fails, old snippet is preserved)
	ctx.WidgetBuilder.BeginPageBuild()
	pb := &pageBuilder{
		moduleLister:         ctx.ModuleLister,
		domainModelReader:    ctx.DomainModelReader,
		pageReader:           ctx.PageReader,
		metadataReader:       ctx.MetadataReader,
		folderManager:        ctx.FolderManager,
		connectionManager:    ctx.ConnectionManager,
		serializationBackend: ctx.Backend,
		moduleID:             moduleID,
		moduleName:           s.Name.Module,
		widgetScope:          make(map[string]model.ID),
		paramScope:           make(map[string]model.ID),
		paramEntityNames:     make(map[string]string),
		execCache:            ctx.Cache,
		fragments:            ctx.Fragments,
		themeRegistry:        ctx.GetThemeRegistry(),
		widgetBackend:        ctx.Backend,
		microflowsRepo:       ctx.Microflows,
		nanoflowsRepo:        ctx.Nanoflows,
		snippetsRepo:         ctx.Snippets,
	}

	// buildSnippetV3 now returns *genPg.Snippet directly (Stage 3.3.5.Cat-B).
	// The builder sets pb.lastContainerID to the resolved folder/module ID.
	genSnippet, err := pb.buildSnippetV3(s)
	ctx.WidgetBuilder.EndPageBuild()
	if err != nil {
		return mdlerrors.NewBackend("build snippet", err)
	}
	containerID := pb.lastContainerID

	// Delete old snippets only after successful build
	for _, id := range snippetsToDelete {
		if err := ctx.PageWriter.DeleteSnippetGen(id); err != nil {
			return mdlerrors.NewBackend("delete existing snippet", err)
		}
	}

	// Create the snippet in the MPR
	if err := ctx.PageWriter.CreateSnippetGen(string(containerID), "Documents", genSnippet); err != nil {
		return mdlerrors.NewBackend("create snippet", err)
	}

	// Track the created snippet so it can be resolved by subsequent snippet references
	ctx.trackCreatedSnippet(s.Name.Module, s.Name.Name, model.ID(genSnippet.ID()), moduleID)

	// Invalidate caches so subsequent findSnippetIDGen calls see the new snippet
	invalidateHierarchy(ctx)
	invalidatePagesGenCache(ctx)

	fmt.Fprintf(ctx.Output, "Created snippet %s\n", s.Name.String())
	return nil
}

// pageModelHasLossyWidget returns true if any widget in the tree maps to a
// BSON encoding that widgetToBSON cannot fully reproduce. Used to gate the
// Task 8 write-path overlay so the gen builder's rich BSON isn't clobbered
// by the slim IR encoding.
func pageModelHasLossyWidget(pm *types.PageModel) bool {
	if pm == nil {
		return false
	}
	for _, n := range pm.Widgets {
		if widgetTreeHasLossyKind(n) {
			return true
		}
	}
	return false
}

func widgetTreeHasLossyKind(n *types.WidgetNode) bool {
	if n == nil {
		return false
	}
	switch n.Kind {
	case types.WidgetDataGrid, types.WidgetGallery, types.WidgetComboBox,
		types.WidgetImage, types.WidgetUnknown:
		return true
	case types.WidgetDataView, types.WidgetListView:
		// DataView/ListView DataSource is a complex nested gen structure
		// (Forms$DataViewSource with SourceVariable+PageVariable) that
		// the IR's dataSourceToBSON cannot reproduce — overlay would
		// erase entity context. Gate to the legacy builder.
		return true
	case types.WidgetButton:
		// Button actions (microflow/nanoflow with parameter mappings) aren't
		// fully represented by the IR's bare OnClick string today — the
		// existing TestRoundtripPage_MicroflowButton* expects the legacy
		// describe format with `Product: $Product` parameter mappings.
		// Treat all buttons as lossy until the IR captures Action+
		// ParameterMappings; the gen builder's rich Action BSON then
		// survives the overlay and legacy describe renders the call.
		return true
	case types.WidgetCheckBox, types.WidgetRadioButtons:
		// renderWidget currently has no dedicated case; legacy path handles
		// these via outputWidgetMDLV3.
		return true
	case types.WidgetScrollView:
		// ScrollContainer children live in CenterRegion.Widgets, not the
		// top-level Widgets field — fromBSON extracts from the wrong field.
		// Fall back to legacy describe until CenterRegion is handled.
		return true
	}
	for _, c := range n.Children {
		if widgetTreeHasLossyKind(c) {
			return true
		}
	}
	if n.DataGrid != nil {
		for _, cw := range n.DataGrid.FilterWidgets {
			if widgetTreeHasLossyKind(cw) {
				return true
			}
		}
		for _, cw := range n.DataGrid.ControlBar {
			if widgetTreeHasLossyKind(cw) {
				return true
			}
		}
	}
	if n.Gallery != nil {
		for _, cw := range n.Gallery.FilterWidgets {
			if widgetTreeHasLossyKind(cw) {
				return true
			}
		}
		for _, cw := range n.Gallery.ContentWidgets {
			if widgetTreeHasLossyKind(cw) {
				return true
			}
		}
	}
	return false
}
