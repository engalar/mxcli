// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/model"
)

// parseRawWidget parses a raw widget map into rawWidget structs.
func (e *Executor) parseRawWidget(w map[string]any) []rawWidget {
	typeName, _ := w["$Type"].(string)
	name, _ := w["Name"].(string)

	// Parse DivContainer as a proper CONTAINER widget with children
	if typeName == "Forms$DivContainer" || typeName == "Pages$DivContainer" ||
		typeName == "Forms$GroupBox" || typeName == "Pages$GroupBox" {
		widget := rawWidget{
			Type: typeName,
			Name: name,
		}
		// Extract CSS class, style, and design properties from Appearance
		if appearance, ok := w["Appearance"].(map[string]any); ok {
			if class, ok := appearance["Class"].(string); ok && class != "" {
				widget.Class = class
			}
			if style, ok := appearance["Style"].(string); ok && style != "" {
				widget.Style = style
			}
			widget.DesignProperties = extractDesignProperties(appearance)
		}
		// Extract GroupBox-specific properties
		if typeName == "Forms$GroupBox" || typeName == "Pages$GroupBox" {
			// Caption is stored as CaptionTemplate (Forms$ClientTemplate)
			if ct, ok := w["CaptionTemplate"].(map[string]any); ok {
				widget.Caption = e.extractTextFromTemplate(ct)
			} else {
				// Fallback to legacy Caption field
				widget.Caption = e.extractTextCaption(w)
			}
			if collapsible, ok := w["Collapsible"].(string); ok {
				widget.Collapsible = collapsible
			}
			if headerMode, ok := w["HeaderMode"].(string); ok {
				widget.HeaderMode = headerMode
			}
		}
		children := getBsonArrayElements(w["Widgets"])
		if children != nil {
			for _, c := range children {
				if cMap, ok := c.(map[string]any); ok {
					widget.Children = append(widget.Children, e.parseRawWidget(cMap)...)
				}
			}
		}
		return []rawWidget{widget}
	}

	widget := rawWidget{
		Type: typeName,
		Name: name,
	}

	// Extract CSS class, style, and design properties from Appearance
	if appearance, ok := w["Appearance"].(map[string]any); ok {
		if class, ok := appearance["Class"].(string); ok && class != "" {
			widget.Class = class
		}
		if style, ok := appearance["Style"].(string); ok && style != "" {
			widget.Style = style
		}
		widget.DesignProperties = extractDesignProperties(appearance)
	}

	switch typeName {
	case "Forms$LayoutGrid", "Pages$LayoutGrid":
		widget.Rows = e.parseLayoutGridRows(w)
		return []rawWidget{widget}

	case "Forms$DynamicText", "Pages$DynamicText":
		widget.Content = e.extractTextContent(w, "Content")
		widget.Parameters = e.extractClientTemplateParameters(w, "Content")
		if rm, ok := w["RenderMode"].(string); ok {
			widget.RenderMode = rm
		}
		return []rawWidget{widget}

	case "Forms$ActionButton", "Pages$ActionButton":
		widget.Caption = e.extractButtonCaption(w)
		widget.Parameters = e.extractButtonCaptionParameters(w)
		widget.ButtonStyle = e.extractButtonStyle(w)
		widget.Action = e.extractButtonAction(w)
		return []rawWidget{widget}

	case "Forms$Text", "Pages$Text":
		widget.Content = e.extractTextCaption(w)
		if rm, ok := w["RenderMode"].(string); ok {
			widget.RenderMode = rm
		}
		return []rawWidget{widget}

	case "Forms$Title", "Pages$Title":
		widget.Caption = e.extractTextCaption(w)
		return []rawWidget{widget}

	case "Forms$DataView", "Pages$DataView":
		widget.Children = e.parseDataViewChildren(w)
		widget.DataSource = e.extractDataViewDataSource(w)
		return []rawWidget{widget}

	case "Forms$TextBox", "Pages$TextBox":
		widget.Caption = e.extractLabelText(w)
		widget.Content = e.extractAttributeRef(w)
		return []rawWidget{widget}

	case "Forms$TextArea", "Pages$TextArea":
		widget.Caption = e.extractLabelText(w)
		widget.Content = e.extractAttributeRef(w)
		return []rawWidget{widget}

	case "Forms$DatePicker", "Pages$DatePicker":
		widget.Caption = e.extractLabelText(w)
		widget.Content = e.extractAttributeRef(w)
		return []rawWidget{widget}

	case "Forms$RadioButtons", "Pages$RadioButtons", "Forms$RadioButtonGroup", "Pages$RadioButtonGroup":
		widget.Type = "Forms$RadioButtons" // Normalize type
		widget.Caption = e.extractLabelText(w)
		widget.Content = e.extractAttributeRef(w)
		return []rawWidget{widget}

	case "Forms$CheckBox", "Pages$CheckBox":
		widget.Caption = e.extractLabelText(w)
		widget.Content = e.extractAttributeRef(w)
		widget.Editable = e.extractEditable(w)
		widget.ReadOnlyStyle = e.extractReadOnlyStyle(w)
		widget.ShowLabel = e.extractShowLabel(w)
		return []rawWidget{widget}

	case "CustomWidgets$CustomWidget":
		widget.Caption = e.extractLabelText(w)
		widget.Content = e.extractCustomWidgetAttribute(w)
		widget.RenderMode = e.extractCustomWidgetType(w) // Store widget type in RenderMode
		// For ComboBox, extract datasource and caption attribute for association mode
		if widget.RenderMode == "COMBOBOX" {
			widget.DataSource = e.extractComboBoxDataSource(w)
			if widget.DataSource != nil {
				widget.CaptionAttribute = e.extractCustomWidgetPropertyAttributeRef(w, "optionsSourceAssociationCaptionAttribute")
			}
		}
		// For DataGrid2, also extract datasource, columns, CONTROLBAR widgets, paging, and selection
		if widget.RenderMode == "DATAGRID2" {
			widget.DataSource = e.extractDataGrid2DataSource(w)
			widget.DataGridColumns = e.extractDataGrid2Columns(w)
			widget.ControlBar = e.extractDataGrid2ControlBar(w)
			widget.PageSize = e.extractCustomWidgetPropertyString(w, "pageSize")
			widget.Pagination = e.extractCustomWidgetPropertyString(w, "pagination")
			widget.PagingPosition = e.extractCustomWidgetPropertyString(w, "pagingPosition")
			widget.ShowPagingButtons = e.extractCustomWidgetPropertyString(w, "showPagingButtons")
			// showNumberOfRows: not yet fully supported in DataGrid2, skip to avoid CE0463
			widget.Selection = e.extractGallerySelection(w)
		}
		// For Gallery, extract datasource, content widgets, filter widgets, and selection mode
		if widget.RenderMode == "GALLERY" {
			widget.DataSource = e.extractGalleryDataSource(w)
			widget.Children = e.extractGalleryContent(w)
			widget.FilterWidgets = e.extractGalleryFilters(w)
			widget.Selection = e.extractGallerySelection(w)
		}
		// For filter widgets, extract filter attributes and expression
		if widget.RenderMode == "TEXTFILTER" || widget.RenderMode == "NUMBERFILTER" || widget.RenderMode == "DROPDOWNFILTER" || widget.RenderMode == "DATEFILTER" {
			widget.FilterAttributes = e.extractFilterAttributes(w)
			widget.FilterExpression = e.extractFilterExpression(w)
		}
		return []rawWidget{widget}

	case "Forms$Label", "Pages$Label":
		widget.Content = e.extractTextCaption(w)
		return []rawWidget{widget}

	case "Forms$NavigationList", "Pages$NavigationList":
		widget.Children = e.parseNavigationListItems(w)
		return []rawWidget{widget}

	case "Forms$Gallery", "Pages$Gallery":
		widget.Children = e.parseGalleryContent(w)
		widget.DataSource = e.extractGalleryDataSource(w)
		return []rawWidget{widget}

	case "Forms$SnippetCallWidget", "Pages$SnippetCallWidget":
		widget.Content = e.extractSnippetRef(w)
		return []rawWidget{widget}

	case "Forms$ListView", "Pages$ListView":
		widget.Children = e.parseListViewContent(w)
		widget.DataSource = e.extractListViewDataSource(w)
		return []rawWidget{widget}

	default:
		// For unknown types, just note them
		return []rawWidget{widget}
	}
}

