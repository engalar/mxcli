// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/pages"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ============================================================================
// Describe Page
// ============================================================================

// describePage handles DESCRIBE PAGE command - outputs MDL V3 syntax.
func (e *Executor) describePage(name ast.QualifiedName) error {
	// Get hierarchy for module/folder resolution
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the page
	allPages, err := e.reader.ListPages()
	if err != nil {
		return fmt.Errorf("failed to list pages: %w", err)
	}

	var foundPage *pages.Page
	for _, p := range allPages {
		modID := h.FindModuleID(p.ContainerID)
		modName := h.GetModuleName(modID)
		if p.Name == name.Name && (name.Module == "" || modName == name.Module) {
			foundPage = p
			break
		}
	}

	if foundPage == nil {
		return fmt.Errorf("page %s not found", name.String())
	}

	// Get module name for the page
	modID := h.FindModuleID(foundPage.ContainerID)
	modName := h.GetModuleName(modID)

	// Output documentation if present
	if foundPage.Documentation != "" {
		lines := strings.Split(foundPage.Documentation, "\n")
		fmt.Fprint(e.output, "/**\n")
		for _, line := range lines {
			fmt.Fprintf(e.output, " * %s\n", line)
		}
		fmt.Fprint(e.output, " */\n")
	}

	// Get title
	title := ""
	if foundPage.Title != nil {
		title = foundPage.Title.GetTranslation("en_US")
		if title == "" {
			for _, text := range foundPage.Title.Translations {
				title = text
				break
			}
		}
	}

	// Get layout from raw data
	layoutName := ""
	rawData, _ := e.reader.GetRawUnit(foundPage.ID)
	if rawData != nil {
		if formCall, ok := rawData["FormCall"].(map[string]any); ok {
			if layoutID := extractBinaryID(formCall["Layout"]); layoutID != "" {
				layoutName = e.resolveLayoutName(model.ID(layoutID))
			} else if formName, ok := formCall["Form"].(string); ok && formName != "" {
				layoutName = formName
			}
		}
	}

	// V3 syntax: CREATE PAGE Module.Page (Title: '...', Layout: ..., Params: { })
	header := fmt.Sprintf("CREATE OR REPLACE PAGE %s.%s", modName, foundPage.Name)
	props := []string{}
	if title != "" {
		props = append(props, fmt.Sprintf("Title: %s", mdlQuote(title)))
	}
	if layoutName != "" {
		props = append(props, fmt.Sprintf("Layout: %s", layoutName))
	}
	if foundPage.URL != "" {
		props = append(props, fmt.Sprintf("Url: %s", mdlQuote(foundPage.URL)))
	}
	if folderPath := h.BuildFolderPath(foundPage.ContainerID); folderPath != "" {
		props = append(props, fmt.Sprintf("Folder: %s", mdlQuote(folderPath)))
	}
	if len(foundPage.Parameters) > 0 {
		params := []string{}
		for _, p := range foundPage.Parameters {
			entityName := p.EntityName
			if entityName == "" {
				entityName = string(p.EntityID)
			}
			params = append(params, fmt.Sprintf("$%s: %s", p.Name, entityName))
		}
		props = append(props, fmt.Sprintf("Params: { %s }", strings.Join(params, ", ")))
	}
	// Output page variables from raw BSON
	if rawData != nil {
		vars := getBsonArrayMaps(rawData["Variables"])
		if len(vars) > 0 {
			varParts := []string{}
			for _, v := range vars {
				varName, _ := v["Name"].(string)
				defaultVal, _ := v["DefaultValue"].(string)
				varTypeName := "Unknown"
				if vt, ok := v["VariableType"].(map[string]any); ok {
					if vtType, ok := vt["$Type"].(string); ok {
						varTypeName = bsonTypeToMDLType(vtType)
					}
				}
				varParts = append(varParts, fmt.Sprintf("$%s: %s = %s", varName, varTypeName, mdlQuote(defaultVal)))
			}
			props = append(props, fmt.Sprintf("Variables: { %s }", strings.Join(varParts, ", ")))
		}
	}

	// Output widgets from raw page data
	rawWidgets := e.getPageWidgetsFromRaw(foundPage.ID)
	if len(rawWidgets) > 0 {
		formatWidgetProps(e.output, "", header, props, " {\n")
		for _, w := range rawWidgets {
			e.outputWidgetMDLV3(w, 1)
		}
		fmt.Fprint(e.output, "}")
	} else {
		formatWidgetProps(e.output, "", header, props, "")
	}

	// Add GRANT VIEW if roles are assigned
	if len(foundPage.AllowedRoles) > 0 {
		roles := make([]string, len(foundPage.AllowedRoles))
		for i, r := range foundPage.AllowedRoles {
			roles[i] = string(r)
		}
		fmt.Fprintf(e.output, "\n\nGRANT VIEW ON PAGE %s.%s TO %s;",
			modName, foundPage.Name, strings.Join(roles, ", "))
	}

	fmt.Fprint(e.output, "\n")
	return nil
}

