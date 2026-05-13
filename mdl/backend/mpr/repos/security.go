// SPDX-License-Identifier: Apache-2.0

// All write paths rely on mmpr.Writer's automatic cache invalidation
// (addendum Blocker 4). Repos do NOT call ReaderCache.Invalidate
// after each write.

package mprrepos

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const (
	projectSecurityTypeName = "Security$ProjectSecurity"
	moduleSecurityTypeName  = "Security$ModuleSecurity"
)

// securityRepo is the direct-mode SecurityRepository covering both
// project-level (singleton) and module-level (per-module, ContainerID
// equals moduleID) security units.
type securityRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewSecurityRepository constructs the direct-mode repository.
func NewSecurityRepository(w *mmpr.Writer) repos.SecurityRepository {
	return &securityRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

// Get returns the singleton ProjectSecurity unit.
func (r *securityRepo) Get() (*genSec.ProjectSecurity, error) {
	refs, err := r.r.ListUnitsByType(projectSecurityTypeName)
	if err != nil {
		return nil, err
	}
	var found string
	for _, ref := range refs {
		if ref.Type != projectSecurityTypeName {
			continue
		}
		if found != "" {
			return nil, fmt.Errorf("multiple %s units found", projectSecurityTypeName)
		}
		found = ref.ID
	}
	if found == "" {
		return nil, fmt.Errorf("no %s unit in project", projectSecurityTypeName)
	}
	bytes, err := r.r.GetRawUnitBytes(found)
	if err != nil {
		return nil, err
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode project security %s: %w", found, err)
	}
	ps, ok := elem.(*genSec.ProjectSecurity)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a ProjectSecurity (got %T)", found, elem)
	}
	return ps, nil
}

// GetModuleSecurity returns the ModuleSecurity unit whose container is
// moduleID. Each module has at most one such unit; the System module
// may not have any (probe: 8 fixture units across 8 non-System modules).
func (r *securityRepo) GetModuleSecurity(moduleID model.ID) (*genSec.ModuleSecurity, error) {
	refs, err := r.r.ListUnitsByType(moduleSecurityTypeName)
	if err != nil {
		return nil, err
	}
	var found string
	for _, ref := range refs {
		if ref.Type != moduleSecurityTypeName {
			continue
		}
		if ref.ContainerID != string(moduleID) {
			continue
		}
		if found != "" {
			return nil, fmt.Errorf("multiple %s units for module %s", moduleSecurityTypeName, moduleID)
		}
		found = ref.ID
	}
	if found == "" {
		return nil, fmt.Errorf("no %s unit for module %s", moduleSecurityTypeName, moduleID)
	}
	bytes, err := r.r.GetRawUnitBytes(found)
	if err != nil {
		return nil, err
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode module security %s: %w", found, err)
	}
	ms, ok := elem.(*genSec.ModuleSecurity)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a ModuleSecurity (got %T)", found, elem)
	}
	return ms, nil
}

// Update writes a ProjectSecurity unit back. Element ID must be set.
func (r *securityRepo) Update(s *genSec.ProjectSecurity) error {
	if s.ID() == "" {
		return fmt.Errorf("ProjectSecurity.Update: ID is empty")
	}
	contents, err := r.enc.Encode(s)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(s.ID()), contents)
}

// UpdateModuleSecurity writes a ModuleSecurity unit back. moduleID is
// captured for diagnostic symmetry with the legacy backend; the actual
// store key is the unit's own ID.
func (r *securityRepo) UpdateModuleSecurity(_ model.ID, s *genSec.ModuleSecurity) error {
	if s.ID() == "" {
		return fmt.Errorf("ModuleSecurity.Update: ID is empty")
	}
	contents, err := r.enc.Encode(s)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(s.ID()), contents)
}

var _ repos.SecurityRepository = (*securityRepo)(nil)
