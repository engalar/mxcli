// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
)

// buildPropertyTypeKeyMap builds a map from PropertyType $ID to PropertyKey for a CustomWidget.
// This resolves TypePointer references in Object.Properties back to their property names.
// If withFallback is true, also checks widgetType["PropertyTypes"] directly (for widgets like
// Gallery/DataGrid2 that may store PropertyTypes at different nesting levels).
func buildPropertyTypeKeyMap(w map[string]any, withFallback bool) map[string]string {
	propTypeKeyMap := make(map[string]string)
	widgetType, ok := w["Type"].(map[string]any)
	if !ok {
		return propTypeKeyMap
	}
	var propTypes []any
	if objType, ok := widgetType["ObjectType"].(map[string]any); ok {
		propTypes = getBsonArrayElements(objType["PropertyTypes"])
	}
	if withFallback && len(propTypes) == 0 {
		propTypes = getBsonArrayElements(widgetType["PropertyTypes"])
	}
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		key := extractString(ptMap["PropertyKey"])
		if key == "" {
			continue
		}
		id := extractBinaryID(ptMap["$ID"])
		if id != "" {
			propTypeKeyMap[id] = key
		}
	}
	return propTypeKeyMap
}

// extractCustomWidgetAttribute extracts the attribute from a CustomWidget (e.g., ComboBox).
func (e *Executor) extractCustomWidgetAttribute(w map[string]any) string {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return ""
	}

	// Search through properties for attributeEnumeration or attributeAssociation
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		// Check for AttributeRef
		attrRef, ok := value["AttributeRef"].(map[string]any)
		if !ok {
			continue
		}
		if attr, ok := attrRef["Attribute"].(string); ok && attr != "" {
			return shortAttributeName(attr)
		}
	}
	return ""
}

// extractCustomWidgetType extracts the widget type ID from a CustomWidget.
func (e *Executor) extractCustomWidgetType(w map[string]any) string {
	typeObj, ok := w["Type"].(map[string]any)
	if !ok {
		return ""
	}
	if widgetID, ok := typeObj["WidgetId"].(string); ok {
		// Return short name based on widget ID (uppercase for MDL keywords)
		switch widgetID {
		case "com.mendix.widget.web.combobox.Combobox":
			return "COMBOBOX"
		case "com.mendix.widget.web.datagrid.Datagrid":
			return "DATAGRID2"
		case "com.mendix.widget.web.gallery.Gallery":
			return "GALLERY"
		case "com.mendix.widget.web.datagridtextfilter.DatagridTextFilter":
			return "TEXTFILTER"
		case "com.mendix.widget.web.datagridnumberfilter.DatagridNumberFilter":
			return "NUMBERFILTER"
		case "com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter":
			return "DROPDOWNFILTER"
		case "com.mendix.widget.web.datagriddatefilter.DatagridDateFilter":
			return "DATEFILTER"
		default:
			// Extract last part of widget ID and uppercase it
			parts := strings.Split(widgetID, ".")
			if len(parts) > 0 {
				return strings.ToUpper(parts[len(parts)-1])
			}
			return strings.ToUpper(widgetID)
		}
	}
	return ""
}

// extractComboBoxDataSource extracts the datasource from a ComboBox CustomWidget in association mode.
// Returns nil for enumeration mode (no datasource).
func (e *Executor) extractComboBoxDataSource(w map[string]any) *rawDataSource {
	// Check if optionsSourceType is "association" first
	sourceType := e.extractCustomWidgetPropertyString(w, "optionsSourceType")
	if sourceType != "association" {
		return nil
	}

	// Extract datasource from optionsSourceAssociationDataSource property
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return nil
	}

	propTypeKeyMap := buildPropertyTypeKeyMap(w, false)

	// Search through properties for optionsSourceAssociationDataSource
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		typePointerID := extractBinaryID(propMap["TypePointer"])
		propKey := propTypeKeyMap[typePointerID]
		if propKey != "optionsSourceAssociationDataSource" {
			continue
		}
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		dsVal, hasDS := value["DataSource"]
		if !hasDS {
			continue
		}
		if ds, ok := dsVal.(map[string]any); ok && ds != nil {
			return e.parseCustomWidgetDataSource(ds)
		}
	}
	return nil
}

