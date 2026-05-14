// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

type PageReader interface {
	Get(id model.ID) (*genPg.Page, error)
	List(moduleID model.ID) ([]*genPg.Page, error)
	ListAll() ([]*genPg.Page, error)
	FindByQualifiedName(qn string) (*genPg.Page, error)

	// GetContainerUUID returns the parent container UUID (folder or
	// module ID) of a page unit. Codec-decoded gen objects do not
	// carry container linkage, so callers retrieve it from the MPR Unit
	// table by UnitID. Returns "" with a non-nil error if the unit is
	// not found.
	GetContainerUUID(id model.ID) (model.ID, error)
}

type PageWriter interface {
	Create(parentUUID string, containmentName string, page *genPg.Page) error
	Update(page *genPg.Page) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

// PageRepository combines reader + writer + the mutator factory for
// large-unit incremental editing (spec section 5 Mutator interfaces).
type PageRepository interface {
	PageReader
	PageWriter
	OpenForMutation(pageID model.ID) (PageMutator, error)
}
