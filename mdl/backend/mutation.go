// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

// ContainerKind represents the type of page container (page, layout, or snippet).
type ContainerKind string

const (
	ContainerPage    ContainerKind = "page"
	ContainerLayout  ContainerKind = "layout"
	ContainerSnippet ContainerKind = "snippet"
)

// InsertPosition represents where a widget is inserted relative to a target.
type InsertPosition string

const (
	InsertBefore InsertPosition = "before"
	InsertAfter  InsertPosition = "after"
)

// PluggablePropertyOp represents the operation type for SetPluggableProperty.
type PluggablePropertyOp string

const (
	PluggableOpAttribute        PluggablePropertyOp = "attribute"
	PluggableOpAssociation      PluggablePropertyOp = "association"
	PluggableOpPrimitive        PluggablePropertyOp = "primitive"
	PluggableOpSelection        PluggablePropertyOp = "selection"
	PluggableOpDataSource       PluggablePropertyOp = "datasource"
	PluggableOpWidgets          PluggablePropertyOp = "widgets"
	PluggableOpTextTemplate     PluggablePropertyOp = "texttemplate"
	PluggableOpAction           PluggablePropertyOp = "action"
	PluggableOpAttributeObjects PluggablePropertyOp = "attributeObjects"
)

// WidgetRef identifies a widget or a column within a widget.
type WidgetRef struct {
	Widget string
	Column string // empty for non-column targeting
}

// IsColumn returns true if this targets a column within a widget.
func (r WidgetRef) IsColumn() bool { return r.Column != "" }

// Name returns the full reference string for error messages.
func (r WidgetRef) Name() string {
	if r.IsColumn() {
		return r.Widget + "." + r.Column
	}
	return r.Widget
}

// PageMutator provides fine-grained mutation operations on a single
// page, layout, or snippet unit. Obtain one via PageMutationBackend.OpenPageForMutation.
// All methods operate on the in-memory representation; call Save to persist.
//
// Widget addressing: most methods accept a widgetRef string (widget name).
// Column-aware operations additionally accept a columnRef string.
// DropWidget uses []WidgetRef to support mixed widget/column targets in a
// single call.
type PageMutator interface {
	// ContainerType returns the kind of container (page, layout, or snippet).
	ContainerType() ContainerKind

	// --- Widget property operations ---

	// SetWidgetProperty sets a simple property on the named widget.
	// For pluggable widget properties, prop is the Mendix property key
	// and value is the string representation.
	SetWidgetProperty(widgetRef string, prop string, value any) error

	// SetWidgetDataSource sets the DataSource on the named widget.
	SetWidgetDataSource(widgetRef string, ds pages.DataSource) error

	// SetColumnProperty sets a property on a column within a grid widget.
	SetColumnProperty(gridRef string, columnRef string, prop string, value any) error

	// --- Widget tree operations ---

	// InsertWidget inserts serialized widgets at the given position
	// relative to the target widget or column. Position is "before" or "after".
	// columnRef is "" for widget targeting; non-empty for column targeting.
	InsertWidget(widgetRef string, columnRef string, position InsertPosition, widgets []pages.Widget) error

	// DropWidget removes widgets by ref from the tree.
	DropWidget(refs []WidgetRef) error

	// ReplaceWidget replaces the target widget or column with the given widgets.
	// columnRef is "" for widget targeting.
	ReplaceWidget(widgetRef string, columnRef string, widgets []pages.Widget) error

	// FindWidget checks if a widget with the given name exists in the tree.
	FindWidget(name string) bool

	// --- Variable operations ---

	// AddVariable adds a local variable to the page/snippet.
	AddVariable(name, dataType, defaultValue string) error

	// DropVariable removes a local variable by name.
	DropVariable(name string) error

	// --- Layout operations ---

	// SetLayout changes the layout reference and remaps placeholder parameters.
	SetLayout(newLayout string, paramMappings map[string]string) error

	// --- Pluggable widget operations ---

	// SetPluggableProperty sets a typed property on a pluggable widget's object.
	// propKey is the Mendix property key, op identifies the operation type,
	// and ctx carries the operation-specific values.
	SetPluggableProperty(widgetRef string, propKey string, op PluggablePropertyOp, ctx PluggablePropertyContext) error

	// --- Introspection ---

	// EnclosingEntity returns the qualified entity name for the given widget's
	// data context, or "" if none.
	EnclosingEntity(widgetRef string) string

	// WidgetScope returns a map of widget name → unit ID for all widgets in the tree.
	WidgetScope() map[string]model.ID

	// ParamScope returns page/snippet parameter maps:
	// paramIDs maps param name → entity ID, paramEntityNames maps param name → qualified entity name.
	ParamScope() (paramIDs map[string]model.ID, paramEntityNames map[string]string)

	// Save persists the mutations to the backend.
	Save() error

	// ── Stage 3.3.5.D0 gen-typed additive siblings ──────────────────────────
	// These accept modelsdk element.Element inputs (codec-encoded to BSON) instead
	// of sdk/pages struct types. Legacy methods above are unchanged — D1+ will
	// migrate callers, E1 will retire the legacy surface.

	// SetWidgetDataSourceGen sets the DataSource on the named widget using a
	// gen-typed DataSource element (codec-encoded to BSON).
	SetWidgetDataSourceGen(widgetRef string, ds element.Element) error

	// InsertWidgetGen inserts gen-typed widget elements at the given position.
	InsertWidgetGen(widgetRef string, columnRef string, position InsertPosition, widgets []element.Element) error

	// ReplaceWidgetGen replaces the target widget or column with gen-typed widgets.
	ReplaceWidgetGen(widgetRef string, columnRef string, widgets []element.Element) error
}

