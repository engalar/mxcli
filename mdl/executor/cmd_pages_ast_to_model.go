// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// pageASTToModel converts a CreatePageStmtV3 AST to a types.PageModel.
// Properties are read from WidgetV3.Properties map; field names are matched
// case-insensitively because Task 3 tests use lowercase ("caption") while
// some renderers/legacy callers use mixed case ("Caption").
func pageASTToModel(ctx *ExecContext, s *ast.CreatePageStmtV3) (*types.PageModel, error) {
	pm := &types.PageModel{
		ModuleName: s.Name.Module,
		Name:       s.Name.Name,
		Title:      s.Title,
		Layout:     s.Layout,
		Folder:     s.Folder,
	}

	for _, p := range s.Parameters {
		pm.Params = append(pm.Params, types.PageParam{
			Name:       p.Name,
			EntityName: pageParamTypeRefString(p),
		})
	}

	for _, w := range s.Widgets {
		node, err := astWidgetToNode(ctx, w, s.Name.Module)
		if err != nil {
			return nil, fmt.Errorf("widget %s: %w", w.Name, err)
		}
		if node != nil {
			pm.Widgets = append(pm.Widgets, node)
		}
	}

	return pm, nil
}

// astWidgetToNode converts a single AST widget to a WidgetNode.
func astWidgetToNode(ctx *ExecContext, w *ast.WidgetV3, moduleName string) (*types.WidgetNode, error) {
	if w == nil {
		return nil, nil
	}

	kind := astWidgetKind(strings.ToLower(w.Type))
	node := &types.WidgetNode{
		Kind: kind,
		Name: w.Name,
	}

	node.Class = propStr(w, "class")
	node.Style = propStr(w, "style")
	node.VisibleIf = propStr(w, "visible")

	for _, child := range w.Children {
		childNode, err := astWidgetToNode(ctx, child, moduleName)
		if err != nil {
			return nil, err
		}
		if childNode != nil {
			node.Children = append(node.Children, childNode)
		}
	}

	switch kind {
	case types.WidgetLayoutCol:
		node.ColWidth = types.ColWidthDef{
			Desktop: propInt(w, "desktopwidth"),
			Tablet:  propInt(w, "tabletwidth"),
			Phone:   propInt(w, "phonewidth"),
		}

	case types.WidgetGroupBox:
		node.Caption = propStr(w, "caption")
		coll := propStr(w, "collapsible")
		header := propStr(w, "headermode")
		if coll != "" || header != "" {
			node.GroupBox = &types.GroupBoxProps{
				Collapsible: coll,
				HeaderMode:  header,
			}
		}

	case types.WidgetTabPage:
		node.Caption = propStr(w, "caption")

	case types.WidgetDataView:
		node.DataSource = astDataSourceToModel(propDS(w, "datasource"))
		if node.DataSource != nil {
			node.EntityCtx = node.DataSource.Entity
		}
		// Re-partition children: footer type → node.Footer, others stay in
		// node.Children. The general loop above already populated node.Children
		// with all children; redo it here so footer widgets (→ BSON
		// FooterWidgets) are separated from main content (→ BSON Widgets).
		node.Children = nil
		node.Footer = nil
		for _, child := range w.Children {
			cn, err := astWidgetToNode(ctx, child, moduleName)
			if err != nil || cn == nil {
				continue
			}
			if strings.EqualFold(child.Type, "footer") {
				node.Footer = append(node.Footer, cn)
			} else {
				node.Children = append(node.Children, cn)
			}
		}

	case types.WidgetListView:
		node.DataSource = astDataSourceToModel(propDS(w, "datasource"))

	case types.WidgetDataGrid:
		node.DataSource = astDataSourceToModel(propDS(w, "datasource"))
		dgp := &types.DataGridProps{}
		for _, c := range w.Children {
			if strings.ToLower(c.Type) == "column" {
				dgp.Columns = append(dgp.Columns, types.ColumnDef{
					Name:      c.Name,
					Attribute: propStr(c, "attribute"),
					Caption:   propStr(c, "caption"),
				})
			}
		}
		// DataGrid columns are extracted into DataGrid.Columns; clear the
		// generic Children slice so the renderer doesn't emit them twice.
		node.Children = nil
		node.DataGrid = dgp

	case types.WidgetButton:
		node.Caption = propStr(w, "caption")
		node.ButtonStyle = propStr(w, "buttonstyle")
		node.OnClick = propActionTarget(w, "action")

	case types.WidgetLabel:
		node.Caption = propStr(w, "caption")

	case types.WidgetText, types.WidgetTitle, types.WidgetDynamicText:
		node.Content = propStr(w, "content")

	case types.WidgetTextBox, types.WidgetTextArea, types.WidgetDatePicker,
		types.WidgetRadioButtons, types.WidgetCheckBox:
		node.EntityAttr = propStr(w, "attribute")
		node.Editable = propStr(w, "editable")

	case types.WidgetSnippet:
		node.Snippet = &types.SnippetProps{SnippetName: propStr(w, "snippet")}

	case types.WidgetComboBox:
		node.DataSource = astDataSourceToModel(propDS(w, "datasource"))
		node.EntityAttr = propStr(w, "attribute")

	case types.WidgetImage:
		node.Image = &types.ImageProps{
			URL:     propStr(w, "imageurl"),
			AltText: propStr(w, "alttext"),
			Width:   propStr(w, "width"),
			Height:  propStr(w, "height"),
		}
	}

	return node, nil
}

