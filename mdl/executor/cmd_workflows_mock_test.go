// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// makeWorkflowsRepo wires a RecordingWorkflowRepository whose ListAll
// returns wfs and whose GetContainerUUID always returns containerID.
// Stage 3.3.3.E0 — gen-typed replacement for the legacy
// MockBackend.ListWorkflowsFunc fixture (codec-decoded gen Workflow
// drops ContainerID, so the repo carries the linkage explicitly).
func makeWorkflowsRepo(wfs []*genWf.Workflow, containerID model.ID) *repostesting.RecordingWorkflowRepository {
	return &repostesting.RecordingWorkflowRepository{
		ListAllFunc:          func() ([]*genWf.Workflow, error) { return wfs, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return containerID, nil },
	}
}

func TestShowWorkflows_Mock(t *testing.T) {
	mod := mkModule("Sales")
	wf := mkWorkflowGen(string(nextID("wf")), "ApproveOrder")

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Workflows = makeWorkflowsRepo([]*genWf.Workflow{wf}, mod.ID)
	assertNoError(t, listWorkflowsGen(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "Qualified Name")
	assertContainsStr(t, out, "Sales.ApproveOrder")
}

func TestDescribeWorkflow_Mock(t *testing.T) {
	mod := mkModule("Sales")
	wf := mkWorkflowGen(string(nextID("wf")), "ApproveOrder")
	param := genWf.NewParameter()
	param.SetEntityQualifiedName("Sales.Order")
	wf.SetParameter(param)

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Workflows = makeWorkflowsRepo([]*genWf.Workflow{wf}, mod.ID)
	assertNoError(t, describeWorkflowGen(ctx, ast.QualifiedName{Module: "Sales", Name: "ApproveOrder"}))

	out := buf.String()
	assertContainsStr(t, out, "create workflow")
	assertContainsStr(t, out, "Sales.ApproveOrder")

	// Roundtrip: DESCRIBE output must be parseable as valid MDL (issue #478)
	_, parseErrs := visitor.Build(out)
	if len(parseErrs) > 0 {
		t.Errorf("describe workflow output is not valid MDL: %v\nOutput:\n%s", parseErrs[0], out)
	}
}

func TestDescribeWorkflow_NotFound(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	ctx.Workflows = makeWorkflowsRepo(nil, "")
	assertError(t, describeWorkflowGen(ctx, ast.QualifiedName{Module: "X", Name: "NoSuch"}))
}

func TestShowWorkflows_FilterByModule(t *testing.T) {
	mod := mkModule("Sales")
	wf := mkWorkflowGen(string(nextID("wf")), "ApproveOrder")

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Workflows = makeWorkflowsRepo([]*genWf.Workflow{wf}, mod.ID)
	assertNoError(t, listWorkflowsGen(ctx, "Sales"))
	assertContainsStr(t, buf.String(), "Sales.ApproveOrder")
}
