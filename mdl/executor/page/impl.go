// SPDX-License-Identifier: Apache-2.0

package page

import (
	"context"
	"fmt"
	"log"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ExecCreatePageFn handles CREATE PAGE statement with V3 syntax.
func ExecCreatePageFn(ctx context.Context, s *ast.CreatePageStmtV3, d PageDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	if len(s.Parameters) > 0 {
		if err := d.CheckFeature("pages", "page_parameters",
			"create page with parameters",
			"pass data via a non-persistent entity or microflow parameter instead"); err != nil {
			return err
		}
	}

	module, err := d.FindOrCreateModule(s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("find module %s", s.Name.Module), err)
	}
	moduleID := module.ID

	existingPgPairs, _ := d.PagesRepo.ListAll()
	var pagesToDelete []model.ID
	var existingAllowedRoles []string
	preserveAllowedRoles := false
	for _, p := range existingPgPairs {
		containerID, cerr := d.PagesRepo.GetContainerUUID(model.ID(p.ID()))
		if cerr != nil {
			continue
		}
		modID := d.GetModuleID(containerID)
		modName := d.GetModuleName(modID)
		if modName == s.Name.Module && p.Name() == s.Name.Name {
			if !s.IsReplace && !s.IsModify && len(pagesToDelete) == 0 {
				return mdlerrors.NewAlreadyExists("page", s.Name.String())
			}
			if len(pagesToDelete) == 0 {
				existingAllowedRoles = append(existingAllowedRoles, p.AllowedRolesQualifiedNames()...)
				preserveAllowedRoles = true
			}
			pagesToDelete = append(pagesToDelete, model.ID(p.ID()))
		}
	}

	d.WidgetBuilder.BeginPageBuild()
	genPage, containerID, err := d.BuildPage(s, moduleID, s.Name.Module)
	d.WidgetBuilder.EndPageBuild()
	if err != nil {
		return mdlerrors.NewBackend("build page", err)
	}

	if preserveAllowedRoles {
		genPage.SetAllowedRolesQualifiedNames(existingAllowedRoles)
	} else {
		defaultRoles := d.DefaultDocumentAccessRoleQNames(module)
		if len(defaultRoles) > 0 {
			genPage.SetAllowedRolesQualifiedNames(defaultRoles)
		}
	}

	if len(pagesToDelete) > 0 {
		genPage.SetID(element.ID(pagesToDelete[0]))
		if err := d.PageWriter.UpdatePageGen(genPage); err != nil {
			return mdlerrors.NewBackend("update page", err)
		}
		for _, id := range pagesToDelete[1:] {
			if err := d.PageWriter.DeletePageGen(id); err != nil {
				return mdlerrors.NewBackend("delete duplicate page", err)
			}
		}
	} else {
		if err := d.PageWriter.CreatePageGen(string(containerID), "Documents", genPage); err != nil {
			return mdlerrors.NewBackend("create page", err)
		}
	}

	if pm, pmErr := d.PageASTToModel(s, s.Name.Module); pmErr == nil && pm != nil {
		if d.PageModelHasLossyWidget(pm) {
		} else if werr := d.PageModelAccess.WritePageModel(model.ID(genPage.ID()), pm); werr != nil {
			log.Printf("warning: WritePageModel overlay failed for %s.%s: %v",
				s.Name.Module, s.Name.Name, werr)
		}
	} else if pmErr != nil {
		log.Printf("warning: pageASTToModel for %s.%s: %v",
			s.Name.Module, s.Name.Name, pmErr)
	}

	d.TrackCreatedPage(s.Name.Module, s.Name.Name, model.ID(genPage.ID()), moduleID)
	d.InvalidatePagesGenCache()

	fmt.Fprintf(d.Output, "Created page %s\n", s.Name.String())
	return nil
}

// ExecCreateSnippetFn handles CREATE SNIPPET statement with V3 syntax.
func ExecCreateSnippetFn(ctx context.Context, s *ast.CreateSnippetStmtV3, d PageDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := d.FindOrCreateModule(s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("find module %s", s.Name.Module), err)
	}
	moduleID := module.ID

	existingSnippetPairs, _ := d.SnippetsRepo.ListAll()
	var snippetsToDelete []model.ID
	for _, snip := range existingSnippetPairs {
		containerID, cerr := d.SnippetsRepo.GetContainerUUID(model.ID(snip.ID()))
		if cerr != nil {
			continue
		}
		modID := d.GetModuleID(containerID)
		modName := d.GetModuleName(modID)
		if modName == s.Name.Module && snip.Name() == s.Name.Name {
			if !s.IsReplace && !s.IsModify && len(snippetsToDelete) == 0 {
				return mdlerrors.NewAlreadyExists("snippet", s.Name.String())
			}
			snippetsToDelete = append(snippetsToDelete, model.ID(snip.ID()))
		}
	}

	d.WidgetBuilder.BeginPageBuild()
	genSnippet, containerID, err := d.BuildSnippet(s, moduleID, s.Name.Module)
	d.WidgetBuilder.EndPageBuild()
	if err != nil {
		return mdlerrors.NewBackend("build snippet", err)
	}

	for _, id := range snippetsToDelete {
		if err := d.PageWriter.DeleteSnippetGen(id); err != nil {
			return mdlerrors.NewBackend("delete existing snippet", err)
		}
	}

	if err := d.PageWriter.CreateSnippetGen(string(containerID), "Documents", genSnippet); err != nil {
		return mdlerrors.NewBackend("create snippet", err)
	}

	d.TrackCreatedSnippet(s.Name.Module, s.Name.Name, model.ID(genSnippet.ID()), moduleID)
	d.InvalidateHierarchy()
	d.InvalidatePagesGenCache()

	fmt.Fprintf(d.Output, "Created snippet %s\n", s.Name.String())
	return nil
}