// formatParametersV3 formats parameter expressions for MDL V3 ContentParams clause.
// Returns format like: {1} = FirstName, {2} = $ParamName.Attribute
// Parameter references keep their $ prefix, entity paths are shown without prefix.
func formatParametersV3(params []string) []string {
	result := make([]string, len(params))
	for i, p := range params {
		// Keep the parameter as-is - extraction already formats correctly:
		// - Entity paths: Module.Entity.Attribute (no $ prefix)
		// - Parameter refs: $ParamName.Attribute (with $ prefix)
		result[i] = fmt.Sprintf("{%d} = %s", i+1, p)
	}
	return result
}

// describeSnippet handles DESCRIBE SNIPPET command - outputs MDL V3 syntax.
func (e *Executor) describeSnippet(name ast.QualifiedName) error {
	// Get hierarchy for module/folder resolution
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the snippet
	allSnippets, err := e.reader.ListSnippets()
	if err != nil {
		return fmt.Errorf("failed to list snippets: %w", err)
	}

	var foundSnippet *pages.Snippet
	for _, s := range allSnippets {
		modID := h.FindModuleID(s.ContainerID)
		modName := h.GetModuleName(modID)
		if s.Name == name.Name && (name.Module == "" || modName == name.Module) {
			foundSnippet = s
			break
		}
	}

	if foundSnippet == nil {
		return fmt.Errorf("snippet %s not found", name.String())
	}

	// Get module name for the snippet
	modID := h.FindModuleID(foundSnippet.ContainerID)
	modName := h.GetModuleName(modID)

	// Output documentation if present
	if foundSnippet.Documentation != "" {
		lines := strings.Split(foundSnippet.Documentation, "\n")
		fmt.Fprint(e.output, "/**\n")
		for _, line := range lines {
			fmt.Fprintf(e.output, " * %s\n", line)
		}
		fmt.Fprint(e.output, " */\n")
	}

	// Get raw data to check for parameters
	rawData, _ := e.reader.GetRawUnit(foundSnippet.ID)
	var params []map[string]any
	if rawData != nil {
		params = getBsonArrayMaps(rawData["Parameters"])
	}

	// Output CREATE SNIPPET statement (V3 syntax)
	fmt.Fprintf(e.output, "CREATE OR REPLACE SNIPPET %s.%s", modName, foundSnippet.Name)
	folderPath := h.BuildFolderPath(foundSnippet.ContainerID)
	if len(params) > 0 || folderPath != "" {
		snippetProps := []string{}
		if len(params) > 0 {
			paramParts := []string{}
			for _, p := range params {
				paramName, _ := p["Name"].(string)
				entityName := extractEntityQualifiedName(p["ParameterType"])
				paramParts = append(paramParts, fmt.Sprintf("$%s: %s", paramName, entityName))
			}
			snippetProps = append(snippetProps, fmt.Sprintf("Params: { %s }", strings.Join(paramParts, ", ")))
		}
		if folderPath != "" {
			snippetProps = append(snippetProps, fmt.Sprintf("Folder: %s", mdlQuote(folderPath)))
		}
		fmt.Fprintf(e.output, " (%s)", strings.Join(snippetProps, ", "))
	}

	// Output widgets from raw snippet data
	rawWidgets := e.getSnippetWidgetsFromRaw(foundSnippet.ID)
	if len(rawWidgets) > 0 {
		fmt.Fprint(e.output, " {\n")
		for _, w := range rawWidgets {
			e.outputWidgetMDLV3(w, 1)
		}
		fmt.Fprint(e.output, "}")
	}

	fmt.Fprint(e.output, "\n")
	return nil
}

