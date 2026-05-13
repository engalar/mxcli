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
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const domainModelTypeName = "DomainModels$DomainModel"

// domainModelRepo is the direct-mode DomainModelRepository.
//
// Each module owns exactly one DomainModel unit; the unit carries the
// module's entities, associations, and annotations as embedded child
// elements (EntitiesItems / AssociationsItems).
type domainModelRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewDomainModelRepository constructs the direct-mode repository.
func NewDomainModelRepository(w *mmpr.Writer) repos.DomainModelRepository {
	return &domainModelRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *domainModelRepo) Get(id model.ID) (*genDm.DomainModel, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("domain model not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode domain model %s: %w", id, err)
	}
	dm, ok := elem.(*genDm.DomainModel)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a DomainModel (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return dm, nil
}

// List returns all DomainModel units whose container chain ends at
// moduleID. With moduleID empty, returns all DomainModels in the
// project.
func (r *domainModelRepo) List(moduleID model.ID) ([]*genDm.DomainModel, error) {
	refs, err := r.r.ListUnitsByType(domainModelTypeName)
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
	result := make([]*genDm.DomainModel, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != domainModelTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		dm, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode domain model %s: %w", ref.ID, err)
		}
		result = append(result, dm)
	}
	return result, nil
}

func (r *domainModelRepo) Create(parentUUID string, containmentName string, dm *genDm.DomainModel) error {
	if dm.ID() == "" {
		dm.SetID(element.ID(mmpr.GenerateID()))
	}
	if dm.TypeName() == "" {
		dm.SetTypeName(domainModelTypeName)
	}
	contents, err := r.enc.Encode(dm)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(dm.ID()), parentUUID, containmentName, dm.TypeName(), contents)
}

func (r *domainModelRepo) Update(dm *genDm.DomainModel) error {
	contents, err := r.enc.Encode(dm)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(dm.ID()), contents)
}

func (r *domainModelRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *domainModelRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.DomainModelRepository = (*domainModelRepo)(nil)