// ExecDropPageFn handles DROP PAGE.
func ExecDropPageFn(ctx context.Context, s *ast.DropPageStmt, d PageDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	pages, err := d.PagesRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}
	for _, p := range pages {
		containerID, cerr := d.PagesRepo.GetContainerUUID(model.ID(p.ID()))
		if cerr != nil {
			continue
		}
		modID := d.GetModuleID(containerID)
		modName := d.GetModuleName(modID)
		if modName == s.Name.Module && p.Name() == s.Name.Name {
			if err := d.PageWriter.DeletePageGen(model.ID(p.ID())); err != nil {
				return mdlerrors.NewBackend("delete page", err)
			}
			d.InvalidatePagesGenCache()
			fmt.Fprintf(d.Output, "Dropped page %s\n", s.Name.String())
			return nil
		}
	}
	return mdlerrors.NewNotFound("page", s.Name.Module+"."+s.Name.Name)
}

// ExecDropSnippetFn handles DROP SNIPPET.
func ExecDropSnippetFn(ctx context.Context, s *ast.DropSnippetStmt, d PageDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	snippets, err := d.SnippetsRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list snippets", err)
	}
	for _, snip := range snippets {
		containerID, cerr := d.SnippetsRepo.GetContainerUUID(model.ID(snip.ID()))
		if cerr != nil {
			continue
		}
		modID := d.GetModuleID(containerID)
		modName := d.GetModuleName(modID)
		if modName == s.Name.Module && snip.Name() == s.Name.Name {
			if err := d.PageWriter.DeleteSnippetGen(model.ID(snip.ID())); err != nil {
				return mdlerrors.NewBackend("delete snippet", err)
			}
			d.InvalidatePagesGenCache()
			fmt.Fprintf(d.Output, "Dropped snippet %s\n", s.Name.String())
			return nil
		}
	}
	return mdlerrors.NewNotFound("snippet", s.Name.Module+"."+s.Name.Name)
}

// ExecCreateLayoutFn handles CREATE [OR MODIFY] LAYOUT.
func ExecCreateLayoutFn(ctx context.Context, s *ast.CreateLayoutStmt, d PageDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := d.FindOrCreateModule(s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("find module %s", s.Name.Module), err)
	}

	existingPairs, _ := d.LayoutsRepo.ListAll()
	var existingID model.ID
	for _, lay := range existingPairs {
		containerID, cerr := d.LayoutsRepo.GetContainerUUID(model.ID(lay.ID()))
		if cerr != nil {
			continue
		}
		modID := d.GetModuleID(containerID)
		modName := d.GetModuleName(modID)
		if modName == s.Name.Module && lay.Name() == s.Name.Name {
			if !s.IsModify && !s.IsReplace {
				return mdlerrors.NewAlreadyExists("layout", s.Name.String())
			}
			existingID = model.ID(lay.ID())
			break
		}
	}

	layout := genPg.NewLayout()
	layout.SetName(s.Name.Name)
	if s.Documentation != "" {
		layout.SetDocumentation(s.Documentation)
	}
	layout.SetCanvasWidth(1198)
	layout.SetCanvasHeight(600)
	layout.SetContent(d.BuildLayoutContent(s))

	if existingID != "" {
		if err := d.PageWriter.DeleteLayoutGen(existingID); err != nil {
			return mdlerrors.NewBackend("delete existing layout", err)
		}
	}

	containerID, err := d.PageWriter.GetContainerID(module.ID, s.Folder)
	if err != nil {
		return mdlerrors.NewBackend("resolve container", err)
	}

	if err := d.PageWriter.CreateLayoutGen(string(containerID), "Documents", layout); err != nil {
		return mdlerrors.NewBackend("create layout", err)
	}

	d.InvalidateHierarchy()
	d.InvalidatePagesGenCache()
	verb := "Created"
	if existingID != "" {
		verb = "Modified"
	}
	fmt.Fprintf(d.Output, "%s layout %s\n", verb, s.Name.String())
	return nil
}
