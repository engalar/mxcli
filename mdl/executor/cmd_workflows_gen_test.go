// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3.A1 unit tests for listWorkflowsGen + activity counters.

package executor

import (
	"bytes"
	"strings"
	"testing"

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
