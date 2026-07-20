// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3.A0/A1 unit tests for the workflows cache helper.
// Uses RecordingWorkflowRepository (mdl/repos/testing) rather than a real
// MPR fixture because no fixture in this repo carries Workflow units.

package executor

import (
	"errors"
	"testing"

	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

func newWorkflowsCacheTestContext(t *testing.T, wfs []*genWf.Workflow, containerID string) *ExecContext {
	t.Helper()
	repo := &repostesting.RecordingWorkflowRepository{
		ListAllFunc: func() ([]*genWf.Workflow, error) { return wfs, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return model.ID(containerID), nil
		},
	}
	ctx := &ExecContext{
		Workflows: repo,
	}
	ctx.ensureCache()
	return ctx
}

func TestListWorkflowsWithContainerGen_NilCtxReturnsNil(t *testing.T) {
	got, err := listWorkflowsWithContainerGen(nil)
	if err != nil {
		t.Fatalf("listWorkflowsWithContainerGen(nil): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil result for nil ctx, got %v", got)
	}
}

func TestListWorkflowsWithContainerGen_NoRepoReturnsNil(t *testing.T) {
	ctx := &ExecContext{}
	ctx.ensureCache()
	got, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listWorkflowsWithContainerGen(): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 entries with no repo, got %d", len(got))
	}
}

func TestListWorkflowsWithContainerGen_CachesAcrossCalls(t *testing.T) {
	wf1 := genWf.NewWorkflow()
	wf1.SetID("ID1")
	wf1.SetName("WF1")

	ctx := newWorkflowsCacheTestContext(t, []*genWf.Workflow{wf1}, "MOD1")

	first, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(first))
	}
	if first[0].Elem.Name() != "WF1" {
		t.Errorf("Name() = %q, want WF1", first[0].Elem.Name())
	}
	if first[0].ContainerID != "MOD1" {
		t.Errorf("ContainerID = %q, want MOD1", first[0].ContainerID)
	}

	second, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(first) != len(second) || &first[0] != &second[0] {
		t.Errorf("cache must return identical slice on repeat call")
	}
}

func TestInvalidateWorkflowsCache_ClearsCache(t *testing.T) {
	wf := genWf.NewWorkflow()
	wf.SetID("X")
	wf.SetName("X")
	ctx := newWorkflowsCacheTestContext(t, []*genWf.Workflow{wf}, "MOD")

	if _, err := listWorkflowsWithContainerGen(ctx); err != nil {
		t.Fatalf("warm-up: %v", err)
	}
	if ctx.Cache.workflowsWithContainerGen == nil {
		t.Fatal("warm-up did not populate cache")
	}
	invalidateWorkflowsCache(ctx)
	if ctx.Cache.workflowsWithContainerGen != nil {
		t.Errorf("invalidate should clear cache")
	}
}

func TestListWorkflowsWithContainerGen_PropagatesListError(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &repostesting.RecordingWorkflowRepository{
		ListAllFunc: func() ([]*genWf.Workflow, error) { return nil, wantErr },
	}
	ctx := &ExecContext{
		Workflows: repo,
	}
	ctx.ensureCache()
	_, err := listWorkflowsWithContainerGen(ctx)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected error to wrap %v, got %v", wantErr, err)
	}
}