// PluggablePropertyContext carries operation-specific values for
// SetPluggableProperty. Only fields relevant to the operation are used.
type PluggablePropertyContext struct {
	AttributePath  string             // "attribute", "association"
	AttributePaths []string           // "attributeObjects"
	AssocPath      string             // "association"
	EntityName     string             // "association"
	PrimitiveVal   string             // "primitive"
	DataSource     pages.DataSource   // "datasource"
	ChildWidgets   []pages.Widget     // "widgets"
	Action         pages.ClientAction // "action"
	TextTemplate   string             // "texttemplate"
	Selection      string             // "selection"
}

// WorkflowMutator provides fine-grained mutation operations on a single
// workflow unit. Obtain one via WorkflowMutationBackend.OpenWorkflowForMutation.
// All methods operate on the in-memory representation; call Save to persist.
//
// Stage 3.3.3.E1 retired the legacy sdk-typed activity inputs; the
// surface is fully gen-typed. Each *Gen-suffixed method accepts
// []element.Element built by buildWorkflowActivitiesGen and serializes
// via SerializeWorkflowActivityGen (codec.Encode + bson.Unmarshal).
type WorkflowMutator interface {
	// --- Top-level property operations ---

	// SetProperty sets a workflow-level property (DisplayName, Description,
	// ExportLevel, DueDate, Parameter, OverviewPage).
	SetProperty(prop string, value string) error

	// SetPropertyWithEntity sets a workflow-level property that references
	// an entity (e.g. Parameter).
	SetPropertyWithEntity(prop string, value string, entity string) error

	// --- Activity operations ---

	// SetActivityProperty sets a property on an activity identified by
	// caption and optional position index.
	SetActivityProperty(activityRef string, atPos int, prop string, value string) error

	// DropActivity removes the referenced activity.
	DropActivity(activityRef string, atPos int) error

	// --- Outcome operations ---

	// DropOutcome removes an outcome by name from the referenced activity.
	DropOutcome(activityRef string, atPos int, outcomeName string) error

	// --- Path operations (parallel split) ---

	// DropPath removes a path by caption from a parallel split activity.
	DropPath(activityRef string, atPos int, pathCaption string) error

	// --- Branch operations (exclusive split) ---

	// DropBranch removes a branch by name from an exclusive split activity.
	DropBranch(activityRef string, atPos int, branchName string) error

	// --- Boundary event operations ---

	// DropBoundaryEvent removes the boundary event from the referenced activity.
	DropBoundaryEvent(activityRef string, atPos int) error

	// --- Gen-typed activity insertion / replacement ---

	InsertAfterActivityGen(activityRef string, atPos int, activities []element.Element) error
	ReplaceActivityGen(activityRef string, atPos int, activities []element.Element) error
	InsertOutcomeGen(activityRef string, atPos int, outcomeName string, activities []element.Element) error
	InsertPathGen(activityRef string, atPos int, pathCaption string, activities []element.Element) error
	InsertBranchGen(activityRef string, atPos int, condition string, activities []element.Element) error
	InsertBoundaryEventGen(activityRef string, atPos int, eventType string, delay string, activities []element.Element) error

	// Save persists the mutations to the backend.
	Save() error
}

