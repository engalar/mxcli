// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// MicroflowReader queries microflows. Implementations decode raw BSON via
// codec.Decoder; freshly-decoded gen objects are returned by value-pointer
// (the caller may mutate freely without affecting the cache).
type MicroflowReader interface {
	Get(id model.ID) (*genMf.Microflow, error)
	List(moduleID model.ID) ([]*genMf.Microflow, error)
	ListAll() ([]*genMf.Microflow, error)
	FindByQualifiedName(qn string) (*genMf.Microflow, error)
	IsRule(qn string) (bool, error)
}

// MicroflowWriter creates/updates/deletes/moves microflows. Container
// lineage is supplied positionally (parentUUID, containmentName) — gen
// objects do not store container identity (addendum Blocker 2).
type MicroflowWriter interface {
	Create(parentUUID string, containmentName string, mf *genMf.Microflow) error
	Update(mf *genMf.Microflow) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type MicroflowRepository interface {
	MicroflowReader
	MicroflowWriter
}
