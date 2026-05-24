// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// mdlQuote wraps a string in single quotes and escapes MDL-sensitive characters.
func mdlQuote(s string) string {
	escaped := strings.NewReplacer(
		"\\", "\\\\",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
		"'", "''",
	).Replace(s)
	return "'" + escaped + "'"
}

// appendDataGridPagingProps appends non-default paging properties for DataGrid2.
func appendDataGridPagingProps(props []string, w rawWidget) []string {
	if w.PageSize != "" && w.PageSize != "20" {
		props = append(props, fmt.Sprintf("pagesize: %s", w.PageSize))
	}
	if w.Pagination != "" && w.Pagination != "buttons" {
		props = append(props, fmt.Sprintf("pagination: %s", w.Pagination))
	}
	if w.PagingPosition != "" && w.PagingPosition != "bottom" {
		props = append(props, fmt.Sprintf("pagingposition: %s", w.PagingPosition))
	}
	if w.ShowPagingButtons != "" && w.ShowPagingButtons != "always" {
		props = append(props, fmt.Sprintf("showpagingbuttons: %s", w.ShowPagingButtons))
	}
	// showNumberOfRows: not yet fully supported in DataGrid2, skip to avoid CE0463
	return props
}

// appendConditionalProps appends VISIBLE IF and EDITABLE IF if present.
func appendConditionalProps(props []string, w rawWidget) []string {
	if w.VisibleIf != "" {
		props = append(props, fmt.Sprintf("visible: [%s]", w.VisibleIf))
	}
	if w.EditableIf != "" {
		props = append(props, fmt.Sprintf("editable: [%s]", w.EditableIf))
	}
	return props
}

// appendAppearanceProps appends Class, Style, DesignProperties, and conditional settings if present.
func appendAppearanceProps(props []string, w rawWidget) []string {
	if w.Class != "" {
		props = append(props, fmt.Sprintf("class: %s", mdlQuote(w.Class)))
	}
	if w.Style != "" {
		props = append(props, fmt.Sprintf("style: %s", mdlQuote(w.Style)))
	}
	if len(w.DesignProperties) > 0 {
		props = append(props, formatDesignPropertiesMDL(w.DesignProperties))
	}
	if w.VisibleIf != "" {
		props = append(props, fmt.Sprintf("visible: [%s]", w.VisibleIf))
	}
	if w.EditableIf != "" {
		props = append(props, fmt.Sprintf("editable: [%s]", w.EditableIf))
	}
	return props
}

// formatDesignPropertiesMDL formats design properties as MDL V3 syntax.
// Toggle → 'Key': ON, Option → 'Key': 'Value'
func formatDesignPropertiesMDL(dps []rawDesignProp) string {
	var entries []string
	for _, dp := range dps {
		switch dp.ValueType {
		case "toggle":
			entries = append(entries, fmt.Sprintf("%s: on", mdlQuote(dp.Key)))
		case "option":
			entries = append(entries, fmt.Sprintf("%s: %s", mdlQuote(dp.Key), mdlQuote(dp.Option)))
		}
	}
	return fmt.Sprintf("designproperties: [%s]", strings.Join(entries, ", "))
}

// formatWidgetProps writes a widget line with automatic multi-line wrapping.
// If the single-line form exceeds 120 chars, each property is written on its own line.
// header is the widget keyword + name (e.g. "DATAGRID ProductGrid"),
// suffix is the trailing content (e.g. "\n" or " {\n").
func formatWidgetProps(w io.Writer, prefix string, header string, props []string, suffix string) {
	if len(props) == 0 {
		fmt.Fprintf(w, "%s%s%s", prefix, header, suffix)
		return
	}
	singleLine := fmt.Sprintf("%s%s (%s)%s", prefix, header, strings.Join(props, ", "), suffix)
	if len(singleLine) <= 120 {
		fmt.Fprint(w, singleLine)
		return
	}
	// Multi-line
	indent := prefix + "  "
	fmt.Fprintf(w, "%s%s (\n", prefix, header)
	for i, p := range props {
		if i < len(props)-1 {
			fmt.Fprintf(w, "%s%s,\n", indent, p)
		} else {
			fmt.Fprintf(w, "%s%s\n", indent, p)
		}
	}
	fmt.Fprintf(w, "%s)%s", prefix, suffix)
}

// outputDataContainerContext writes a comment showing available variables inside a data container.
// isList indicates list containers (DataGrid2, ListView, Gallery) where a selection variable is available.
func outputDataContainerContext(w io.Writer, prefix string, widgetName string, entityRef string, isList bool) {
	if entityRef == "" {
		return
	}
	parts := []string{fmt.Sprintf("$currentObject (%s)", entityRef)}
	if isList && widgetName != "" {
		parts = append(parts, fmt.Sprintf("$%s (selection)", widgetName))
	}
	fmt.Fprintf(w, "%s-- Context: %s\n", prefix, strings.Join(parts, ", "))
}

