// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// WorkflowReader queries workflows. Implementations decode raw BSON via
// codec.Decoder; freshly-decoded gen objects are returned by value-pointer
// (the caller may mutate freely without affecting the cache).
type WorkflowReader interface {
	Get(id model.ID) (*genWf.Workflow, error)
	List(moduleID model.ID) ([]*genWf.Workflow, error)
	ListAll() ([]*genWf.Workflow, error)
	FindByQualifiedName(qn string) (*genWf.Workflow, error)

	// GetContainerUUID returns the parent container UUID (folder or
	// module ID) of a workflow unit. Codec-decoded gen objects do not
	// carry container linkage, so callers retrieve it from the MPR Unit
	// table by UnitID. Returns "" with a non-nil error if the unit is
	// not found.
	GetContainerUUID(id model.ID) (model.ID, error)
}

// WorkflowWriter creates/updates/deletes/moves workflows. Container
// lineage is supplied positionally (parentUUID, containmentName) — gen
// objects do not store container identity (addendum Blocker 2).
type WorkflowWriter interface {
	Create(parentUUID string, containmentName string, wf *genWf.Workflow) error
	Update(wf *genWf.Workflow) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type WorkflowRepository interface {
	WorkflowReader
	WorkflowWriter
	OpenForMutation(workflowID model.ID) (WorkflowMutator, error)
}
