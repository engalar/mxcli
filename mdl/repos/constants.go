// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genCo "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
)

// ConstantReader / ConstantWriter / ConstantRepository — signatures
// intentionally minimal until Stage 3 cutover.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// and produce an MPR implementation.
type ConstantReader interface {
	Get(id model.ID) (*genCo.Constant, error)
	List(moduleID model.ID) ([]*genCo.Constant, error)
}

type ConstantWriter interface {
	Create(parentUUID string, containmentName string, c *genCo.Constant) error
	Update(c *genCo.Constant) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type ConstantRepository interface {
	ConstantReader
	ConstantWriter
}
