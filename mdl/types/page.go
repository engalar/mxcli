// SPDX-License-Identifier: Apache-2.0

package types

// PageModel is the MDL-level intermediate representation of a page, snippet,
// or layout. Both the write path (AST→PageModel→BSON) and the read path
// (BSON→PageModel→MDL) share this type, keeping the two paths in sync.
type PageModel struct {
	ModuleName string
	Name       string
	Title      string
	Layout     string // e.g. "Atlas_Core.Atlas_Default"
	Folder     string // e.g. "Ticket/Search"
	Params     []PageParam
	Variables  []PageVariable
	Widgets    []*WidgetNode
}

// PageParam represents a page parameter ($Foo: Module.Entity).
type PageParam struct {
	Name       string
	EntityName string
}

// PageVariable represents a page-level variable (non-persistent object).
type PageVariable struct {
	Name       string
	EntityName string
	IsList     bool
}

// WidgetKind identifies the MDL-level widget keyword.
type WidgetKind string

const (
	WidgetContainer    WidgetKind = "container"
	WidgetScrollView   WidgetKind = "scrollview"
	WidgetGroupBox     WidgetKind = "groupbox"
	WidgetLayoutGrid   WidgetKind = "layoutgrid"
	WidgetLayoutRow    WidgetKind = "row"
	WidgetLayoutCol    WidgetKind = "column"
	WidgetTabContainer WidgetKind = "tabcontainer"
	WidgetTabPage      WidgetKind = "tabpage"
	WidgetDataView     WidgetKind = "dataview"
	WidgetListView     WidgetKind = "listview"
	WidgetGallery      WidgetKind = "gallery"
	WidgetButton       WidgetKind = "button"
	WidgetTextBox      WidgetKind = "textbox"
	WidgetTextArea     WidgetKind = "textarea"
	WidgetDatePicker   WidgetKind = "datepicker"
	WidgetRadioButtons WidgetKind = "radiobuttons"
	WidgetCheckBox     WidgetKind = "checkbox"
	WidgetLabel        WidgetKind = "label"
	WidgetText         WidgetKind = "text"
	WidgetDynamicText  WidgetKind = "dynamictext"
	WidgetTitle        WidgetKind = "title"
	WidgetNavList      WidgetKind = "navigationlist"
	WidgetSnippet      WidgetKind = "snippet"
	WidgetDataGrid     WidgetKind = "datagrid"     // CustomWidget type=datagrid2
	WidgetComboBox     WidgetKind = "combobox"     // CustomWidget type=combobox
	WidgetImage        WidgetKind = "image"        // CustomWidget type=image
	WidgetUnknown      WidgetKind = "unknown"      // unrecognised pluggable widget
	WidgetPlaceholder  WidgetKind = "placeholder"  // Forms$Placeholder — layout only
)

// WidgetNode is a single node in the page widget tree.
type WidgetNode struct {
	Kind     WidgetKind
	Name     string
	Children []*WidgetNode

	// Footer holds footer widgets (DataView only). Separate from Children
	// because Mendix stores them in FooterWidgets in BSON, not Widgets.
	Footer []*WidgetNode

	// Data binding
	DataSource *DataSourceDef
	EntityAttr string // attribute path (textbox, datepicker, …)
	EntityCtx  string // entity type provided to children by a dataview

	// Display
	Caption string // button/label caption
	Content string // static text content

	// Layout (column)
	ColWidth ColWidthDef

	// Actions
	OnClick     string // qualified microflow/nanoflow/page name
	ButtonStyle string // Primary | Success | Warning | Danger | Default | Link | Icon

	// Input widget properties
	Editable   string // Always | Never | Conditional
	EditableIf string // expression when Editable==Conditional
	ShowLabel  bool
	LabelPos   string // Left | Top
	ReadOnly   string // Inherit | Control | Text

	// Conditional visibility
	VisibleIf string

	// Appearance
	Class       string
	Style       string
	DesignProps []DesignProp

	// Kind-specific sub-structs (nil when not applicable)
	GroupBox *GroupBoxProps
	DataGrid *DataGridProps
	Gallery  *GalleryProps
	Image    *ImageProps
	Snippet  *SnippetProps
	Unknown  *UnknownProps
}

// ColWidthDef holds responsive column widths (1-12; 0 = not set).
type ColWidthDef struct {
	Desktop, Tablet, Phone int
}

// DataSourceDef describes a widget's data source.
type DataSourceDef struct {
	Kind            DataSourceKind
	Reference       string // qualified name for mf/nf/param
	Entity          string // entity for database sources
	XPathConstraint string
	SortColumns     []SortDef
}

// DataSourceKind enumerates supported data source types.
type DataSourceKind string

const (
	DataSourceDatabase  DataSourceKind = "database"
	DataSourceMicroflow DataSourceKind = "microflow"
	DataSourceNanoflow  DataSourceKind = "nanoflow"
	DataSourceParameter DataSourceKind = "parameter"
	DataSourceSelection DataSourceKind = "selection"
)

// SortDef represents a single sort column.
type SortDef struct {
	Attribute string
	Order     string // ASC | DESC
}

// DesignProp represents a Mendix design property.
type DesignProp struct {
	Key, Option string
	ValueType   string // toggle | option
}

// GroupBoxProps holds groupbox-specific properties.
type GroupBoxProps struct {
	Collapsible string // No | YesInitiallyExpanded | YesInitiallyCollapsed
	HeaderMode  string // Div | H1 … H6
}

// DataGridProps holds datagrid-specific properties.
type DataGridProps struct {
	Columns       []ColumnDef
	FilterWidgets []*WidgetNode
	ControlBar    []*WidgetNode
	PageSize      int
	Pagination    string // buttons | virtualScrolling | loadMore
	PagingPos     string // bottom | top | both
}

// ColumnDef describes a single DataGrid column.
type ColumnDef struct {
	Name, Attribute, Caption                 string
	ShowContentAs                            string // attribute | customContent | dynamicText
	ContentWidgets                           []*WidgetNode
	DynamicText                              string
	Alignment                                string // left | center | right
	WrapText, Sortable, Resizable, Draggable bool
	Hidable                                  string // yes | hidden | no
	ColumnWidth                              string // autoFill | autoFit | manual
	Size, Visible, CellClass, Tooltip        string
}

// GalleryProps holds gallery-specific properties.
type GalleryProps struct {
	DesktopColumns, TabletColumns, PhoneColumns int
	Selection                                   string // Single | Multi | None
	FilterWidgets                               []*WidgetNode
	ContentWidgets                              []*WidgetNode
}

// ImageProps holds pluggable-image-specific properties.
type ImageProps struct {
	URL, AltText  string
	Width, Height string
	WidthUnit     string // auto | pixels | percentage
	HeightUnit    string // auto | pixels | percentage | viewport
	DisplayAs     string // fullImage | thumbnail
	Responsive    bool
	ImageType     string // image | imageUrl | icon
	OnClickType   string // action | enlarge
}

// SnippetProps holds snippet-call-specific properties.
type SnippetProps struct {
	SnippetName string // qualified name
}

// UnknownProps holds data for unrecognised pluggable widgets.
type UnknownProps struct {
	WidgetID      string // e.g. com.mendix.widget.custom.switch.Switch
	ExplicitProps []ExplicitProp
}

// ExplicitProp is a single non-default property from an unknown widget.
type ExplicitProp struct {
	Key, Value string
	IsRef      bool
}
