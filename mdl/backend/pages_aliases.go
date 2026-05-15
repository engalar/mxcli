// SPDX-License-Identifier: Apache-2.0

package backend

import "github.com/mendixlabs/mxcli/sdk/pages"

// Stage 3.3.5 migration aliases for sdk/pages symbols used by executor.
// This keeps sdk/pages imports centralized at backend boundaries while
// executor files progressively move to gen models.
type (
	Widget                         = pages.Widget
	Page                           = pages.Page
	Layout                         = pages.Layout
	Snippet                        = pages.Snippet
	BuildingBlock                  = pages.BuildingBlock
	PageTemplate                   = pages.PageTemplate
	LayoutCall                     = pages.LayoutCall
	LayoutCallArgument             = pages.LayoutCallArgument
	PageParameter                  = pages.PageParameter
	SnippetParameter               = pages.SnippetParameter
	LocalVariable                  = pages.LocalVariable
	BaseWidget                     = pages.BaseWidget
	Container                      = pages.Container
	LayoutGrid                     = pages.LayoutGrid
	LayoutGridRow                  = pages.LayoutGridRow
	LayoutGridColumn               = pages.LayoutGridColumn
	TabContainer                   = pages.TabContainer
	TabPage                        = pages.TabPage
	GroupBox                       = pages.GroupBox
	DataView                       = pages.DataView
	DataGridColumn                 = pages.DataGridColumn
	ListView                       = pages.ListView
	TextBox                        = pages.TextBox
	TextArea                       = pages.TextArea
	DatePicker                     = pages.DatePicker
	DropDown                       = pages.DropDown
	CheckBox                       = pages.CheckBox
	Text                           = pages.Text
	DynamicText                    = pages.DynamicText
	Title                          = pages.Title
	ActionButton                   = pages.ActionButton
	RadioButtons                   = pages.RadioButtons
	NavigationList                 = pages.NavigationList
	NavigationListItem             = pages.NavigationListItem
	SnippetCallWidget              = pages.SnippetCallWidget
	SnippetParamMapping            = pages.SnippetParamMapping
	StaticImage                    = pages.StaticImage
	DynamicImage                   = pages.DynamicImage
	CustomWidget                   = pages.CustomWidget
	DataSource                     = pages.DataSource
	PropertyTypeIDEntry            = pages.PropertyTypeIDEntry
	DataViewSource                 = pages.DataViewSource
	DatabaseSource                 = pages.DatabaseSource
	MicroflowSource                = pages.MicroflowSource
	NanoflowSource                 = pages.NanoflowSource
	AssociationSource              = pages.AssociationSource
	ListenToWidgetSource           = pages.ListenToWidgetSource
	GridSort                       = pages.GridSort
	ClientAction                   = pages.ClientAction
	SaveChangesClientAction        = pages.SaveChangesClientAction
	CancelChangesClientAction      = pages.CancelChangesClientAction
	ClosePageClientAction          = pages.ClosePageClientAction
	DeleteClientAction             = pages.DeleteClientAction
	CreateObjectClientAction       = pages.CreateObjectClientAction
	PageClientAction               = pages.PageClientAction
	PageClientParameterMapping     = pages.PageClientParameterMapping
	MicroflowClientAction          = pages.MicroflowClientAction
	MicroflowParameterMapping      = pages.MicroflowParameterMapping
	NanoflowClientAction           = pages.NanoflowClientAction
	NanoflowParameterMapping       = pages.NanoflowParameterMapping
	LinkClientAction               = pages.LinkClientAction
	SignOutClientAction            = pages.SignOutClientAction
	SetTaskOutcomeClientAction     = pages.SetTaskOutcomeClientAction
	ClientTemplate                 = pages.ClientTemplate
	ClientTemplateParameter        = pages.ClientTemplateParameter
	ConditionalVisibilitySettings  = pages.ConditionalVisibilitySettings
	ConditionalEditabilitySettings = pages.ConditionalEditabilitySettings
	DesignPropertyValue            = pages.DesignPropertyValue
	ButtonStyle                    = pages.ButtonStyle
	ContainerRenderMode            = pages.ContainerRenderMode
	TextRenderMode                 = pages.TextRenderMode
)

const (
	ButtonStyleDefault      = pages.ButtonStyleDefault
	LinkTypeWeb             = pages.LinkTypeWeb
	SortDirectionAscending  = pages.SortDirectionAscending
	SortDirectionDescending = pages.SortDirectionDescending
	TextRenderModeText      = pages.TextRenderModeText
)