// describeLayout handles DESCRIBE LAYOUT command - outputs MDL-style representation.
func (e *Executor) describeLayout(name ast.QualifiedName) error {
	// Get hierarchy for module/folder resolution
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the layout
	allLayouts, err := e.reader.ListLayouts()
	if err != nil {
		return fmt.Errorf("failed to list layouts: %w", err)
	}

	var foundLayout *pages.Layout
	for _, l := range allLayouts {
		modID := h.FindModuleID(l.ContainerID)
		modName := h.GetModuleName(modID)
		if l.Name == name.Name && (name.Module == "" || modName == name.Module) {
			foundLayout = l
			break
		}
	}

	if foundLayout == nil {
		return fmt.Errorf("layout %s not found", name.String())
	}

	// Get module name for the layout
	modID := h.FindModuleID(foundLayout.ContainerID)
	modName := h.GetModuleName(modID)

	// Output documentation if present
	if foundLayout.Documentation != "" {
		lines := strings.Split(foundLayout.Documentation, "\n")
		fmt.Fprint(e.output, "/**\n")
		for _, line := range lines {
			fmt.Fprintf(e.output, " * %s\n", line)
		}
		fmt.Fprint(e.output, " */\n")
	}

	// Output layout type comment
	layoutTypeStr := string(foundLayout.LayoutType)
	if layoutTypeStr == "" {
		layoutTypeStr = "Responsive"
	}

	fmt.Fprintf(e.output, "-- Layout Type: %s\n", layoutTypeStr)
	fmt.Fprintf(e.output, "-- This is a layout document. Layouts define the structure that pages are built upon.\n")
	fmt.Fprintf(e.output, "-- Layouts cannot be created via MDL; they must be created in Studio Pro.\n\n")

	// Output as a comment showing the layout name
	fmt.Fprintf(e.output, "-- LAYOUT %s.%s\n", modName, foundLayout.Name)

	// Output widgets from raw layout data
	rawWidgets := e.getLayoutWidgetsFromRaw(foundLayout.ID)
	if len(rawWidgets) > 0 {
		fmt.Fprint(e.output, "-- Widget structure:\n")
		for _, w := range rawWidgets {
			e.outputWidgetMDLV3Comment(w, 0)
		}
	}

	fmt.Fprint(e.output, "\n")
	return nil
}

// getLayoutWidgetsFromRaw extracts widgets from raw layout BSON.
func (e *Executor) getLayoutWidgetsFromRaw(layoutID model.ID) []rawWidget {
	// Get raw layout data
	rawData, err := e.reader.GetRawUnit(layoutID)
	if err != nil {
		return nil
	}

	// Layouts have a Widget field containing the root widget
	widgetData, ok := rawData["Widget"].(map[string]any)
	if !ok {
		return nil
	}

	return e.parseRawWidget(widgetData)
}

// outputWidgetMDLV3Comment outputs a widget as MDL V3 comment.
func (e *Executor) outputWidgetMDLV3Comment(w rawWidget, indent int) {
	prefix := strings.Repeat("  ", indent)
	fmt.Fprintf(e.output, "%s-- %s %s\n", prefix, w.Type, w.Name)

	// Output children
	for _, child := range w.Children {
		e.outputWidgetMDLV3Comment(child, indent+1)
	}
}

