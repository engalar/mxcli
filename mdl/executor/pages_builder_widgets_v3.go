// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/bsonutil"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// buildWidgetV3 converts a V3 AST widget to an element.Element.
//
// Keyword dispatch (Phase 2 — issue #539): the keywordDispatchTable encodes
// our editorial policy for dual-stack keywords (e.g. DATAGRID → pluggable
// Datagrid 2.x). Today the existing switch cases handle this correctly via
// the hand-coded builders (buildDataGridV3 already produces pluggable BSON).
// The dispatch table is consumed by inspection commands and DESCRIBE-side
// keyword resolution rather than overriding write-side routing here.
func (pb *pageBuilder) buildWidgetV3(w *ast.WidgetV3) (element.Element, error) {
	typeLower := strings.ToLower(w.Type)

	// Explicit error cases: not valid as standalone widgets.
	switch typeLower {
	case "legacydatagrid":
		return nil, mdlerrors.NewUnsupported(
			"LEGACYDATAGRID (native Forms$DataGrid) is not yet implemented. " +
				"Use DATAGRID for the pluggable equivalent on Mendix 11+, " +
				"or open the project in Studio Pro to add native datagrids manually.")
	case "tabpage":
		return nil, mdlerrors.NewValidation("tabpage must be a direct child of tabcontainer")
	case "item":
		return nil, mdlerrors.NewValidation("item must be a direct child of navigationlist")
	}

	// Registered built-in widget types.
	if fn, ok := widgetBuilders[typeLower]; ok {
		widget, err := fn(pb, w)
		if err != nil {
			return nil, err
		}
		applyWidgetAppearanceGen(widget, w, pb.themeRegistry)
		applyConditionalSettingsGen(widget, w)
		applyWidgetDefaults(widget)
		return widget, nil
	}

	// Pluggable widget fallback: look up by type ID in the registry.
	pb.initPluggableEngine()
	if pb.widgetRegistry != nil {
		if def, ok := pb.widgetRegistry.Get(strings.ToUpper(w.Type)); ok {
			cw, err := pb.pluggableEngine.Build(def, w)
			if err != nil {
				return nil, err
			}
			return pb.customWidgetToElement(cw)
		}
		if w.Type == "pluggablewidget" || w.Type == "customwidget" {
			if widgetType, ok := w.Properties["WidgetType"].(string); ok { // nolint:describe-raw-bson // migrated verbatim from former default switch case; AST property map, not BSON
				if def, ok := pb.widgetRegistry.GetByWidgetID(widgetType); ok {
					cw, err := pb.pluggableEngine.Build(def, w)
					if err != nil {
						return nil, err
					}
					return pb.customWidgetToElement(cw)
				}
				return nil, mdlerrors.NewNotFoundMsg("widget", widgetType,
					"no definition for widget "+widgetType+" (run 'mxcli widget init -p app.mpr')")
			}
		}
	}
	if pb.pluggableEngineErr != nil {
		return nil, mdlerrors.NewUnsupported(fmt.Sprintf("unsupported widget type: %s (%v)", w.Type, pb.pluggableEngineErr))
	}
	return nil, mdlerrors.NewUnsupported("unsupported widget type: " + w.Type)
}

// customWidgetToElement returns the element.Element from a *backend.GenCustomWidgetElem.
// Build() now returns *GenCustomWidgetElem which implements element.Element directly.
func (pb *pageBuilder) customWidgetToElement(cw *backend.GenCustomWidgetElem) (element.Element, error) {
	if cw == nil {
		return nil, fmt.Errorf("customWidgetToElement: nil GenCustomWidgetElem")
	}
	elem := cw.AsElement()
	applyWidgetDefaults(elem)
	return elem, nil
}

// applyConditionalSettingsGen sets ConditionalVisibilitySettings and
// ConditionalEditabilitySettings on a gen widget element.
func applyConditionalSettingsGen(widget element.Element, w *ast.WidgetV3) {
	if visibleIf := w.GetStringProp("VisibleIf"); visibleIf != "" {
		cvs := genPg.NewConditionalVisibilitySettings()
		assignFreshID(cvs)
		cvs.SetExpression(visibleIf)
		type cvsSetter interface {
			SetConditionalVisibilitySettings(element.Element)
		}
		if s, ok := widget.(cvsSetter); ok {
			s.SetConditionalVisibilitySettings(cvs)
		}
	}

	if editableIf := w.GetStringProp("EditableIf"); editableIf != "" {
		ces := genPg.NewConditionalEditabilitySettings()
		assignFreshID(ces)
		ces.SetExpression(editableIf)
		type cesSetter interface {
			SetConditionalEditabilitySettings(element.Element)
		}
		if s, ok := widget.(cesSetter); ok {
			s.SetConditionalEditabilitySettings(ces)
		}
	}
}

// applyWidgetDefaults sets common default values on any widget.
func applyWidgetDefaults(w element.Element) {
	type hasCondVis interface{ ConditionalVisibilitySettings() element.Element }
	type setCondVis interface{ SetConditionalVisibilitySettings(element.Element) }
	if w, ok := w.(setCondVis); ok {
		if cv, ok2 := w.(hasCondVis); !ok2 || cv.ConditionalVisibilitySettings() == nil {
			w.SetConditionalVisibilitySettings(nil)
		}
	}
	type setTabIdx interface{ SetTabIndex(int32) }
	if w, ok := w.(setTabIdx); ok {
		w.SetTabIndex(0)
	}
	type hasApp interface{ Appearance() element.Element }
	type setApp interface{ SetAppearance(element.Element) }
	if w, ok := w.(setApp); ok {
		if ha, ok2 := w.(hasApp); ok2 && ha.Appearance() == nil {
			w.SetAppearance(newDefaultAppearance())
		}
	}
}

