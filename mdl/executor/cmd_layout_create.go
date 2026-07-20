// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

const (
	layoutCanvasWidth  int32 = 1198
	layoutCanvasHeight int32 = 600
)

func ExecCreateOrModifyLayoutFn(ctx context.Context, s *ast.CreateLayoutStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	return execCreateOrModifyLayoutImplDeps(ctx, deps, s)
}

func execCreateOrModifyLayoutImplDeps(ctx context.Context, deps *HandlerDeps, s *ast.CreateLayoutStmt) error {
	module, err := findOrCreateModuleDeps(ctx, deps, s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("find module %s", s.Name.Module), err)
	}

	h, err := getHierarchyDeps(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	existingPairs, _ := listLayoutsWithContainerGenDeps(deps)
	var existingID model.ID
	for _, pair := range existingPairs {
		modID := h.FindModuleID(model.ID(pair.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && pair.Elem.Name() == s.Name.Name {
			if !s.IsModify && !s.IsReplace {
				return mdlerrors.NewAlreadyExists("layout", s.Name.String())
			}
			existingID = model.ID(pair.Elem.ID())
			break
		}
	}

	layout := genPg.NewLayout()
	layout.SetName(s.Name.Name)
	if s.Documentation != "" {
		layout.SetDocumentation(s.Documentation)
	}
	layout.SetCanvasWidth(layoutCanvasWidth)
	layout.SetCanvasHeight(layoutCanvasHeight)
	layout.SetContent(buildLayoutContent(s))

	if existingID != "" {
		if err := deps.PageWriter.DeleteLayoutGen(existingID); err != nil {
			return mdlerrors.NewBackend("delete existing layout", err)
		}
	}

	containerID, err := deps.PageWriter.GetContainerID(module.ID, s.Folder)
	if err != nil {
		return mdlerrors.NewBackend("resolve container", err)
	}

	if err := deps.PageWriter.CreateLayoutGen(string(containerID), "Documents", layout); err != nil {
		return mdlerrors.NewBackend("create layout", err)
	}

	invalidateHierarchyDeps(deps)
	invalidatePagesGenCacheDeps(deps)

	verb := "Created"
	if existingID != "" {
		verb = "Modified"
	}
	fmt.Fprintf(deps.Output, "%s layout %s\n", verb, s.Name.String())
	return nil
}



func buildLayoutContent(s *ast.CreateLayoutStmt) element.Element {
	if strings.HasPrefix(strings.ToLower(s.LayoutType), "native") {
		return buildNativeLayoutContent(s)
	}
	return buildWebLayoutContent(s)
}

func buildWebLayoutContent(s *ast.CreateLayoutStmt) *genPg.WebLayoutContent {
	content := genPg.NewWebLayoutContent()
	lt := s.LayoutType
	if lt == "" {
		lt = "Responsive"
	}
	content.SetLayoutType(lt)

	for _, w := range s.Widgets {
		switch w.Kind {
		case ast.LayoutWidgetScrollContainer:
			content.AddWidgets(buildScrollContainer(w))
		case ast.LayoutWidgetPlaceholder:
			ph := genPg.NewPlaceholder()
			ph.SetName(w.PlaceholderName)
			content.AddWidgets(ph)
		}
	}
	return content
}

func buildNativeLayoutContent(s *ast.CreateLayoutStmt) *genPg.NativeLayoutContent {
	content := genPg.NewNativeLayoutContent()
	lt := s.LayoutType
	if lt == "" {
		lt = "Default"
	}
	content.SetLayoutType(lt)

	for _, w := range s.Widgets {
		if w.Kind == ast.LayoutWidgetPlaceholder {
			ph := genPg.NewPlaceholder()
			ph.SetName(w.PlaceholderName)
			content.AddWidgets(ph)
		}
	}
	return content
}

func buildScrollContainer(w *ast.LayoutWidgetV3) *genPg.ScrollContainer {
	sc := genPg.NewScrollContainer()
	sc.SetName(w.Name)

	for _, region := range w.Regions {
		r := genPg.NewScrollContainerRegion()
		for _, ph := range region.Placeholders {
			p := genPg.NewPlaceholder()
			p.SetName(ph.Name)
			r.AddWidgets(p)
		}
		switch strings.ToLower(region.Name) {
		case "center":
			sc.SetCenter(r)
		case "top":
			sc.SetTop(r)
		case "bottom":
			sc.SetBottom(r)
		case "left":
			sc.SetLeft(r)
		case "right":
			sc.SetRight(r)
		}
	}
	return sc
}
