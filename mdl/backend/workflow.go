// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// WorkflowBackend provides workflow operations.
//
// Stage 3.3.3.E1 retired the legacy sdk-typed surface. The gen-typed
// methods are the only public API; DeleteWorkflow is pure ID-based and
// stays unchanged.
type WorkflowBackend interface {
	// Gen-typed surface (production wiring in mdl/backend/mpr/backend.go;
	// mock stubs in mdl/backend/mock/mock_workflow.go return descriptive
	// errors per master plan §5 P3 / CLAUDE.md MockBackend audit rule).
	ListWorkflowsGen() ([]*genWf.Workflow, error)
	GetWorkflowGen(id model.ID) (*genWf.Workflow, error)
	CreateWorkflowGen(parentUUID, containmentName string, wf *genWf.Workflow) error
	UpdateWorkflowGen(wf *genWf.Workflow) error

	// DeleteWorkflow remains sdk-free: model.ID is the only input and
	// implementations route through msdkWriter.DeleteUnit. Kept on the
	// gen-typed surface for symmetry with the rest of the FullBackend.
	DeleteWorkflow(id model.ID) error
}
