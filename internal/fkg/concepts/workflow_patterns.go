// internal/fkg/concepts/workflow_patterns.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&WorkflowPatternsAdapter{}) }

type WorkflowPatternsAdapter struct{}

func (a *WorkflowPatternsAdapter) Name() string { return "fkg:workflow-patterns" }
func (a *WorkflowPatternsAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelPattern, LabelImplDetail}}
}
func (a *WorkflowPatternsAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *WorkflowPatternsAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		// ── Patterns ──────────────────────────────────────────────────────────
		patternNode("approval-workflow", "Approval Workflow Pattern",
			"User task with outcomes → decision routing → multi-user task with majority completion → boundary events for escalation"),
		patternNode("boundary-events", "Boundary Events Pattern",
			"Non-interrupting timer for reminders and interrupting timer for auto-rejection on SLA breach"),

		// ── Step nodes ─────────────────────────────────────────────────────────
		stepNode("wf-create-workflow", "Create workflow definition",
			"CREATE OR REPLACE WORKFLOW with context parameter entity",
			1, "create", "Workflow", "HD.WF_TicketEscalation",
			"create or replace workflow HD.WF_TicketEscalation parameter $WorkflowContext: HD.EscalationRequest { ... }"),
		stepNode("wf-add-user-task", "Add user task",
			"User task with page, targeting users microflow, and approval/rejection outcomes",
			2, "configure", "UserTask", "Primary Review",
			"user task UT_PrimaryReview 'Primary Review' page HD.EscalationReview_Form targeting users microflow HD.WFA_GetManagerAssignees outcomes 'Approve' { ... } 'Reject' { ... };"),
		stepNode("wf-add-decision", "Add decision routing",
			"Decision node for conditional routing based on workflow context",
			3, "configure", "Decision", "approval check",
			"decision '$WorkflowContext/Approved = true' outcomes true { ... } false { ... };"),
		stepNode("wf-add-call-workflow", "Add sub-workflow call",
			"CALL WORKFLOW for nested escalation path (sub-workflow)",
			4, "configure", "SubWorkflow", "Manager Review",
			"call workflow HD.WF_SUB_ManagerReview;"),
		stepNode("wf-add-boundary-timer", "Add boundary timer event",
			"Non-interrupting or interrupting timer for SLA-based escalation",
			5, "configure", "BoundaryEvent", "escalation timer",
			"alter workflow HD.WF_TicketEscalation insert boundary event on ... non interrupting timer 'addHours([%CurrentDateTime%], 12)' { call microflow HD.WFS_SendReminder; };"),
		stepNode("wf-call-from-mf", "Call workflow from microflow",
			"Invoke workflow from microflow, passing context entity",
			6, "wire", "Microflow", "TriggerWorkflow",
			"call workflow HD.WF_TicketEscalation (EscalationRequest = $ER);"),

		// ── Edges: pattern → concept ──────────────────────────────────────────
		edge("workflow", "pattern:approval-workflow", HasPattern),
		edge("workflow", "pattern:boundary-events", HasPattern),

		// ── Edges: pattern → step ───────────────────────────────────────────
		edge("pattern:approval-workflow", "step:wf-create-workflow", HasSyntax),
		edge("pattern:approval-workflow", "step:wf-add-user-task", HasSyntax),
		edge("pattern:approval-workflow", "step:wf-add-decision", HasSyntax),
		edge("pattern:approval-workflow", "step:wf-add-call-workflow", HasSyntax),
		edge("pattern:boundary-events", "step:wf-add-boundary-timer", HasSyntax),
		edge("pattern:boundary-events", "step:wf-call-from-mf", HasSyntax),
	})
}
