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
