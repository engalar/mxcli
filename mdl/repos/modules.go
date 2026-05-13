// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
)

// ModuleReader / ModuleWriter / ModuleRepository — signatures
// intentionally minimal until Stage 3 cutover. Modules are persisted
// under "Projects$Module" units; the Module gen type lives in the
// projects gen package alongside Folder + ModuleDocument.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// (CREATE/RENAME/DROP MODULE) and produce an MPR implementation.
type ModuleReader interface {
	Get(id model.ID) (*genPr.Module, error)
	ListAll() ([]*genPr.Module, error)
	FindByName(name string) (*genPr.Module, error)
}

type ModuleWriter interface {
	Create(parentUUID string, containmentName string, m *genPr.Module) error
	Update(m *genPr.Module) error
	Delete(id model.ID) error
}

type ModuleRepository interface {
	ModuleReader
	ModuleWriter
}
