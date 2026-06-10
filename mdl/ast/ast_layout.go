// SPDX-License-Identifier: Apache-2.0

package ast

// LayoutWidgetKind identifies the kind of widget in a layout body.
type LayoutWidgetKind int

const (
	LayoutWidgetScrollContainer LayoutWidgetKind = iota
	LayoutWidgetPlaceholder                      // standalone placeholder (native layouts)
)

// CreateLayoutStmt represents a CREATE [OR MODIFY] LAYOUT statement.
type CreateLayoutStmt struct {
	Name          QualifiedName
	LayoutType    string
	Folder        string
	Documentation string
	Widgets       []*LayoutWidgetV3
	IsModify      bool
	IsReplace     bool
}

func (s *CreateLayoutStmt) isStatement() {}

// LayoutWidgetV3 is a top-level widget in a layout body.
type LayoutWidgetV3 struct {
	Kind            LayoutWidgetKind
	Name            string
	Regions         []*LayoutRegionV3
	PlaceholderName string // for standalone placeholders (native layouts)
}

// LayoutRegionV3 represents a named region inside a scroll container.
type LayoutRegionV3 struct {
	Name         string // "center", "top", "bottom", "left", "right"
	Placeholders []*LayoutPlaceholderV3
}

// LayoutPlaceholderV3 is a named placeholder slot within a layout region.
type LayoutPlaceholderV3 struct {
	Name string
}
