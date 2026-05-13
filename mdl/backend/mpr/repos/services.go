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

// servicesRepo is the direct-mode ServiceRepository (umbrella over
// REST clients, OData services, business events, JS/Java actions,
// app services, etc.). The fixture probe shows zero Services$* units;
// tests exercise wiring only. Stage 3 may split this into per-service
// sub-repositories based on executor needs.
type servicesRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewServiceRepository constructs the direct-mode repository.
func NewServiceRepository(w *mmpr.Writer) repos.ServiceRepository {
	return &servicesRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *servicesRepo) Get(id model.ID) (element.Element, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("service unit not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode service unit %s: %w", id, err)
	}
	return elem, nil
}

// ListByType walks Reader.ListUnitsByType filtering on the exact type
// name. typeName must include the storage prefix (e.g.
// "Services$ConsumedODataService").
func (r *servicesRepo) ListByType(typeName string) ([]element.Element, error) {
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
			return nil, fmt.Errorf("decode service unit %s: %w", ref.ID, err)
		}
		result = append(result, elem)
	}
	return result, nil
}

func (r *servicesRepo) Create(parentUUID string, containmentName string, svc element.Element) error {
	if svc == nil {
		return fmt.Errorf("Service.Create: nil element")
	}
	if svc.ID() == "" {
		return fmt.Errorf("Service.Create: element ID is empty (caller must set; element.Element interface has no SetID)")
	}
	if svc.TypeName() == "" {
		return fmt.Errorf("Service.Create: TypeName is empty (caller must set Services$ConsumedODataService / etc.)")
	}
	contents, err := r.enc.Encode(svc)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(svc.ID()), parentUUID, containmentName, svc.TypeName(), contents)
}

func (r *servicesRepo) Update(svc element.Element) error {
	if svc == nil {
		return fmt.Errorf("Service.Update: nil element")
	}
	if svc.ID() == "" {
		return fmt.Errorf("Service.Update: ID is empty")
	}
	contents, err := r.enc.Encode(svc)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(svc.ID()), contents)
}

func (r *servicesRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *servicesRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.ServiceRepository = (*servicesRepo)(nil)
