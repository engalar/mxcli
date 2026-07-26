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
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
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
	sc.SetLayoutMode("Headline")
	sc.SetWidth(int32(960))
	sc.SetWidthMode("Auto")
	sc.SetAlignment("Center")
	sc.SetScrollBehavior("PerRegion")
	sc.SetNativeHideScrollbars(false)
	sc.SetTabIndex(int32(0))

	appearance := genPg.NewAppearance()
	sc.SetAppearance(appearance)

	for _, region := range w.Regions {
		r := genPg.NewScrollContainerRegion()
		switch strings.ToLower(region.Name) {
		case "center":
			ap := genPg.NewAppearance()
			ap.SetClass("region-content")
			r.SetAppearance(ap)
			r.SetSize(int32(200))
			r.SetSizeMode("Auto")
			r.SetToggleMode("None")
		case "left", "right":
			ap := genPg.NewAppearance()
			ap.SetClass("region-sidebar")
			r.SetAppearance(ap)
			r.SetSize(int32(232))
			r.SetSizeMode("Pixels")
			r.SetToggleMode("None")
		case "top":
			ap := genPg.NewAppearance()
			ap.SetClass("region-topbar")
			r.SetAppearance(ap)
			r.SetSize(int32(200))
			r.SetSizeMode("Auto")
			r.SetToggleMode("None")
		}
		for _, ph := range region.Placeholders {
			p := genPg.NewPlaceholder()
			p.SetName(ph.Name)
			r.AddWidgets(p)
		}
		for _, w := range region.Widgets {
			widget := buildWidgetForLayout(w)
			if widget != nil {
				r.AddWidgets(widget)
			}
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

	return mprrepos.FixScrollContainerBSON(sc)
}

// buildWidgetForLayout converts an AST widget to a genPg element for use in layout regions.
// Only supports widget types needed for sidebar navigation.
func buildWidgetForLayout(w *ast.WidgetV3) element.Element {
	switch strings.ToLower(w.Type) {
	case "layoutgrid":
		return buildLayoutGridForLayout(w)
	case "button", "actionbutton":
		return buildActionButtonForLayout(w)
	default:
		return nil
	}
}

func buildLayoutGridForLayout(w *ast.WidgetV3) element.Element {
	lg := genPg.NewLayoutGrid()
	assignFreshID(lg)
	lg.SetName(w.Name)
	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "row" {
			row := buildLayoutGridRowForLayout(child)
			if row != nil {
				lg.AddRows(row)
			}
		}
	}
	return lg
}

func buildLayoutGridRowForLayout(w *ast.WidgetV3) element.Element {
	row := genPg.NewLayoutGridRow()
	assignFreshID(row)
	if row.Appearance() == nil {
		row.SetAppearance(genPg.NewAppearance())
	}
	row.SetConditionalVisibilitySettings(nil)
	row.SetHorizontalAlignment("None")
	row.SetVerticalAlignment("None")
	row.SetSpacingBetweenColumns(true)
	if w.Name != "" {
		row.SetClass("_mdlRow:" + w.Name)
	}
	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "column" {
			col := buildLayoutGridColumnForLayout(child)
			if col != nil {
				row.AddColumns(col)
			}
		}
	}
	return row
}

func buildLayoutGridColumnForLayout(w *ast.WidgetV3) element.Element {
	col := genPg.NewLayoutGridColumn()
	assignFreshID(col)
	if col.Appearance() == nil {
		col.SetAppearance(genPg.NewAppearance())
	}
	col.SetVerticalAlignment("None")
	if w.Name != "" {
		col.SetClass("_mdlCol:" + w.Name)
	}
	col.SetWeight(-1)
	col.SetTabletWeight(-1)
	col.SetPhoneWeight(-1)
	col.SetPreviewWidth(-1)
	dw := w.GetDesktopWidth()
	if dw != nil {
		if v, ok := dw.(int); ok {
			col.SetWeight(int32(v))
		}
	}
	for _, child := range w.Children {
		widget := buildWidgetForLayout(child)
		if widget != nil {
			col.AddWidgets(widget)
		}
	}
	return col
}

func buildActionButtonForLayout(w *ast.WidgetV3) element.Element {
	btn := genPg.NewActionButton()
	assignFreshID(btn)
	btn.SetName(w.Name)
	btn.SetButtonStyle("Link")
	if caption := w.GetCaption(); caption != "" {
		ct := genPg.NewClientTemplate()
		assignFreshID(ct)
		ct.SetTemplate(genSimpleText(caption))
		fallback := genTexts.NewText()
		assignFreshID(fallback)
		ct.SetFallback(fallback)
		btn.SetCaption(ct)
	}
	if action := w.GetAction(); action != nil {
		act := buildActionForLayout(action)
		if act != nil {
			btn.SetAction(act)
		}
	}
	return btn
}

func buildActionForLayout(action *ast.ActionV3) element.Element {
	switch action.Type {
	case "showPage":
		act := genPg.NewPageClientAction()
		assignFreshID(act)
		ps := genPg.NewPageSettings()
		assignFreshID(ps)
		ps.SetPageQualifiedName(action.Target)
		act.SetPageSettings(ps)
		act.SetDisabledDuringExecution(false)
		act.SetNumberOfPagesToClose2("1")
		return act
	default:
		return nil
	}
}
