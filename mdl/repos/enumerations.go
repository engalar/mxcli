// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genEn "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
)

// EnumerationReader / EnumerationWriter / EnumerationRepository —
// signatures intentionally minimal until Stage 3 cutover.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// and produce an MPR implementation.
type EnumerationReader interface {
	Get(id model.ID) (*genEn.Enumeration, error)
	List(moduleID model.ID) ([]*genEn.Enumeration, error)
}

type EnumerationWriter interface {
	Create(parentUUID string, containmentName string, e *genEn.Enumeration) error
	Update(e *genEn.Enumeration) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type EnumerationRepository interface {
	EnumerationReader
	EnumerationWriter
}