// applyWidgetAppearanceGen sets Class, Style, and DesignProperties on a gen widget.
// Uses the gen Appearance sub-element for design properties.
func applyWidgetAppearanceGen(widget element.Element, w *ast.WidgetV3, theme *ThemeRegistry) {
	class, style := w.GetClass(), w.GetStyle()
	astProps := w.GetDesignProperties()

	// Always ensure every widget that supports Appearance has a Forms$Appearance in its
	// BSON — Studio Pro 11.6.6 requires it on every widget or the widget is invisible.
	// Input widgets (TextBox, TextArea, …) already have Appearance set via
	// applyFormWidgetDefaults; for container widgets (DataView, DivContainer, …) we
	// ensure it here if not yet populated.
	type appearanceGetSetter interface {
		Appearance() element.Element
		SetAppearance(element.Element)
	}
	if as, ok := widget.(appearanceGetSetter); ok && as.Appearance() == nil {
		as.SetAppearance(newDefaultAppearance())
	}

	// If no explicit styling, the default Appearance above is sufficient.
	if class == "" && style == "" && len(astProps) == 0 {
		return
	}

	// Apply class/style directly on the widget (most gen widgets have these setters)
	if class != "" {
		type classSetter interface{ SetClass(string) }
		if s, ok := widget.(classSetter); ok {
			s.SetClass(class)
		}
	}
	if style != "" {
		type styleSetter interface{ SetStyle(string) }
		if s, ok := widget.(styleSetter); ok {
			s.SetStyle(style)
		}
	}

	// Apply design properties via Appearance element
	if len(astProps) > 0 {
		appearance := newDefaultAppearance()
		if class != "" {
			appearance.SetClass(class)
		}
		if style != "" {
			appearance.SetStyle(style)
		}

		for _, p := range astProps {
			switch strings.ToLower(p.Value) {
			case "on":
				appearance.AddDesignProperties(newDesignPropertyEntry(p.Key, genPg.NewToggleDesignPropertyValue()))
			case "off":
				// OFF means toggle absence — skip
			default:
				opt := genPg.NewOptionDesignPropertyValue()
				opt.SetOption(p.Value)
				appearance.AddDesignProperties(newDesignPropertyEntry(p.Key, opt))
			}
		}

		type appearanceSetter interface {
			SetAppearance(element.Element)
		}
		if s, ok := widget.(appearanceSetter); ok {
			s.SetAppearance(appearance)
		}
	}
}

// =============================================================================
// V3 Client Action Builder
// =============================================================================

// buildClientActionV3 converts a V3 Action AST to a gen element.Element.
func (pb *pageBuilder) buildClientActionV3(action *ast.ActionV3) (element.Element, error) {
	if fn, ok := actionBuilders[action.Type]; ok {
		return fn(pb, action)
	}
	return nil, mdlerrors.NewUnsupported("unsupported action type: " + action.Type)
}

// ============================================================================
// Individual Widget Builders (V3 Gen-native)
// ============================================================================

func (pb *pageBuilder) buildLayoutGridV3(w *ast.WidgetV3) (element.Element, error) {
	lg := genPg.NewLayoutGrid()
	assignFreshID(lg)
	lg.SetName(w.Name)

	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "row" {
			row, err := pb.buildLayoutGridRowV3(child)
			if err != nil {
				return nil, err
			}
			lg.AddRows(row)
		}
	}

	return lg, nil
}

func (pb *pageBuilder) buildLayoutGridRowV3(w *ast.WidgetV3) (element.Element, error) {
	row := genPg.NewLayoutGridRow()
	assignFreshID(row)
	if row.Appearance() == nil {
		row.SetAppearance(newDefaultAppearance())
	}
	row.SetConditionalVisibilitySettings(nil)
	row.SetHorizontalAlignment("None")
	row.SetVerticalAlignment("None")
	row.SetSpacingBetweenColumns(true)
	if w.Name != "" {
		row.SetClass("_mdlRow:" + w.Name)
	}

	var desktopSum int
	var hasExplicit bool
	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "column" {
			col, err := pb.buildLayoutGridColumnV3(child)
			if err != nil {
				return nil, err
			}
			row.AddColumns(col)
			if dw := child.GetDesktopWidth(); dw != nil {
				if v, ok := dw.(int); ok {
					desktopSum += v
					hasExplicit = true
				}
			}
		}
	}

	// Studio Pro requires that explicit desktop widths sum to exactly 12 (CE0535).
	// Warn immediately so the user can correct before opening in Studio Pro.
	if hasExplicit && desktopSum != 12 {
		log.Printf("WARNING [%s]: LAYOUTGRID row desktop column widths sum to %d, but Studio Pro requires exactly 12 (CE0535).\n"+
			"  Adjust desktopwidth values so they add up to 12 (e.g. 8+4, 6+6, 4+4+4).",
			w.Name, desktopSum)
	}

	return row, nil
}

func (pb *pageBuilder) buildLayoutGridColumnV3(w *ast.WidgetV3) (element.Element, error) {
	col := genPg.NewLayoutGridColumn()
	assignFreshID(col)
	if col.Appearance() == nil {
		col.SetAppearance(newDefaultAppearance())
	}
	col.SetVerticalAlignment("None")
	if w.Name != "" {
		col.SetClass("_mdlCol:" + w.Name)
	}

	// Studio Pro defaults all size fields to -1 (AutoFill) when unset.
	// Apply the same defaults so our BSON matches SP output.
	col.SetWeight(-1)
	col.SetTabletWeight(-1)
	col.SetPhoneWeight(-1)
	col.SetPreviewWidth(-1)

	// Handle DesktopWidth
	if dw := w.GetDesktopWidth(); dw != nil {
		switch v := dw.(type) {
		case int:
			col.SetWeight(int32(v))
		case string:
			if strings.EqualFold(v, "AutoFill") {
				col.SetWeight(-1)
			}
		}
	}

	// Handle TabletWidth
	if tw := w.Properties["TabletWidth"]; tw != nil {
		switch v := tw.(type) {
		case int:
			col.SetTabletWeight(int32(v))
		case string:
			if strings.EqualFold(v, "AutoFill") {
				col.SetTabletWeight(-1)
			}
		}
	}

	// Handle PhoneWidth
	if pw := w.Properties["PhoneWidth"]; pw != nil {
		switch v := pw.(type) {
		case int:
			col.SetPhoneWeight(int32(v))
		case string:
			if strings.EqualFold(v, "AutoFill") {
				col.SetPhoneWeight(-1)
			}
		}
	}

	// Build child widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		col.AddWidgets(widget)
	}

	return col, nil
}

