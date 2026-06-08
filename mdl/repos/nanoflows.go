// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	// Nanoflow root type lives in the microflows gen package
	// (modelsdk/gen/microflows/types.go:type Nanoflow struct), not in
	// modelsdk/gen/nanoflows which only contains parameter-value types.
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// NanoflowReader / NanoflowWriter / NanoflowRepository — signatures
// intentionally minimal until Stage 3 cutover. Mirror the legacy
// microflow backend (nanoflows are persisted under
// "Microflows$Nanoflow" units alongside microflows).
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// and produce an MPR implementation.
type NanoflowReader interface {
	Get(id model.ID) (*genMf.Nanoflow, error)
	List(moduleID model.ID) ([]*genMf.Nanoflow, error)
	ListAll() ([]*genMf.Nanoflow, error)
	FindByQualifiedName(qn string) (*genMf.Nanoflow, error)

	// GetContainerUUID returns the parent container UUID of a nanoflow
	// unit (folder or module ID). Codec-decoded gen objects do not carry
	// container linkage, so we retrieve it from the MPR Unit table by
	// UnitID. Returns "" with a non-nil error if the unit is not found.
	GetContainerUUID(id model.ID) (model.ID, error)
}

type NanoflowWriter interface {
	Create(parentUUID string, containmentName string, nf *genMf.Nanoflow) error
	Update(nf *genMf.Nanoflow) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type NanoflowRepository interface {
	NanoflowReader
	NanoflowWriter
}