// extractDataGrid2DataSource extracts the datasource from a DataGrid2 CustomWidget.
func (e *Executor) extractDataGrid2DataSource(w map[string]any) *rawDataSource {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return nil
	}

	// Search through properties for datasource
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		// Check for DataSource
		ds, ok := value["DataSource"].(map[string]any)
		if !ok || ds == nil {
			continue
		}

		dsType := extractString(ds["$Type"])
		switch dsType {
		case "Forms$DatabaseSource":
			entityRef, ok := ds["EntityRef"].(map[string]any)
			if ok && entityRef != nil {
				entity := extractString(entityRef["Entity"])
				if entity != "" {
					return &rawDataSource{Type: "database", Reference: entity}
				}
			}
		case "CustomWidgets$CustomWidgetXPathSource":
			// CustomWidget datasource format - EntityRef contains Entity as qualified name
			result := &rawDataSource{Type: "database"}
			entityRef, ok := ds["EntityRef"].(map[string]any)
			if ok && entityRef != nil {
				result.Reference = extractString(entityRef["Entity"])
			}
			// Extract XPathConstraint
			result.XPathConstraint = extractString(ds["XPathConstraint"])
			// Extract sorting from SortBar - support multiple sort columns
			if sortBar, ok := ds["SortBar"].(map[string]any); ok {
				sortItems := getBsonArrayElements(sortBar["SortItems"])
				for _, item := range sortItems {
					sortItem, ok := item.(map[string]any)
					if !ok {
						continue
					}
					col := rawSortColumn{Order: "ASC"}
					// Extract attribute from AttributeRef
					if attrRef, ok := sortItem["AttributeRef"].(map[string]any); ok {
						col.Attribute = shortAttributeName(extractString(attrRef["Attribute"]))
					}
					// Extract sort order
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
			if microflow != "" {
				return &rawDataSource{Type: "microflow", Reference: microflow}
			}
		case "Forms$EntityPathSource", "Forms$DataViewSource":
			entityPath := extractString(ds["EntityPath"])
			if entityPath != "" {
				return &rawDataSource{Type: "parameter", Reference: entityPath}
			}
		}
	}
	return nil
}

// extractDataGrid2Columns extracts the columns from a DataGrid2 CustomWidget.
func (e *Executor) extractDataGrid2Columns(w map[string]any) []rawDataGridColumn {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return nil
	}

	// Build column property key map from Type.ObjectType.PropertyTypes -> columns -> ValueType.ObjectType.PropertyTypes
	colPropKeyMap := e.buildColumnPropertyKeyMap(w)

	// Search through properties for columns
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		// Check for Objects array (columns are stored as Objects)
		objects := getBsonArrayElements(value["Objects"])
		if len(objects) == 0 {
			continue
		}

		var columns []rawDataGridColumn
		for _, colObj := range objects {
			colMap, ok := colObj.(map[string]any)
			if !ok {
				continue
			}
			col := e.extractDataGrid2Column(colMap, colPropKeyMap)
			if col.Attribute != "" || col.Caption != "" {
				columns = append(columns, col)
			}
		}
		if len(columns) > 0 {
			return columns
		}
	}
	return nil
}

// buildColumnPropertyKeyMap builds a map from TypePointer ID to property key
// for column-level properties (alignment, wrapText, etc.) from the widget Type.
func (e *Executor) buildColumnPropertyKeyMap(w map[string]any) map[string]string {
	result := make(map[string]string)
	widgetType, ok := w["Type"].(map[string]any)
	if !ok {
		return result
	}
	objType, ok := widgetType["ObjectType"].(map[string]any)
	if !ok {
		return result
	}
	// Find the "columns" property type
	propTypes := getBsonArrayElements(objType["PropertyTypes"])
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		key := extractString(ptMap["PropertyKey"])
		if key != "columns" {
			continue
		}
		// Get ValueType.ObjectType.PropertyTypes for column-level properties
		valueType, ok := ptMap["ValueType"].(map[string]any)
		if !ok {
			break
		}
		colObjType, ok := valueType["ObjectType"].(map[string]any)
		if !ok {
			break
		}
		colPropTypes := getBsonArrayElements(colObjType["PropertyTypes"])
		for _, cpt := range colPropTypes {
			cptMap, ok := cpt.(map[string]any)
			if !ok {
				continue
			}
			colKey := extractString(cptMap["PropertyKey"])
			if colKey == "" {
				continue
			}
			id := extractBinaryID(cptMap["$ID"])
			if id != "" {
				result[id] = colKey
			}
		}
		break
	}
	return result
}

