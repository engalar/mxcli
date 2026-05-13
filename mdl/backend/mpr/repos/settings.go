// SPDX-License-Identifier: Apache-2.0

// All write paths rely on mmpr.Writer's automatic cache invalidation
// (addendum Blocker 4). Repos do NOT call ReaderCache.Invalidate
// after each write.

package mprrepos

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genSet "github.com/mendixlabs/mxcli/modelsdk/gen/settings"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// projectSettingsTypeName is the BSON $Type for the singleton project
// settings unit. Fixture probe: 1 exact match.
const projectSettingsTypeName = "Settings$ProjectSettings"

// moduleSettingsTypeName is the BSON $Type Mendix uses for the
// per-module settings unit. Note: the storage prefix is
// "Projects$ModuleSettings" (lives in modelsdk/gen/projects), not
// "Settings$ModuleSettings" — the stub doc-comment in
// mdl/repos/settings.go reflects an older guess.
// Fixture probe: 8 exact matches, one per non-System module, with
// ContainerID == moduleID.
const moduleSettingsTypeName = "Projects$ModuleSettings"

// projectSettingsRepo is the direct-mode singleton ProjectSettings
// repository.
type projectSettingsRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewProjectSettingsRepository constructs the direct-mode repository.
func NewProjectSettingsRepository(w *mmpr.Writer) repos.ProjectSettingsRepository {
	return &projectSettingsRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

// Get returns the singleton Settings$ProjectSettings unit. Returns an
// error if zero or multiple are found (the fixture has exactly one).
func (r *projectSettingsRepo) Get() (*genSet.ProjectSettings, error) {
	refs, err := r.r.ListUnitsByType(projectSettingsTypeName)
	if err != nil {
		return nil, err
	}
	var found string
	for _, ref := range refs {
		if ref.Type != projectSettingsTypeName {
			continue
		}
		if found != "" {
			return nil, fmt.Errorf("multiple %s units found", projectSettingsTypeName)
		}
		found = ref.ID
	}
	if found == "" {
		return nil, fmt.Errorf("no %s unit in project", projectSettingsTypeName)
	}
	bytes, err := r.r.GetRawUnitBytes(found)
	if err != nil {
		return nil, err
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode project settings %s: %w", found, err)
	}
	ps, ok := elem.(*genSet.ProjectSettings)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a ProjectSettings (got %T)", found, elem)
	}
	return ps, nil
}

// Update writes a ProjectSettings unit back. The ID must be set
// (decoded units always have one).
func (r *projectSettingsRepo) Update(s *genSet.ProjectSettings) error {
	if s.ID() == "" {
		return fmt.Errorf("ProjectSettings.Update: ID is empty")
	}
	contents, err := r.enc.Encode(s)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(s.ID()), contents)
}

var _ repos.ProjectSettingsRepository = (*projectSettingsRepo)(nil)

// moduleSettingsRepo is the direct-mode ModuleSettings repository
// (per-module). Each module has at most one Projects$ModuleSettings
// unit whose ContainerID equals the module's UUID.
type moduleSettingsRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewModuleSettingsRepository constructs the direct-mode repository.
func NewModuleSettingsRepository(w *mmpr.Writer) repos.ModuleSettingsRepository {
	return &moduleSettingsRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

// Get returns the ModuleSettings unit whose container is moduleID, or
// an error if none / more than one exist.
func (r *moduleSettingsRepo) Get(moduleID model.ID) (element.Element, error) {
	refs, err := r.r.ListUnitsByType(moduleSettingsTypeName)
	if err != nil {
		return nil, err
	}
	var found string
	for _, ref := range refs {
		if ref.Type != moduleSettingsTypeName {
			continue
		}
		if ref.ContainerID != string(moduleID) {
			continue
		}
		if found != "" {
			return nil, fmt.Errorf("multiple %s units for module %s", moduleSettingsTypeName, moduleID)
		}
		found = ref.ID
	}
	if found == "" {
		return nil, fmt.Errorf("no %s unit for module %s", moduleSettingsTypeName, moduleID)
	}
	bytes, err := r.r.GetRawUnitBytes(found)
	if err != nil {
		return nil, err
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode module settings %s: %w", found, err)
	}
	return elem, nil
}

// Update writes a ModuleSettings element back. The element must have a
// non-empty ID (decoded units always do).
func (r *moduleSettingsRepo) Update(_ model.ID, s element.Element) error {
	if s.ID() == "" {
		return fmt.Errorf("ModuleSettings.Update: element ID is empty")
	}
	contents, err := r.enc.Encode(s)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(s.ID()), contents)
}

var _ repos.ModuleSettingsRepository = (*moduleSettingsRepo)(nil)
