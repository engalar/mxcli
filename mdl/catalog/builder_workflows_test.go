// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3.C3 — fixtures migrated to gen-typed Workflow.

package catalog

import (
	"testing"

	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

func TestCountWorkflowActivityTypesGen(t *testing.T) {
	// Build helpers — gen types use NewX() factories.
	emptyWf := func() *genWf.Workflow {
		wf := genWf.NewWorkflow()
		return wf
	}
	wfWithFlow := func(activities ...func() interface{ TypeName() string }) *genWf.Workflow {
		wf := genWf.NewWorkflow()
		flow := genWf.NewFlow()
		for _, mk := range activities {
			a := mk()
			// AddActivities expects element.Element; per-mk closures
			// return a typed *XxxActivity — convert via type-switch
			// done implicitly because element.Element is an interface
			// satisfied by every gen type.
			switch v := a.(type) {
			case *genWf.UserTask:
				flow.AddActivities(v)
			case *genWf.SingleUserTaskActivity:
				flow.AddActivities(v)
			case *genWf.MultiUserTaskActivity:
				flow.AddActivities(v)
			case *genWf.CallMicroflowTask:
				flow.AddActivities(v)
			case *genWf.CallMicroflowActivity:
				flow.AddActivities(v)
			case *genWf.ExclusiveSplitActivity:
				flow.AddActivities(v)
			case *genWf.ParallelSplitActivity:
				flow.AddActivities(v)
			}
		}
		wf.SetFlow(flow)
		return wf
	}

	tests := []struct {
		name                               string
		wf                                 *genWf.Workflow
		wantTotal, wantUT, wantMF, wantDec int
	}{
		{name: "nil flow", wf: emptyWf(), wantTotal: 0},
		{name: "empty flow", wf: wfWithFlow(), wantTotal: 0},
		{
			name: "user task (SingleUserTaskActivity)",
			wf: wfWithFlow(func() interface{ TypeName() string } {
				return genWf.NewSingleUserTaskActivity()
			}),
			wantTotal: 1, wantUT: 1,
		},
		{
			name: "user task (legacy UserTask storage)",
			wf: wfWithFlow(func() interface{ TypeName() string } {
				return genWf.NewUserTask()
			}),
			wantTotal: 1, wantUT: 1,
		},
		{
			name: "multi user task",
			wf: wfWithFlow(func() interface{ TypeName() string } {
				return genWf.NewMultiUserTaskActivity()
			}),
			wantTotal: 1, wantUT: 1,
		},
		{
			name: "call microflow task (legacy storage)",
			wf: wfWithFlow(func() interface{ TypeName() string } {
				return genWf.NewCallMicroflowTask()
			}),
			wantTotal: 1, wantMF: 1,
		},
		{
			name: "call microflow activity (gen storage)",
			wf: wfWithFlow(func() interface{ TypeName() string } {
				return genWf.NewCallMicroflowActivity()
			}),
			wantTotal: 1, wantMF: 1,
		},
		{
			name: "exclusive split counts as decision",
			wf: wfWithFlow(func() interface{ TypeName() string } {
				return genWf.NewExclusiveSplitActivity()
			}),
			wantTotal: 1, wantDec: 1,
		},
		{
			name: "nested activities in user task outcomes",
			wf: func() *genWf.Workflow {
				wf := genWf.NewWorkflow()
				flow := genWf.NewFlow()
				ut := genWf.NewSingleUserTaskActivity()
				oc := genWf.NewUserTaskOutcome()
				subFlow := genWf.NewFlow()
				subFlow.AddActivities(genWf.NewCallMicroflowTask())
				oc.SetFlow(subFlow)
				ut.AddOutcomes(oc)
				flow.AddActivities(ut)
				wf.SetFlow(flow)
				return wf
			}(),
			wantTotal: 2, wantUT: 1, wantMF: 1,
		},
		{
			name: "parallel split recurses into outcomes",
			wf: func() *genWf.Workflow {
				wf := genWf.NewWorkflow()
				flow := genWf.NewFlow()
				ps := genWf.NewParallelSplitActivity()
				oc1 := genWf.NewParallelSplitOutcome()
				sub1 := genWf.NewFlow()
				sub1.AddActivities(genWf.NewSingleUserTaskActivity())
				oc1.SetFlow(sub1)
				ps.AddOutcomes(oc1)
				oc2 := genWf.NewParallelSplitOutcome()
				sub2 := genWf.NewFlow()
				sub2.AddActivities(genWf.NewExclusiveSplitActivity())
				oc2.SetFlow(sub2)
				ps.AddOutcomes(oc2)
				flow.AddActivities(ps)
				wf.SetFlow(flow)
				return wf
			}(),
			wantTotal: 3, wantUT: 1, wantDec: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, ut, mf, dec := countWorkflowActivityTypesGen(tt.wf)
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			if ut != tt.wantUT {
				t.Errorf("userTasks = %d, want %d", ut, tt.wantUT)
			}
			if mf != tt.wantMF {
				t.Errorf("microflowCalls = %d, want %d", mf, tt.wantMF)
			}
			if dec != tt.wantDec {
				t.Errorf("decisions = %d, want %d", dec, tt.wantDec)
			}
		})
	}
}