// PageMutationBackend provides page/layout/snippet mutation capabilities.
type PageMutationBackend interface {
	// OpenPageForMutation loads a page, layout, or snippet unit and returns
	// a mutator for applying changes. Call Save() on the returned mutator
	// to persist.
	OpenPageForMutation(unitID model.ID) (PageMutator, error)
}

// WorkflowMutationBackend provides workflow mutation capabilities.
type WorkflowMutationBackend interface {
	// OpenWorkflowForMutation loads a workflow unit and returns a mutator
	// for applying changes. Call Save() on the returned mutator to persist.
	OpenWorkflowForMutation(unitID model.ID) (WorkflowMutator, error)
}

// WidgetSerializationBackend provides widget and activity serialization
// for CREATE paths where the executor builds domain objects that need
// to be converted to the storage format.
type WidgetSerializationBackend interface {
	// SerializeWidget converts a domain Widget to its storage representation.
	// The returned value is opaque to the caller; it is only used as input
	// to mutation operations or passed to the backend for persistence.
	SerializeWidget(w pages.Widget) (any, error)

	// SerializeClientAction converts a domain ClientAction to storage format.
	SerializeClientAction(a pages.ClientAction) (any, error)

	// SerializeDataSource converts a domain DataSource to storage format.
	SerializeDataSource(ds pages.DataSource) (any, error)

	// SerializeWorkflowActivityGen converts a gen-typed workflow activity
	// element to storage format. Routes through codec.Encode +
	// bson.Unmarshal so the BSON shape matches the codec round-trip
	// produced by CreateWorkflowGen / UpdateWorkflowGen — divergence
	// here would also break the gen-typed CREATE WORKFLOW round-trip.
	SerializeWorkflowActivityGen(a element.Element) (any, error)
}

// WidgetObjectBuilder provides storage-agnostic operations on a loaded pluggable widget template.
// The executor calls these methods with domain-typed values; the backend handles
// all storage-specific manipulation internally.
//
// Workflow: LoadTemplate → apply operations → EnsureRequiredObjectLists → Finalize
type WidgetObjectBuilder interface {
	// --- Property operations ---
	// Each operation finds the property by key (via TypePointer matching) and updates its value.
	// Set* methods do not return errors — invalid operations are logged as warnings
	// and deferred to Finalize, which returns the aggregate result.

	SetAttribute(propertyKey string, attributePath string)
	SetAssociation(propertyKey string, assocPath string, entityName string)
	SetPrimitive(propertyKey string, value string)
	SetSelection(propertyKey string, value string)
	SetExpression(propertyKey string, value string)
	SetDataSource(propertyKey string, ds pages.DataSource)
	SetChildWidgets(propertyKey string, children []pages.Widget)
	SetTextTemplate(propertyKey string, text string)
	SetTextTemplateWithParams(propertyKey string, text string, entityContext string)
	SetAction(propertyKey string, action pages.ClientAction)
	SetAttributeObjects(propertyKey string, attributePaths []string)

	// --- Template metadata ---

	// PropertyTypeIDs returns the property type metadata for the loaded template.
	PropertyTypeIDs() map[string]pages.PropertyTypeIDEntry

	// --- Object list defaults ---

	// EnsureRequiredObjectLists auto-populates required empty object lists.
	EnsureRequiredObjectLists()

	// --- Gallery-specific ---

	// CloneGallerySelectionProperty clones the itemSelection property with a new Selection value.
	CloneGallerySelectionProperty(propertyKey string, selectionMode string)

	// --- Finalize ---

	// Finalize builds the CustomWidget from the mutated template.
	// Returns the widget with RawType/RawObject populated from internal state.
	Finalize(id model.ID, name string, label string, editable string) *pages.CustomWidget
}

