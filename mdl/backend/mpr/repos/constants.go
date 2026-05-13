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
	genCo "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const constantTypeName = "Constants$Constant"

// constantRepo is the direct-mode ConstantRepository.
type constantRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewConstantRepository constructs the direct-mode repository.
func NewConstantRepository(w *mmpr.Writer) repos.ConstantRepository {
	return &constantRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *constantRepo) Get(id model.ID) (*genCo.Constant, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("constant not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode constant %s: %w", id, err)
	}
	c, ok := elem.(*genCo.Constant)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Constant (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return c, nil
}

func (r *constantRepo) List(moduleID model.ID) ([]*genCo.Constant, error) {
	refs, err := r.r.ListUnitsByType(constantTypeName)
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
	result := make([]*genCo.Constant, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != constantTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		c, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode constant %s: %w", ref.ID, err)
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *constantRepo) Create(parentUUID string, containmentName string, c *genCo.Constant) error {
	if c.ID() == "" {
		c.SetID(element.ID(mmpr.GenerateID()))
	}
	if c.TypeName() == "" {
		c.SetTypeName(constantTypeName)
	}
	contents, err := r.enc.Encode(c)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(c.ID()), parentUUID, containmentName, c.TypeName(), contents)
}

func (r *constantRepo) Update(c *genCo.Constant) error {
	contents, err := r.enc.Encode(c)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(c.ID()), contents)
}

func (r *constantRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *constantRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.ConstantRepository = (*constantRepo)(nil)
