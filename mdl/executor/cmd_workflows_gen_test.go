// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3.A1 unit tests for listWorkflowsGen + activity counters.

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

func newListWorkflowsTestCtx(t *testing.T, wfs []*genWf.Workflow) *ExecContext {
	t.Helper()
	repo := &repostesting.RecordingWorkflowRepository{
		ListAllFunc: func() ([]*genWf.Workflow, error) { return wfs, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return "MOD", nil
		},
	}
	var buf bytes.Buffer
	ctx := &ExecContext{
		Workflows: repo,
		Output:    &buf,
		Format:    FormatTable,
	}
	ctx.ensureCache()
	return ctx
}

func TestListWorkflowsGen_OutputsHeaderAndSummary(t *testing.T) {
	wf := genWf.NewWorkflow()
	wf.SetID("ID1")
	wf.SetName("ApproveWF")

	ctx := newListWorkflowsTestCtx(t, []*genWf.Workflow{wf})
	if err := listWorkflowsGen(ctx, ""); err != nil {
		t.Fatalf("listWorkflowsGen: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "Activities") {
		t.Errorf("missing Activities header: %q", out)
	}
	if !strings.Contains(out, "ApproveWF") {
		t.Errorf("missing workflow name: %q", out)
	}
	if !strings.Contains(out, "1 workflows") {
		t.Errorf("missing summary: %q", out)
	}
}

func TestListWorkflowsGen_FilterByModule(t *testing.T) {
	wf := genWf.NewWorkflow()
	wf.SetID("ID1")
	wf.SetName("X")

	// Without a backend hierarchy the module name resolves to "" so the
	// filter "OtherModule" must exclude the row.
	ctx := newListWorkflowsTestCtx(t, []*genWf.Workflow{wf})
	if err := listWorkflowsGen(ctx, "OtherModule"); err != nil {
		t.Fatalf("listWorkflowsGen with filter: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "0 workflows") {
		t.Errorf("filter should exclude all entries; got %q", out)
	}
}

func TestListWorkflowsGen_NilWorkflowSkipped(t *testing.T) {
	ctx := newListWorkflowsTestCtx(t, []*genWf.Workflow{nil, nil})
	if err := listWorkflowsGen(ctx, ""); err != nil {
		t.Fatalf("listWorkflowsGen: %v", err)
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(out, "0 workflows") {
		t.Errorf("nil workflows should be skipped; got %q", out)
	}
}

func TestCountWorkflowActivitiesGen_EmptyFlow(t *testing.T) {
	wf := genWf.NewWorkflow()
	total, ut, dec := countWorkflowActivitiesGen(wf)
	if total != 0 || ut != 0 || dec != 0 {
		t.Errorf("empty flow: got (%d,%d,%d), want all zero", total, ut, dec)
	}
}

func TestCountWorkflowActivitiesGen_NilWorkflowReturnsZero(t *testing.T) {
	total, ut, dec := countWorkflowActivitiesGen(nil)
	if total != 0 || ut != 0 || dec != 0 {
		t.Errorf("nil wf: got (%d,%d,%d), want all zero", total, ut, dec)
	}
}

func TestFormatWorkflowActivitiesGen_JumpTo(t *testing.T) {
	flow := genWf.NewFlow()
	jt := genWf.NewJumpToActivity()
	jt.SetName("J1")
	jt.SetCaption("go back")
	jt.SetTargetActivityQualifiedName("Demo.Approve.Outcome1")
	flow.AddActivities(jt)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "jump to Demo.Approve.Outcome1 comment 'go back';") {
		t.Errorf("missing jump-to line: %q", joined)
	}
}

func TestFormatWorkflowActivitiesGen_WaitForTimerWithDelay(t *testing.T) {
	flow := genWf.NewFlow()
	wt := genWf.NewWaitForTimerActivity()
	wt.SetName("WT1")
	wt.SetCaption("hold")
	wt.SetDelay("dateTime(2026,1,1)")
	flow.AddActivities(wt)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "wait for timer 'dateTime(2026,1,1)' comment 'hold';") {
		t.Errorf("missing wait-for-timer line: %q", joined)
	}
}

func TestFormatWorkflowActivitiesGen_WaitForTimerNoDelay(t *testing.T) {
	flow := genWf.NewFlow()
	wt := genWf.NewWaitForTimerActivity()
	wt.SetName("WT2")
	wt.SetCaption("pause")
	flow.AddActivities(wt)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "wait for timer comment 'pause';") {
		t.Errorf("missing wait-for-timer (no delay) line: %q", joined)
	}
}