// DataGridColumnSpec carries pre-resolved column data for DataGrid2 construction.
// All attribute paths are fully qualified. Child widgets are already built as
// domain objects; the backend serializes them to storage format internally.
type DataGridColumnSpec struct {
	Attribute    string         // Fully qualified attribute path (empty for action/custom-content columns)
	Caption      string         // Column header caption
	ChildWidgets []pages.Widget // Pre-built child widgets (for custom-content columns)
	FilterWidget pages.Widget   // Pre-built filter widget for the column's filter slot (optional)
	Properties   map[string]any // Column properties (Sortable, Resizable, Visible, etc.)
}

// DataGridSpec carries all inputs needed to build a DataGrid2 widget object.
type DataGridSpec struct {
	DataSource    pages.DataSource
	Columns       []DataGridColumnSpec
	HeaderWidgets []pages.Widget // Pre-built CONTROLBAR widgets for filtersPlaceholder
	// Paging overrides (empty string = use template default)
	PagingOverrides map[string]string // camelCase widget key → string value
	SelectionMode   string            // empty = no override
}

// FilterWidgetSpec carries inputs for building a filter widget.
type FilterWidgetSpec struct {
	WidgetID   string // e.g. pages.WidgetIDDataGridTextFilter
	FilterName string // widget name
}

// WidgetBuilderBackend provides pluggable widget construction capabilities.
type WidgetBuilderBackend interface {
	// LoadWidgetTemplate loads a widget template by ID and returns a builder
	// for applying property operations. projectPath is used for runtime template
	// augmentation from .mpk files.
	LoadWidgetTemplate(widgetID string, projectPath string) (WidgetObjectBuilder, error)

	// SerializeWidgetToOpaque converts a domain Widget to an opaque form
	// suitable for passing to WidgetObjectBuilder.SetChildWidgets.
	// This replaces the direct mpr.SerializeWidget call.
	SerializeWidgetToOpaque(w pages.Widget) any

	// SerializeDataSourceToOpaque converts a domain DataSource to an opaque
	// form suitable for embedding in widget properties.
	SerializeDataSourceToOpaque(ds pages.DataSource) any

	// BuildCreateAttributeObject creates an attribute object for filter widgets.
	// Returns an opaque value to be collected into attribute object lists.
	BuildCreateAttributeObject(attributePath string, objectTypeID, propertyTypeID, valueTypeID string) (any, error)

	// BuildDataGrid2Widget builds a complete DataGrid2 CustomWidget from domain-typed inputs.
	// The backend loads the template, constructs the storage object with columns,
	// datasource, header widgets, paging, and selection, and returns a fully
	// assembled CustomWidget. Returns the widget with an opaque RawType/RawObject.
	BuildDataGrid2Widget(id model.ID, name string, spec DataGridSpec, projectPath string) (*pages.CustomWidget, error)

	// BuildFilterWidget builds a filter widget (text, number, date, or dropdown filter)
	// for use inside DataGrid2 filtersPlaceholder or CONTROLBAR sections.
	BuildFilterWidget(spec FilterWidgetSpec, projectPath string) (pages.Widget, error)
}
