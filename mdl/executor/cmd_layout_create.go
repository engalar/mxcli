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

func execCreateOrModifyLayoutFn(ctx context.Context, s *ast.CreateLayoutStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	return execCreateOrModifyLayoutImpl(ectx, s)
}

func execCreateOrModifyLayoutImpl(ctx *ExecContext, s *ast.CreateLayoutStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := findOrCreateModule(ctx, s.Name.Module)
	if err != nil {
		return mdlerrors.NewBackend(fmt.Sprintf("find module %s", s.Name.Module), err)
	}

	existingPairs, _ := listLayoutsWithContainerGen(ctx)
	var existingID model.ID
	for _, pair := range existingPairs {
		modID := getModuleID(ctx, model.ID(pair.ContainerID))
		modName := getModuleName(ctx, modID)
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
		if err := ctx.PageWriter.DeleteLayoutGen(existingID); err != nil {
			return mdlerrors.NewBackend("delete existing layout", err)
		}
	}

	containerID, err := ctx.PageWriter.GetContainerID(module.ID, s.Folder)
	if err != nil {
		return mdlerrors.NewBackend("resolve container", err)
	}

	if err := ctx.PageWriter.CreateLayoutGen(string(containerID), "Documents", layout); err != nil {
		return mdlerrors.NewBackend("create layout", err)
	}

	invalidateHierarchy(ctx)
	invalidatePagesGenCache(ctx)

	verb := "Created"
	if existingID != "" {
		verb = "Modified"
	}
	fmt.Fprintf(ctx.Output, "%s layout %s\n", verb, s.Name.String())
	return nil
}

func execCreateOrModifyLayout(ctx *ExecContext, s *ast.CreateLayoutStmt) error {
	return execCreateOrModifyLayoutFn(ctx, s, execContextToDeps(ctx))
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
