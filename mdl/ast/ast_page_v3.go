// SPDX-License-Identifier: Apache-2.0

package ast

import "strings"

// =============================================================================
// V3 Page AST Types
// =============================================================================
//
// V3 follows the pattern: WIDGET name (Prop: Value) { children }
//
// Key differences from V2:
// - Page header uses single () block with Params:, Title:, Layout:, Url:
// - DataSource: replaces -> for containers
// - Attribute: replaces -> for input widgets and columns
// - Action: replaces -> for buttons
// - All properties are explicit key-value pairs
//

// CreatePageStmtV3 represents a V3 page creation statement.
// V3 syntax: CREATE PAGE Module.Page (Title: '...', Layout: ...) { widgets }
type CreatePageStmtV3 struct {
	Name          QualifiedName
	Parameters    []PageParameter // From Params: { } block
	Variables     []PageVariable  // From Variables: { } block
	Title         string
	Layout        string
	URL           string
	Folder        string
	Widgets       []*WidgetV3
	Documentation string
	IsReplace     bool // CREATE OR REPLACE
	IsModify      bool // CREATE OR MODIFY
	Excluded      bool // @excluded — document excluded from project
}

func (s *CreatePageStmtV3) isStatement()     {}
func (s *CreatePageStmtV3) TypeName() string { return "CreatePageStmtV3" }

// CreateSnippetStmtV3 represents a V3 snippet creation statement.
type CreateSnippetStmtV3 struct {
	Name          QualifiedName
	Parameters    []PageParameter // From Params: { } block
	Variables     []PageVariable  // From Variables: { } block
	Folder        string
	Widgets       []*WidgetV3
	Documentation string
	IsReplace     bool
	IsModify      bool
}

func (s *CreateSnippetStmtV3) isStatement()     {}
func (s *CreateSnippetStmtV3) TypeName() string { return "CreateSnippetStmtV3" }

// WidgetV3 represents a V3 widget with explicit properties.
// Pattern: WIDGET name (Props) { children }
type WidgetV3 struct {
	Type       string         // Widget type: TEXTBOX, DATAVIEW, etc.
	Name       string         // Required widget name
	Properties map[string]any // All properties as key-value pairs
	Children   []*WidgetV3    // Child widgets
}

// DataSourceV3 represents a V3 datasource expression.
type DataSourceV3 struct {
	Type            string          // "parameter", "database", "microflow", "nanoflow", "association", "selection"
	Reference       string          // Entity name, flow name, widget name, or parameter name
	ContextVariable string          // Context variable name (for association source: $currentObject → "currentObject")
	Args            []FlowArgV3     // Arguments for microflow/nanoflow calls
	Where           string          // XPath constraint (for database source)
	OrderBy         []OrderByItemV3 // Sort order (for database source)
}

// FlowArgV3 represents an argument for microflow/nanoflow/page calls.
type FlowArgV3 struct {
	Name  string // Parameter name
	Value any    // Value (expression)
}

// OrderByItemV3 represents a sort column.
type OrderByItemV3 struct {
	Attribute string // Attribute path
	Direction string // "ASC" or "DESC"
}

// ActionV3 represents a V3 action expression.
type ActionV3 struct {
	Type         string      // "save", "cancel", "close", "delete", "create", "showPage", "microflow", "nanoflow", "openLink", "signOut", "completeTask"
	Target       string      // Entity, page, or flow qualified name (for create/showPage/microflow/nanoflow)
	Args         []FlowArgV3 // Arguments for showPage/microflow calls
	ThenAction   *ActionV3   // For CREATE_OBJECT ... THEN ...
	ClosePage    bool        // For SAVE_CHANGES CLOSE_PAGE
	LinkURL      string      // For OPEN_LINK
	OutcomeValue string      // For COMPLETE_TASK
}

// ColumnV3 represents a V3 datagrid column.
type ColumnV3 struct {
	Name       string      // Column name
	Attr       string      // Bound attribute
	Caption    string      // Header caption
	CanSort    bool        // Sortable
	CanFilter  bool        // Filterable
	Children   []*WidgetV3 // For action columns or custom content
	Properties map[string]any
}

// ParamAssignmentV3 represents a template parameter: {1} = value
type ParamAssignmentV3 struct {
	Index int // Parameter index (1, 2, 3, ...)
	Value any // Expression value
}

// DesignPropertyEntryV3 represents a single design property key-value pair.
type DesignPropertyEntryV3 struct {
	Key   string // e.g., "Spacing top"
	Value string // e.g., "Large", "ON", "OFF"
}

// Helper functions to extract typed properties from WidgetV3