func TestFormatWorkflowActivitiesGen_StartEndOmitted(t *testing.T) {
	flow := genWf.NewFlow()
	flow.AddActivities(genWf.NewStartWorkflowActivity())
	flow.AddActivities(genWf.NewEndWorkflowActivity())
	flow.AddActivities(genWf.NewEndOfParallelSplitPathActivity())
	flow.AddActivities(genWf.NewEndOfBoundaryEventPathActivity())

	lines := formatWorkflowActivitiesGen(flow, "  ")
	if len(lines) != 0 {
		t.Errorf("Start/End activities must be omitted; got %v", lines)
	}
}

func TestFormatWorkflowActivitiesGen_FloatingAnnotation(t *testing.T) {
	flow := genWf.NewFlow()
	a := genWf.NewFloatingAnnotation()
	a.SetDescription("note 1")
	flow.AddActivities(a)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "annotation 'note 1';") {
		t.Errorf("missing floating-annotation line: %q", joined)
	}
}

func TestFormatWorkflowActivitiesGen_FloatingAnnotationEmptyOmitted(t *testing.T) {
	flow := genWf.NewFlow()
	flow.AddActivities(genWf.NewFloatingAnnotation())

	lines := formatWorkflowActivitiesGen(flow, "  ")
	if len(lines) != 0 {
		t.Errorf("empty annotation must be omitted; got %v", lines)
	}
}

func TestDescribeWorkflowToStringGen_BasicShape(t *testing.T) {
	wf := genWf.NewWorkflow()
	wf.SetID("WF1")
	wf.SetName("Approve")
	wf.SetDocumentation("Approval workflow")
	wf.SetTitle("Approve order")
	wf.SetExportLevel("public")
	wf.SetOverviewPageQualifiedName("Demo.Overview")
	wf.SetDueDate("[%CurrentDateTime%]")

	flow := genWf.NewFlow()
	wf.SetFlow(flow)
	ut := genWf.NewSingleUserTaskActivity()
	ut.SetName("Step1")
	ut.SetCaption("first")
	flow.AddActivities(ut)

	ctx := newListWorkflowsTestCtx(t, []*genWf.Workflow{wf})
	out, ranges, err := describeWorkflowToStringGen(ctx, ast.QualifiedName{Module: "", Name: "Approve"})
	if err != nil {
		t.Fatalf("describeWorkflowToStringGen: %v", err)
	}
	if ranges != nil {
		t.Errorf("expected nil ELK ranges (legacy parity), got %v", ranges)
	}
	for _, want := range []string{
		"create workflow .Approve",
		"display 'Approve order'",
		"export level public",
		"overview page Demo.Overview",
		"due date '[%CurrentDateTime%]'",
		"begin",
		"user task Step1 'first'",
		"end workflow",
		"/",
		"/**",
		"Approval workflow",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("describe output missing %q in:\n%s", want, out)
		}
	}
}

func TestDescribeWorkflowToStringGen_NotFound(t *testing.T) {
	ctx := newListWorkflowsTestCtx(t, nil)
	_, _, err := describeWorkflowToStringGen(ctx, ast.QualifiedName{Module: "Demo", Name: "Missing"})
	if err == nil {
		t.Error("expected error for missing workflow")
	}
}

func TestFormatUserTaskGen_BasicShape(t *testing.T) {
	flow := genWf.NewFlow()
	ut := genWf.NewSingleUserTaskActivity()
	ut.SetName("Approve")
	ut.SetCaption("approve order")
	ut.SetDueDate("[%CurrentDateTime%]")
	flow.AddActivities(ut)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "user task Approve 'approve order'") {
		t.Errorf("missing header: %q", joined)
	}
	if !strings.Contains(joined, "due date '[%CurrentDateTime%]'") {
		t.Errorf("missing due date: %q", joined)
	}
}

func TestFormatUserTaskGen_MultiKeyword(t *testing.T) {
	flow := genWf.NewFlow()
	ut := genWf.NewMultiUserTaskActivity()
	ut.SetName("Review")
	ut.SetCaption("multi review")
	flow.AddActivities(ut)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "multi user task Review 'multi review'") {
		t.Errorf("multi keyword missing: %q", joined)
	}
}

