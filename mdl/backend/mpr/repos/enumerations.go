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
	genEn "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const enumerationTypeName = "Enumerations$Enumeration"

// enumerationRepo is the direct-mode EnumerationRepository.
type enumerationRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewEnumerationRepository constructs the direct-mode repository.
func NewEnumerationRepository(w *mmpr.Writer) repos.EnumerationRepository {
	return &enumerationRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *enumerationRepo) Get(id model.ID) (*genEn.Enumeration, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("enumeration not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode enumeration %s: %w", id, err)
	}
	e, ok := elem.(*genEn.Enumeration)
	if !ok {
		return nil, fmt.Errorf("unit %s is not an Enumeration (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return e, nil
}

func (r *enumerationRepo) List(moduleID model.ID) ([]*genEn.Enumeration, error) {
	refs, err := r.r.ListUnitsByType(enumerationTypeName)
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
	result := make([]*genEn.Enumeration, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != enumerationTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		e, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode enumeration %s: %w", ref.ID, err)
		}
		result = append(result, e)
	}
	return result, nil
}

func (r *enumerationRepo) Create(parentUUID string, containmentName string, e *genEn.Enumeration) error {
	if e.ID() == "" {
		e.SetID(element.ID(mmpr.GenerateID()))
	}
	if e.TypeName() == "" {
		e.SetTypeName(enumerationTypeName)
	}
	contents, err := r.enc.Encode(e)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(e.ID()), parentUUID, containmentName, e.TypeName(), contents)
}

func (r *enumerationRepo) Update(e *genEn.Enumeration) error {
	contents, err := r.enc.Encode(e)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(e.ID()), contents)
}

func (r *enumerationRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *enumerationRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.EnumerationRepository = (*enumerationRepo)(nil)