// getSnippetWidgetsFromRaw extracts widgets from raw snippet BSON.
func (e *Executor) getSnippetWidgetsFromRaw(snippetID model.ID) []rawWidget {
	// Get raw snippet data
	rawData, err := e.reader.GetRawUnit(snippetID)
	if err != nil {
		return nil
	}

	// Handle both snippet formats:
	// - Studio Pro uses "Widgets" (plural): a top-level array of widgets
	// - mxcli uses "Widget" (singular): a single container whose "Widgets" field holds children
	var widgetsArray []any
	if wa := getBsonArrayElements(rawData["Widgets"]); wa != nil {
		widgetsArray = wa
	} else if widgetContainer, ok := rawData["Widget"].(map[string]any); ok {
		widgetsArray = getBsonArrayElements(widgetContainer["Widgets"])
	}
	if widgetsArray == nil {
		return nil
	}

	var result []rawWidget
	for _, w := range widgetsArray {
		if wMap, ok := w.(map[string]any); ok {
			result = append(result, e.parseRawWidget(wMap)...)
		}
	}
	return result
}

// extractEntityQualifiedName extracts the entity qualified name from a parameter type.
func extractEntityQualifiedName(paramType any) string {
	if paramType == nil {
		return "Unknown"
	}
	ptMap, ok := paramType.(map[string]any)
	if !ok {
		return "Unknown"
	}

	// Check for EntityType or ObjectType (snippet parameters use DataTypes$ObjectType)
	if entityType, ok := ptMap["$Type"].(string); ok {
		if entityType == "Pages$EntityType" || entityType == "Forms$EntityType" || entityType == "DataTypes$ObjectType" {
			if entityRef, ok := ptMap["Entity"].(string); ok && entityRef != "" {
				return entityRef
			}
		}
	}
	return "Unknown"
}

// getBsonArrayMaps extracts []map[string]interface{} from BSON array types.
func getBsonArrayMaps(v any) []map[string]any {
	if v == nil {
		return nil
	}
	switch arr := v.(type) {
	case []any:
		var result []map[string]any
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
		return result
	case primitive.A:
		var result []map[string]any
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
		return result
	default:
		return nil
	}
}

// resolveLayoutName resolves a layout ID to its qualified name.
func (e *Executor) resolveLayoutName(layoutID model.ID) string {
	layouts, err := e.reader.ListLayouts()
	if err != nil {
		return string(layoutID)
	}

	h, err := e.getHierarchy()
	if err != nil {
		return string(layoutID)
	}

	for _, l := range layouts {
		if l.ID == layoutID {
			return h.GetQualifiedName(l.ContainerID, l.Name)
		}
	}
	return string(layoutID)
}

// rawSortColumn represents a sort column for describe output.
type rawSortColumn struct {
	Attribute string // Qualified name or simple identifier
	Order     string // "ASC" or "DESC"
}

// rawDataSource represents a data source for describe output.
type rawDataSource struct {
	Type            string          // "microflow", "nanoflow", "parameter", "database"
	Reference       string          // Qualified name or parameter name
	XPathConstraint string          // XPath constraint (WHERE clause)
	SortColumns     []rawSortColumn // Multiple sort columns
}

// rawDataGridColumn represents a DataGrid2 column for describe output.
type rawDataGridColumn struct {
	Name              string // Widget name (generated if not stored)
	Attribute         string
	Caption           string
	CaptionParams     []string    // Parameters for template placeholders in caption
	ShowContentAs     string      // "attribute", "customContent", or "dynamicText"
	ContentWidgets    []rawWidget // Widgets inside the column (for custom content)
	DynamicText       string      // Template text for dynamicText mode
	DynamicTextParams []string    // Parameters for dynamicText template
	Alignment         string      // "left", "center", or "right" (empty = default "left")
	WrapText          string      // "true" or "false" (empty = default "false")
	Sortable          string      // "true" or "false"
	Resizable         string      // "true" or "false"
	Draggable         string      // "true" or "false"
	Hidable           string      // "yes", "hidden", or "no"
	ColumnWidth       string      // "autoFill", "autoFit", or "manual"
	Size              string      // e.g. "200" (default "1")
	Visible           string      // expression, e.g. "true"
	DynamicCellClass  string      // expression
	Tooltip           string      // text
}

