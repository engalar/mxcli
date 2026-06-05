// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// DomainModelWithContainer pairs a DomainModel with the ID of its direct
// parent unit (always the owning module — DomainModels are never in folders).
type DomainModelWithContainer struct {
	DM          *genDm.DomainModel
	ContainerID model.ID
}

// DomainModelReader / DomainModelWriter / DomainModelRepository —
// signatures intentionally minimal until Stage 3 cutover. Each module
// owns exactly one DomainModel unit ("DomainModels$DomainModel").
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// (entity/attribute/association mutation lives here) and produce an MPR
// implementation.
type DomainModelReader interface {
	Get(id model.ID) (*genDm.DomainModel, error)
	List(moduleID model.ID) ([]*genDm.DomainModel, error)
	// ListAllWithContainerID returns every DomainModel in the project together
	// with its direct parent unit ID (the module). One DB+file pass; preferred
	// over repeated List(moduleID) calls to avoid O(N²) file reads.
	ListAllWithContainerID() ([]DomainModelWithContainer, error)
}

type DomainModelWriter interface {
	Create(parentUUID string, containmentName string, dm *genDm.DomainModel) error
	Update(dm *genDm.DomainModel) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type DomainModelRepository interface {
	DomainModelReader
	DomainModelWriter
}