func (e *Executor) parseLayoutGridRows(w map[string]any) []rawWidgetRow {
	rows := getBsonArrayElements(w["Rows"])
	if rows == nil {
		return nil
	}

	var result []rawWidgetRow
	for _, r := range rows {
		rMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		row := rawWidgetRow{}
		cols := getBsonArrayElements(rMap["Columns"])
		for _, c := range cols {
			cMap, ok := c.(map[string]any)
			if !ok {
				continue
			}
			col := rawWidgetColumn{}
			// Get width
			if weight, ok := cMap["Weight"].(int32); ok {
				col.Width = int(weight)
			} else if weight, ok := cMap["DesktopWeight"].(int32); ok {
				col.Width = int(weight)
			}
			// Get widgets
			colWidgets := getBsonArrayElements(cMap["Widgets"])
			for _, cw := range colWidgets {
				if cwMap, ok := cw.(map[string]any); ok {
					col.Widgets = append(col.Widgets, e.parseRawWidget(cwMap)...)
				}
			}
			row.Columns = append(row.Columns, col)
		}
		result = append(result, row)
	}
	return result
}

// parseNavigationListItems extracts items from a NavigationList widget.
func (e *Executor) parseNavigationListItems(w map[string]any) []rawWidget {
	items := getBsonArrayElements(w["Items"])
	if items == nil {
		return nil
	}

	var result []rawWidget
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		rw := rawWidget{
			Type: "NavigationListItem",
		}

		// Read Name field
		rw.Name, _ = itemMap["Name"].(string)

		// Parse all child widgets from the Widgets array
		widgets := getBsonArrayElements(itemMap["Widgets"])
		for _, widget := range widgets {
			wMap, ok := widget.(map[string]any)
			if !ok {
				continue
			}
			parsed := e.parseRawWidget(wMap)
			rw.Children = append(rw.Children, parsed...)
		}

		// Extract action
		rw.Action = e.extractNavigationListItemAction(itemMap)

		// Extract style from Appearance class
		if appearance, ok := itemMap["Appearance"].(map[string]any); ok {
			if class, ok := appearance["Class"].(string); ok && class != "" {
				rw.ButtonStyle = class
			}
		}

		result = append(result, rw)
	}
	return result
}

