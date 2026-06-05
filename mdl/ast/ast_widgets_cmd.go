// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Widget Commands (SHOW WIDGETS, UPDATE WIDGETS)
// ============================================================================

// ShowWidgetsStmt represents: SHOW WIDGETS [WHERE ...] [IN module]
type ShowWidgetsStmt struct {
	Filters  []WidgetFilter
	InModule string
}

func (s *ShowWidgetsStmt) isStatement() {}

// ShowInstalledWidgetsStmt represents: SHOW INSTALLED WIDGETS
// Scans the project's widgets/ directory for .mpk files, listing all
// widget definitions regardless of page instantiation.
type ShowInstalledWidgetsStmt struct{}

func (s *ShowInstalledWidgetsStmt) isStatement() {}

// UpdateWidgetsStmt represents: UPDATE WIDGETS SET ... WHERE ... [IN module] [DRY RUN]
type UpdateWidgetsStmt struct {
	Assignments []WidgetPropertyAssignment
	Filters     []WidgetFilter
	InModule    string
	DryRun      bool
}

func (s *UpdateWidgetsStmt) isStatement() {}

// WidgetFilter represents a filter condition for widget queries.
type WidgetFilter struct {
	Field    string // "WidgetType", "Name", etc.
	Operator string // "=" or "LIKE"
	Value    string
}

// WidgetPropertyAssignment represents 'path' = value in UPDATE WIDGETS.
type WidgetPropertyAssignment struct {
	PropertyPath string
	Value        any // string, int64, float64, bool, or nil
}