// buildContainerWithRowV3 creates a DivContainer holding a LayoutGrid with one row.
func (pb *pageBuilder) buildContainerWithRowV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	lg := genPg.NewLayoutGrid()
	assignFreshID(lg)
	lg.SetName(w.Name + "_grid")

	row, err := pb.buildLayoutGridRowV3(w)
	if err != nil {
		return nil, err
	}
	lg.AddRows(row)
	container.AddWidgets(lg)

	return container, nil
}

// buildContainerWithColumnV3 creates a DivContainer holding a LayoutGrid with one column.
func (pb *pageBuilder) buildContainerWithColumnV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	lg := genPg.NewLayoutGrid()
	assignFreshID(lg)
	lg.SetName(w.Name + "_grid")

	row := genPg.NewLayoutGridRow()
	assignFreshID(row)

	col, err := pb.buildLayoutGridColumnV3(w)
	if err != nil {
		return nil, err
	}
	row.AddColumns(col)
	lg.AddRows(row)
	container.AddWidgets(lg)

	return container, nil
}

func (pb *pageBuilder) buildContainerV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	// Handle RenderMode
	if rm := w.GetRenderMode(); rm != "" {
		container.SetRenderMode(rm)
	}

	// Build child widgets
	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.AddWidgets(widget)
	}

	return container, nil
}

func (pb *pageBuilder) buildTabContainerV3(w *ast.WidgetV3) (element.Element, error) {
	tc := genPg.NewTabContainer()
	assignFreshID(tc)
	tc.SetName(w.Name)

	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "tabpage" {
			tp, err := pb.buildTabPageV3(child)
			if err != nil {
				return nil, err
			}
			tc.AddTabPages(tp)
		}
	}

	if err := pb.registerWidgetName(w.Name, model.ID(tc.ID())); err != nil {
		return nil, err
	}

	return tc, nil
}

func (pb *pageBuilder) buildTabPageV3(w *ast.WidgetV3) (element.Element, error) {
	tp := genPg.NewTabPage()
	assignFreshID(tp)
	tp.SetName(w.Name)

	// TabPage.Caption must be Texts$Text, NOT Forms$ClientTemplate.
	// genSimpleLabel wraps in ClientTemplate → StorageLoadException ("ClientTemplate cannot be converted to Text").
	// Same fix as page.Title (see comment above genSimpleText call in buildPageV3).
	if caption := w.GetCaption(); caption != "" {
		tp.SetCaption(genSimpleText(caption))
	}

	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		tp.AddWidgets(widget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(tp.ID())); err != nil {
		return nil, err
	}

	return tp, nil
}

func (pb *pageBuilder) buildGroupBoxV3(w *ast.WidgetV3) (element.Element, error) {
	gb := genPg.NewGroupBox()
	assignFreshID(gb)
	gb.SetName(w.Name)
	gb.SetCollapsible("No")
	gb.SetHeaderMode("Div")

	// GroupBox.Caption must be Texts$Text, NOT Forms$ClientTemplate (same as TabPage.Caption).
	if caption := w.GetCaption(); caption != "" {
		gb.SetCaption(genSimpleText(caption))
	}

	if collapsible := w.GetStringProp("Collapsible"); collapsible != "" {
		switch strings.ToLower(collapsible) {
		case "yesexpanded", "yesinitiallyexpanded", "yes":
			gb.SetCollapsible("YesInitiallyExpanded")
		case "yescollapsed", "yesinitiallycollapsed":
			gb.SetCollapsible("YesInitiallyCollapsed")
		case "no":
			gb.SetCollapsible("No")
		default:
			gb.SetCollapsible(collapsible)
		}
	}

	if headerMode := w.GetStringProp("HeaderMode"); headerMode != "" {
		gb.SetHeaderMode(headerMode)
	}

	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		gb.AddWidgets(widget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(gb.ID())); err != nil {
		return nil, err
	}

	return gb, nil
}

// buildFooterV3 creates a DivContainer (footer) from V3 syntax.
func (pb *pageBuilder) buildFooterV3(w *ast.WidgetV3) (element.Element, error) {
	footer := genPg.NewDivContainer()
	assignFreshID(footer)
	footer.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		footer.AddWidgets(childWidget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(footer.ID())); err != nil {
		return nil, err
	}

	return footer, nil
}

// buildHeaderV3 creates a DivContainer (header) from V3 syntax.
func (pb *pageBuilder) buildHeaderV3(w *ast.WidgetV3) (element.Element, error) {
	header := genPg.NewDivContainer()
	assignFreshID(header)
	header.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		header.AddWidgets(childWidget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(header.ID())); err != nil {
		return nil, err
	}

	return header, nil
}

// buildControlBarV3 creates a DivContainer (control bar) from V3 syntax.
func (pb *pageBuilder) buildControlBarV3(w *ast.WidgetV3) (element.Element, error) {
	controlBar := genPg.NewDivContainer()
	assignFreshID(controlBar)
	controlBar.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		controlBar.AddWidgets(childWidget)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(controlBar.ID())); err != nil {
		return nil, err
	}

	return controlBar, nil
}