// extractDataGrid2Column extracts a single column's info from its WidgetObject.
// DataGrid2 columns have several properties:
// - "header": TextTemplate for column header caption (with optional parameters)
// - "attribute": AttributeRef for the attribute binding
// - "showContentAs": enum value ("attribute", "dynamicText", "customContent")
// - "content": Widgets array for custom content
// - "dynamicText": TextTemplate for dynamic text (when showContentAs = "dynamicText")
// - "alignment": enum value ("left", "center", "right")
// - "wrapText": boolean ("true", "false")
func (e *Executor) extractDataGrid2Column(colObj map[string]any, colPropKeyMap map[string]string) rawDataGridColumn {
	col := rawDataGridColumn{}

	// Track if we've found the header to avoid overwriting with dynamicText's TextTemplate
	foundHeader := false

	props := getBsonArrayElements(colObj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}

		// Resolve property key via TypePointer if available
		propKey := ""
		if len(colPropKeyMap) > 0 {
			typePointerID := extractBinaryID(propMap["TypePointer"])
			propKey = colPropKeyMap[typePointerID]
		}

		// Extract alignment and wrapText by property key
		if propKey == "alignment" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				col.Alignment = primVal
			}
			continue
		}
		if propKey == "wrapText" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				col.WrapText = primVal
			}
			continue
		}
		if propKey == "sortable" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				col.Sortable = primVal
			}
			continue
		}
		if propKey == "resizable" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				col.Resizable = primVal
			}
			continue
		}
		if propKey == "draggable" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				col.Draggable = primVal
			}
			continue
		}
		if propKey == "hidable" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				col.Hidable = primVal
			}
			continue
		}
		if propKey == "width" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				col.ColumnWidth = primVal
			}
			continue
		}
		if propKey == "size" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				col.Size = primVal
			}
			continue
		}
		if propKey == "visible" {
			if expr := extractString(value["Expression"]); expr != "" {
				col.Visible = expr
			}
			continue
		}
		if propKey == "columnClass" {
			if expr := extractString(value["Expression"]); expr != "" {
				col.DynamicCellClass = expr
			}
			continue
		}
		if propKey == "tooltip" {
			if textTemplate, ok := value["TextTemplate"].(map[string]any); ok && textTemplate != nil {
				if template, ok := textTemplate["Template"].(map[string]any); ok && template != nil {
					items := getBsonArrayElements(template["Items"])
					for _, item := range items {
						itemMap, ok := item.(map[string]any)
						if !ok {
							continue
						}
						if text := extractString(itemMap["Text"]); text != "" {
							col.Tooltip = text
							break
						}
					}
				}
			}
			continue
		}

		// Check for AttributeRef (attribute property)
		if col.Attribute == "" {
			if attrRef, ok := value["AttributeRef"].(map[string]any); ok && attrRef != nil {
				attr := extractString(attrRef["Attribute"])
				if attr != "" {
					// Extract just the attribute name from qualified path
					parts := strings.Split(attr, ".")
					if len(parts) > 0 {
						col.Attribute = parts[len(parts)-1]
					}
				}
			}
		}

		// Check for PrimitiveValue (could be showContentAs enum)
		if col.ShowContentAs == "" {
			if primVal := extractString(value["PrimitiveValue"]); primVal != "" {
				// Check if it's a showContentAs enum value
				if primVal == "attribute" || primVal == "dynamicText" || primVal == "customContent" {
					col.ShowContentAs = primVal
				}
			}
		}

		// Check for Widgets array (content property for custom widgets)
		if len(col.ContentWidgets) == 0 {
			widgets := getBsonArrayElements(value["Widgets"])
			if len(widgets) > 0 {
				for _, w := range widgets {
					if wMap, ok := w.(map[string]any); ok {
						col.ContentWidgets = append(col.ContentWidgets, e.parseRawWidget(wMap)...)
					}
				}
			}
		}

		// Check for TextTemplate (could be header or dynamicText property)
		if textTemplate, ok := value["TextTemplate"].(map[string]any); ok && textTemplate != nil {
			template, ok := textTemplate["Template"].(map[string]any)
			if ok && template != nil {
				items := getBsonArrayElements(template["Items"])
				for _, item := range items {
					itemMap, ok := item.(map[string]any)
					if !ok {
						continue
					}
					if text := extractString(itemMap["Text"]); text != "" {
						if !foundHeader {
							// First TextTemplate with text is the header
							col.Caption = text
							col.CaptionParams = e.extractTextTemplateParameters(textTemplate)
							foundHeader = true
						} else if col.DynamicText == "" {
							// Second TextTemplate is dynamicText (if showContentAs = dynamicText)
							col.DynamicText = text
							col.DynamicTextParams = e.extractTextTemplateParameters(textTemplate)
						}
						break
					}
				}
			}
		}
	}
	return col
}

