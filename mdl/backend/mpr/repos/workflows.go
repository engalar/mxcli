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
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const workflowTypeName = "Workflows$Workflow"

// workflowRepo is the direct-mode WorkflowRepository.
//
// The fixture probe shows zero Workflow units. CRUD is exercised via
// fresh-construct + InsertUnit roundtrip; OpenForMutation returns a
// minimal Stage-2.6 mutator that applies the same Option A
// decode-edit-encode pattern as pageMutator (re-encode on Commit) but
// without a generic activity walker — the mutator surfaces explicit
// "not yet implemented" errors for InsertActivity / DeleteActivity /
// ReplaceActivity / SetActivityProperty until Stage 3 brings a real
// workflow tree walker.
type workflowRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewWorkflowRepository constructs the direct-mode repository.
func NewWorkflowRepository(w *mmpr.Writer) repos.WorkflowRepository {
	return &workflowRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *workflowRepo) Get(id model.ID) (*genWf.Workflow, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode workflow %s: %w", id, err)
	}
	wf, ok := elem.(*genWf.Workflow)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Workflow (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return wf, nil
}

func (r *workflowRepo) List(moduleID model.ID) ([]*genWf.Workflow, error) {
	refs, err := r.r.ListUnitsByType(workflowTypeName)
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
	result := make([]*genWf.Workflow, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != workflowTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		wf, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode workflow %s: %w", ref.ID, err)
		}
		result = append(result, wf)
	}
	return result, nil
}

func (r *workflowRepo) ListAll() ([]*genWf.Workflow, error) {
	refs, err := r.r.ListUnitsByType(workflowTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genWf.Workflow, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != workflowTypeName {
			continue
		}
		wf, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode workflow %s: %w", ref.ID, err)
		}
		result = append(result, wf)
	}
	return result, nil
}

func (r *workflowRepo) FindByQualifiedName(qn string) (*genWf.Workflow, error) {
	moduleName, simpleName, ok := splitQN(qn)
	if !ok {
		return nil, fmt.Errorf("FindByQualifiedName: invalid qualified name %q (want Module.Name)", qn)
	}
	mods, err := r.r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string, len(mods))
	for _, m := range mods {
		moduleMap[m.ID] = m.Name
	}
	parents, err := r.r.BuildContainerParent()
	if err != nil {
		return nil, err
	}
	refs, err := r.r.ListUnitsByType(workflowTypeName)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if ref.Type != workflowTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		wf, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, err
		}
		if wf.Name() == simpleName {
			return wf, nil
		}
	}
	return nil, nil
}

func (r *workflowRepo) GetContainerUUID(id model.ID) (model.ID, error) {
	if id == "" {
		return "", fmt.Errorf("GetContainerUUID: empty id")
	}
	bin := mmpr.IDToBsonBinary(string(id))
	if len(bin.Data) != 16 {
		return "", fmt.Errorf("GetContainerUUID: invalid id %q", id)
	}
	var blob []byte
	err := r.r.DB().QueryRow("SELECT ContainerID FROM Unit WHERE UnitID = ?", bin.Data).Scan(&blob)
	if err != nil {
		return "", fmt.Errorf("GetContainerUUID(%s): %w", id, err)
	}
	return model.ID(mmpr.BlobToUUID(blob)), nil
}

func (r *workflowRepo) Create(parentUUID string, containmentName string, wf *genWf.Workflow) error {
	if wf.ID() == "" {
		wf.SetID(element.ID(mmpr.GenerateID()))
	}
	if wf.TypeName() == "" {
		wf.SetTypeName(workflowTypeName)
	}
	contents, err := r.enc.Encode(wf)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(wf.ID()), parentUUID, containmentName, wf.TypeName(), contents)
}

func (r *workflowRepo) Update(wf *genWf.Workflow) error {
	contents, err := r.enc.Encode(wf)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(wf.ID()), contents)
}

func (r *workflowRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *workflowRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

// OpenForMutation returns a Stage 2.6 workflowMutator that supports
// Commit (re-encodes the cached workflow) but not the
// activity-tree edits — those return explicit "not yet implemented"
// errors until Stage 3 wires a real walker.
func (r *workflowRepo) OpenForMutation(id model.ID) (repos.WorkflowMutator, error) {
	wf, err := r.Get(id)
	if err != nil {
		return nil, fmt.Errorf("OpenForMutation(%s): %w", id, err)
	}
	return &workflowMutator{repo: r, wf: wf}, nil
}

var _ repos.WorkflowRepository = (*workflowRepo)(nil)

// workflowMutator is the Stage 2.6 minimal Option-A mutator. Activity
// walking is deferred to Stage 3 — the four edit methods surface
// "not implemented" so callers can detect the gap explicitly instead
// of silently no-oping.
type workflowMutator struct {
	repo *workflowRepo
	wf   *genWf.Workflow
}

func (m *workflowMutator) SetActivityProperty(_ model.ID, _ string, _ any) error {
	return fmt.Errorf("WorkflowMutator.SetActivityProperty: not implemented in Stage 2.6 (Stage 3 will land an activity walker)")
}

func (m *workflowMutator) InsertActivity(_ model.ID, _ string, _ element.Element) error {
	return fmt.Errorf("WorkflowMutator.InsertActivity: not implemented in Stage 2.6 (Stage 3 will land an activity walker)")
}

func (m *workflowMutator) DeleteActivity(_ model.ID) error {
	return fmt.Errorf("WorkflowMutator.DeleteActivity: not implemented in Stage 2.6 (Stage 3 will land an activity walker)")
}

func (m *workflowMutator) ReplaceActivity(_ model.ID, _ element.Element) error {
	return fmt.Errorf("WorkflowMutator.ReplaceActivity: not implemented in Stage 2.6 (Stage 3 will land an activity walker)")
}

// Commit persists the cached workflow via workflowRepo.Update — the
// canonical Option A decode-edit-encode round-trip.
func (m *workflowMutator) Commit() error { return m.repo.Update(m.wf) }

var _ repos.WorkflowMutator = (*workflowMutator)(nil)
