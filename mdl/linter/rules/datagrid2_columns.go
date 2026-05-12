// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
)

// DataGrid2ColumnRule checks DataGrid2 (pluggable widget) columns for the common
// misconfiguration where sorting is enabled but no attribute is bound.
//
// Root cause: Mendix DataGrid2 columns default to Sortable=true. If a column
// has no Attribute binding and Sortable remains true, Studio Pro (and mx check)
// reports: "An attribute is required when column sorting is enabled."
//
// This also catches entirely empty DataGrid2 widgets (no columns at all), which
// produce placeholder columns with the same error when the project is opened.
type DataGrid2ColumnRule struct{}

// NewDataGrid2ColumnRule creates a new DataGrid2 column validation rule.
func NewDataGrid2ColumnRule() *DataGrid2ColumnRule {
	return &DataGrid2ColumnRule{}
}

func (r *DataGrid2ColumnRule) ID() string                       { return "MPR009" }
func (r *DataGrid2ColumnRule) Name() string                     { return "DataGrid2Column" }
func (r *DataGrid2ColumnRule) Category() string                 { return "correctness" }
func (r *DataGrid2ColumnRule) DefaultSeverity() linter.Severity { return linter.SeverityError }

func (r *DataGrid2ColumnRule) Description() string {
	return "Checks DataGrid2 columns: every sortable column must have an Attribute binding"
}

const datagrid2WidgetType = "com.mendix.widget.web.datagrid.Datagrid"

// datagrid2Issue describes a single DataGrid2 problem found in BSON.
type datagrid2Issue struct {
	// WidgetName is the name of the DataGrid2 widget (e.g., "dgContractors").
	WidgetName string
	// ColumnCaption is the user-visible caption of the offending column.
	ColumnCaption string
	// NoColumns is true when the DataGrid2 has zero columns (empty stub).
	NoColumns bool
}

// Check scans all pages/snippets that contain DataGrid2 widgets and verifies
// that each column with sorting enabled has an attribute binding.
func (r *DataGrid2ColumnRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	if reader == nil {
		return nil
	}

	// Collect unique containers (pages/snippets) that have at least one DataGrid2.
	type containerInfo struct {
		ID            string
		QualifiedName string
		Type          string
		ModuleName    string
	}
	seen := make(map[string]containerInfo)

	for w := range ctx.Widgets() {
		if ctx.IsExcluded(w.ModuleName) {
			continue
		}
		if w.WidgetType != datagrid2WidgetType {
			continue
		}
		if _, ok := seen[w.ContainerID]; !ok {
			seen[w.ContainerID] = containerInfo{
				ID:            w.ContainerID,
				QualifiedName: w.ContainerQualifiedName,
				Type:          w.ContainerType,
				ModuleName:    w.ModuleName,
			}
		}
	}

	var violations []linter.Violation

	for _, c := range seen {
		rawData, err := reader.GetRawUnit(model.ID(c.ID))
		if err != nil || rawData == nil {
			continue
		}

		docType := "page"
		if c.Type == "SNIPPET" {
			docType = "snippet"
		}

		issues := findDataGrid2Issues(rawData)
		for _, issue := range issues {
			var msg, suggestion string
			if issue.NoColumns {
				msg = fmt.Sprintf(
					"DataGrid2 '%s' in %s has no columns — Studio Pro will generate broken placeholder columns",
					issue.WidgetName, c.QualifiedName)
				suggestion = "Add column definitions with Attribute: <name> for each entity attribute to display"
			} else {
				msg = fmt.Sprintf(
					"DataGrid2 '%s' column '%s' in %s has Sortable:true but no Attribute binding — Studio Pro error CE",
					issue.WidgetName, issue.ColumnCaption, c.QualifiedName)
				suggestion = "Add Attribute: <attributeName> to the column, or set Sortable: false for action-only columns"
			}

			violations = append(violations, linter.Violation{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message:  msg,
				Location: linter.Location{
					Module:       c.ModuleName,
					DocumentType: docType,
					DocumentName: docNameFromQualified(c.QualifiedName),
					DocumentID:   c.ID,
				},
				Suggestion: suggestion,
			})
		}
	}

	return violations
}

