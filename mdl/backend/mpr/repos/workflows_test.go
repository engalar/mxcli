// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// Workflows domain has 0 fixture units. Get/List against existing
// state are skipped explicitly; CRUD round-trip exercises the write
// path via fresh-construct + InsertUnit.

func TestWorkflowRepo_List_NoFixture(t *testing.T) {
	w := openTestWriter(t)
	repo := NewWorkflowRepository(w)
	got, err := repo.List("")
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List on workflow-less fixture: got %d, want 0", len(got))
	}
}

func TestWorkflowRepo_Get_NotFound(t *testing.T) {
	w := openTestWriter(t)
	repo := NewWorkflowRepository(w)
	if _, err := repo.Get("nonexistent-workflow-id"); err == nil {
		t.Error("Get(nonexistent): want error, got nil")
	}
}

func TestWorkflowRepo_CreateUpdateDeleteCycle(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, _ := r.ListModules()
	var parentID string
	for _, m := range mods {
		if m.Name == "MyFirstModule" {
			parentID = m.ID
			break
		}
	}
	if parentID == "" {
		t.Skip("fixture missing MyFirstModule")
	}

	repo := NewWorkflowRepository(w)
	baseline, err := repo.List("")
	if err != nil {
		t.Fatalf("List baseline: %v", err)
	}

	newWf := newEmptyWorkflow(t, "RepoWorkflowCycleTest")
	if err := repo.Create(parentID, "Documents", newWf); err != nil {
		t.Fatalf("Create: %v", err)
	}
	afterCreate, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Create: %v", err)
	}
	if len(afterCreate) != len(baseline)+1 {
		t.Errorf("after Create: count = %d, want %d", len(afterCreate), len(baseline)+1)
	}

	got, err := repo.Get(model.ID(newWf.ID()))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got.SetName("RepoWorkflowCycleTestUpdated")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, err := repo.Get(model.ID(newWf.ID()))
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got2.Name() != "RepoWorkflowCycleTestUpdated" {
		t.Errorf("post-Update Name = %q, want RepoWorkflowCycleTestUpdated", got2.Name())
	}

	if err := repo.Delete(model.ID(newWf.ID())); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	afterDelete, err := repo.List("")
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	if len(afterDelete) != len(baseline) {
		t.Errorf("after Delete: count = %d, want baseline %d", len(afterDelete), len(baseline))
	}
}

func TestWorkflowRepo_OpenForMutation_CommitOnly(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, _ := r.ListModules()
	var parentID string
	for _, m := range mods {
		if m.Name == "MyFirstModule" {
			parentID = m.ID
			break
		}
	}
	if parentID == "" {
		t.Skip("fixture missing MyFirstModule")
	}

	repo := NewWorkflowRepository(w)
	wf := newEmptyWorkflow(t, "RepoWorkflowMutatorTest")
	if err := repo.Create(parentID, "Documents", wf); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = repo.Delete(model.ID(wf.ID())) }()

	mut, err := repo.OpenForMutation(model.ID(wf.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}

	// Activity-tree edits are deferred to Stage 3 — they must surface
	// explicit errors instead of silently no-oping.
	if err := mut.SetActivityProperty("a-1", "P", 7); err == nil {
		t.Error("SetActivityProperty: want not-implemented error, got nil")
	}
	if err := mut.InsertActivity("a-1", "Items", nil); err == nil {
		t.Error("InsertActivity: want not-implemented error, got nil")
	}
	if err := mut.DeleteActivity("a-1"); err == nil {
		t.Error("DeleteActivity: want not-implemented error, got nil")
	}
	if err := mut.ReplaceActivity("a-1", nil); err == nil {
		t.Error("ReplaceActivity: want not-implemented error, got nil")
	}

	// Commit re-encodes the cached workflow — works on Stage 2.6.
	if err := mut.Commit(); err != nil {
		t.Errorf("Commit: %v", err)
	}
}

func newEmptyWorkflow(t *testing.T, name string) *genWf.Workflow {
	t.Helper()
	wf := genWf.NewWorkflow()
	wf.SetID(element.ID(mmpr.GenerateID()))
	wf.SetTypeName(workflowTypeName)
	wf.SetName(name)
	return wf
}