// extractNavigationListItemAction extracts action from a NavigationListItem.
// NavigationListItem uses Forms$FormAction with FormSettings.Form for page references,
// which differs from ActionButton's action format.
func (e *Executor) extractNavigationListItemAction(w map[string]any) string {
	action, ok := w["Action"].(map[string]any)
	if !ok {
		return ""
	}
	typeName, _ := action["$Type"].(string)
	switch typeName {
	case "Forms$FormAction", "Pages$FormAction":
		// Extract page reference from FormSettings (Studio Pro format)
		if formSettings, ok := action["FormSettings"].(map[string]any); ok {
			if formName, ok := formSettings["Form"].(string); ok && formName != "" {
				return "SHOW_PAGE '" + formName + "'"
			}
		}
		// Fall back to PageSettings.Form (string name)
		if pageSettings, ok := action["PageSettings"].(map[string]any); ok {
			if pageName, ok := pageSettings["Form"].(string); ok && pageName != "" {
				return "SHOW_PAGE '" + pageName + "'"
			}
		}
		// Fall back to Page field (binary ID from mxcli serialization)
		if pageID := extractBinaryID(action["Page"]); pageID != "" {
			pageName := e.getPageQualifiedName(model.ID(pageID))
			if pageName != "" {
				return "SHOW_PAGE '" + pageName + "'"
			}
		}
		return "SHOW_PAGE"
	default:
		// Delegate to the standard action extractor
		return e.extractButtonAction(w)
	}
}