func TestFormatUserTaskGen_OutcomesEmptyAndPopulated(t *testing.T) {
	flow := genWf.NewFlow()
	ut := genWf.NewSingleUserTaskActivity()
	ut.SetName("Check")
	ut.SetCaption("check thing")
	o1 := genWf.NewUserTaskOutcome()
	o1.SetValue("approved")
	ut.AddOutcomes(o1)
	o2 := genWf.NewUserTaskOutcome()
	o2.SetCaption("rejected")
	subFlow := genWf.NewFlow()
	subFlow.AddActivities(genWf.NewJumpToActivity())
	o2.SetFlow(subFlow)
	ut.AddOutcomes(o2)
	flow.AddActivities(ut)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "outcomes") {
		t.Errorf("missing outcomes header: %q", joined)
	}
	if !strings.Contains(joined, "'approved' { }") {
		t.Errorf("missing empty outcome: %q", joined)
	}
	if !strings.Contains(joined, "'rejected' {") {
		t.Errorf("missing populated outcome opener: %q", joined)
	}
}

func TestFormatCallMicroflowGen_NoMappings(t *testing.T) {
	flow := genWf.NewFlow()
	cm := genWf.NewCallMicroflowActivity()
	cm.SetName("DoIt")
	cm.SetCaption("do thing")
	cm.SetMicroflowQualifiedName("Demo.Action")
	flow.AddActivities(cm)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "call microflow Demo.Action; -- do thing") {
		t.Errorf("expected call microflow line; got: %q", joined)
	}
}

func TestFormatCallMicroflowGen_WithMappings(t *testing.T) {
	flow := genWf.NewFlow()
	cm := genWf.NewCallMicroflowActivity()
	cm.SetName("DoIt")
	cm.SetMicroflowQualifiedName("Demo.Action")
	pm := genWf.NewMicroflowCallParameterMapping()
	pm.SetParameterQualifiedName("Demo.Action.X")
	pm.SetExpression("$WorkflowContext")
	cm.AddParameterMappings(pm)
	flow.AddActivities(cm)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "with (X = '$WorkflowContext')") {
		t.Errorf("missing parameter mapping: %q", joined)
	}
}

func TestFormatExclusiveSplitGen_WithExpression(t *testing.T) {
	flow := genWf.NewFlow()
	es := genWf.NewExclusiveSplitActivity()
	es.SetName("Split")
	es.SetCaption("decide")
	es.SetExpression("$x = 1")
	o := genWf.NewBooleanConditionOutcome()
	o.SetValue(true)
	es.AddOutcomes(o)
	flow.AddActivities(es)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "decision '$x = 1' -- decide") {
		t.Errorf("missing decision line: %q", joined)
	}
	if !strings.Contains(joined, "true -> { }") {
		t.Errorf("missing boolean outcome: %q", joined)
	}
}

func TestFormatParallelSplitGen(t *testing.T) {
	flow := genWf.NewFlow()
	ps := genWf.NewParallelSplitActivity()
	ps.SetName("Fork")
	ps.SetCaption("fork it")
	o1 := genWf.NewParallelSplitOutcome()
	subFlow := genWf.NewFlow()
	subFlow.AddActivities(genWf.NewJumpToActivity())
	o1.SetFlow(subFlow)
	ps.AddOutcomes(o1)
	o2 := genWf.NewParallelSplitOutcome()
	ps.AddOutcomes(o2)
	flow.AddActivities(ps)

	lines := formatWorkflowActivitiesGen(flow, "  ")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "parallel split -- fork it") {
		t.Errorf("missing parallel split header: %q", joined)
	}
	if !strings.Contains(joined, "path 1 {") || !strings.Contains(joined, "path 2 {") {
		t.Errorf("missing path entries: %q", joined)
	}
}

func TestCountWorkflowActivitiesGen_UserTaskAndDecision(t *testing.T) {
	wf := genWf.NewWorkflow()
	flow := genWf.NewFlow()
	wf.SetFlow(flow)

	ut := genWf.NewSingleUserTaskActivity()
	flow.AddActivities(ut)

	dec := genWf.NewExclusiveSplitActivity()
	flow.AddActivities(dec)

	total, userTasks, decisions := countWorkflowActivitiesGen(wf)
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if userTasks != 1 {
		t.Errorf("userTasks = %d, want 1", userTasks)
	}
	if decisions != 1 {
		t.Errorf("decisions = %d, want 1", decisions)
	}
}
