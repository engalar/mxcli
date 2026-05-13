// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

func TestRecordingWorkflowRepository_RecordsCreate(t *testing.T) {
	rec := &RecordingWorkflowRepository{}
	wf := genWf.NewWorkflow()
	if err := rec.Create("parent-uuid", "Documents", wf); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(rec.Created) != 1 || rec.Created[0].Workflow != wf {
		t.Errorf("Created = %+v", rec.Created)
	}
}

func TestRecordingWorkflowRepository_OpenForMutation_DefaultMutator(t *testing.T) {
	rec := &RecordingWorkflowRepository{}
	mut, err := rec.OpenForMutation("wf-1")
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	if _, ok := mut.(*RecordingWorkflowMutator); !ok {
		t.Errorf("default mutator type = %T, want *RecordingWorkflowMutator", mut)
	}
	if len(rec.OpenedIDs) != 1 || rec.OpenedIDs[0] != model.ID("wf-1") {
		t.Errorf("OpenedIDs = %v", rec.OpenedIDs)
	}
}

func TestRecordingWorkflowRepository_OpenForMutation_InjectedMutator(t *testing.T) {
	injected := &RecordingWorkflowMutator{}
	rec := &RecordingWorkflowRepository{
		OpenForMutationFunc: func(model.ID) (repos.WorkflowMutator, error) {
			return injected, nil
		},
	}
	mut, err := rec.OpenForMutation("wf-2")
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	if mut != injected {
		t.Errorf("OpenForMutation returned %T, want injected pointer", mut)
	}
}

func TestRecordingWorkflowMutator_AllMethodsRecord(t *testing.T) {
	m := &RecordingWorkflowMutator{}
	_ = m.SetActivityProperty("a", "P", 7)
	_ = m.InsertActivity("p", "slot", nil)
	_ = m.DeleteActivity("d")
	_ = m.ReplaceActivity("r", nil)
	_ = m.Commit()
	_ = m.Commit()

	if len(m.SetActivityPropertyCalls) != 1 ||
		len(m.InsertActivityCalls) != 1 ||
		len(m.DeleteActivityCalls) != 1 ||
		len(m.ReplaceActivityCalls) != 1 ||
		m.CommitCalls != 2 {
		t.Errorf("counts: setProp=%d insert=%d delete=%d replace=%d commit=%d",
			len(m.SetActivityPropertyCalls), len(m.InsertActivityCalls),
			len(m.DeleteActivityCalls), len(m.ReplaceActivityCalls), m.CommitCalls)
	}
}

func TestRecordingWorkflowRepository_FuncOverride(t *testing.T) {
	want := errors.New("inject")
	rec := &RecordingWorkflowRepository{
		CreateFunc: func(WorkflowCreateCall) error { return want },
	}
	err := rec.Create("p", "c", genWf.NewWorkflow())
	if !errors.Is(err, want) {
		t.Errorf("Create err = %v, want %v", err, want)
	}
}