// parseDataViewChildren extracts child widgets from a DataView.
func (e *Executor) parseDataViewChildren(w map[string]any) []rawWidget {
	var result []rawWidget

	// Get main widgets
	widgets := getBsonArrayElements(w["Widgets"])
	for _, child := range widgets {
		if childMap, ok := child.(map[string]any); ok {
			result = append(result, e.parseRawWidget(childMap)...)
		}
	}

	// Get footer widgets
	footerWidgets := getBsonArrayElements(w["FooterWidgets"])
	if len(footerWidgets) > 0 {
		// Create a special footer container with synthetic name
		footer := rawWidget{Type: "Footer", Name: "footer1"}
		for _, child := range footerWidgets {
			if childMap, ok := child.(map[string]any); ok {
				footer.Children = append(footer.Children, e.parseRawWidget(childMap)...)
			}
		}
		result = append(result, footer)
	}

	return result
}

// extractDataViewDataSource extracts the data source from a DataView widget.
func (e *Executor) extractDataViewDataSource(w map[string]any) *rawDataSource {
	ds, ok := w["DataSource"].(map[string]any)
	if !ok {
		return nil
	}

	dsType, _ := ds["$Type"].(string)

	switch dsType {
	case "Forms$MicroflowSource":
		// Extract microflow name from MicroflowSettings
		if settings, ok := ds["MicroflowSettings"].(map[string]any); ok {
			if mfName, ok := settings["Microflow"].(string); ok && mfName != "" {
				return &rawDataSource{Type: "microflow", Reference: mfName}
			}
		}
	case "Forms$NanoflowSource":
		// Extract nanoflow name from NanoflowSettings
		if settings, ok := ds["NanoflowSettings"].(map[string]any); ok {
			if nfName, ok := settings["Nanoflow"].(string); ok && nfName != "" {
				return &rawDataSource{Type: "nanoflow", Reference: nfName}
			}
		}
	case "Forms$DataViewSource":
		// Page parameter source - extract from SourceVariable
		if srcVar, ok := ds["SourceVariable"].(map[string]any); ok {
			if paramName, ok := srcVar["PageParameter"].(string); ok && paramName != "" {
				return &rawDataSource{Type: "parameter", Reference: paramName}
			}
		}
	case "Forms$DatabaseSource":
		// Database/XPath source - for now just note it's a database source
		return &rawDataSource{Type: "database", Reference: ""}
	}

	return nil
}

// extractLabelText extracts the label text from an input widget.
func (e *Executor) extractLabelText(w map[string]any) string {
	labelTemplate, ok := w["LabelTemplate"].(map[string]any)
	if !ok {
		return ""
	}
	return e.extractTextFromTemplate(labelTemplate)
}

// extractEditable extracts the Editable setting from an input widget.
// Returns "Always", "Never", or "Conditional".
func (e *Executor) extractEditable(w map[string]any) string {
	if editable, ok := w["Editable"].(string); ok {
		return editable
	}
	return ""
}

// extractReadOnlyStyle extracts the ReadOnlyStyle from an input widget.
// Returns "Inherit", "Control", or "Text".
func (e *Executor) extractReadOnlyStyle(w map[string]any) string {
	if style, ok := w["ReadOnlyStyle"].(string); ok {
		return style
	}
	return ""
}

// extractShowLabel extracts whether the label is visible from LabelTemplate.
func (e *Executor) extractShowLabel(w map[string]any) bool {
	labelTemplate, ok := w["LabelTemplate"].(map[string]any)
	if !ok {
		return true // Default to showing label
	}
	// Check for TextVisible field - false means "Show label: No"
	if textVisible, ok := labelTemplate["TextVisible"].(bool); ok {
		return textVisible
	}
	return true // Default
}