// findDataGrid2Issues walks a raw BSON page/snippet document and returns all
// DataGrid2 column configuration problems found.
func findDataGrid2Issues(rawData map[string]any) []datagrid2Issue {
	// Pages: FormCall → Arguments → Widgets
	if formCall, ok := rawData["FormCall"].(map[string]any); ok {
		args := getBsonArray(formCall["Arguments"])
		var result []datagrid2Issue
		for _, arg := range args {
			if argMap, ok := arg.(map[string]any); ok {
				for _, w := range getBsonArray(argMap["Widgets"]) {
					if wMap, ok := w.(map[string]any); ok {
						result = append(result, findDataGrid2IssuesRecursive(wMap)...)
					}
				}
			}
		}
		return result
	}

	// Snippets: Widgets at top level
	var result []datagrid2Issue
	for _, w := range getBsonArray(rawData["Widgets"]) {
		if wMap, ok := w.(map[string]any); ok {
			result = append(result, findDataGrid2IssuesRecursive(wMap)...)
		}
	}
	return result
}

// findDataGrid2IssuesRecursive walks a widget subtree, inspecting every
// CustomWidgets$CustomWidget that carries a "columns" property list.
func findDataGrid2IssuesRecursive(w map[string]any) []datagrid2Issue {
	var result []datagrid2Issue

	widgetType := extractStr(w["$Type"])

	if widgetType == "CustomWidgets$CustomWidget" {
		obj, _ := w["Object"].(map[string]any)
		if obj != nil {
			issues := checkDataGrid2Object(extractStr(w["Name"]), obj)
			result = append(result, issues...)
		}
	}

	// Recurse into child widgets (handles LayoutGrid, DataView, etc.)
	for _, child := range getBsonArray(w["Widgets"]) {
		if childMap, ok := child.(map[string]any); ok {
			result = append(result, findDataGrid2IssuesRecursive(childMap)...)
		}
	}
	for _, row := range getBsonArray(w["Rows"]) {
		if rowMap, ok := row.(map[string]any); ok {
			for _, col := range getBsonArray(rowMap["Columns"]) {
				if colMap, ok := col.(map[string]any); ok {
					for _, cw := range getBsonArray(colMap["Widgets"]) {
						if cwMap, ok := cw.(map[string]any); ok {
							result = append(result, findDataGrid2IssuesRecursive(cwMap)...)
						}
					}
				}
			}
		}
	}
	for _, fw := range getBsonArray(w["FooterWidgets"]) {
		if fwMap, ok := fw.(map[string]any); ok {
			result = append(result, findDataGrid2IssuesRecursive(fwMap)...)
		}
	}
	for _, tp := range getBsonArray(w["TabPages"]) {
		if tpMap, ok := tp.(map[string]any); ok {
			for _, tw := range getBsonArray(tpMap["Widgets"]) {
				if twMap, ok := tw.(map[string]any); ok {
					result = append(result, findDataGrid2IssuesRecursive(twMap)...)
				}
			}
		}
	}
	// Recurse into nested custom widget property widgets
	if obj, ok := w["Object"].(map[string]any); ok {
		for _, prop := range getBsonArray(obj["Properties"]) {
			if propMap, ok := prop.(map[string]any); ok {
				if value, ok := propMap["Value"].(map[string]any); ok {
					for _, pw := range getBsonArray(value["Widgets"]) {
						if pwMap, ok := pw.(map[string]any); ok {
							result = append(result, findDataGrid2IssuesRecursive(pwMap)...)
						}
					}
				}
			}
		}
	}

	return result
}