// rawWidget represents a widget from raw BSON data for MDL output.
type rawWidget struct {
	Type            string
	Name            string
	Content         string
	Caption         string
	RenderMode      string
	Action          string
	ButtonStyle     string
	Selection       string // For Gallery selection mode (Single, Multi, None)
	Class           string // CSS class from Appearance
	Style           string // Inline CSS style from Appearance
	Parameters      []string
	Children        []rawWidget
	FilterWidgets   []rawWidget // For Gallery filter widgets
	ControlBar      []rawWidget // For DataGrid2 CONTROLBAR widgets
	Rows            []rawWidgetRow
	DataSource      *rawDataSource
	DataGridColumns []rawDataGridColumn // For DataGrid2 widgets
	// Input widget properties
	Editable      string // "Always", "Never", "Conditional"
	ReadOnlyStyle string // "Inherit", "Control", "Text"
	ShowLabel     bool   // Whether label is shown (from LabelTemplate visibility)
	LabelPosition string // "Left", "Top", etc.
	// Filter widget properties
	FilterAttributes []string // Attributes to filter on
	FilterExpression string   // Default filter expression (contains, startsWith, etc.)
	// Paging properties (DataGrid2)
	PageSize          string // e.g. "20", "50"
	Pagination        string // "buttons", "virtualScrolling", "loadMore"
	PagingPosition    string // "bottom", "top", "both"
	ShowPagingButtons string // "always", "auto"
	// Gallery column properties
	DesktopColumns string // e.g. "9", "4"
	TabletColumns  string // e.g. "4", "2"
	PhoneColumns   string // e.g. "2", "1"
	// ComboBox association mode properties
	CaptionAttribute string // Display attribute for association-mode ComboBox
	// GroupBox properties
	Collapsible string // "No", "YesInitiallyExpanded", "YesInitiallyCollapsed"
	HeaderMode  string // "Div", "H1"-"H6"
	// Design properties from Appearance
	DesignProperties []rawDesignProp
	// Explicit widget properties (for generic PLUGGABLEWIDGET output)
	ExplicitProperties []rawExplicitProp
	// Full widget ID (e.g. "com.mendix.widget.custom.switch.Switch")
	WidgetID string
}

// rawExplicitProp represents a non-default property extracted from a CustomWidget.
type rawExplicitProp struct {
	Key   string
	Value string // attribute short name or primitive value
	IsRef bool   // true if this is an attribute reference, false for primitive
}

// rawDesignProp represents a parsed design property from BSON.
type rawDesignProp struct {
	Key       string // Design property key, e.g., "Spacing top"
	ValueType string // "toggle" or "option"
	Option    string // For "option" type: the selected option value
}

type rawWidgetRow struct {
	Columns []rawWidgetColumn
}

type rawWidgetColumn struct {
	Width   int
	Widgets []rawWidget
}

// toBsonArray converts various BSON array types to []interface{}.
func toBsonArray(v any) []any {
	switch arr := v.(type) {
	case []any:
		return arr
	case primitive.A:
		// primitive.A is already []interface{} under the hood
		return []any(arr)
	default:
		return nil
	}
}

// getBsonArrayElements extracts array elements from BSON array format.
// BSON arrays have format [typeIndicator, item1, item2, ...] where typeIndicator is a number.
func getBsonArrayElements(v any) []any {
	arr := toBsonArray(v)
	if len(arr) == 0 {
		return nil
	}
	// Check if first element is a type indicator (integer)
	if _, ok := arr[0].(int32); ok {
		return arr[1:]
	}
	if _, ok := arr[0].(int); ok {
		return arr[1:]
	}
	// No type indicator, return as-is
	return arr
}