// extractTextFromTemplate extracts text from a ClientTemplate.
// ClientTemplate structure: Template.Items[] contains Texts$Translation with Text field
func (e *Executor) extractTextFromTemplate(template map[string]any) string {
	// For ClientTemplate (Forms$ClientTemplate), the text is in Template.Items[].Text
	if innerTemplate, ok := template["Template"].(map[string]any); ok {
		items := getBsonArrayElements(innerTemplate["Items"])
		for _, item := range items {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := itemMap["Text"].(string); ok && text != "" {
				return text
			}
		}
	}
	// Fallback: direct Items array (for legacy or different template types)
	items := getBsonArrayElements(template["Items"])
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

// shortAttributeName strips the qualified prefix from a BSON attribute path.
// "Module.Entity.Attribute" → "Attribute". The entity context is established
// by the enclosing DATAVIEW, so DESCRIBE outputs only the bare name.
func shortAttributeName(attr string) string {
	if idx := strings.LastIndex(attr, "."); idx >= 0 {
		return attr[idx+1:]
	}
	return attr
}

// extractAttributeRef extracts the attribute reference from an input widget.
// Returns just the attribute name (last segment).
func (e *Executor) extractAttributeRef(w map[string]any) string {
	attrRef, ok := w["AttributeRef"].(map[string]any)
	if !ok {
		return ""
	}
	attr, ok := attrRef["Attribute"].(string)
	if !ok {
		return ""
	}
	return shortAttributeName(attr)
}

// parseGalleryContent extracts the content widget from a Gallery.
func (e *Executor) parseGalleryContent(w map[string]any) []rawWidget {
	content := w["ContentWidget"]
	if content == nil {
		return nil
	}
	contentMap, ok := content.(map[string]any)
	if !ok {
		return nil
	}
	return e.parseRawWidget(contentMap)
}

// parseListViewContent extracts the content widgets from a ListView.
func (e *Executor) parseListViewContent(w map[string]any) []rawWidget {
	widgets := getBsonArrayElements(w["Widgets"])
	if widgets == nil {
		return nil
	}
	var result []rawWidget
	for _, wgt := range widgets {
		wgtMap, ok := wgt.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, e.parseRawWidget(wgtMap)...)
	}
	return result
}

// extractListViewDataSource extracts the datasource from a ListView widget.
func (e *Executor) extractListViewDataSource(w map[string]any) *rawDataSource {
	ds, ok := w["DataSource"].(map[string]any)
	if !ok || ds == nil {
		return nil
	}

	dsType := extractString(ds["$Type"])
	switch dsType {
	case "Forms$ListViewXPathSource":
		result := &rawDataSource{Type: "database"}
		entityRef, ok := ds["EntityRef"].(map[string]any)
		if ok && entityRef != nil {
			result.Reference = extractString(entityRef["Entity"])
		}
		result.XPathConstraint = extractString(ds["XPathConstraint"])
		// Extract sorting from Sort field
		if sortObj, ok := ds["Sort"].(map[string]any); ok {
			sortPaths := getBsonArrayElements(sortObj["Paths"])
			for _, item := range sortPaths {
				sortItem, ok := item.(map[string]any)
				if !ok {
					continue
				}
				col := rawSortColumn{Order: "ASC"}
				if attrRef, ok := sortItem["AttributeRef"].(map[string]any); ok {
					col.Attribute = shortAttributeName(extractString(attrRef["Attribute"]))
				}
				sortOrder := extractString(sortItem["SortOrder"])
				if sortOrder == "Descending" {
					col.Order = "DESC"
				}
				if col.Attribute != "" {
					result.SortColumns = append(result.SortColumns, col)
				}
			}
		}
		if result.Reference != "" {
			return result
		}
	case "Forms$MicroflowSource":
		microflow := extractString(ds["Microflow"])
		if mfSettings, ok := ds["MicroflowSettings"].(map[string]any); ok && microflow == "" {
			microflow = extractString(mfSettings["Microflow"])
		}
		if microflow != "" {
			return &rawDataSource{Type: "microflow", Reference: microflow}
		}
	case "Forms$NanoflowSource":
		nanoflow := extractString(ds["Nanoflow"])
		if nfSettings, ok := ds["NanoflowSettings"].(map[string]any); ok && nanoflow == "" {
			nanoflow = extractString(nfSettings["Nanoflow"])
		}
		if nanoflow != "" {
			return &rawDataSource{Type: "nanoflow", Reference: nanoflow}
		}
	}
	return nil
}