// extractDataGrid2ControlBar extracts the CONTROLBAR widgets from a DataGrid2 CustomWidget.
// DataGrid2 stores header/filter widgets in the 'filtersPlaceholder' property, same as Gallery.
func (e *Executor) extractDataGrid2ControlBar(w map[string]any) []rawWidget {
	return e.extractGalleryWidgetsByPropertyKey(w, "filtersPlaceholder")
}

// extractTextTemplateParameters extracts parameters from a TextTemplate (Forms$ClientTemplate).
func (e *Executor) extractTextTemplateParameters(textTemplate map[string]any) []string {
	params := getBsonArrayElements(textTemplate["Parameters"])
	if params == nil || len(params) == 0 {
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
					parts := strings.Split(attr, ".")
					attrName := parts[len(parts)-1]
					result = append(result, "$"+sourceVarName+"."+attrName)
				} else {
					// No SourceVariable - use short attribute name
					result = append(result, shortAttributeName(attr))
				}
				continue
			}
		}
		// Parameter exists but has no binding
		result = append(result, "<unbound>")
	}
	return result
}

// extractGalleryDataSource extracts the datasource from a Gallery widget.
// Handles both Forms$Gallery and CustomWidgets$CustomWidget Gallery formats.
func (e *Executor) extractGalleryDataSource(w map[string]any) *rawDataSource {
	// First check for CustomWidget Gallery format (datasource in Object.Properties)
	if obj, ok := w["Object"].(map[string]any); ok {
		props := getBsonArrayElements(obj["Properties"])
		for _, prop := range props {
			propMap, ok := prop.(map[string]any)
			if !ok {
				continue
			}
			value, ok := propMap["Value"].(map[string]any)
			if !ok {
				continue
			}
			// Check for DataSource field in Value - only process if not nil
			dsVal, hasDS := value["DataSource"]
			if !hasDS {
				continue
			}
			if ds, ok := dsVal.(map[string]any); ok && ds != nil {
				result := e.parseCustomWidgetDataSource(ds)
				if result != nil {
					return result
				}
			}
		}
	}

	// Fall back to Forms$Gallery format (DataSource at top level)
	ds, ok := w["DataSource"].(map[string]any)
	if !ok || ds == nil {
		return nil
	}

	dsType := extractString(ds["$Type"])
	switch dsType {
	case "Forms$DatabaseSource":
		result := &rawDataSource{Type: "database"}
		entityRef, ok := ds["EntityRef"].(map[string]any)
		if ok && entityRef != nil {
			result.Reference = extractString(entityRef["Entity"])
		}
		result.XPathConstraint = extractString(ds["XPathConstraint"])
		// Extract sorting
		if sortBar, ok := ds["SortBar"].(map[string]any); ok {
			sortItems := getBsonArrayElements(sortBar["SortItems"])
			for _, item := range sortItems {
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
		if microflow != "" {
			return &rawDataSource{Type: "microflow", Reference: microflow}
		}
	case "Forms$EntityPathSource", "Forms$DataViewSource":
		entityPath := extractString(ds["EntityPath"])
		if entityPath != "" {
			return &rawDataSource{Type: "parameter", Reference: entityPath}
		}
	}
	return nil
}

// parseCustomWidgetDataSource parses datasource from CustomWidget property format.
func (e *Executor) parseCustomWidgetDataSource(ds map[string]any) *rawDataSource {
	dsType := extractString(ds["$Type"])
	switch dsType {
	case "CustomWidgets$CustomWidgetXPathSource":
		result := &rawDataSource{Type: "database"}
		entityRef, ok := ds["EntityRef"].(map[string]any)
		if ok && entityRef != nil {
			result.Reference = extractString(entityRef["Entity"])
		}
		result.XPathConstraint = extractString(ds["XPathConstraint"])
		// Extract sorting if present
		if sortBar, ok := ds["SortBar"].(map[string]any); ok {
			sortItems := getBsonArrayElements(sortBar["SortItems"])
			for _, item := range sortItems {
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
		return result
	case "Forms$MicroflowSource":
		// Pluggable widgets use Forms$MicroflowSource with MicroflowSettings
		if settings, ok := ds["MicroflowSettings"].(map[string]any); ok {
			microflow := extractString(settings["Microflow"])
			if microflow != "" {
				return &rawDataSource{Type: "microflow", Reference: microflow}
			}
		}
	case "Forms$NanoflowSource":
		// Pluggable widgets use Forms$NanoflowSource with NanoflowSettings
		if settings, ok := ds["NanoflowSettings"].(map[string]any); ok {
			nanoflow := extractString(settings["Nanoflow"])
			if nanoflow != "" {
				return &rawDataSource{Type: "nanoflow", Reference: nanoflow}
			}
		}
	case "CustomWidgets$CustomWidgetNanoflowSource":
		nanoflow := extractString(ds["Nanoflow"])
		if nanoflow != "" {
			return &rawDataSource{Type: "nanoflow", Reference: nanoflow}
		}
	}
	return nil
}

// extractGalleryContent extracts the content widgets from a CustomWidget Gallery.
func (e *Executor) extractGalleryContent(w map[string]any) []rawWidget {
	return e.extractGalleryWidgetsByPropertyKey(w, "content")
}

// extractGalleryWidgetsByPropertyKey extracts widgets from a named property of a CustomWidget Gallery.
func (e *Executor) extractGalleryWidgetsByPropertyKey(w map[string]any, targetKey string) []rawWidget {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return nil
	}

	// Build a map from PropertyType ID to PropertyKey (with fallback for Gallery/DataGrid2)
	propTypeKeyMap := buildPropertyTypeKeyMap(w, true)

	// Search through properties for the named property
	props := getBsonArrayElements(obj["Properties"])

	// First pass: try to match by property key
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		// Check property key via TypePointer - can be string, binary, or map with $Subtype
		typePointerID := extractBinaryID(propMap["TypePointer"])
		propKey := propTypeKeyMap[typePointerID]

		// Skip if not the target property
		if propKey != targetKey {
			continue
		}

		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		// Check for Widgets array
		widgetsArr := getBsonArrayElements(value["Widgets"])
		if len(widgetsArr) == 0 {
			continue
		}

		var result []rawWidget
		for _, wgt := range widgetsArr {
			wgtMap, ok := wgt.(map[string]any)
			if !ok {
				continue
			}
			result = append(result, e.parseRawWidget(wgtMap)...)
		}
		return result
	}

	// Fallback: if no property key map, scan all properties with Widgets
	// This handles cases where PropertyKey field isn't available
	if len(propTypeKeyMap) == 0 && targetKey == "content" {
		for _, prop := range props {
			propMap, ok := prop.(map[string]any)
			if !ok {
				continue
			}
			value, ok := propMap["Value"].(map[string]any)
			if !ok {
				continue
			}
			// Check for Widgets array
			widgetsArr := getBsonArrayElements(value["Widgets"])
			if len(widgetsArr) == 0 {
				continue
			}
			var result []rawWidget
			for _, wgt := range widgetsArr {
				wgtMap, ok := wgt.(map[string]any)
				if !ok {
					continue
				}
				result = append(result, e.parseRawWidget(wgtMap)...)
			}
			if len(result) > 0 {
				return result
			}
		}
	}

	return nil
}

// extractGalleryFilters extracts the filter widgets from a CustomWidget Gallery.
func (e *Executor) extractGalleryFilters(w map[string]any) []rawWidget {
	return e.extractGalleryWidgetsByPropertyKey(w, "filtersPlaceholder")
}

// extractGallerySelection extracts the selection mode from a CustomWidget Gallery.
func (e *Executor) extractGallerySelection(w map[string]any) string {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return ""
	}

	// Search through properties for one with Selection != "None"
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		// Check for Selection field
		if sel, ok := value["Selection"].(string); ok && sel != "None" && sel != "" {
			return sel
		}
	}
	return ""
}