func (pb *pageBuilder) buildDataViewV3(w *ast.WidgetV3) (element.Element, error) {
	dv := genPg.NewDataView()
	assignFreshID(dv)
	dv.SetName(w.Name)

	if ds := w.GetDataSource(); ds != nil {
		dataSource, entityName, err := pb.buildDataSourceV3(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		dv.SetDataSource(dataSource)

		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()

		if w.Name != "" && entityName != "" {
			pb.paramEntityNames[w.Name] = entityName
		}
	}

	// Build child widgets, separating FOOTER widgets
	for _, child := range w.Children {
		if child.Type == "footer" {
			dv.SetShowFooter(true)
			for _, fw := range child.Children {
				widget, err := pb.buildWidgetV3(fw)
				if err != nil {
					return nil, err
				}
				dv.AddFooterWidgets(widget)
			}
			continue
		}
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		dv.AddWidgets(childWidget)
	}

	// Also build footer widgets from Properties (legacy support)
	if footerWidgets, ok := w.Properties["Footer"].([]*ast.WidgetV3); ok {
		dv.SetShowFooter(true)
		for _, fw := range footerWidgets {
			widget, err := pb.buildWidgetV3(fw)
			if err != nil {
				return nil, err
			}
			dv.AddFooterWidgets(widget)
		}
	}

	if err := pb.registerWidgetName(w.Name, model.ID(dv.ID())); err != nil {
		return nil, err
	}

	return dv, nil
}

func (pb *pageBuilder) buildDataGridV3(w *ast.WidgetV3) (element.Element, error) {
	widgetID := model.ID(types.GenerateID())

	// Build datasource BSON (pre-serialized for DataGridSpec)
	var datasourceBSON bson.D
	if ds := w.GetDataSource(); ds != nil {
		dsDoc, entityName, err := pb.buildDataGridDataSourceBSON(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		datasourceBSON = dsDoc

		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()
	}

	// Extract column definitions and CONTROLBAR widgets from children
	var columns []backend.DataGridColumnSpec
	var headerWidgetsBSON []bson.D
	var columnsFilterable bool
	for _, child := range w.Children {
		switch strings.ToLower(child.Type) {
		case "column":
			attr := child.GetAttribute()
			if attr == "" && child.Name != "" && len(child.Children) == 0 {
				attr = child.Name
			}
			col := backend.DataGridColumnSpec{
				Attribute:  pb.resolveAttributePath(attr),
				Caption:    child.GetCaption(),
				Properties: child.Properties,
			}
			for _, grandchild := range child.Children {
				if dataGridFilterWidgetID(grandchild.Type) != "" {
					// Column-level filter: enable DataGrid2's built-in column filtering.
					// We do NOT embed a separate CustomWidget in the column's filter slot —
					// that causes CE0463 "widget definition changed" because the embedded
					// type schema doesn't match the installed widget's schema.
					// Instead we set columnsFilterable so the DataGrid2 property is "true",
					// which activates the native per-column filter UI.
					columnsFilterable = true
				} else {
					// Build child widget as pre-serialized BSON
					childDoc, err := pb.buildWidgetBSON(grandchild)
					if err != nil {
						return nil, mdlerrors.NewBackend("build column child widget", err)
					}
					if childDoc != nil {
						col.ChildWidgetsBSON = append(col.ChildWidgetsBSON, childDoc)
					}
				}
			}
			columns = append(columns, col)
		case "controlbar":
			for _, controlBarChild := range child.Children {
				childDoc, err := pb.buildWidgetBSON(controlBarChild)
				if err != nil {
					return nil, mdlerrors.NewBackend("build controlbar widget", err)
				}
				if childDoc != nil {
					headerWidgetsBSON = append(headerWidgetsBSON, childDoc)
				}
			}
		}
	}

	// Collect paging overrides from AST properties
	pagingOverrides := make(map[string]string)
	for mdlKey, widgetKey := range dataGridPagingPropMap {
		if v := w.GetStringProp(mdlKey); v != "" {
			pagingOverrides[widgetKey] = v
		} else if iv := w.GetIntProp(mdlKey); iv > 0 {
			pagingOverrides[widgetKey] = fmt.Sprintf("%d", iv)
		} else if bv, ok := w.Properties[mdlKey]; ok {
			if boolVal, isBool := bv.(bool); isBool {
				if boolVal {
					pagingOverrides[widgetKey] = "yes"
				} else {
					pagingOverrides[widgetKey] = "no"
				}
			}
		}
	}

	spec := backend.DataGridSpec{
		DataSourceBSON:    datasourceBSON,
		Columns:           columns,
		HeaderWidgetsBSON: headerWidgetsBSON,
		PagingOverrides:   pagingOverrides,
		SelectionMode:     w.GetSelection(),
		ColumnsFilterable: columnsFilterable,
	}

	// Use gen-native DataGrid2 builder
	grid, err := pb.widgetBackend.BuildDataGrid2WidgetGen(widgetID, w.Name, spec, pb.backend.Path())
	if err != nil {
		return nil, err
	}

	if err := pb.registerWidgetName(w.Name, model.ID(grid.AsElement().ID())); err != nil {
		return nil, err
	}

	return grid.AsElement(), nil
}

// buildWidgetBSON builds a pre-serialized bson.D widget for use in DataGridSpec.
// Used internally by buildDataGridV3 for column child widgets and controlbar widgets.
func (pb *pageBuilder) buildWidgetBSON(w *ast.WidgetV3) (bson.D, error) {
	switch strings.ToLower(w.Type) {
	case "button", "actionbutton":
		buttonStyle := "Default"
		if style := w.GetButtonStyle(); style != "" {
			buttonStyle = style
		}
		// Build action BSON
		actionBSON := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$NoAction"},
			{Key: "DisabledDuringExecution", Value: true},
		}
		if action := w.GetAction(); action != nil {
			ab, err := pb.genClientActionToBsonD(action)
			if err != nil {
				return nil, mdlerrors.NewBackend("build action", err)
			}
			if ab != nil {
				actionBSON = ab
			}
		}
		// Build caption template
		captionTemplate := buildMinimalClientTemplate(w.GetCaption())
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$ActionButton"},
			{Key: "Action", Value: actionBSON},
			{Key: "Appearance", Value: buildMinimalAppearance()},
			{Key: "AriaRole", Value: "Button"},
			{Key: "ButtonStyle", Value: buttonStyle},
			{Key: "CaptionTemplate", Value: captionTemplate},
			{Key: "ConditionalVisibilitySettings", Value: nil},
			{Key: "Icon", Value: nil},
			{Key: "Name", Value: w.Name},
			{Key: "NativeAccessibilitySettings", Value: nil},
			{Key: "RenderType", Value: "Button"},
			{Key: "TabIndex", Value: int64(0)},
			{Key: "Tooltip", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Texts$Text"},
				{Key: "Items", Value: bson.A{int32(3)}},
			}},
		}
		return doc, nil
	case "textfilter", "numberfilter", "datefilter", "dropdownfilter":
		filterWidgetID := dataGridFilterWidgetID(w.Type)
		rawAttrs := w.GetAttributes()
		resolvedAttrs := make([]string, 0, len(rawAttrs))
		for _, a := range rawAttrs {
			resolvedAttrs = append(resolvedAttrs, pb.resolveAttributePath(a))
		}
		fw, err := pb.widgetBackend.BuildFilterWidgetGen(backend.FilterWidgetSpec{
			WidgetID:   filterWidgetID,
			FilterName: w.Name,
			FilterType: w.GetFilterType(),
			Attributes: resolvedAttrs,
		}, pb.backend.Path())
		if err != nil {
			return nil, mdlerrors.NewBackend("build controlbar filter widget", err)
		}
		return pb.serializeGenWidgetToBsonD(fw)
	case "pluggablewidget", "customwidget":
		widgetType, ok := w.Properties["WidgetType"].(string)
		if !ok || widgetType == "" {
			return nil, mdlerrors.NewValidation("pluggable widget in datagrid column: missing WidgetType")
		}
		pb.initPluggableEngine()
		if pb.pluggableEngine == nil {
			return nil, mdlerrors.NewUnsupported("pluggable widget engine not available")
		}
		if def, ok := pb.widgetRegistry.GetByWidgetID(widgetType); ok {
			cw, err := pb.pluggableEngine.Build(def, w)
			if err != nil {
				return nil, mdlerrors.NewBackend("build pluggable widget in datagrid column", err)
			}
			return pb.serializeGenWidgetToBsonD(cw)
		}
		return nil, mdlerrors.NewNotFoundMsg("widget", widgetType,
			"no definition for widget "+widgetType)
	default:
		// For other widget types in DataGrid context, create a minimal DivContainer
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$DivContainer"},
			{Key: "Appearance", Value: buildMinimalAppearance()},
			{Key: "ConditionalVisibilitySettings", Value: nil},
			{Key: "Name", Value: w.Name},
			{Key: "NativeAccessibilitySettings", Value: nil},
			{Key: "OnClickAction", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$NoAction"},
				{Key: "DisabledDuringExecution", Value: true},
			}},
			{Key: "RenderMode", Value: "Div"},
			{Key: "ScreenReaderHidden", Value: false},
			{Key: "TabIndex", Value: int64(0)},
			{Key: "Widgets", Value: bson.A{int32(2)}},
		}
		return doc, nil
	}
}

