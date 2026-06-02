// SPDX-License-Identifier: Apache-2.0

// Package pagerender renders a types.PageModel to MDL V3 text. It lives in its
// own package so both the executor and the canonical/page layer can render
// without importing each other (which would create an import cycle: executor
// imports canonical/page to register its codec). It depends only on the
// standard library and mdl/types.
package pagerender

import (
	"fmt"
	"io"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// PageModelToMDL renders a PageModel to MDL V3 text and writes it to w.
func PageModelToMDL(w io.Writer, pm *types.PageModel, modName, pageName string) {
	fmt.Fprintf(w, "create or modify page %s.%s (", modName, pageName)
	fmt.Fprintf(w, "\n  title: '%s'", escapeMDLString(pm.Title))
	if pm.Layout != "" {
		fmt.Fprintf(w, ",\n  layout: %s", pm.Layout)
	}
	if pm.Folder != "" {
		fmt.Fprintf(w, ",\n  folder: '%s'", pm.Folder)
	}
	if len(pm.Params) > 0 {
		parts := make([]string, len(pm.Params))
		for i, p := range pm.Params {
			parts[i] = fmt.Sprintf("$%s: %s", p.Name, p.EntityName)
		}
		fmt.Fprintf(w, ",\n  params: { %s }", strings.Join(parts, ", "))
	}
	fmt.Fprintf(w, "\n) {\n")

	for _, widget := range pm.Widgets {
		RenderWidget(w, widget, 1)
	}

	fmt.Fprintf(w, "}")
}

// RenderWidget renders a single widget node (and its children) as MDL text.
func RenderWidget(w io.Writer, node *types.WidgetNode, depth int) {
	if node == nil {
		return
	}
	indent := strings.Repeat("  ", depth)

	switch node.Kind {
	case types.WidgetContainer, types.WidgetScrollView:
		kw := "container"
		if node.Kind == types.WidgetScrollView {
			kw = "scrollcontainer"
		}
		fmt.Fprintf(w, "%s%s %s", indent, kw, node.Name)
		renderAppearanceInline(w, node)
		renderVisibility(w, node)
		fmt.Fprintf(w, " {\n")
		for _, c := range node.Children {
			RenderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetLayoutGrid:
		fmt.Fprintf(w, "%slayoutgrid %s {\n", indent, node.Name)
		for _, r := range node.Children {
			RenderWidget(w, r, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetLayoutRow:
		fmt.Fprintf(w, "%srow %s {\n", indent, node.Name)
		for _, c := range node.Children {
			RenderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetLayoutCol:
		cw := node.ColWidth
		fmt.Fprintf(w, "%scolumn %s", indent, node.Name)
		// -1 = AutoFill, 0 = unset (omit), >0 = explicit width 1-12.
		if cw.Desktop != 0 || cw.Tablet != 0 || cw.Phone != 0 {
			fmt.Fprintf(w, " (")
			sep := ""
			fmtWeight := func(name string, v int) {
				if v == 0 {
					return
				}
				if v == -1 {
					fmt.Fprintf(w, "%s%s: AutoFill", sep, name)
				} else {
					fmt.Fprintf(w, "%s%s: %d", sep, name, v)
				}
				sep = ", "
			}
			fmtWeight("DesktopWidth", cw.Desktop)
			fmtWeight("TabletWidth", cw.Tablet)
			fmtWeight("PhoneWidth", cw.Phone)
			fmt.Fprintf(w, ")")
		}
		fmt.Fprintf(w, " {\n")
		for _, c := range node.Children {
			RenderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetGroupBox:
		fmt.Fprintf(w, "%sgroupbox %s", indent, node.Name)
		props := []string{}
		if node.Caption != "" {
			props = append(props, fmt.Sprintf("caption: '%s'", escapeMDLString(node.Caption)))
		}
		if node.GroupBox != nil && node.GroupBox.Collapsible != "" && node.GroupBox.Collapsible != "No" {
			props = append(props, fmt.Sprintf("collapsible: %s", node.GroupBox.Collapsible))
		}
		if len(props) > 0 {
			fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
		}
		fmt.Fprintf(w, " {\n")
		for _, c := range node.Children {
			RenderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetTabContainer:
		fmt.Fprintf(w, "%stabcontainer %s {\n", indent, node.Name)
		for _, tp := range node.Children {
			RenderWidget(w, tp, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetTabPage:
		caption := ""
		if node.Caption != "" {
			caption = fmt.Sprintf(" (caption: '%s')", escapeMDLString(node.Caption))
		}
		fmt.Fprintf(w, "%stabpage %s%s {\n", indent, node.Name, caption)
		for _, c := range node.Children {
			RenderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetDataView:
		fmt.Fprintf(w, "%sdataview %s", indent, node.Name)
		if node.DataSource != nil {
			fmt.Fprintf(w, " (DataSource: %s)", renderDataSource(node.DataSource))
		}
		fmt.Fprintf(w, " {\n")
		for _, c := range node.Children {
			RenderWidget(w, c, depth+1)
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetDataGrid:
		fmt.Fprintf(w, "%sdatagrid %s", indent, node.Name)
		if node.DataSource != nil {
			fmt.Fprintf(w, " (DataSource: %s)", renderDataSource(node.DataSource))
		}
		fmt.Fprintf(w, " {\n")
		if node.DataGrid != nil {
			for _, col := range node.DataGrid.Columns {
				renderDataGridColumn(w, col, depth+1)
			}
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetButton:
		fmt.Fprintf(w, "%sactionbutton %s", indent, node.Name)
		props := []string{}
		if node.Caption != "" {
			props = append(props, fmt.Sprintf("caption: '%s'", escapeMDLString(node.Caption)))
		}
		if node.OnClick != "" {
			props = append(props, fmt.Sprintf("action: microflow %s", node.OnClick))
		}
		if node.ButtonStyle != "" && node.ButtonStyle != "Default" {
			props = append(props, fmt.Sprintf("style: %s", node.ButtonStyle))
		}
		if len(props) > 0 {
			fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
		}
		fmt.Fprintf(w, "\n")

	case types.WidgetTextBox:
		fmt.Fprintf(w, "%stextbox %s", indent, node.Name)
		renderInputProps(w, node)
		fmt.Fprintf(w, "\n")

	case types.WidgetTextArea:
		fmt.Fprintf(w, "%stextarea %s", indent, node.Name)
		renderInputProps(w, node)
		fmt.Fprintf(w, "\n")

	case types.WidgetDatePicker:
		fmt.Fprintf(w, "%sdatepicker %s", indent, node.Name)
		renderInputProps(w, node)
		fmt.Fprintf(w, "\n")

	case types.WidgetLabel:
		// MDL grammar has no `label` keyword; render labels as staticttext
		// preserving the Caption as Content.
		fmt.Fprintf(w, "%sstatictext %s", indent, node.Name)
		if node.Caption != "" {
			fmt.Fprintf(w, " (Content: '%s')", escapeMDLString(node.Caption))
		}
		fmt.Fprintf(w, "\n")

	case types.WidgetText, types.WidgetTitle, types.WidgetDynamicText:
		kw := "statictext"
		switch node.Kind {
		case types.WidgetTitle:
			kw = "title"
		case types.WidgetDynamicText:
			kw = "dynamictext"
		}
		fmt.Fprintf(w, "%s%s %s", indent, kw, node.Name)
		if node.Content != "" {
			fmt.Fprintf(w, " (Content: '%s')", escapeMDLString(node.Content))
		}
		fmt.Fprintf(w, "\n")

	case types.WidgetSnippet:
		name := ""
		if node.Snippet != nil {
			name = node.Snippet.SnippetName
		}
		fmt.Fprintf(w, "%ssnippetcall %s (Snippet: %s)\n", indent, node.Name, name)

	case types.WidgetGallery:
		fmt.Fprintf(w, "%sgallery %s", indent, node.Name)
		if node.DataSource != nil {
			fmt.Fprintf(w, " (DataSource: %s)", renderDataSource(node.DataSource))
		}
		fmt.Fprintf(w, " {\n")
		if node.Gallery != nil {
			for _, c := range node.Gallery.ContentWidgets {
				RenderWidget(w, c, depth+1)
			}
		}
		fmt.Fprintf(w, "%s}\n", indent)

	case types.WidgetUnknown:
		widgetID := ""
		if node.Unknown != nil {
			widgetID = node.Unknown.WidgetID
		}
		fmt.Fprintf(w, "%s-- unsupported widget: %s (%s)\n", indent, node.Name, widgetID)

	default:
		fmt.Fprintf(w, "%s-- unhandled kind: %s %s\n", indent, node.Kind, node.Name)
	}
}

func renderDataSource(ds *types.DataSourceDef) string {
	switch ds.Kind {
	case types.DataSourceDatabase:
		s := "database " + ds.Entity
		if ds.XPathConstraint != "" {
			s += fmt.Sprintf(" where '%s'", ds.XPathConstraint)
		}
		return s
	case types.DataSourceMicroflow:
		return "microflow " + ds.Reference
	case types.DataSourceNanoflow:
		return "nanoflow " + ds.Reference
	case types.DataSourceParameter:
		// Parameter sources use the bare $Var form per dataSourceExprV3 grammar.
		ref := ds.Reference
		if ref == "" {
			return ""
		}
		if !strings.HasPrefix(ref, "$") {
			ref = "$" + ref
		}
		return ref
	case types.DataSourceSelection:
		return "selection " + ds.Reference
	}
	return ""
}

func renderDataGridColumn(w io.Writer, col types.ColumnDef, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(w, "%scolumn %s", indent, col.Name)
	props := []string{}
	if col.Attribute != "" {
		props = append(props, fmt.Sprintf("Attribute: %s", col.Attribute))
	}
	if col.Caption != "" {
		props = append(props, fmt.Sprintf("Caption: '%s'", escapeMDLString(col.Caption)))
	}
	if len(props) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
	}
	fmt.Fprintf(w, "\n")
}

func renderInputProps(w io.Writer, node *types.WidgetNode) {
	props := []string{}
	if node.EntityAttr != "" {
		props = append(props, fmt.Sprintf("Attribute: %s", node.EntityAttr))
	}
	if node.Editable != "" && node.Editable != "Always" {
		props = append(props, fmt.Sprintf("editable: %s", node.Editable))
	}
	if len(props) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
	}
}

func renderAppearanceInline(w io.Writer, node *types.WidgetNode) {
	props := []string{}
	if node.Class != "" {
		props = append(props, fmt.Sprintf("class: '%s'", node.Class))
	}
	if node.Style != "" {
		props = append(props, fmt.Sprintf("style: '%s'", node.Style))
	}
	if len(props) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(props, ", "))
	}
}

func renderVisibility(w io.Writer, node *types.WidgetNode) {
	if node.VisibleIf != "" {
		fmt.Fprintf(w, " visible if '%s'", node.VisibleIf)
	}
}

func escapeMDLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
