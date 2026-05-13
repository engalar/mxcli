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

// agentsRepo is the direct-mode AgentRepository (umbrella over
// Agent / KnowledgeBase / ConsumedMCPService / Model).
//
// modelsdk/gen has no agents package yet (the schema regen for Mendix
// 11.9+ AgentEditorCommons hasn't landed). Reads go through
// dec.Decode which falls back to LazyDoc for unregistered types.
//
// The fixture probe shows zero Agents$* units — these tests therefore
// exercise only the wiring (List of empty + Create round-trip is
// covered via the Recording mock). Stage 3 will introduce typed
// sub-repositories once the gen tree exposes Agent root types.
type agentsRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewAgentRepository constructs the direct-mode repository.
func NewAgentRepository(w *mmpr.Writer) repos.AgentRepository {
	return &agentsRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *agentsRepo) Get(id model.ID) (element.Element, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("agent unit not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode agent unit %s: %w", id, err)
	}
	return elem, nil
}

// ListByType walks Reader.ListUnitsByType filtering on the exact type
// name. typeName must include the storage prefix (e.g. "Agents$Agent").
func (r *agentsRepo) ListByType(typeName string) ([]element.Element, error) {
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
			return nil, fmt.Errorf("decode agent unit %s: %w", ref.ID, err)
		}
		result = append(result, elem)
	}
	return result, nil
}

func (r *agentsRepo) Create(parentUUID string, containmentName string, agent element.Element) error {
	if agent == nil {
		return fmt.Errorf("Agent.Create: nil element")
	}
	// element.Element exposes ID() and TypeName() but not the setters
	// (those live on *element.Base). Callers must construct the
	// concrete type with both fields populated; the Stage 3 typed
	// sub-repositories will hide this requirement once Agents gen
	// types exist.
	if agent.ID() == "" {
		return fmt.Errorf("Agent.Create: element ID is empty (caller must set; element.Element interface has no SetID)")
	}
	if agent.TypeName() == "" {
		return fmt.Errorf("Agent.Create: TypeName is empty (caller must set Agents$Agent / KnowledgeBase / etc.)")
	}
	contents, err := r.enc.Encode(agent)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(agent.ID()), parentUUID, containmentName, agent.TypeName(), contents)
}

func (r *agentsRepo) Update(agent element.Element) error {
	if agent == nil {
		return fmt.Errorf("Agent.Update: nil element")
	}
	if agent.ID() == "" {
		return fmt.Errorf("Agent.Update: ID is empty")
	}
	contents, err := r.enc.Encode(agent)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(agent.ID()), contents)
}

func (r *agentsRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *agentsRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.AgentRepository = (*agentsRepo)(nil)