// genClientActionToBsonD builds a pre-serialized bson.D client action for DataGrid button widgets.
// It delegates to buildClientActionV3 (the gen path) and serializes the result via the codec,
// eliminating hand-rolled BSON that was prone to wrong type names and missing fields.
func (pb *pageBuilder) genClientActionToBsonD(action *ast.ActionV3) (bson.D, error) {
	elem, err := pb.buildClientActionV3(action)
	if err != nil {
		return nil, err
	}
	if elem == nil {
		return nil, nil
	}
	opaque := pb.widgetBackend.SerializeGenElemToOpaque(elem)
	if opaque == nil {
		return nil, fmt.Errorf("genClientActionToBsonD: serialize returned nil for action type %q", action.Type)
	}
	var doc bson.D
	switch v := opaque.(type) {
	case bson.Raw:
		if err := bson.Unmarshal([]byte(v), &doc); err != nil {
			return nil, fmt.Errorf("genClientActionToBsonD: unmarshal: %w", err)
		}
	case bson.D:
		doc = v
	default:
		return nil, fmt.Errorf("genClientActionToBsonD: unexpected opaque type %T", opaque)
	}
	return doc, nil
}

// serializeGenWidgetToBsonD encodes a GenCustomWidgetElem to bson.D.
// Used for pre-serializing filter widgets into DataGridColumnSpec.FilterWidgetBSON.
func (pb *pageBuilder) serializeGenWidgetToBsonD(fw *backend.GenCustomWidgetElem) (bson.D, error) {
	if fw == nil {
		return nil, nil
	}
	opaque := pb.widgetBackend.SerializeGenElemToOpaque(fw.AsElement())
	if opaque == nil {
		return nil, fmt.Errorf("serializeGenWidgetToBsonD: encode returned nil")
	}
	// SerializeGenElemToOpaque returns bson.Raw — unmarshal to bson.D for embedding.
	var doc bson.D
	switch v := opaque.(type) {
	case bson.Raw:
		if err := bson.Unmarshal([]byte(v), &doc); err != nil {
			return nil, fmt.Errorf("serializeGenWidgetToBsonD: unmarshal: %w", err)
		}
	case bson.D:
		doc = v
	default:
		return nil, fmt.Errorf("serializeGenWidgetToBsonD: unexpected opaque type %T", opaque)
	}
	return doc, nil
}

// buildMinimalClientTemplate builds a minimal Forms$ClientTemplate BSON with optional text.
func buildMinimalClientTemplate(text string) bson.D {
	var templateItems bson.A
	if text != "" {
		templateItems = bson.A{int32(3), bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Texts$Translation"},
			{Key: "LanguageCode", Value: "en_US"},
			{Key: "Text", Value: text},
		}}
	} else {
		templateItems = bson.A{int32(3)}
	}
	return bson.D{
		{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
		{Key: "$Type", Value: "Forms$ClientTemplate"},
		{Key: "Fallback", Value: bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: bson.A{int32(3)}},
		}},
		{Key: "Parameters", Value: bson.A{int32(2)}},
		{Key: "Template", Value: bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: templateItems},
		}},
	}
}

