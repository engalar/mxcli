// internal/fkg/concepts/microflow_patterns.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&MicroflowPatternsAdapter{}) }

type MicroflowPatternsAdapter struct{}

func (a *MicroflowPatternsAdapter) Name() string { return "fkg:microflow-patterns" }
func (a *MicroflowPatternsAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelPattern, LabelImplDetail}}
}
func (a *MicroflowPatternsAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *MicroflowPatternsAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		// ── Patterns ──────────────────────────────────────────────────────────
		patternNode("state-machine-sla", "State Machine + SLA Pattern",
			"Entity status lifecycle with SLA deadline computation and validation pre-checks"),
		patternNode("validation-feedback", "Validation Feedback Pattern",
			"Pre-condition validation with feedback to UI and early return"),

		// ── ImplDetail nodes ──────────────────────────────────────────────────
		implDetailNode("validation-feedback", "Validation Feedback", "validation feedback $Ticket/Subject message 'Subject is required'"),
		implDetailNode("show-message", "Show Message", "show message 'Ticket submitted' type success"),
		implDetailNode("addhours", "addHours()", "addHours('[%CurrentDateTime%]', @HD.SLA_HIGH_HOURS) — date arithmetic"),
		implDetailNode("currentdatetime", "[%CurrentDateTime%]", "Current datetime token for microflow expressions"),
		implDetailNode("loop", "Loop", "loop $T in $AllTickets { ... } — iterate over list"),
		implDetailNode("retrieve-filter", "Retrieve with filter", "retrieve $Tickets from HD.Ticket where [Status = 'Open'] sort by Subject asc limit 0"),
		implDetailNode("change-object", "Change Object", "change $Ticket (Status = HD.TicketStatus.Open, SLADueAt = ...)"),
		implDetailNode("commit-object", "Commit Object", "commit $Ticket — persist changes to database"),

		// ── Edges: Pattern → concept ──────────────────────────────────────────
		edge("microflow", "pattern:state-machine-sla", HasPattern),
		edge("microflow", "pattern:validation-feedback", HasPattern),

		// ── Step nodes ─────────────────────────────────────────────────────────
		stepNode("sla-define-entity", "Add status and SLA fields to entity",
			"Add TicketStatus enum attribute and SLADueAt datetime to entity",
			1, "configure", "Entity", "HD.Ticket",
			"Status: HD.TicketStatus default Draft; SLADueAt: datetime"),
		stepNode("sla-create-mf", "Create state transition microflow",
			"CREATE OR MODIFY MICROFLOW with parameter, validation, and state change",
			2, "create", "Microflow", "HD.ACT_Ticket_Submit",
			"create or modify microflow HD.ACT_Ticket_Submit ($Ticket: HD.Ticket) returns boolean { ... }"),
		stepNode("sla-add-validation", "Add pre-condition validation",
			"Validation feedback before state transition, early return on failure",
			3, "configure", "Validation", "Subject check",
			"if $Ticket/Subject = '' { validation feedback $Ticket/Subject message 'Required'; return false; }"),
		stepNode("sla-compute-deadline", "Compute SLA deadline",
			"Calculate SLADueAt using addHours with constant reference",
			4, "configure", "SLA", "deadline",
			"change $Ticket (Status = HD.TicketStatus.Open, SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_HIGH_HOURS));"),
		stepNode("sla-commit", "Commit and return",
			"Commit changes to database and return success",
			5, "configure", "Commit", "$Ticket",
			"commit $Ticket; return true;"),

		stepNode("validation-check-preconditions", "Check pre-conditions",
			"Validate required fields and business rules before proceeding",
			1, "configure", "Validation", "preconditions",
			"if $Ticket/Subject = '' or $Ticket/Description = '' { ... }"),
		stepNode("validation-return-feedback", "Return feedback to UI",
			"Show validation message and early return false",
			2, "configure", "Feedback", "UI message",
			"validation feedback $Ticket/Subject message 'Subject is required'; return false;"),

		// ── Nanoflow pattern (P2) ──────────────────────────────────────────────
		patternNode("nanoflow-quick-create", "Nanoflow Quick-Create Pattern",
			"Client-side object creation: create → commit → return, no server round-trip"),

		stepNode("nf-create", "Create Nanoflow definition",
			"CREATE OR MODIFY NANOFLOW with parameters and return type",
			1, "create", "Nanoflow", "HD.NF_Ticket_QuickCreate",
			"create or modify nanoflow HD.NF_Ticket_QuickCreate ($Customer: HD.Customer, $Subject: string) returns HD.Ticket as $Ticket { ... }"),
		stepNode("nf-create-obj", "Create object client-side",
			"Use create + commit for client-side object creation without microflow",
			2, "configure", "Object", "HD.Ticket",
			"$Ticket = create HD.Ticket (Subject = $Subject, Status = HD.TicketStatus.Draft); commit $Ticket;"),
		stepNode("nf-return", "Return result",
			"Return the created object to the nanoflow caller (page or microflow)",
			3, "configure", "Return", "$Ticket",
			"return $Ticket;"),
		stepNode("nf-search", "Client-side search",
			"Retrieve with contains() filter and client-side sort/limit",
			4, "configure", "Search", "Tickets",
			"retrieve $Tickets from HD.Ticket where [contains(Subject, $Search/SubjectKeyword)] sort by Subject asc limit 100;"),

		edge("nanoflow", "pattern:nanoflow-quick-create", HasPattern),
		edge("pattern:nanoflow-quick-create", "step:nf-create", HasSyntax),
		edge("pattern:nanoflow-quick-create", "step:nf-create-obj", HasSyntax),
		edge("pattern:nanoflow-quick-create", "step:nf-return", HasSyntax),
		edge("pattern:nanoflow-quick-create", "step:nf-search", HasSyntax),

		// ── Edges: pattern → step ───────────────────────────────────────────
		edge("pattern:state-machine-sla", "step:sla-define-entity", HasSyntax),
		edge("pattern:state-machine-sla", "step:sla-create-mf", HasSyntax),
		edge("pattern:state-machine-sla", "step:sla-add-validation", HasSyntax),
		edge("pattern:state-machine-sla", "step:sla-compute-deadline", HasSyntax),
		edge("pattern:state-machine-sla", "step:sla-commit", HasSyntax),
		edge("pattern:validation-feedback", "step:validation-check-preconditions", HasSyntax),
		edge("pattern:validation-feedback", "step:validation-return-feedback", HasSyntax),
	})
}
