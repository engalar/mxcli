// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// LayoutReader / LayoutWriter / LayoutRepository — signatures
// intentionally minimal until Stage 3 cutover. Layouts are persisted
// under "Pages$Layout" units in the same packaging as pages.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// and produce an MPR implementation.
type LayoutReader interface {
	Get(id model.ID) (*genPg.Layout, error)
	List(moduleID model.ID) ([]*genPg.Layout, error)
}

type LayoutWriter interface {
	Create(parentUUID string, containmentName string, layout *genPg.Layout) error
	Update(layout *genPg.Layout) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type LayoutRepository interface {
	LayoutReader
	LayoutWriter
}