// extractFilterAttributes extracts the filter attributes from a TextFilter/NumberFilter widget.
func (e *Executor) extractFilterAttributes(w map[string]any) []string {
	// Use the generic property extraction helper
	return e.extractCustomWidgetPropertyAttributes(w, "attributes")
}

// extractFilterExpression extracts the default filter expression from a TextFilter widget.
func (e *Executor) extractFilterExpression(w map[string]any) string {
	return e.extractCustomWidgetPropertyString(w, "defaultFilter")
}

// extractCustomWidgetPropertyAttributeRef extracts an AttributeRef value from a named CustomWidget property.
func (e *Executor) extractCustomWidgetPropertyAttributeRef(w map[string]any, propertyKey string) string {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return ""
	}

	propTypeKeyMap := buildPropertyTypeKeyMap(w, false)

	// Search through properties for the named property
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		typePointerID := extractBinaryID(propMap["TypePointer"])
		propKey := propTypeKeyMap[typePointerID]
		if propKey != propertyKey {
			continue
		}
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		if attrRef, ok := value["AttributeRef"].(map[string]any); ok && attrRef != nil {
			if attr, ok := attrRef["Attribute"].(string); ok && attr != "" {
				return shortAttributeName(attr)
			}
		}
	}
	return ""
}

// extractCustomWidgetPropertyAssociation extracts an association name from a named
// CustomWidget property that was written by opAssociation (setAssociationRef).
// The association is stored as EntityRef.Steps[1].Association (qualified path);
// this function returns only the short name (last segment after the final dot).
//
// This is the symmetric counterpart of extractCustomWidgetPropertyAttributeRef,
// handling the EntityRef storage format instead of AttributeRef.
func (e *Executor) extractCustomWidgetPropertyAssociation(w map[string]any, propertyKey string) string {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return ""
	}

	propTypeKeyMap := buildPropertyTypeKeyMap(w, false)

	// Find the named property and extract EntityRef.Steps[1].Association
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		typePointerID := extractBinaryID(propMap["TypePointer"])
		if propTypeKeyMap[typePointerID] != propertyKey {
			continue
		}
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}
		entityRef, ok := value["EntityRef"].(map[string]any)
		if !ok || entityRef == nil {
			return ""
		}
		steps := getBsonArrayElements(entityRef["Steps"])
		// Steps layout: [int32(2), step0, step1, ...] — first element is version marker
		for _, step := range steps {
			stepMap, ok := step.(map[string]any)
			if !ok {
				continue
			}
			if assoc := extractString(stepMap["Association"]); assoc != "" {
				return shortAttributeName(assoc)
			}
		}
	}
	return ""
}