// astWidgetKind maps a (lowercased) AST widget type string to WidgetKind.
// Accepts both the grammar's canonical keywords (actionbutton, tabpage,
// statictext, snippetcall, scrollcontainer) and the IR shorthand used by
// Task 3 tests (button, tab, label, snippet, scrollview) for compatibility.
func astWidgetKind(astType string) types.WidgetKind {
	switch astType {
	case "container":
		return types.WidgetContainer
	case "scrollview", "scrollcontainer":
		return types.WidgetScrollView
	case "groupbox":
		return types.WidgetGroupBox
	case "layoutgrid":
		return types.WidgetLayoutGrid
	case "row":
		return types.WidgetLayoutRow
	case "column":
		return types.WidgetLayoutCol
	case "tabcontainer":
		return types.WidgetTabContainer
	case "tab", "tabpage":
		return types.WidgetTabPage
	case "dataview":
		return types.WidgetDataView
	case "listview":
		return types.WidgetListView
	case "datagrid":
		return types.WidgetDataGrid
	case "gallery":
		return types.WidgetGallery
	case "button", "actionbutton":
		return types.WidgetButton
	case "textbox":
		return types.WidgetTextBox
	case "textarea":
		return types.WidgetTextArea
	case "datepicker":
		return types.WidgetDatePicker
	case "radiobuttons":
		return types.WidgetRadioButtons
	case "checkbox":
		return types.WidgetCheckBox
	case "label", "statictext":
		return types.WidgetLabel
	case "text":
		return types.WidgetText
	case "title":
		return types.WidgetTitle
	case "dynamictext":
		return types.WidgetDynamicText
	case "snippet", "snippetcall":
		return types.WidgetSnippet
	case "combobox":
		return types.WidgetComboBox
	case "image":
		return types.WidgetImage
	case "navigationlist":
		return types.WidgetNavList
	}
	return types.WidgetUnknown
}

// astDataSourceToModel converts an AST data source node to DataSourceDef.
func astDataSourceToModel(ds *ast.DataSourceV3) *types.DataSourceDef {
	if ds == nil {
		return nil
	}
	def := &types.DataSourceDef{XPathConstraint: ds.Where}
	switch ds.Type {
	case "database":
		def.Kind = types.DataSourceDatabase
		def.Entity = ds.Reference
	case "microflow":
		def.Kind = types.DataSourceMicroflow
		def.Reference = ds.Reference
	case "nanoflow":
		def.Kind = types.DataSourceNanoflow
		def.Reference = ds.Reference
	case "parameter":
		def.Kind = types.DataSourceParameter
		def.Reference = ds.Reference
	case "selection":
		def.Kind = types.DataSourceSelection
		def.Reference = ds.Reference
	default:
		return nil
	}
	return def
}

// propStr reads a string property by case-insensitive key from a widget's
// Properties map. Returns "" if missing or not a string.
func propStr(w *ast.WidgetV3, key string) string {
	if w == nil || w.Properties == nil {
		return ""
	}
	for k, v := range w.Properties {
		if strings.EqualFold(k, key) {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// propInt reads an int property by case-insensitive key. Accepts int, int32,
// int64, or numeric strings.
func propInt(w *ast.WidgetV3, key string) int {
	if w == nil || w.Properties == nil {
		return 0
	}
	for k, v := range w.Properties {
		if strings.EqualFold(k, key) {
			switch n := v.(type) {
			case int:
				return n
			case int32:
				return int(n)
			case int64:
				return int(n)
			case float64:
				return int(n)
			case string:
				if strings.EqualFold(n, "AutoFill") {
					return -1
				}
				if i, err := strconv.Atoi(n); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

// propDS reads a *ast.DataSourceV3 property by case-insensitive key.
func propDS(w *ast.WidgetV3, key string) *ast.DataSourceV3 {
	if w == nil || w.Properties == nil {
		return nil
	}
	for k, v := range w.Properties {
		if strings.EqualFold(k, key) {
			if ds, ok := v.(*ast.DataSourceV3); ok {
				return ds
			}
		}
	}
	return nil
}

// propActionTarget extracts the qualified target name from an ActionV3
// property (microflow/nanoflow/showPage targets all stored in Target).
func propActionTarget(w *ast.WidgetV3, key string) string {
	if w == nil || w.Properties == nil {
		return ""
	}
	for k, v := range w.Properties {
		if strings.EqualFold(k, key) {
			if act, ok := v.(*ast.ActionV3); ok && act != nil {
				return act.Target
			}
		}
	}
	return ""
}

// pageParamTypeRefString renders the entity type of a PageParameter as the
// qualified name string used in MDL syntax. PageParameter exposes EntityType
// (entity refs) and Type (primitives + enums); only entity refs round-trip
// to a single qualified name today — primitive page params are deferred to
// Task 9 cleanup.
func pageParamTypeRefString(p ast.PageParameter) string {
	if p.EntityType.Name != "" {
		return p.EntityType.String()
	}
	// For primitives, DataType.Kind carries the token but printing it back to
	// MDL requires a small formatter not yet ported into this package. Leave
	// blank for now; PageParameter currently used by tests are entity-typed.
	return ""
}