// GetStringProp returns a string property or empty string if not found.
// Lookup is case-insensitive on fallback: the parser stores generic
// IDENTIFIER property keys verbatim (e.g. "collapsible") but builders
// historically look up Title-Case names (e.g. "Collapsible"). Try the
// requested key first, then fold to case-insensitive scan so the
// canonical MDL lowercase convention round-trips correctly.
func (w *WidgetV3) GetStringProp(key string) string {
	if v, ok := w.Properties[key].(string); ok {
		return v
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

// GetIntProp returns an int property or 0 if not found.
func (w *WidgetV3) GetIntProp(key string) int {
	switch v := w.Properties[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// GetBoolProp returns a bool property or false if not found.
func (w *WidgetV3) GetBoolProp(key string) bool {
	if v, ok := w.Properties[key].(bool); ok {
		return v
	}
	return false
}

// GetDataSource returns the DataSource property or nil if not found.
func (w *WidgetV3) GetDataSource() *DataSourceV3 {
	if v, ok := w.Properties["DataSource"].(*DataSourceV3); ok {
		return v
	}
	return nil
}

// GetAction returns the Action property or nil if not found.
func (w *WidgetV3) GetAction() *ActionV3 {
	if v, ok := w.Properties["Action"].(*ActionV3); ok {
		return v
	}
	return nil
}

// GetAttribute returns the Attribute property (attribute path) or empty string.
func (w *WidgetV3) GetAttribute() string {
	return w.GetStringProp("Attribute")
}

// GetBinds returns the Binds property (attribute path) or empty string.
// Deprecated: use GetAttribute instead.
func (w *WidgetV3) GetBinds() string {
	if v, ok := w.Properties["Binds"].(string); ok {
		return v
	}
	return ""
}

// GetLabel returns the Label property or empty string.
func (w *WidgetV3) GetLabel() string {
	return w.GetStringProp("Label")
}

// GetCaption returns the Caption property or empty string.
func (w *WidgetV3) GetCaption() string {
	return w.GetStringProp("Caption")
}

// GetContent returns the Content property or empty string.
func (w *WidgetV3) GetContent() string {
	return w.GetStringProp("Content")
}

// GetRenderMode returns the RenderMode property or empty string.
func (w *WidgetV3) GetRenderMode() string {
	return w.GetStringProp("RenderMode")
}

// GetButtonStyle returns the ButtonStyle property or empty string.
func (w *WidgetV3) GetButtonStyle() string {
	return w.GetStringProp("ButtonStyle")
}

// GetClass returns the Class property or empty string.
func (w *WidgetV3) GetClass() string {
	return w.GetStringProp("Class")
}

// GetStyle returns the Style property or empty string.
func (w *WidgetV3) GetStyle() string {
	return w.GetStringProp("Style")
}

// GetDesktopWidth returns the DesktopWidth property.
func (w *WidgetV3) GetDesktopWidth() any {
	return w.Properties["DesktopWidth"]
}

// GetContentParams returns ContentParams or nil.
func (w *WidgetV3) GetContentParams() []ParamAssignmentV3 {
	if v, ok := w.Properties["ContentParams"].([]ParamAssignmentV3); ok {
		return v
	}
	return nil
}

// GetCaptionParams returns CaptionParams or nil.
func (w *WidgetV3) GetCaptionParams() []ParamAssignmentV3 {
	if v, ok := w.Properties["CaptionParams"].([]ParamAssignmentV3); ok {
		return v
	}
	return nil
}

// GetAttributes returns the Attributes property as a string slice (for filter widgets).
func (w *WidgetV3) GetAttributes() []string {
	if v, ok := w.Properties["Attributes"].([]string); ok {
		return v
	}
	return nil
}

// GetFilterType returns the FilterType property (for filter widgets).
func (w *WidgetV3) GetFilterType() string {
	return w.GetStringProp("FilterType")
}

// GetAttr returns the Attr property (for COLUMN widgets) or empty string.
func (w *WidgetV3) GetAttr() string {
	return w.GetStringProp("Attr")
}

// GetSnippet returns the Snippet property (qualified name) or empty string.
func (w *WidgetV3) GetSnippet() string {
	return w.GetStringProp("Snippet")
}

// SnippetCallParam represents one parameter mapping in a SNIPPETCALL Params: block.
type SnippetCallParam struct {
	ParamName string // Parameter name as written (may include leading $)
	Variable  string // Variable being passed, always includes leading $
}

// GetSnippetParams returns the Params mappings for a SNIPPETCALL widget, or nil.
func (w *WidgetV3) GetSnippetParams() []SnippetCallParam {
	if v, ok := w.Properties["Params"].([]SnippetCallParam); ok {
		return v
	}
	return nil
}

// GetSelection returns the Selection mode or empty string.
func (w *WidgetV3) GetSelection() string {
	return w.GetStringProp("Selection")
}

// GetDesignProperties returns the DesignProperties or nil.
func (w *WidgetV3) GetDesignProperties() []DesignPropertyEntryV3 {
	if v, ok := w.Properties["DesignProperties"].([]DesignPropertyEntryV3); ok {
		return v
	}
	return nil
}
