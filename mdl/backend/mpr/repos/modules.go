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
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// moduleTypeName is the BSON $Type Mendix uses for module units. The
// gen "Module" struct has a storage alias to "Projects$ModuleImpl"
// (initModule sets TypeName to ModuleImpl). The fixture probe
// confirms 8 exact ModuleImpl matches; "Projects$Module" prefix
// pulls in 16 (ModuleImpl + ModuleSettings) and yields 0 exact.
const moduleTypeName = "Projects$ModuleImpl"

// moduleRepo is the direct-mode ModuleRepository.
type moduleRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewModuleRepository constructs the direct-mode repository.
func NewModuleRepository(w *mmpr.Writer) repos.ModuleRepository {
	return &moduleRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *moduleRepo) Get(id model.ID) (*genPr.Module, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("module not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode module %s: %w", id, err)
	}
	m, ok := elem.(*genPr.Module)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Module (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return m, nil
}

// ListAll returns every Projects$ModuleImpl unit.
func (r *moduleRepo) ListAll() ([]*genPr.Module, error) {
	refs, err := r.r.ListUnitsByType(moduleTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genPr.Module, 0, len(refs))
	for _, ref := range refs {
		// Strict $Type filter — the "Projects$" prefix collides with
		// ModuleSettings + Project + Folder, ListUnitsByType is
		// prefix-matched.
		if ref.Type != moduleTypeName {
			continue
		}
		m, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode module %s: %w", ref.ID, err)
		}
		result = append(result, m)
	}
	return result, nil
}

// FindByName scans ListAll for an exact name match. Returns (nil, nil)
// if no module matches.
func (r *moduleRepo) FindByName(name string) (*genPr.Module, error) {
	all, err := r.ListAll()
	if err != nil {
		return nil, err
	}
	for _, m := range all {
		if m.Name() == name {
			return m, nil
		}
	}
	return nil, nil
}

func (r *moduleRepo) Create(parentUUID string, containmentName string, m *genPr.Module) error {
	if m.ID() == "" {
		m.SetID(element.ID(mmpr.GenerateID()))
	}
	if m.TypeName() == "" {
		m.SetTypeName(moduleTypeName)
	}
	contents, err := r.enc.Encode(m)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(m.ID()), parentUUID, containmentName, m.TypeName(), contents)
}

func (r *moduleRepo) Update(m *genPr.Module) error {
	contents, err := r.enc.Encode(m)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(m.ID()), contents)
}

func (r *moduleRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

var _ repos.ModuleRepository = (*moduleRepo)(nil)