// getPageWidgetsFromRaw extracts widgets from raw page BSON.
func (e *Executor) getPageWidgetsFromRaw(pageID model.ID) []rawWidget {
	// Get raw page data
	rawData, err := e.reader.GetRawUnit(pageID)
	if err != nil {
		return nil
	}

	// Parse FormCall.Arguments to get widgets
	formCall, ok := rawData["FormCall"].(map[string]any)
	if !ok {
		return nil
	}

	// Handle both []interface{} and primitive.A types
	args := getBsonArrayElements(formCall["Arguments"])
	if args == nil {
		return nil
	}

	var widgets []rawWidget
	for _, arg := range args {
		argMap, ok := arg.(map[string]any)
		if !ok {
			continue
		}
		argWidgets := getBsonArrayElements(argMap["Widgets"])
		for _, w := range argWidgets {
			if wMap, ok := w.(map[string]any); ok {
				parsed := e.parseRawWidget(wMap)
				for _, pw := range parsed {
					// Unwrap the conditionalVisibilityWidget wrapper that
					// mxcli (and Studio Pro) adds as a layout placeholder
					// container. Without this, DESCRIBE PAGE shows a phantom
					// CONTAINER wrapping all widgets.
					if isConditionalVisibilityWrapper(pw) {
						widgets = append(widgets, pw.Children...)
					} else {
						widgets = append(widgets, pw)
					}
				}
			}
		}
	}
	return widgets
}

// isConditionalVisibilityWrapper returns true if the widget is a DivContainer
// named "conditionalVisibilityWidget*" — the transparent wrapper that layouts
// use to hold placeholder content. We unwrap it so DESCRIBE output is clean
// and round-trippable without phantom CONTAINER nesting.
func isConditionalVisibilityWrapper(w rawWidget) bool {
	if w.Type != "Forms$DivContainer" && w.Type != "Pages$DivContainer" {
		return false
	}
	return strings.HasPrefix(w.Name, "conditionalVisibilityWidget") &&
		w.Class == "" && w.Style == "" && len(w.DesignProperties) == 0
}

// extractBinaryID extracts a UUID string from a BSON binary or string value.
// GUIDs use little-endian byte order for the first three groups:
// - First 4 bytes (group 1): little-endian
// - Next 2 bytes (group 2): little-endian
// - Next 2 bytes (group 3): little-endian
// - Last 8 bytes: big-endian
func extractBinaryID(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return formatGUID(val)
	case primitive.Binary:
		return formatGUID(val.Data)
	default:
		return ""
	}
}

// formatGUID converts a 16-byte GUID to its string representation with proper byte ordering.
func formatGUID(data []byte) string {
	if len(data) != 16 {
		return string(data)
	}
	// Reverse first 4 bytes (group 1)
	g1 := []byte{data[3], data[2], data[1], data[0]}
	// Reverse next 2 bytes (group 2)
	g2 := []byte{data[5], data[4]}
	// Reverse next 2 bytes (group 3)
	g3 := []byte{data[7], data[6]}
	// Last 8 bytes stay in order
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		g1, g2, g3, data[8:10], data[10:16])
}

// wrapStringLiteralExpression wraps a string value in single quotes for Mendix expression format.
// If the value already looks like an expression (starts with $ for variable, or contains operators),
// it is returned as-is.
func wrapStringLiteralExpression(value string) string {
	// If it's already quoted, return as-is
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value
	}
	// If it's a variable reference, return as-is
	if strings.HasPrefix(value, "$") {
		return value
	}
	// If it looks like a number, return as-is
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return value
	}
	// If it's a boolean, return as-is
	if value == "true" || value == "false" {
		return value
	}
	// Otherwise wrap in single quotes as a string literal
	return "'" + value + "'"
}
