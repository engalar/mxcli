// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Styling Commands (SHOW DESIGN PROPERTIES, DESCRIBE STYLING, ALTER STYLING)
// ============================================================================

// ShowDesignPropertiesStmt represents: SHOW DESIGN PROPERTIES [FOR widgetType]
type ShowDesignPropertiesStmt struct {
	WidgetType string // Empty = all widget types, or specific type like "CONTAINER"
}

func (s *ShowDesignPropertiesStmt) isStatement()     {}
func (s *ShowDesignPropertiesStmt) TypeName() string { return "ShowDesignProperties" }

// DescribeStylingStmt represents: DESCRIBE STYLING ON PAGE/SNIPPET Module.Name [WIDGET widgetName]
type DescribeStylingStmt struct {
	ContainerType string        // "PAGE" or "SNIPPET"
	ContainerName QualifiedName // Page or snippet qualified name
	WidgetName    string        // Empty = all widgets with styling
}

func (s *DescribeStylingStmt) isStatement()     {}
func (s *DescribeStylingStmt) TypeName() string { return "DescribeStyling" }

// AlterStylingStmt represents: ALTER STYLING ON PAGE/SNIPPET Module.Name WIDGET widgetName SET/CLEAR ...
type AlterStylingStmt struct {
	ContainerType    string              // "PAGE" or "SNIPPET"
	ContainerName    QualifiedName       // Page or snippet qualified name
	WidgetName       string              // Widget name to modify
	Assignments      []StylingAssignment // SET assignments
	ClearDesignProps bool                // CLEAR DESIGN PROPERTIES
}

func (s *AlterStylingStmt) isStatement()     {}
func (s *AlterStylingStmt) TypeName() string { return "AlterStyling" }

// StylingAssignment represents a single property assignment in ALTER STYLING SET.
type StylingAssignment struct {
	Property string // "Class", "Style", or design property key (e.g., "Spacing top")
	Value    string // Value string (CSS class, style, or option name)
	IsToggle bool   // true for ON/OFF values
	ToggleOn bool   // true for ON, false for OFF (only meaningful when IsToggle is true)
}