// buildMinimalAppearance builds a minimal Forms$Appearance BSON.
func buildMinimalAppearance() bson.D {
	return bson.D{
		{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
		{Key: "$Type", Value: "Forms$Appearance"},
		{Key: "Class", Value: ""},
		{Key: "DesignProperties", Value: bson.A{int32(3)}},
		{Key: "DynamicClasses", Value: ""},
		{Key: "Style", Value: ""},
	}
}

func (pb *pageBuilder) buildListViewV3(w *ast.WidgetV3) (element.Element, error) {
	lv := genPg.NewListView()
	assignFreshID(lv)
	lv.SetName(w.Name)
	lv.SetPageSize(20)

	if ds := w.GetDataSource(); ds != nil {
		dataSource, entityName, err := pb.buildListViewDataSourceV3(ds)
		if err != nil {
			return nil, mdlerrors.NewBackend("build datasource", err)
		}
		lv.SetDataSource(dataSource)

		oldContext := pb.entityContext
		pb.entityContext = entityName
		defer func() { pb.entityContext = oldContext }()

		if w.Name != "" && entityName != "" {
			pb.paramEntityNames[w.Name] = entityName
		}
	}

	if err := pb.registerWidgetName(w.Name, model.ID(lv.ID())); err != nil {
		return nil, err
	}

	for _, child := range w.Children {
		widget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		lv.AddWidgets(widget)
	}

	return lv, nil
}

func (pb *pageBuilder) buildTextBoxV3(w *ast.WidgetV3) (element.Element, error) {
	tb := genPg.NewTextBox()
	assignFreshID(tb)
	tb.SetName(w.Name)
	applyFormWidgetDefaults(tb)

	if attr := w.GetAttribute(); attr != "" {
		tb.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		tb.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(tb.ID())); err != nil {
		return nil, err
	}

	return tb, nil
}

func (pb *pageBuilder) buildTextAreaV3(w *ast.WidgetV3) (element.Element, error) {
	ta := genPg.NewTextArea()
	assignFreshID(ta)
	ta.SetName(w.Name)
	applyFormWidgetDefaults(ta)

	if attr := w.GetAttribute(); attr != "" {
		ta.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		ta.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(ta.ID())); err != nil {
		return nil, err
	}

	return ta, nil
}

func (pb *pageBuilder) buildDatePickerV3(w *ast.WidgetV3) (element.Element, error) {
	dp := genPg.NewDatePicker()
	assignFreshID(dp)
	dp.SetName(w.Name)
	applyFormWidgetDefaults(dp)

	if attr := w.GetAttribute(); attr != "" {
		dp.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		dp.SetLabel(genSimpleLabel(label))
	}

	// ConditionalVisibility/Editability must be explicit null — DatePicker does not
	// implement textLike so applyFormWidgetDefaults does not set them, but Mendix
	// requires these fields to validate attribute type (CE2421 without them).
	dp.SetConditionalVisibilitySettings(nil)
	dp.SetConditionalEditabilitySettings(nil)
	// FormattingInfo is required for DatePicker — without it Mendix cannot validate
	// the attribute type and raises CE2421. All fields must be set to match Studio Pro.
	fi := genPg.NewFormattingInfo()
	assignFreshID(fi)
	fi.SetDateFormat("DateTime")
	fi.SetCustomDateFormat("")
	fi.SetDecimalPrecision(2)
	fi.SetEnumFormat("Text")
	fi.SetGroupDigits(false)
	dp.SetFormattingInfo(fi)

	if err := pb.registerWidgetName(w.Name, model.ID(dp.ID())); err != nil {
		return nil, err
	}

	return dp, nil
}

// buildFileManagerV3 builds a Forms$FileManager widget (MDL: fileinput).
// FileManager does not implement the formInputDefaults interface (no OnChangeAction,
// Validation, SourceVariable, etc.), so the mandatory fields it does share with
// standard inputs are set explicitly here instead of via applyFormWidgetDefaults.
func (pb *pageBuilder) buildFileManagerV3(w *ast.WidgetV3) (element.Element, error) {
	fm := genPg.NewFileManager()
	assignFreshID(fm)
	fm.SetName(w.Name)

	fm.SetEditable("Always")
	fm.SetAppearance(newDefaultAppearance())
	fm.SetScreenReaderLabel(nil)
	fm.SetTabIndex(0)
	fm.SetConditionalVisibilitySettings(nil)
	fm.SetConditionalEditabilitySettings(nil)

	if ext := w.GetStringProp("allowedExtensions"); ext != "" {
		fm.SetAllowedExtensions(ext)
	}

	if size := w.GetIntProp("maxFileSize"); size > 0 {
		fm.SetMaxFileSize(int32(size))
	}

	if label := w.GetLabel(); label != "" {
		fm.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(fm.ID())); err != nil {
		return nil, err
	}

	return fm, nil
}

func (pb *pageBuilder) buildDropdownV3(w *ast.WidgetV3) (element.Element, error) {
	dd := genPg.NewDropDown()
	assignFreshID(dd)
	dd.SetName(w.Name)
	applyFormWidgetDefaults(dd)

	if attr := w.GetAttribute(); attr != "" {
		dd.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		dd.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(dd.ID())); err != nil {
		return nil, err
	}

	return dd, nil
}

func (pb *pageBuilder) buildCheckBoxV3(w *ast.WidgetV3) (element.Element, error) {
	cb := genPg.NewCheckBox()
	assignFreshID(cb)
	cb.SetName(w.Name)
	applyFormWidgetDefaults(cb)

	if attr := w.GetAttribute(); attr != "" {
		cb.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		cb.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(cb.ID())); err != nil {
		return nil, err
	}

	return cb, nil
}

// buildRadioButtonsV3 creates a RadioButtonGroup from V3 syntax.
func (pb *pageBuilder) buildRadioButtonsV3(w *ast.WidgetV3) (element.Element, error) {
	rb := genPg.NewRadioButtonGroup()
	assignFreshID(rb)
	rb.SetName(w.Name)
	applyFormWidgetDefaults(rb)

	if attr := w.GetAttribute(); attr != "" {
		rb.SetAttributeRef(newAttributeRef(pb.resolveAttributePath(attr)))
	}

	if label := w.GetLabel(); label != "" {
		rb.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(rb.ID())); err != nil {
		return nil, err
	}

	return rb, nil
}

func (pb *pageBuilder) buildTextWidgetV3(w *ast.WidgetV3) (element.Element, error) {
	lbl := genPg.NewLabel()
	assignFreshID(lbl)
	lbl.SetName(w.Name)

	if content := w.GetContent(); content != "" {
		lbl.SetCaption(genSimpleLabel(content))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(lbl.ID())); err != nil {
		return nil, err
	}

	return lbl, nil
}

func (pb *pageBuilder) buildDynamicTextV3(w *ast.WidgetV3) (element.Element, error) {
	dt := genPg.NewDynamicText()
	assignFreshID(dt)
	dt.SetName(w.Name)

	if rm := w.GetRenderMode(); rm != "" {
		dt.SetRenderMode(rm)
	}

	content := w.GetContent()
	explicitParams := w.GetContentParams()

	var autoGeneratedParams []string
	if content != "" && explicitParams == nil {
		isEntityPath := false
		if strings.Contains(content, ".") && !strings.HasPrefix(content, "$") {
			isEntityPath = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*$`).MatchString(content)
		}
		if strings.HasPrefix(content, "$") || isEntityPath {
			autoGeneratedParams = append(autoGeneratedParams, content)
			content = "{1}"
		}
	}

	if content == "" {
		// Empty content would produce a {1} template with 0 parameters,
		// causing Studio Pro CE0720 "Place holder index 1 is greater than 0".
		// Use a single space as a safe placeholder (CLAUDE.md note).
		content = " "
	}

	ct := genPg.NewClientTemplate()
	assignFreshID(ct)
	ct.SetTemplate(genSimpleText(content))
	fallbackText := genTexts.NewText()
	assignFreshID(fallbackText)
	ct.SetFallback(fallbackText)

	// Add auto-generated parameters first
	for _, attrRef := range autoGeneratedParams {
		param := genPg.NewClientTemplateParameter()
		assignFreshID(param)
		pb.resolveTemplateAttributePathFull(attrRef, param)
		ct.AddParameters(param)
	}

	// Handle explicit ContentParams
	if explicitParams != nil {
		for _, p := range explicitParams {
			param := genPg.NewClientTemplateParameter()
			assignFreshID(param)
			if strVal, ok := p.Value.(string); ok {
				if strings.HasPrefix(strVal, "'") || strings.HasPrefix(strVal, "\"") {
					param.SetExpression(strVal)
				} else if strings.HasPrefix(strVal, "$") || strings.Contains(strVal, ".") {
					pb.resolveTemplateAttributePathFull(strVal, param)
				} else {
					pb.resolveTemplateAttributePathFull(strVal, param)
				}
			}
			ct.AddParameters(param)
		}
	}

	dt.SetContent(ct)

	if err := pb.registerWidgetName(w.Name, model.ID(dt.ID())); err != nil {
		return nil, err
	}

	return dt, nil
}

func (pb *pageBuilder) buildTitleV3(w *ast.WidgetV3) (element.Element, error) {
	title := genPg.NewTitle()
	assignFreshID(title)
	title.SetName(w.Name)

	if content := w.GetContent(); content != "" {
		setRawBSONField(title, "Caption", content)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(title.ID())); err != nil {
		return nil, err
	}

	return title, nil
}

func (pb *pageBuilder) buildButtonV3(w *ast.WidgetV3) (element.Element, error) {
	btn := genPg.NewActionButton()
	assignFreshID(btn)
	btn.SetName(w.Name)
	btn.SetButtonStyle("Default")

	if caption := w.GetCaption(); caption != "" {
		// Build caption ClientTemplate
		ct := genPg.NewClientTemplate()
		assignFreshID(ct)
		ct.SetTemplate(genSimpleText(caption))
		fallback := genTexts.NewText()
		assignFreshID(fallback)
		ct.SetFallback(fallback)

		// Handle CaptionParams
		if params := w.GetCaptionParams(); params != nil {
			for _, p := range params {
				param := genPg.NewClientTemplateParameter()
				assignFreshID(param)
				if strVal, ok := p.Value.(string); ok {
					if strings.HasPrefix(strVal, "'") || strings.HasPrefix(strVal, "\"") {
						param.SetExpression(strVal)
					} else if strings.HasPrefix(strVal, "$") || strings.Contains(strVal, ".") {
						param.SetAttributePath(pb.resolveTemplateAttributePath(strVal))
					} else {
						param.SetExpression("'" + strVal + "'")
					}
				}
				ct.AddParameters(param)
			}
		}
		btn.SetCaption(ct)
	}

	if style := w.GetButtonStyle(); style != "" {
		btn.SetButtonStyle(style)
	}

	if action := w.GetAction(); action != nil {
		act, err := pb.buildClientActionV3(action)
		if err != nil {
			return nil, mdlerrors.NewBackend("build action", err)
		}
		btn.SetAction(act)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(btn.ID())); err != nil {
		return nil, err
	}

	return btn, nil
}

// buildNavigationListV3 creates a NavigationList widget from V3 syntax.
func (pb *pageBuilder) buildNavigationListV3(w *ast.WidgetV3) (element.Element, error) {
	nl := genPg.NewNavigationList()
	assignFreshID(nl)
	nl.SetName(w.Name)

	for _, child := range w.Children {
		if strings.ToLower(child.Type) == "item" {
			item, err := pb.buildNavigationListItemV3(child)
			if err != nil {
				return nil, err
			}
			nl.AddItems(item)
		}
	}

	if err := pb.registerWidgetName(w.Name, model.ID(nl.ID())); err != nil {
		return nil, err
	}

	return nl, nil
}

// buildNavigationListItemV3 creates a NavigationListItem from V3 syntax.
func (pb *pageBuilder) buildNavigationListItemV3(w *ast.WidgetV3) (element.Element, error) {
	if w.Name == "" {
		return nil, mdlerrors.NewValidation("item inside navigationlist requires a name")
	}

	item := genPg.NewNavigationListItem()
	assignFreshID(item)
	// NavigationListItem has no SetName in gen — inject via raw BSON
	setRawBSONField(item, "Name", w.Name)

	if err := pb.registerWidgetName(w.Name, model.ID(item.ID())); err != nil {
		return nil, err
	}

	if action := w.GetAction(); action != nil {
		clientAction, err := pb.buildClientActionV3(action)
		if err != nil {
			return nil, err
		}
		item.SetAction(clientAction)
	}

	// Build explicit child widgets or create a DynamicText from Caption
	if len(w.Children) > 0 {
		for _, child := range w.Children {
			childWidget, err := pb.buildWidgetV3(child)
			if err != nil {
				return nil, err
			}
			item.AddWidgets(childWidget)
		}
	} else if caption := w.GetCaption(); caption != "" {
		// Create DynamicText for caption (matches old SDK serializer behavior)
		dt := genPg.NewDynamicText()
		assignFreshID(dt)
		dt.SetName("text_" + w.Name)
		ct := genPg.NewClientTemplate()
		assignFreshID(ct)
		ct.SetTemplate(genSimpleText(caption))
		fallback := genTexts.NewText()
		assignFreshID(fallback)
		ct.SetFallback(fallback)
		dt.SetContent(ct)
		item.AddWidgets(dt)
	}

	return item, nil
}

// buildSnippetCallV3 creates a SnippetCallWidget from V3 syntax.
func (pb *pageBuilder) buildSnippetCallV3(w *ast.WidgetV3) (element.Element, error) {
	scw := genPg.NewSnippetCallWidget()
	assignFreshID(scw)
	scw.SetName(w.Name)

	snippetName := w.GetSnippet()
	if snippetName != "" {
		snippetID, err := pb.resolveSnippetRef(snippetName)
		if err != nil {
			return nil, mdlerrors.NewBackend(fmt.Sprintf("resolve snippet %s", snippetName), err)
		}
		_ = snippetID

		// Build SnippetCall sub-element
		sc := genPg.NewSnippetCall()
		assignFreshID(sc)
		sc.SetSnippetQualifiedName(snippetName)

		// Validate and wire up parameter mappings
		if err := pb.buildSnippetCallParamsGen(sc, snippetName, w.GetSnippetParams()); err != nil {
			return nil, err
		}

		scw.SetSnippetCall(sc)
	}

	if err := pb.registerWidgetName(w.Name, model.ID(scw.ID())); err != nil {
		return nil, err
	}

	return scw, nil
}

// buildSnippetCallParamsGen validates the supplied param mappings against the
// snippet's declared parameters and populates sc.ParameterMappings.
func (pb *pageBuilder) buildSnippetCallParamsGen(sc *genPg.SnippetCall, snippetQName string, supplied []ast.SnippetCallParam) error {
	snippets, err := pb.backend.ListSnippetsGen()
	if err != nil {
		return err
	}

	var targetSnippet *genPg.Snippet
	for _, s := range snippets {
		if s == nil {
			continue
		}
		name := s.Name()
		if name != "" && (name == snippetQName || strings.HasSuffix(snippetQName, "."+name)) {
			targetSnippet = s
			break
		}
	}
	if targetSnippet == nil || len(targetSnippet.ParametersItems()) == 0 {
		return nil
	}

	suppliedByName := make(map[string]string, len(supplied))
	for _, p := range supplied {
		name := strings.TrimPrefix(p.ParamName, "$")
		suppliedByName[name] = p.Variable
	}

	for _, rawParam := range targetSnippet.ParametersItems() {
		sp, ok := rawParam.(*genPg.SnippetParameter)
		if !ok || sp == nil {
			continue
		}
		paramName := sp.Name()
		argument, found := suppliedByName[paramName]
		if !found {
			return mdlerrors.NewValidationf(
				"snippet %s requires parameter $%s — add Params: {%s: $<variable>} to the SNIPPETCALL",
				snippetQName, paramName, paramName,
			)
		}

		pm := genPg.NewSnippetParameterMapping()
		assignFreshID(pm)
		// Full qualified name: Module.SnippetName.ParamName
		pm.SetParameterQualifiedName(snippetQName + "." + paramName)

		// Build PageVariable using gen modelsdk to hold the argument reference.
		varName := strings.TrimPrefix(argument, "$")
		pv := genPg.NewPageVariable()
		assignFreshID(pv)
		if _, isParam := pb.paramEntityNames[varName]; isParam {
			pv.SetPageParameterQualifiedName(varName)
		} else {
			pv.SetLocalVariableQualifiedName(varName)
		}
		pm.SetVariable(pv)
		sc.AddParameterMappings(pm)
	}

	return nil
}

// buildTemplateV3 creates a DivContainer to hold template content.
func (pb *pageBuilder) buildTemplateV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.AddWidgets(childWidget)
	}

	return container, nil
}

// buildFilterV3 creates a DivContainer to hold filter widgets.
func (pb *pageBuilder) buildFilterV3(w *ast.WidgetV3) (element.Element, error) {
	container := genPg.NewDivContainer()
	assignFreshID(container)
	container.SetName(w.Name)

	for _, child := range w.Children {
		childWidget, err := pb.buildWidgetV3(child)
		if err != nil {
			return nil, err
		}
		container.AddWidgets(childWidget)
	}

	return container, nil
}

func (pb *pageBuilder) buildStaticImageV3(w *ast.WidgetV3) (element.Element, error) {
	img := genPg.NewStaticImageViewer()
	assignFreshID(img)
	img.SetName(w.Name)
	img.SetResponsive(true)

	if width := w.GetIntProp("Width"); width > 0 {
		img.SetWidth(int32(width))
	}
	if height := w.GetIntProp("Height"); height > 0 {
		img.SetHeight(int32(height))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(img.ID())); err != nil {
		return nil, err
	}

	return img, nil
}

func (pb *pageBuilder) buildDynamicImageV3(w *ast.WidgetV3) (element.Element, error) {
	img := genPg.NewDynamicImageViewer()
	assignFreshID(img)
	img.SetName(w.Name)
	img.SetResponsive(true)

	if width := w.GetIntProp("Width"); width > 0 {
		img.SetWidth(int32(width))
	}
	if height := w.GetIntProp("Height"); height > 0 {
		img.SetHeight(int32(height))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(img.ID())); err != nil {
		return nil, err
	}

	return img, nil
}

// dataGridFilterWidgetID maps a MDL filter type keyword to its pluggable widget ID.
// Returns "" for non-filter widget types.
func dataGridFilterWidgetID(widgetType string) string {
	switch strings.ToLower(widgetType) {
	case "textfilter":
		return widgetIDDataGridTextFilter
	case "numberfilter":
		return widgetIDDataGridNumberFilter
	case "datefilter":
		return widgetIDDataGridDateFilter
	case "dropdownfilter":
		return widgetIDDataGridDropdownFilter
	}
	return ""
}

// dataGridPagingPropMap maps lowercase MDL property names to camelCase widget property keys.
// The visitor stores generic widget properties with the literal text from the grammar (lowercase),
// so these keys must match the lowercased MDL keyword form.
var dataGridPagingPropMap = map[string]string{
	"pagesize":          "pageSize",
	"pagination":        "pagination",
	"pagingposition":    "pagingPosition",
	"showpagingbuttons": "showPagingButtons",
}
