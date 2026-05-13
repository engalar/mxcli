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
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// mappingsRepo is the direct-mode MappingRepository (umbrella over
// Import / Export / ML mappings). The fixture probe shows zero
// Mappings$* units; tests exercise wiring only. Stage 3 will split
// into typed sub-repositories per kind.
type mappingsRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewMappingRepository constructs the direct-mode repository.
func NewMappingRepository(w *mmpr.Writer) repos.MappingRepository {
	return &mappingsRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *mappingsRepo) Get(id model.ID) (element.Element, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("mapping unit not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode mapping unit %s: %w", id, err)
	}
	return elem, nil
}

// ListByType walks Reader.ListUnitsByType filtering on the exact type
// name. typeName must include the storage prefix (e.g.
// "Mappings$ImportMapping").
func (r *mappingsRepo) ListByType(typeName string) ([]element.Element, error) {
	if typeName == "" {
		return nil, fmt.Errorf("ListByType: typeName must not be empty")
	}
	refs, err := r.r.ListUnitsByType(typeName)
	if err != nil {
		return nil, err
	}
	result := make([]element.Element, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != typeName {
			continue
		}
		elem, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode mapping unit %s: %w", ref.ID, err)
		}
		result = append(result, elem)
	}
	return result, nil
}

func (r *mappingsRepo) Create(parentUUID string, containmentName string, mapping element.Element) error {
	if mapping == nil {
		return fmt.Errorf("Mapping.Create: nil element")
	}
	if mapping.ID() == "" {
		return fmt.Errorf("Mapping.Create: element ID is empty (caller must set; element.Element interface has no SetID)")
	}
	if mapping.TypeName() == "" {
		return fmt.Errorf("Mapping.Create: TypeName is empty (caller must set Mappings$ImportMapping / ExportMapping / etc.)")
	}
	contents, err := r.enc.Encode(mapping)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(mapping.ID()), parentUUID, containmentName, mapping.TypeName(), contents)
}

func (r *mappingsRepo) Update(mapping element.Element) error {
	if mapping == nil {
		return fmt.Errorf("Mapping.Update: nil element")
	}
	if mapping.ID() == "" {
		return fmt.Errorf("Mapping.Update: ID is empty")
	}
	contents, err := r.enc.Encode(mapping)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(mapping.ID()), contents)
}

func (r *mappingsRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *mappingsRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.MappingRepository = (*mappingsRepo)(nil)
