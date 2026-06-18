// internal/fkg/concepts/page_patterns.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&PagePatternsAdapter{}) }

type PagePatternsAdapter struct{}

func (a *PagePatternsAdapter) Name() string { return "fkg:page-patterns" }
func (a *PagePatternsAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelPattern, LabelImplDetail}}
}
func (a *PagePatternsAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *PagePatternsAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		// ── Patterns ──────────────────────────────────────────────────────────
		patternNode("overview-page", "Overview Page Pattern",
			"DataGrid CRUD overview: DataGrid + controlbar + actionbutton for create and view"),
		patternNode("master-detail", "Master-Detail Pattern",
			"Split page: DataView header with related DataGrid or ListView below"),
		patternNode("popup-form", "Popup Form Pattern",
			"Microflow creates object → show page with params → save_changes close_page"),

		// ── ImplDetail nodes ──────────────────────────────────────────────────
		implDetailNode("actionbutton", "ActionButton", "Page button with action: show_page, microflow, create_object, save_changes"),
		implDetailNode("dynamictext", "DynamicText", "Text display with contentparams template rendering"),
		implDetailNode("textbox", "TextBox", "Single-line text input bound to string attribute"),
		implDetailNode("textarea", "TextArea", "Multi-line text input bound to string attribute"),
		implDetailNode("combobox", "ComboBox", "Dropdown selector bound to enumeration or association"),
		implDetailNode("layoutgrid", "LayoutGrid", "Responsive grid: layoutgrid → row → column with desktopwidth"),
		implDetailNode("textfilter", "TextFilter", "DataGrid column text search filter"),
		implDetailNode("dropdownfilter", "DropDownFilter", "DataGrid column dropdown filter"),
		implDetailNode("controlbar", "ControlBar", "DataGrid top toolbar with action buttons"),
		implDetailNode("footer", "Footer", "Form bottom action area for save/cancel buttons"),
		implDetailNode("page-params", "Page Params", "params: { $Ticket: HD.Ticket } — typed parameter passing to pages"),
		implDetailNode("contentparams", "Content Params", "contentparams: [{1}=Subject] — template substitution in dynamic text"),
		implDetailNode("rendermode", "RenderMode", "H2, H3, H4 heading levels for dynamictext"),
		implDetailNode("buttonstyle", "ButtonStyle", "primary, secondary, success, warning, danger, link UI variants"),

		// ── Edges: Pattern → concept relation ─────────────────────────────────
		edge("page", "pattern:overview-page", HasPattern),
		edge("page", "pattern:master-detail", HasPattern),
		edge("page", "pattern:popup-form", HasPattern),

		// ── Step nodes (consumed by Guide() for ordered implementation steps) ──
		stepNode("popup-create-mf", "Create supporting microflow",
			"Create microflow to pre-populate object and open page with params",
			1, "create", "Microflow", "NF_CreateTicket",
			"create or modify microflow HD.NF_CreateTicket returns HD.Ticket as $Ticket { ... }"),
		stepNode("popup-create-page", "Create popup page",
			"Create page with DataView for the new object, params match microflow return",
			2, "create", "Page", "HD.Ticket_New",
			"create or replace page HD.Ticket_New (title:'New Ticket', layout: Atlas_Core.PopupLayout, params:{$Ticket: HD.Ticket}) { ... }"),
		stepNode("popup-wire-action", "Wire create action",
			"Add actionbutton with microflow → show_page chain",
			3, "wire", "ActionButton", "btnCreate",
			"actionbutton btnCreate (caption:'New', action: create_object HD.Ticket then show_page HD.Ticket_New)"),

		stepNode("overview-create-page", "Create overview page",
			"CREATE OR REPLACE PAGE with DataGrid datasource",
			1, "create", "Page", "HD.Ticket_Overview",
			"create or replace page HD.Ticket_Overview (title:'Tickets', layout:Atlas_Core.Atlas_Default) { datagrid ... }"),
		stepNode("overview-configure-columns", "Configure columns and filters",
			"Add columns with textfilter/dropdownfilter, configure PageSize and PagingPosition",
			2, "configure", "DataGrid", "dgTickets",
			"column colSubject (attribute:Subject, caption:'Subject') { textfilter fSubject }"),

		// ── Edges: pattern → step ───────────────────────────────────────────
		edge("pattern:popup-form", "step:popup-create-mf", HasSyntax),
		edge("pattern:popup-form", "step:popup-create-page", HasSyntax),
		edge("pattern:popup-form", "step:popup-wire-action", HasSyntax),
		edge("pattern:overview-page", "step:overview-create-page", HasSyntax),
		edge("pattern:overview-page", "step:overview-configure-columns", HasSyntax),
	})
}