// checkDataGrid2Object inspects a CustomWidget Object to find DataGrid2 column issues.
// Returns nil if the widget is not a DataGrid2 or has no problems.
func checkDataGrid2Object(widgetName string, obj map[string]any) []datagrid2Issue {
	// Find the "columns" property in the widget's property list.
	props := getBsonArray(obj["Properties"])
	for _, prop := range props {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		if extractStr(propMap["Key"]) != "columns" {
			continue
		}
		// Found the columns property — this is a DataGrid2.
		value, ok := propMap["Value"].(map[string]any)
		if !ok {
			return nil
		}
		columns := getBsonArray(value["Objects"])
		if len(columns) == 0 {
			// No columns at all: mxcli-created stub, will become broken on open.
			return []datagrid2Issue{{WidgetName: widgetName, NoColumns: true}}
		}
		return checkColumnObjects(widgetName, columns)
	}
	return nil
}

// checkColumnObjects inspects each column object for the sortable-without-attribute issue.
func checkColumnObjects(widgetName string, columns []any) []datagrid2Issue {
	var result []datagrid2Issue

	for _, col := range columns {
		colMap, ok := col.(map[string]any)
		if !ok {
			continue
		}
		colProps := getBsonArray(colMap["Properties"])

		var (
			caption         string
			sortable        = true  // DataGrid2 default
			hasAttribute    = false
			showContentAs   = "attribute" // DataGrid2 default
		)

		for _, p := range colProps {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			key := extractStr(pm["Key"])
			val, _ := pm["Value"].(map[string]any)
			if val == nil {
				continue
			}

			switch key {
			case "caption":
				// Caption is a translatableString object; extract the default text.
				caption = extractColumnCaption(val)
			case "sortable":
				sortable = extractBoolValue(val, true)
			case "attribute":
				hasAttribute = isAttributeBound(val)
			case "showContentAs":
				showContentAs = extractEnumValue(val, "attribute")
			}
		}

		// Action-only columns (showContentAs = customContent) don't need an attribute.
		if showContentAs == "customContent" {
			continue
		}
		// If sortable and no attribute binding, report.
		if sortable && !hasAttribute {
			result = append(result, datagrid2Issue{
				WidgetName:    widgetName,
				ColumnCaption: caption,
			})
		}
	}

	return result
}

// extractColumnCaption pulls the caption text from a DataGrid2 column caption value.
// The caption is stored as a translatableString (map with a "Translations" list).
func extractColumnCaption(val map[string]any) string {
	// Try direct Value string first
	if s := extractStr(val["Value"]); s != "" {
		return s
	}
	// TranslatableString: look at Translations[0].Text
	translations := getBsonArray(val["Translations"])
	for _, t := range translations {
		if tm, ok := t.(map[string]any); ok {
			if text := extractStr(tm["Text"]); text != "" {
				return text
			}
		}
	}
	return "(unnamed)"
}

// extractBoolValue returns the boolean value from a WidgetBooleanValue BSON map.
func extractBoolValue(val map[string]any, defaultVal bool) bool {
	v, ok := val["Value"]
	if !ok {
		return defaultVal
	}
	switch b := v.(type) {
	case bool:
		return b
	case int32:
		return b != 0
	case int64:
		return b != 0
	}
	return defaultVal
}

// extractEnumValue returns the enumeration string from a WidgetEnumerationValue BSON map.
func extractEnumValue(val map[string]any, defaultVal string) string {
	if s := extractStr(val["Value"]); s != "" {
		return s
	}
	return defaultVal
}

// isAttributeBound returns true if the WidgetAttributeValue has a non-nil AttributeRef.
func isAttributeBound(val map[string]any) bool {
	ref, ok := val["AttributeRef"]
	if !ok || ref == nil {
		return false
	}
	// AttributeRef is a map — check it is non-empty.
	if refMap, ok := ref.(map[string]any); ok {
		return len(refMap) > 0
	}
	return false
}