// outputWidgetMDLV3 outputs a widget in MDL V3 syntax.
// V3 syntax uses WIDGET Name (Props) { children } format.
func outputWidgetMDLV3(ctx *ExecContext, w rawWidget, indent int) {
	prefix := strings.Repeat("  ", indent)

	switch w.Type {
	case "Forms$ScrollContainer", "Pages$ScrollContainer":
		header := fmt.Sprintf("scrollcontainer %s", w.Name)
		props := appendAppearanceProps(nil, w)
		if len(w.Children) > 0 {
			formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
			for _, child := range w.Children {
				outputWidgetMDLV3(ctx, child, indent+1)
			}
			fmt.Fprintf(ctx.Output, "%s}\n", prefix)
		} else {
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		}

	case "Forms$TabControl", "Pages$TabControl":
		header := fmt.Sprintf("tabcontainer %s", w.Name)
		props := appendAppearanceProps(nil, w)
		if len(w.Children) > 0 {
			formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
			for _, child := range w.Children {
				outputWidgetMDLV3(ctx, child, indent+1)
			}
			fmt.Fprintf(ctx.Output, "%s}\n", prefix)
		} else {
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		}

	case "Pages$TabPage":
		header := fmt.Sprintf("tabpage %s", w.Name)
		var props []string
		if w.TabCaption != "" {
			props = append(props, fmt.Sprintf("caption: %s", mdlQuote(w.TabCaption)))
		}
		if len(w.Children) > 0 {
			formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
			for _, child := range w.Children {
				outputWidgetMDLV3(ctx, child, indent+1)
			}
			fmt.Fprintf(ctx.Output, "%s}\n", prefix)
		} else {
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		}

	case "Forms$DivContainer", "Pages$DivContainer":
		header := fmt.Sprintf("container %s", w.Name)
		props := appendAppearanceProps(nil, w)
		if len(w.Children) > 0 {
			formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
			for _, child := range w.Children {
				outputWidgetMDLV3(ctx, child, indent+1)
			}
			fmt.Fprintf(ctx.Output, "%s}\n", prefix)
		} else {
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		}

	case "Forms$GroupBox", "Pages$GroupBox":
		header := fmt.Sprintf("groupbox %s", w.Name)
		props := []string{}
		if w.Caption != "" {
			props = append(props, fmt.Sprintf("caption: %s", mdlQuote(w.Caption)))
		}
		if w.HeaderMode != "" && w.HeaderMode != "Div" {
			props = append(props, fmt.Sprintf("headermode: %s", w.HeaderMode))
		}
		if w.Collapsible != "" && w.Collapsible != "No" {
			switch w.Collapsible {
			case "YesInitiallyExpanded":
				props = append(props, "collapsible: YesExpanded")
			case "YesInitiallyCollapsed":
				props = append(props, "collapsible: YesCollapsed")
			default:
				props = append(props, fmt.Sprintf("collapsible: %s", w.Collapsible))
			}
		}
		props = appendAppearanceProps(props, w)
		if len(w.Children) > 0 {
			formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
			for _, child := range w.Children {
				outputWidgetMDLV3(ctx, child, indent+1)
			}
			fmt.Fprintf(ctx.Output, "%s}\n", prefix)
		} else {
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		}

	case "Forms$LayoutGrid", "Pages$LayoutGrid":
		header := "layoutgrid"
		if w.Name != "" {
			header += " " + w.Name
		}
		props := appendAppearanceProps(nil, w)
		formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
		for rowIdx, row := range w.Rows {
			fmt.Fprintf(ctx.Output, "%s  row row%d {\n", prefix, rowIdx+1)
			for colIdx, col := range row.Columns {
				var colProps []string
				widthStr := "AutoFill"
				if col.Width > 0 && col.Width <= 12 {
					widthStr = fmt.Sprintf("%d", col.Width)
				}
				colProps = append(colProps, "desktopwidth: "+widthStr)
				if col.TabletWidth > 0 && col.TabletWidth <= 12 {
					colProps = append(colProps, fmt.Sprintf("tabletwidth: %d", col.TabletWidth))
				}
				if col.PhoneWidth > 0 && col.PhoneWidth <= 12 {
					colProps = append(colProps, fmt.Sprintf("phonewidth: %d", col.PhoneWidth))
				}
				fmt.Fprintf(ctx.Output, "%s    column col%d (%s) {\n", prefix, colIdx+1, strings.Join(colProps, ", "))
				for _, cw := range col.Widgets {
					outputWidgetMDLV3(ctx, cw, indent+3)
				}
				fmt.Fprintf(ctx.Output, "%s    }\n", prefix)
			}
			fmt.Fprintf(ctx.Output, "%s  }\n", prefix)
		}
		fmt.Fprintf(ctx.Output, "%s}\n", prefix)

	case "Forms$DynamicText", "Pages$DynamicText":
		header := fmt.Sprintf("dynamictext %s", w.Name)
		props := []string{}
		if w.Content != "" {
			props = append(props, fmt.Sprintf("content: %s", mdlQuote(w.Content)))
		}
		if w.RenderMode != "" && w.RenderMode != "Text" {
			props = append(props, fmt.Sprintf("rendermode: %s", w.RenderMode))
		}
		if len(w.Parameters) > 0 {
			props = append(props, fmt.Sprintf("contentparams: [%s]", strings.Join(formatParametersV3(w.Parameters), ", ")))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "Forms$ActionButton", "Pages$ActionButton":
		header := fmt.Sprintf("actionbutton %s", w.Name)
		props := []string{}
		if w.Caption != "" {
			props = append(props, fmt.Sprintf("caption: %s", mdlQuote(w.Caption)))
		}
		if len(w.Parameters) > 0 {
			props = append(props, fmt.Sprintf("contentparams: [%s]", strings.Join(formatParametersV3(w.Parameters), ", ")))
		}
		if w.Action != "" {
			props = append(props, fmt.Sprintf("action: %s", w.Action))
		}
		if w.ButtonStyle != "" && w.ButtonStyle != "Default" {
			props = append(props, fmt.Sprintf("buttonstyle: %s", w.ButtonStyle))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "Forms$Text", "Pages$Text":
		props := []string{}
		if w.Content != "" {
			props = append(props, fmt.Sprintf("content: %s", mdlQuote(w.Content)))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, "statictext", props, "\n")

	case "Forms$Title", "Pages$Title":
		header := fmt.Sprintf("title %s", w.Name)
		props := []string{}
		if w.Caption != "" {
			props = append(props, fmt.Sprintf("content: %s", mdlQuote(w.Caption)))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "Forms$DataView", "Pages$DataView":
		header := fmt.Sprintf("dataview %s", w.Name)
		props := []string{}
		if w.DataSource != nil {
			switch w.DataSource.Type {
			case "microflow":
				props = append(props, fmt.Sprintf("datasource: microflow %s", w.DataSource.Reference))
			case "nanoflow":
				props = append(props, fmt.Sprintf("datasource: nanoflow %s", w.DataSource.Reference))
			case "parameter":
				props = append(props, fmt.Sprintf("datasource: $%s", w.DataSource.Reference))
			}
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
		outputDataContainerContext(ctx.Output, prefix+"  ", w.Name, w.EntityContext, false)
		for _, child := range w.Children {
			outputWidgetMDLV3(ctx, child, indent+1)
		}
		fmt.Fprintf(ctx.Output, "%s}\n", prefix)

	case "Forms$TextBox", "Pages$TextBox":
		header := fmt.Sprintf("textbox %s", w.Name)
		props := []string{}
		if w.Caption != "" {
			props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
		}
		if w.Content != "" {
			props = append(props, fmt.Sprintf("attribute: %s", w.Content))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "Forms$TextArea", "Pages$TextArea":
		header := fmt.Sprintf("textarea %s", w.Name)
		props := []string{}
		if w.Caption != "" {
			props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
		}
		if w.Content != "" {
			props = append(props, fmt.Sprintf("attribute: %s", w.Content))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "Forms$DatePicker", "Pages$DatePicker":
		header := fmt.Sprintf("datepicker %s", w.Name)
		props := []string{}
		if w.Caption != "" {
			props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
		}
		if w.Content != "" {
			props = append(props, fmt.Sprintf("attribute: %s", w.Content))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "Forms$RadioButtons", "Pages$RadioButtons":
		header := fmt.Sprintf("radiobuttons %s", w.Name)
		props := []string{}
		if w.Caption != "" {
			props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
		}
		if w.Content != "" {
			props = append(props, fmt.Sprintf("attribute: %s", w.Content))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "Forms$CheckBox", "Pages$CheckBox":
		header := fmt.Sprintf("checkbox %s", w.Name)
		props := []string{}
		if w.Caption != "" {
			props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
		}
		if w.Content != "" {
			props = append(props, fmt.Sprintf("attribute: %s", w.Content))
		}
		// Show Editable if not default "Always"
		if w.Editable != "" && w.Editable != "Always" {
			props = append(props, fmt.Sprintf("editable: %s", w.Editable))
		}
		// Show ReadOnlyStyle if not default "Inherit"
		if w.ReadOnlyStyle != "" && w.ReadOnlyStyle != "Inherit" {
			props = append(props, fmt.Sprintf("readonlystyle: %s", w.ReadOnlyStyle))
		}
		// Show ShowLabel if false (not showing label)
		if !w.ShowLabel {
			props = append(props, "showlabel: No")
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "CustomWidgets$CustomWidget":
		widgetType := w.RenderMode // We stored widget type in RenderMode
		if widgetType == "" {
			widgetType = "customwidget"
		}
		// Handle DataGrid2 specially with datasource and columns
		if widgetType == "datagrid2" && (w.DataSource != nil || len(w.DataGridColumns) > 0) {
			header := fmt.Sprintf("datagrid %s", w.Name)
			props := []string{}
			if w.DataSource != nil {
				switch w.DataSource.Type {
				case "database":
					dsVal := fmt.Sprintf("database from %s", w.DataSource.Reference)
					if w.DataSource.XPathConstraint != "" {
						xpath := w.DataSource.XPathConstraint
						if len(xpath) >= 2 && xpath[0] == '[' && xpath[len(xpath)-1] == ']' {
							xpath = xpath[1 : len(xpath)-1]
						}
						dsVal += fmt.Sprintf(" where %s", xpath)
					}
					if len(w.DataSource.SortColumns) > 0 {
						var sortParts []string
						for _, col := range w.DataSource.SortColumns {
							sortParts = append(sortParts, col.Attribute+" "+col.Order)
						}
						dsVal += fmt.Sprintf(" sort by %s", strings.Join(sortParts, ", "))
					}
					props = append(props, fmt.Sprintf("datasource: %s", dsVal))
				case "microflow":
					props = append(props, fmt.Sprintf("datasource: microflow %s", w.DataSource.Reference))
				case "parameter":
					props = append(props, fmt.Sprintf("datasource: %s", w.DataSource.Reference))
				}
			}
			// Add selection mode if specified
			if w.Selection != "" {
				props = append(props, fmt.Sprintf("selection: %s", w.Selection))
			}
			// Add paging properties if non-default
			props = appendDataGridPagingProps(props, w)
			props = appendAppearanceProps(props, w)
			// Output CONTROLBAR and columns as children
			hasContent := len(w.ControlBar) > 0 || len(w.DataGridColumns) > 0
			if hasContent {
				formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
				outputDataContainerContext(ctx.Output, prefix+"  ", w.Name, w.EntityContext, true)
				// Output CONTROLBAR section if control bar widgets present
				if len(w.ControlBar) > 0 {
					fmt.Fprintf(ctx.Output, "%s  controlbar controlBar1 {\n", prefix)
					for _, cb := range w.ControlBar {
						outputWidgetMDLV3(ctx, cb, indent+2)
					}
					fmt.Fprintf(ctx.Output, "%s  }\n", prefix)
				}
				// Output columns — derive name from attribute or caption, fall back to col%d
				for i, col := range w.DataGridColumns {
					colName := deriveColumnName(col, i)
					outputDataGrid2ColumnV3(ctx, prefix+"  ", colName, col)
				}
				fmt.Fprintf(ctx.Output, "%s}\n", prefix)
			} else {
				formatWidgetProps(ctx.Output, prefix, header, props, "\n")
			}
		} else if widgetType == "gallery" {
			// Handle Gallery specially with datasource, selection, filter and content widgets
			header := fmt.Sprintf("gallery %s", w.Name)
			props := []string{}
			if w.DataSource != nil {
				switch w.DataSource.Type {
				case "database":
					dsVal := fmt.Sprintf("database from %s", w.DataSource.Reference)
					if w.DataSource.XPathConstraint != "" {
						xpath := w.DataSource.XPathConstraint
						if len(xpath) >= 2 && xpath[0] == '[' && xpath[len(xpath)-1] == ']' {
							xpath = xpath[1 : len(xpath)-1]
						}
						dsVal += fmt.Sprintf(" where %s", xpath)
					}
					// Add SORT BY if present
					if len(w.DataSource.SortColumns) > 0 {
						var sortParts []string
						for _, col := range w.DataSource.SortColumns {
							sortParts = append(sortParts, col.Attribute+" "+col.Order)
						}
						dsVal += fmt.Sprintf(" sort by %s", strings.Join(sortParts, ", "))
					}
					props = append(props, fmt.Sprintf("datasource: %s", dsVal))
				case "microflow":
					props = append(props, fmt.Sprintf("datasource: microflow %s", w.DataSource.Reference))
				}
			}
			// Add column counts if non-default
			if w.DesktopColumns != "" && w.DesktopColumns != "1" {
				props = append(props, fmt.Sprintf("desktopcolumns: %s", w.DesktopColumns))
			}
			if w.TabletColumns != "" && w.TabletColumns != "1" {
				props = append(props, fmt.Sprintf("tabletcolumns: %s", w.TabletColumns))
			}
			if w.PhoneColumns != "" && w.PhoneColumns != "1" {
				props = append(props, fmt.Sprintf("phonecolumns: %s", w.PhoneColumns))
			}
			// Add Selection mode if specified
			if w.Selection != "" {
				props = append(props, fmt.Sprintf("selection: %s", w.Selection))
			}
			props = appendAppearanceProps(props, w)
			// Output filter and content widgets
			hasContent := len(w.Children) > 0 || len(w.FilterWidgets) > 0
			if hasContent {
				formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
				outputDataContainerContext(ctx.Output, prefix+"  ", w.Name, w.EntityContext, true)
				// Output FILTER section if filter widgets present
				if len(w.FilterWidgets) > 0 {
					fmt.Fprintf(ctx.Output, "%s  filter filter1 {\n", prefix)
					for _, filter := range w.FilterWidgets {
						outputWidgetMDLV3(ctx, filter, indent+2)
					}
					fmt.Fprintf(ctx.Output, "%s  }\n", prefix)
				}
				// Output TEMPLATE section if content widgets present
				if len(w.Children) > 0 {
					fmt.Fprintf(ctx.Output, "%s  template template1 {\n", prefix)
					for _, child := range w.Children {
						outputWidgetMDLV3(ctx, child, indent+2)
					}
					fmt.Fprintf(ctx.Output, "%s  }\n", prefix)
				}
				fmt.Fprintf(ctx.Output, "%s}\n", prefix)
			} else {
				formatWidgetProps(ctx.Output, prefix, header, props, "\n")
			}
		} else if widgetType == "image" {
			header := fmt.Sprintf("image %s", w.Name)
			props := []string{}
			if w.ImageType != "" && w.ImageType != "image" {
				props = append(props, fmt.Sprintf("imagetype: %s", w.ImageType))
			}
			if w.ImageUrl != "" {
				props = append(props, fmt.Sprintf("imageurl: %s", mdlQuote(w.ImageUrl)))
			}
			if w.AlternativeText != "" {
				props = append(props, fmt.Sprintf("alternativetext: %s", mdlQuote(w.AlternativeText)))
			}
			if w.WidthUnit != "" && w.WidthUnit != "auto" {
				props = append(props, fmt.Sprintf("widthunit: %s", w.WidthUnit))
			}
			if w.ImageWidth != "" && w.ImageWidth != "100" {
				props = append(props, fmt.Sprintf("width: %s", w.ImageWidth))
			}
			if w.HeightUnit != "" && w.HeightUnit != "auto" {
				props = append(props, fmt.Sprintf("heightunit: %s", w.HeightUnit))
			}
			if w.ImageHeight != "" && w.ImageHeight != "100" {
				props = append(props, fmt.Sprintf("height: %s", w.ImageHeight))
			}
			if w.DisplayAs != "" && w.DisplayAs != "fullImage" {
				props = append(props, fmt.Sprintf("displayas: %s", w.DisplayAs))
			}
			if w.Responsive != "" && w.Responsive != "true" {
				props = append(props, fmt.Sprintf("responsive: %s", w.Responsive))
			}
			if w.OnClickType == "enlarge" {
				props = append(props, "onclicktype: enlarge")
			}
			if w.Action != "" {
				props = append(props, fmt.Sprintf("onclick: %s", w.Action))
			}
			props = appendConditionalProps(props, w)
			props = appendAppearanceProps(props, w)
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		} else if len(w.ExplicitProperties) > 0 && w.WidgetID != "" {
			// Generic pluggable widget with explicit properties
			header := fmt.Sprintf("pluggablewidget '%s' %s", w.WidgetID, w.Name)
			props := []string{}
			if w.Caption != "" {
				props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
			}
			for _, ep := range w.ExplicitProperties {
				props = append(props, fmt.Sprintf("%s: %s", ep.Key, ep.Value))
			}
			props = appendAppearanceProps(props, w)
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		} else {
			header := fmt.Sprintf("%s %s", widgetType, w.Name)
			props := []string{}
			if w.Caption != "" {
				props = append(props, fmt.Sprintf("label: %s", mdlQuote(w.Caption)))
			}
			if w.Content != "" {
				props = append(props, fmt.Sprintf("attribute: %s", w.Content))
			}
			// Show DataSource and CaptionAttribute for ComboBox association mode
			if w.DataSource != nil && widgetType == "combobox" {
				switch w.DataSource.Type {
				case "database":
					props = append(props, fmt.Sprintf("datasource: database from %s", w.DataSource.Reference))
				case "microflow":
					props = append(props, fmt.Sprintf("datasource: microflow %s", w.DataSource.Reference))
				}
				if w.CaptionAttribute != "" {
					props = append(props, fmt.Sprintf("captionattribute: %s", w.CaptionAttribute))
				}
			}
			// Show filter attributes for filter widgets
			if len(w.FilterAttributes) > 0 {
				props = append(props, fmt.Sprintf("attributes: [%s]", strings.Join(w.FilterAttributes, ", ")))
			}
			// Show filter expression if not default
			if w.FilterExpression != "" && w.FilterExpression != "contains" {
				props = append(props, fmt.Sprintf("filtertype: %s", w.FilterExpression))
			}
			props = appendAppearanceProps(props, w)
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		}

	case "Forms$NavigationList", "Pages$NavigationList":
		fmt.Fprintf(ctx.Output, "%snavigationlist %s {\n", prefix, w.Name)
		for _, child := range w.Children {
			itemHeader := fmt.Sprintf("item %s", child.Name)
			props := []string{}
			if child.Action != "" {
				props = append(props, fmt.Sprintf("action: %s", child.Action))
			}
			if child.ButtonStyle != "" && child.ButtonStyle != "Default" {
				props = append(props, fmt.Sprintf("buttonstyle: %s", child.ButtonStyle))
			}
			formatWidgetProps(ctx.Output, prefix+"  ", itemHeader, props, " {\n")
			for _, cw := range child.Children {
				outputWidgetMDLV3(ctx, cw, indent+2)
			}
			fmt.Fprintf(ctx.Output, "%s  }\n", prefix)
		}
		fmt.Fprintf(ctx.Output, "%s}\n", prefix)

	case "Forms$Label", "Pages$Label":
		fmt.Fprintf(ctx.Output, "%sstatictext (Content: %s)\n", prefix, mdlQuote(w.Content))

	case "Forms$Gallery", "Pages$Gallery":
		header := fmt.Sprintf("gallery %s", w.Name)
		props := []string{}
		if w.DataSource != nil {
			switch w.DataSource.Type {
			case "database":
				dsVal := fmt.Sprintf("database from %s", w.DataSource.Reference)
				if w.DataSource.XPathConstraint != "" {
					xpath := w.DataSource.XPathConstraint
					if len(xpath) >= 2 && xpath[0] == '[' && xpath[len(xpath)-1] == ']' {
						xpath = xpath[1 : len(xpath)-1]
					}
					dsVal += fmt.Sprintf(" where %s", xpath)
				}
				if len(w.DataSource.SortColumns) > 0 {
					var sortParts []string
					for _, col := range w.DataSource.SortColumns {
						sortParts = append(sortParts, col.Attribute+" "+col.Order)
					}
					dsVal += fmt.Sprintf(" sort by %s", strings.Join(sortParts, ", "))
				}
				props = append(props, fmt.Sprintf("datasource: %s", dsVal))
			case "microflow":
				props = append(props, fmt.Sprintf("datasource: microflow %s", w.DataSource.Reference))
			case "parameter":
				props = append(props, fmt.Sprintf("datasource: %s", w.DataSource.Reference))
			}
		}
		props = appendAppearanceProps(props, w)
		if len(w.Children) > 0 {
			formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
			outputDataContainerContext(ctx.Output, prefix+"  ", w.Name, w.EntityContext, true)
			for _, child := range w.Children {
				outputWidgetMDLV3(ctx, child, indent+1)
			}
			fmt.Fprintf(ctx.Output, "%s}\n", prefix)
		} else {
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		}

	case "Forms$SnippetCallWidget", "Pages$SnippetCallWidget":
		header := fmt.Sprintf("snippetcall %s", w.Name)
		props := []string{}
		if w.Content != "" {
			props = append(props, fmt.Sprintf("snippet: %s", w.Content))
		}
		props = appendAppearanceProps(props, w)
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")

	case "Footer":
		fmt.Fprintf(ctx.Output, "%sfooter %s {\n", prefix, w.Name)
		for _, child := range w.Children {
			outputWidgetMDLV3(ctx, child, indent+1)
		}
		fmt.Fprintf(ctx.Output, "%s}\n", prefix)

	case "Forms$ListView", "Pages$ListView":
		// ListView (also used for Gallery serialization)
		header := fmt.Sprintf("listview %s", w.Name)
		props := []string{}
		if w.DataSource != nil {
			switch w.DataSource.Type {
			case "database":
				dsVal := fmt.Sprintf("database from %s", w.DataSource.Reference)
				if w.DataSource.XPathConstraint != "" {
					xpath := w.DataSource.XPathConstraint
					if len(xpath) >= 2 && xpath[0] == '[' && xpath[len(xpath)-1] == ']' {
						xpath = xpath[1 : len(xpath)-1]
					}
					dsVal += fmt.Sprintf(" where %s", xpath)
				}
				props = append(props, fmt.Sprintf("datasource: %s", dsVal))
			case "microflow":
				props = append(props, fmt.Sprintf("datasource: microflow %s", w.DataSource.Reference))
			case "nanoflow":
				props = append(props, fmt.Sprintf("datasource: nanoflow %s", w.DataSource.Reference))
			case "parameter":
				props = append(props, fmt.Sprintf("datasource: %s", w.DataSource.Reference))
			}
		}
		props = appendAppearanceProps(props, w)
		if len(w.Children) > 0 {
			formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
			outputDataContainerContext(ctx.Output, prefix+"  ", w.Name, w.EntityContext, true)
			for _, child := range w.Children {
				outputWidgetMDLV3(ctx, child, indent+1)
			}
			fmt.Fprintf(ctx.Output, "%s}\n", prefix)
		} else {
			formatWidgetProps(ctx.Output, prefix, header, props, "\n")
		}

	default:
		// Output unknown widget type as comment
		fmt.Fprintf(ctx.Output, "%s-- %s", prefix, w.Type)
		if w.Name != "" {
			fmt.Fprintf(ctx.Output, " (%s)", w.Name)
		}
		fmt.Fprint(ctx.Output, "\n")
	}
}

// deriveColumnName produces a semantic column name from the column's attribute
// or caption. Always prefixes with "col" so the result is never a reserved
// keyword (e.g. "Status" → "colStatus"). Falls back to "col%d".
func deriveColumnName(col rawDataGridColumn, index int) string {
	if col.Attribute != "" {
		// Use the short attribute name (last segment after dot), with "col" prefix
		// to avoid reserved-keyword collisions (Status, Title, Content, etc.).
		parts := strings.Split(col.Attribute, ".")
		return "col" + parts[len(parts)-1]
	}
	if col.Caption != "" {
		// Sanitize caption to a valid identifier: keep alphanumeric, replace rest with underscore
		sanitized := strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
				return r
			}
			return '_'
		}, col.Caption)
		// Trim leading/trailing underscores and collapse multiples
		result := strings.TrimFunc(sanitized, func(r rune) bool { return r == '_' })
		if result != "" {
			return "col" + result
		}
	}
	return fmt.Sprintf("col%d", index+1)
}

// outputDataGrid2ColumnV3 outputs a single DataGrid2 column in V3 MDL syntax.
func outputDataGrid2ColumnV3(ctx *ExecContext, prefix, colName string, col rawDataGridColumn) {
	// Build the main column properties
	var props []string
	if col.Attribute != "" {
		props = append(props, fmt.Sprintf("attribute: %s", col.Attribute))
	}
	if col.Caption != "" {
		props = append(props, fmt.Sprintf("caption: %s", mdlQuote(col.Caption)))
	}
	if len(col.CaptionParams) > 0 {
		props = append(props, fmt.Sprintf("captionparams: [%s]", strings.Join(formatParametersV3(col.CaptionParams), ", ")))
	}
	// Add ShowContentAs if not default "attribute"
	if col.ShowContentAs != "" && col.ShowContentAs != "attribute" {
		props = append(props, fmt.Sprintf("ShowContentAs: %s", col.ShowContentAs))
	}
	// Add DynamicText content when ShowContentAs is dynamicText
	if col.ShowContentAs == "dynamicText" && col.DynamicText != "" {
		props = append(props, fmt.Sprintf("content: %s", mdlQuote(col.DynamicText)))
		if len(col.DynamicTextParams) > 0 {
			props = append(props, fmt.Sprintf("contentparams: [%s]", strings.Join(formatParametersV3(col.DynamicTextParams), ", ")))
		}
	}
	// Add column styling properties if non-default
	if col.Alignment != "" && col.Alignment != "left" {
		props = append(props, fmt.Sprintf("Alignment: %s", col.Alignment))
	}
	if col.WrapText == "true" {
		props = append(props, "WrapText: true")
	}
	// Sortable: default depends on whether attribute is bound
	if col.Sortable != "" {
		defaultSortable := "true"
		if col.Attribute == "" {
			defaultSortable = "false"
		}
		if col.Sortable != defaultSortable {
			props = append(props, fmt.Sprintf("Sortable: %s", col.Sortable))
		}
	}
	if col.Resizable == "false" {
		props = append(props, "Resizable: false")
	}
	if col.Draggable == "false" {
		props = append(props, "Draggable: false")
	}
	if col.Hidable != "" && col.Hidable != "yes" {
		props = append(props, fmt.Sprintf("Hidable: %s", col.Hidable))
	}
	if col.ColumnWidth != "" && col.ColumnWidth != "autoFill" {
		props = append(props, fmt.Sprintf("ColumnWidth: %s", col.ColumnWidth))
	}
	if col.ColumnWidth == "manual" && col.Size != "" && col.Size != "1" {
		props = append(props, fmt.Sprintf("Size: %s", col.Size))
	}
	if col.Visible != "" && col.Visible != "true" {
		props = append(props, fmt.Sprintf("visible: %s", mdlQuote(col.Visible)))
	}
	if col.DynamicCellClass != "" {
		props = append(props, fmt.Sprintf("DynamicCellClass: %s", mdlQuote(col.DynamicCellClass)))
	}
	if col.Tooltip != "" {
		props = append(props, fmt.Sprintf("Tooltip: %s", mdlQuote(col.Tooltip)))
	}

	// Check if we have content widgets to display
	header := fmt.Sprintf("column %s", colName)
	hasContent := len(col.ContentWidgets) > 0

	if hasContent {
		// Output column with content block
		formatWidgetProps(ctx.Output, prefix, header, props, " {\n")
		for _, widget := range col.ContentWidgets {
			outputWidgetMDLV3(ctx, widget, len(prefix)/2+1)
		}
		fmt.Fprintf(ctx.Output, "%s}\n", prefix)
	} else {
		// Output simple column line
		formatWidgetProps(ctx.Output, prefix, header, props, "\n")
	}
}

func extractTextContent(ctx *ExecContext, w map[string]any, field string) string {
	content, ok := w[field].(map[string]any)
	if !ok {
		return ""
	}
	// Path: Content.Template.Items[] where Items contains Translation objects
	// Structure: Content -> Template -> Items -> [version, Translation{Text: "value"}]
	template, ok := content["Template"].(map[string]any)
	if !ok {
		return ""
	}
	items := getBsonArrayElements(template["Items"])
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		// Translation objects have Text directly as a string
		if text, ok := itemMap["Text"].(string); ok {
			return text
		}
	}
	return ""
}

func extractButtonCaption(ctx *ExecContext, w map[string]any) string {
	// Try Caption first (legacy format)
	if caption := extractTextContent(ctx, w, "Caption"); caption != "" {
		return caption
	}
	// Try CaptionTemplate (modern format used by ActionButton)
	return extractTextContent(ctx, w, "CaptionTemplate")
}

// extractButtonCaptionParameters extracts parameters from ActionButton caption.
// Tries CaptionTemplate first (modern format), then Caption (legacy format).
func extractButtonCaptionParameters(ctx *ExecContext, w map[string]any) []string {
	// Try CaptionTemplate first (modern format used by ActionButton)
	if params := extractClientTemplateParameters(ctx, w, "CaptionTemplate"); params != nil {
		return params
	}
	// Fall back to Caption (legacy format)
	return extractClientTemplateParameters(ctx, w, "Caption")
}

func extractButtonStyle(ctx *ExecContext, w map[string]any) string {
	if style, ok := w["ButtonStyle"].(string); ok {
		return style
	}
	return "Default"
}

func extractButtonAction(ctx *ExecContext, w map[string]any) string {
	action, ok := w["Action"].(map[string]any)
	if !ok {
		// Try primitive.M type
		if actionM, okM := w["Action"].(primitive.M); okM {
			action = map[string]any(actionM)
		} else {
			return ""
		}
	}
	typeName, _ := action["$Type"].(string)
	switch typeName {
	case "Forms$SaveChangesClientAction", "Pages$SaveChangesClientAction":
		result := "save_changes"
		if closePage, ok := action["ClosePage"].(bool); ok && closePage {
			result += " close_page"
		}
		return result
	case "Forms$CancelChangesClientAction", "Pages$CancelChangesClientAction":
		result := "cancel_changes"
		if closePage, ok := action["ClosePage"].(bool); ok && closePage {
			result += " close_page"
		}
		return result
	case "Forms$ClosePageClientAction", "Pages$ClosePageClientAction":
		return "close_page"
	case "Forms$DeleteClientAction", "Pages$DeleteClientAction":
		result := "delete_object"
		if closePage, ok := action["ClosePage"].(bool); ok && closePage {
			result += " close_page"
		}
		return result
	case "Forms$CreateObjectClientAction", "Pages$CreateObjectClientAction":
		result := "create_object"
		// Extract entity reference
		if entityRef, ok := action["EntityRef"].(map[string]any); ok {
			if entityName, ok := entityRef["Entity"].(string); ok && entityName != "" {
				result += " " + entityName
			}
		}
		// Extract page reference from PageSettings (Forms$FormSettings)
		if pageSettings, ok := action["PageSettings"].(map[string]any); ok {
			// The page is stored in "Form" field as a qualified name string (BY_NAME_REFERENCE)
			if pageName, ok := pageSettings["Form"].(string); ok && pageName != "" {
				pageAction := "show_page " + pageName
				// Extract page parameters
				params := extractPageParameters(ctx, pageSettings)
				if params != "" {
					pageAction += "(" + params + ")"
				}
				result += " then " + pageAction
			}
		}
		return result
	case "Forms$FormAction", "Pages$FormAction":
		// SHOW_PAGE action - page reference is in FormSettings.Form (string name)
		// or PageSettings.Form, or Page field (binary ID for legacy)
		if formSettings, ok := action["FormSettings"].(map[string]any); ok {
			if pageName, ok := formSettings["Form"].(string); ok && pageName != "" {
				result := "show_page " + pageName
				params := extractPageParameters(ctx, formSettings)
				if params != "" {
					result += "(" + params + ")"
				}
				return result
			}
		}
		if pageSettings, ok := action["PageSettings"].(map[string]any); ok {
			if pageName, ok := pageSettings["Form"].(string); ok && pageName != "" {
				result := "show_page " + pageName
				params := extractPageParameters(ctx, pageSettings)
				if params != "" {
					result += "(" + params + ")"
				}
				return result
			}
		}
		// Fall back to Page field (binary ID from legacy serialization)
		if pageID := extractBinaryID(action["Page"]); pageID != "" {
			pageName := getPageQualifiedName(ctx, model.ID(pageID))
			if pageName != "" {
				return "show_page " + pageName
			}
		}
		return "show_page"
	case "Forms$MicroflowAction", "Forms$MicroflowClientAction", "Pages$MicroflowClientAction":
		// Decode using gen types — no raw map field access.
		if mf := decodeMicroflowClientAction(action); mf != nil {
			result := formatPageMicroflowActionGen(mf)
			// ClosePage is not in the gen type (it's raw-injected), so check the raw map.
			if closePage, ok := action["ClosePage"].(bool); ok && closePage {
				result += " close_page"
			}
			return result
		}
		return "microflow"
	case "Forms$CallNanoflowClientAction", "Pages$CallNanoflowClientAction":
		if nfName, ok := action["Nanoflow"].(string); ok && nfName != "" {
			result := "call_nanoflow " + nfName
			// Extract parameter mappings (directly in the action)
			params := extractNanoflowParameters(ctx, action)
			if params != "" {
				result += "(" + params + ")"
			}
			return result
		}
		return "call_nanoflow"
	case "Forms$SetTaskOutcomeClientAction", "Pages$SetTaskOutcomeClientAction":
		outcomeValue, _ := action["OutcomeValue"].(string)
		return "complete_task '" + strings.ReplaceAll(outcomeValue, "'", "''") + "'"
	case "Forms$NoClientAction", "Pages$NoClientAction":
		return ""
	default:
		return ""
	}
}

// getPageQualifiedName resolves a page ID to its qualified name.
func getPageQualifiedName(ctx *ExecContext, pageID model.ID) string {
	if pageID == "" {
		return ""
	}
	pgPairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return ""
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return ""
	}
	for _, pair := range pgPairs {
		p := pair.Elem
		if model.ID(p.ID()) == pageID {
			modName := h.GetModuleName(h.FindModuleID(model.ID(pair.ContainerID)))
			return modName + "." + p.Name()
		}
	}
	return ""
}

// extractPageParameters extracts page parameter mappings from a FormSettings/PageSettings object.
// Returns formatted string like "Product: $currentObject" or empty string if no params.
func extractPageParameters(ctx *ExecContext, settings map[string]any) string {
	mappings := getBsonArrayElements(settings["ParameterMappings"])
	if len(mappings) == 0 {
		return ""
	}

	var params []string
	for _, mapping := range mappings {
		mappingMap, ok := mapping.(map[string]any)
		if !ok {
			continue
		}

		// Get parameter name from Parameter field (BY_NAME_REFERENCE: "PageName.ParamName")
		paramRef := extractString(mappingMap["Parameter"])
		if paramRef == "" {
			continue
		}
		// Extract just the parameter name (last part after the dot)
		parts := strings.Split(paramRef, ".")
		paramName := parts[len(parts)-1]
		if paramName == "" {
			continue
		}

		// Get the value - check for $currentObject (WidgetValue), Argument (variable or expression)
		value := ""

		// Check for WidgetValue (represents $currentObject in list widgets)
		if widgetVal, ok := mappingMap["WidgetValue"].(map[string]any); ok && widgetVal != nil {
			// $Type is Pages$WidgetValue or similar - this represents current row object
			if valType := extractString(widgetVal["$Type"]); valType != "" {
				value = "$currentObject"
			}
		}

		// Check for Argument (variable reference or expression stored as string)
		if value == "" {
			if arg := extractString(mappingMap["Argument"]); arg != "" {
				value = arg // e.g., "$Product" or an expression
			}
		}

		// Check for Variable reference (older format - Variable as a map with Name)
		if value == "" {
			if varRef, ok := mappingMap["Variable"].(map[string]any); ok && varRef != nil {
				if varName := extractString(varRef["Name"]); varName != "" {
					value = "$" + varName
				}
			}
		}

		if value != "" {
			params = append(params, paramName+": "+value)
		}
	}

	return strings.Join(params, ", ")
}

// extractMicroflowParameters extracts microflow parameter mappings from a MicroflowSettings object.
// Returns formatted string like "Product = $currentObject" or empty string if no params.
func extractMicroflowParameters(ctx *ExecContext, settings map[string]any) string {
	mappings := getBsonArrayElements(settings["ParameterMappings"])
	if len(mappings) == 0 {
		return ""
	}

	var params []string
	for _, mapping := range mappings {
		mappingMap, ok := mapping.(map[string]any)
		if !ok {
			continue
		}

		// Get parameter name from Parameter field (BY_NAME_REFERENCE: "Module.Microflow.ParamName")
		paramRef := extractString(mappingMap["Parameter"])
		if paramRef == "" {
			continue
		}
		// Extract just the parameter name (last part after the dots)
		parts := strings.Split(paramRef, ".")
		paramName := parts[len(parts)-1]
		if paramName == "" {
			continue
		}

		// Get the value - check for $currentObject (WidgetValue), Expression, or Variable
		value := ""

		// Check for WidgetValue (represents $currentObject in list widgets)
		if widgetVal, ok := mappingMap["WidgetValue"].(map[string]any); ok && widgetVal != nil {
			if valType := extractString(widgetVal["$Type"]); valType != "" {
				value = "$currentObject"
			}
		}

		// Check for Expression (used in Pages$MicroflowParameterMapping)
		if value == "" {
			if expr := extractString(mappingMap["Expression"]); expr != "" {
				value = expr // e.g., "$Product" or an expression
			}
		}

		// Check for Variable reference (older format - Variable as a map with Name)
		if value == "" {
			if varRef, ok := mappingMap["Variable"].(map[string]any); ok && varRef != nil {
				if varName := extractString(varRef["Name"]); varName != "" {
					value = "$" + varName
				}
			}
		}

		if value != "" {
			params = append(params, paramName+" = "+value)
		}
	}

	return strings.Join(params, ", ")
}

// extractNanoflowParameters extracts nanoflow parameter mappings from an action object.
// Returns formatted string like "Product = $currentObject" or empty string if no params.
func extractNanoflowParameters(ctx *ExecContext, action map[string]any) string {
	mappings := getBsonArrayElements(action["ParameterMappings"])
	if len(mappings) == 0 {
		return ""
	}

	var params []string
	for _, mapping := range mappings {
		mappingMap, ok := mapping.(map[string]any)
		if !ok {
			continue
		}

		// Get parameter name from Parameter field (BY_NAME_REFERENCE: "Module.Nanoflow.ParamName")
		paramRef := extractString(mappingMap["Parameter"])
		if paramRef == "" {
			continue
		}
		// Extract just the parameter name (last part after the dots)
		parts := strings.Split(paramRef, ".")
		paramName := parts[len(parts)-1]
		if paramName == "" {
			continue
		}

		// Get the value - check for $currentObject (WidgetValue), Expression, or Variable
		value := ""

		// Check for WidgetValue (represents $currentObject in list widgets)
		if widgetVal, ok := mappingMap["WidgetValue"].(map[string]any); ok && widgetVal != nil {
			if valType := extractString(widgetVal["$Type"]); valType != "" {
				value = "$currentObject"
			}
		}

		// Check for Expression (used in Pages$NanoflowParameterMapping)
		if value == "" {
			if expr := extractString(mappingMap["Expression"]); expr != "" {
				value = expr // e.g., "$Product" or an expression
			}
		}

		// Check for Variable reference (older format - Variable as a map with Name)
		if value == "" {
			if varRef, ok := mappingMap["Variable"].(map[string]any); ok && varRef != nil {
				if varName := extractString(varRef["Name"]); varName != "" {
					value = "$" + varName
				}
			}
		}

		if value != "" {
			params = append(params, paramName+" = "+value)
		}
	}

	return strings.Join(params, ", ")
}

func extractTextCaption(ctx *ExecContext, w map[string]any) string {
	caption, ok := w["Caption"].(map[string]any)
	if !ok {
		return ""
	}
	items := getBsonArrayElements(caption["Items"])
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if text, ok := itemMap["Text"].(string); ok {
			return text
		}
	}
	return ""
}

// extractClientTemplateParameters extracts parameter values from a ClientTemplate field (Content or Caption).
func extractClientTemplateParameters(ctx *ExecContext, w map[string]any, fieldName string) []string {
	template, ok := w[fieldName].(map[string]any)
	if !ok {
		return nil
	}
	params := getBsonArrayElements(template["Parameters"])
	if params == nil {
		return nil
	}
	var result []string
	for _, p := range params {
		pMap, ok := p.(map[string]any)
		if !ok {
			continue
		}
		// Check for Expression first (literal value)
		if expr, ok := pMap["Expression"].(string); ok && expr != "" {
			result = append(result, expr)
			continue
		}

		// Check for SourceVariable (page/snippet parameter reference)
		// If present, output as $paramName.Attribute
		sourceVarName := ""
		if srcVar, ok := pMap["SourceVariable"].(map[string]any); ok && srcVar != nil {
			if paramName, ok := srcVar["PageParameter"].(string); ok && paramName != "" {
				sourceVarName = paramName
			}
		}

		// Check for AttributeRef
		if attrRef, ok := pMap["AttributeRef"].(map[string]any); ok && attrRef != nil {
			if attr, ok := attrRef["Attribute"].(string); ok {
				if sourceVarName != "" {
					// Has SourceVariable - this is a page parameter reference
					// Extract just the attribute name from the path
					// attr is like "Module.Entity.Attribute", we want just "Attribute"
					parts := strings.Split(attr, ".")
					attrName := parts[len(parts)-1]
					// Use $ParamName.Attribute format to indicate parameter reference
					result = append(result, "$"+sourceVarName+"."+attrName)
				} else {
					// No SourceVariable - use short attribute name
					result = append(result, shortAttributeName(attr))
				}
				continue
			}
		}
		// Parameter exists but has no binding - mark as unbound
		result = append(result, "<unbound>")
	}
	return result
}

func (e *Executor) outputWidgetMDLV3(w rawWidget, indent int) {
	outputWidgetMDLV3(e.newExecContext(context.Background()), w, indent)
}

// decodeMicroflowClientAction marshals a raw BSON map back to bytes and
// decodes it via the gen-type registry, returning the typed value or nil.
func decodeMicroflowClientAction(action map[string]any) *genPg.MicroflowClientAction {
	raw, err := bson.Marshal(action)
	if err != nil {
		return nil
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(raw))
	if err != nil {
		return nil
	}
	mca, _ := elem.(*genPg.MicroflowClientAction)
	return mca
}

// formatPageMicroflowActionGen formats a MicroflowClientAction using the gen
// typed API. Output: `microflow Module.Name (Param: $Value)` — the canonical
// MDLPage.g4 page-action syntax (MICROFLOW keyword, colon-separated args).
func formatPageMicroflowActionGen(mca *genPg.MicroflowClientAction) string {
	settings, ok := mca.MicroflowSettings().(*genPg.MicroflowSettings)
	if !ok || settings == nil {
		return "microflow"
	}
	mfQN := settings.MicroflowQualifiedName()
	if mfQN == "" {
		return "microflow"
	}

	var params []string
	for _, item := range settings.ParameterMappingsItems() {
		pm, ok := item.(*genPg.MicroflowParameterMapping)
		if !ok || pm == nil {
			continue
		}
		paramRef := pm.ParameterQualifiedName()
		if paramRef == "" {
			continue
		}
		parts := strings.Split(paramRef, ".")
		paramName := parts[len(parts)-1]

		value := pm.Expression()
		if value == "" {
			continue
		}
		params = append(params, paramName+": "+value)
	}

	result := "microflow " + mfQN
	if len(params) > 0 {
		result += "(" + strings.Join(params, ", ") + ")"
	}
	return result
}
