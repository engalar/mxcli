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
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// layoutTypeName is the BSON $Type Mendix uses for layouts. Note the
// stub doc-comment in mdl/repos/layouts.go references "Pages$Layout"
// (stale) — the actual storage name is "Forms$Layout" (legacy "Form"
// terminology, matching pageTypeName).
const layoutTypeName = "Forms$Layout"

// layoutRepo is the direct-mode LayoutRepository: reads via the
// concrete *mmpr.Reader, writes via a writeSink (no transaction).
type layoutRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewLayoutRepository constructs the direct-mode repository.
func NewLayoutRepository(w *mmpr.Writer) repos.LayoutRepository {
	return &layoutRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *layoutRepo) Get(id model.ID) (*genPg.Layout, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("layout not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode layout %s: %w", id, err)
	}
	layout, ok := elem.(*genPg.Layout)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Layout (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return layout, nil
}

// List returns all layouts whose unit chains up to moduleID. When
// moduleID is empty, returns every layout in the project.
func (r *layoutRepo) List(moduleID model.ID) ([]*genPg.Layout, error) {
	refs, err := r.r.ListUnitsByType(layoutTypeName)
	if err != nil {
		return nil, err
	}
	var moduleMap map[string]string
	var parents map[string]string
	wantName := ""
	if moduleID != "" {
		mods, err := r.r.ListModules()
		if err != nil {
			return nil, err
		}
		moduleMap = make(map[string]string, len(mods))
		for _, m := range mods {
			moduleMap[m.ID] = m.Name
			if m.ID == string(moduleID) {
				wantName = m.Name
			}
		}
		if wantName == "" {
			return nil, fmt.Errorf("module not found: %s", moduleID)
		}
		parents, err = r.r.BuildContainerParent()
		if err != nil {
			return nil, err
		}
	}
	result := make([]*genPg.Layout, 0, len(refs))
	for _, ref := range refs {
		// Strict $Type filter — ListUnitsByType is prefix-matched.
		if ref.Type != layoutTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		layout, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode layout %s: %w", ref.ID, err)
		}
		result = append(result, layout)
	}
	return result, nil
}

func (r *layoutRepo) Create(parentUUID string, containmentName string, layout *genPg.Layout) error {
	if layout.ID() == "" {
		layout.SetID(element.ID(mmpr.GenerateID()))
	}
	if layout.TypeName() == "" {
		layout.SetTypeName(layoutTypeName)
	}
	contents, err := r.enc.Encode(layout)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(layout.ID()), parentUUID, containmentName, layout.TypeName(), contents)
}

func (r *layoutRepo) Update(layout *genPg.Layout) error {
	contents, err := r.enc.Encode(layout)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(layout.ID()), contents)
}

func (r *layoutRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *layoutRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.LayoutRepository = (*layoutRepo)(nil)
