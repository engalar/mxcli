// SPDX-License-Identifier: Apache-2.0

package pages

import (
	"github.com/mendixlabs/mxcli/model"
)

// Widget is the base interface for all page widgets.
type Widget interface {
	GetID() model.ID
	GetTypeName() string
	GetName() string
}

// DesignPropertyValue represents a design property (from Atlas UI theme).
// ValueType determines the BSON serialization type:
//   - "toggle" → Forms$ToggleDesignPropertyValue (Toggle type, no value)
//   - "option" → Forms$OptionDesignPropertyValue (Dropdown type, uses Option field)
//   - "custom" → Forms$CustomDesignPropertyValue (ToggleButtonGroup/ColorPicker, uses Option as Value)
type DesignPropertyValue struct {
	Key       string // Design property key, e.g., "Shadow"
	ValueType string // "toggle", "option", or "custom"
	Option    string // Selected value (for "option"/"custom" types)
}

// BaseWidget provides common fields for all widgets.
type BaseWidget struct {
	model.BaseElement
	Name                   string                          `json:"name"`
	Class                  string                          `json:"class,omitempty"`
	Style                  string                          `json:"style,omitempty"`
	TabIndex               int                             `json:"tabIndex,omitempty"`
	DesignProperties       []DesignPropertyValue           `json:"designProperties,omitempty"`
	ConditionalVisibility  *ConditionalVisibilitySettings  `json:"-"` // Set via VISIBLE IF
	ConditionalEditability *ConditionalEditabilitySettings `json:"-"` // Set via EDITABLE IF
}

// GetName returns the widget's name.
func (w *BaseWidget) GetName() string {
	return w.Name
}

// GetBaseWidget returns a pointer to the BaseWidget for accessing conditional settings.
func (w *BaseWidget) GetBaseWidget() *BaseWidget {
	return w
}

// SetAppearance sets the CSS class and inline style on the widget.
func (w *BaseWidget) SetAppearance(class, style string) {
	w.Class = class
	w.Style = style
}

// SetDesignProperties sets the design properties on the widget.
func (w *BaseWidget) SetDesignProperties(props []DesignPropertyValue) {
	w.DesignProperties = props
}

// Placeholder Widgets

// LayoutPlaceholder represents a placeholder in a layout.
type LayoutPlaceholder struct {
	BaseWidget
}

// ConditionalVisibilitySettings represents visibility conditions.
type ConditionalVisibilitySettings struct {
	model.BaseElement
	Expression     string        `json:"expression,omitempty"`
	ModuleRoles    []model.ID    `json:"moduleRoles,omitempty"`
	SourceVariable *PageVariable `json:"sourceVariable,omitempty"`
	Attribute      model.ID      `json:"attribute,omitempty"`
}

// ConditionalEditabilitySettings represents editability conditions.
type ConditionalEditabilitySettings struct {
	model.BaseElement
	Expression string `json:"expression,omitempty"`
}

// PageVariable represents a page variable reference.
type PageVariable struct {
	model.BaseElement
	UseAllPages bool     `json:"useAllPages"`
	PageID      model.ID `json:"pageId,omitempty"`
	Widget      string   `json:"widget,omitempty"`
}