// extractSnippetRef extracts the snippet reference from a SnippetCallWidget.
func (e *Executor) extractSnippetRef(w map[string]any) string {
	// First try the FormCall.Form path (used for BY_NAME_REFERENCE)
	if formCall, ok := w["FormCall"].(map[string]any); ok {
		if form, ok := formCall["Form"].(string); ok && form != "" {
			return form
		}
		// Try binary ID and resolve to name
		if formID := extractBinaryID(formCall["Form"]); formID != "" {
			// Try to resolve the snippet name from ID
			snippets, err := e.reader.ListSnippets()
			if err == nil {
				for _, s := range snippets {
					if string(s.ID) == formID {
						moduleName := ""
						if modules, err := e.reader.ListModules(); err == nil {
							for _, m := range modules {
								if m.ID == s.ContainerID {
									moduleName = m.Name
									break
								}
							}
						}
						if moduleName != "" {
							return moduleName + "." + s.Name
						}
						return s.Name
					}
				}
			}
		}
	}
	// Fallback to direct Snippet field
	return extractString(w["Snippet"])
}

// extractDesignProperties extracts design properties from an Appearance map.
// The DesignProperties field is a BSON array: [version, prop1, prop2, ...]
// Studio Pro uses a nested format where each prop has $Type "Forms$DesignPropertyValue"
// with Key and a Value sub-map containing the actual Toggle/Option type.
// We also handle the flat format (Toggle/Option directly) for backward compatibility.
func extractDesignProperties(appearance map[string]any) []rawDesignProp {
	dpArray := getBsonArrayElements(appearance["DesignProperties"])
	if len(dpArray) == 0 {
		return nil
	}

	var result []rawDesignProp
	for _, item := range dpArray {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typeName, _ := itemMap["$Type"].(string)
		key, _ := itemMap["Key"].(string)
		if key == "" {
			continue
		}

		switch typeName {
		case "Forms$DesignPropertyValue":
			// Studio Pro nested format: Value sub-map contains the actual type
			valueMap, ok := itemMap["Value"].(map[string]any)
			if !ok {
				continue
			}
			innerType, _ := valueMap["$Type"].(string)
			switch innerType {
			case "Forms$ToggleDesignPropertyValue":
				result = append(result, rawDesignProp{
					Key:       key,
					ValueType: "toggle",
				})
			case "Forms$OptionDesignPropertyValue":
				option, _ := valueMap["Option"].(string)
				result = append(result, rawDesignProp{
					Key:       key,
					ValueType: "option",
					Option:    option,
				})
			case "Forms$CustomDesignPropertyValue":
				value, _ := valueMap["Value"].(string)
				result = append(result, rawDesignProp{
					Key:       key,
					ValueType: "option", // Treat custom (ToggleButtonGroup) as option for display
					Option:    value,
				})
			}
		case "Forms$ToggleDesignPropertyValue":
			// Flat format (backward compat with mxcli-written pages)
			result = append(result, rawDesignProp{
				Key:       key,
				ValueType: "toggle",
			})
		case "Forms$OptionDesignPropertyValue":
			// Flat format (backward compat with mxcli-written pages)
			option, _ := itemMap["Option"].(string)
			result = append(result, rawDesignProp{
				Key:       key,
				ValueType: "option",
				Option:    option,
			})
		}
	}
	return result
}
