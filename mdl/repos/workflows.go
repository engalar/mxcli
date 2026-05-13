// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// WorkflowReader / WorkflowWriter / WorkflowRepository — signatures
// intentionally minimal until Stage 3 cutover. Workflows are large
// composite units whose ALTER paths require the WorkflowMutator
// (analogous to PageMutator) — see workflow_mutator.go.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// and produce an MPR implementation.
type WorkflowReader interface {
	Get(id model.ID) (*genWf.Workflow, error)
	List(moduleID model.ID) ([]*genWf.Workflow, error)
}

type WorkflowWriter interface {
	Create(parentUUID string, containmentName string, wf *genWf.Workflow) error
	Update(wf *genWf.Workflow) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

// WorkflowRepository combines reader + writer; the mutator factory
// (OpenForMutation → WorkflowMutator) is added in Task 3 once the
// WorkflowMutator interface exists. Stage 2 task ordering avoids forward
// references in this file.
type WorkflowRepository interface {
	WorkflowReader
	WorkflowWriter
}
