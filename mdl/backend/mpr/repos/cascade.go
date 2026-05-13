// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// cascadeService implements repos.CascadeService end-to-end via modelsdk
// (no sdk/mpr dependency). DeleteModule walks the Unit table containment
// graph via mmpr.Writer.DeleteChildUnits + DeleteUnit. DeleteFolder
// guards against deleting a non-empty folder via a direct SQL count query
// against the modelsdk Reader's underlying DB.
type cascadeService struct {
	w *mmpr.Writer
	r *mmpr.Reader
}

// NewCascadeService constructs a CascadeService backed by the given Writer.
func NewCascadeService(w *mmpr.Writer) repos.CascadeService {
	return &cascadeService{
		w: w,
		r: w.ConcreteReader(),
	}
}

var _ repos.CascadeService = (*cascadeService)(nil)

func (s *cascadeService) DeleteModule(moduleID model.ID) error {
	if err := s.w.DeleteChildUnits(string(moduleID)); err != nil {
		return fmt.Errorf("delete child units of module %s: %w", moduleID, err)
	}
	return s.w.DeleteUnit(string(moduleID))
}

func (s *cascadeService) DeleteFolder(folderID model.ID) error {
	// Mirror legacy semantics (mdl/backend/mpr/modules_modelsdk.go
	// deleteFolderViaModelsdk): refuse to delete a non-empty folder.
	db := s.r.DB()
	blob := mmpr.IDToBsonBinary(string(folderID)).Data
	var count int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM Unit WHERE ContainerID = ? AND UnitID != ContainerID",
		blob,
	).Scan(&count); err != nil {
		return fmt.Errorf("check folder contents: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("folder %s is not empty: contains %d child unit(s)", folderID, count)
	}
	return s.w.DeleteUnit(string(folderID))
}
