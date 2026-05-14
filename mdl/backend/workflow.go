// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// WorkflowBackend provides workflow operations.
//
// Stage 3.3.3.C1 adds the gen-typed sibling surface (additive). The
// legacy sdk-typed methods stay until Phase E1 retires them once every
// consumer migrates.
type WorkflowBackend interface {
	ListWorkflows() ([]*workflows.Workflow, error)
	GetWorkflow(id model.ID) (*workflows.Workflow, error)
	CreateWorkflow(wf *workflows.Workflow) error
	UpdateWorkflow(wf *workflows.Workflow) error
	DeleteWorkflow(id model.ID) error

	// Gen-typed surface (production wiring in mdl/backend/mpr/backend.go;
	// mock stubs in mdl/backend/mock/mock_workflow.go return descriptive
	// errors per master plan §5 P3 / CLAUDE.md MockBackend audit rule).
	ListWorkflowsGen() ([]*genWf.Workflow, error)
	GetWorkflowGen(id model.ID) (*genWf.Workflow, error)
	CreateWorkflowGen(parentUUID, containmentName string, wf *genWf.Workflow) error
	UpdateWorkflowGen(wf *genWf.Workflow) error
}
