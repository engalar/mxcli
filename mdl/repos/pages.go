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
}

type PageWriter interface {
	Create(parentUUID string, containmentName string, page *genPg.Page) error
	Update(page *genPg.Page) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

// PageRepository combines reader + writer; the mutator factory
// (OpenForMutation → PageMutator) is added in Task 3 once the
// PageMutator interface exists. Stage 2 task ordering avoids forward
// references in this file.
type PageRepository interface {
	PageReader
	PageWriter
}
