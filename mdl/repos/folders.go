// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
)

// FolderReader / FolderWriter / FolderRepository — signatures
// intentionally minimal until Stage 3 cutover. Folders are project-
// scoped tree nodes that the legacy FolderBackend returns by module ID.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// (MOVE folder, CREATE folder, etc.) and produce an MPR implementation.
type FolderReader interface {
	Get(id model.ID) (*genPr.Folder, error)
	List(moduleID model.ID) ([]*genPr.Folder, error)
}

type FolderWriter interface {
	Create(parentUUID string, containmentName string, f *genPr.Folder) error
	Update(f *genPr.Folder) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type FolderRepository interface {
	FolderReader
	FolderWriter
}