// extractCustomWidgetPropertyString extracts a string property value from a CustomWidget.
func (e *Executor) extractCustomWidgetPropertyString(w map[string]any, propertyKey string) string {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return ""
	}

	propTypeKeyMap := buildPropertyTypeKeyMap(w, false)

	// Search through properties for the named property
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		// Check property key via TypePointer
		typePointerID := extractBinaryID(propMap["TypePointer"])
		propKey := propTypeKeyMap[typePointerID]
		if propKey != propertyKey {
			continue
		}

		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}

		// Extract PrimitiveValue for string properties
		if pv, ok := value["PrimitiveValue"].(string); ok && pv != "" {
			return pv
		}
	}
	return ""
}

// extractCustomWidgetPropertyAttributes extracts attribute references from a CustomWidget property.
func (e *Executor) extractCustomWidgetPropertyAttributes(w map[string]any, propertyKey string) []string {
	obj, ok := w["Object"].(map[string]any)
	if !ok {
		return nil
	}

	propTypeKeyMap := buildPropertyTypeKeyMap(w, false)

	// Search through properties for the named property
	props := getBsonArrayElements(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		// Check property key via TypePointer
		typePointerID := extractBinaryID(propMap["TypePointer"])
		propKey := propTypeKeyMap[typePointerID]
		if propKey != propertyKey {
			continue
		}

		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			continue
		}

		// Extract from Objects array (each object has an AttributeRef)
		objects := getBsonArrayElements(value["Objects"])
		var result []string
		for _, objItem := range objects {
			objMap, ok := objItem.(map[string]any)
			if !ok {
				continue
			}
			// Look for Properties inside each object
			objProps := getBsonArrayElements(objMap["Properties"])
			for _, objProp := range objProps {
				objPropMap, ok := objProp.(map[string]any)
				if !ok {
					continue
				}
				objValue, ok := objPropMap["Value"].(map[string]any)
				if !ok {
					continue
				}
				// Check for AttributeRef
				if attrRef, ok := objValue["AttributeRef"].(map[string]any); ok && attrRef != nil {
					if attr, ok := attrRef["Attribute"].(string); ok && attr != "" {
						result = append(result, shortAttributeName(attr))
					}
				}
			}
		}
		return result
	}
	return nil
}
