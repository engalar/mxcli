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
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const nanoflowTypeName = "Microflows$Nanoflow"

// nanoflowRepo is the direct-mode NanoflowRepository: reads via the
// concrete *mmpr.Reader, writes via a writeSink (no transaction).
type nanoflowRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewNanoflowRepository constructs the direct-mode repository.
func NewNanoflowRepository(w *mmpr.Writer) repos.NanoflowRepository {
	return &nanoflowRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *nanoflowRepo) Get(id model.ID) (*genMf.Nanoflow, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("nanoflow not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode nanoflow %s: %w", id, err)
	}
	nf, ok := elem.(*genMf.Nanoflow)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Nanoflow (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return nf, nil
}

// List returns all nanoflows whose unit chains up to moduleID. When
// moduleID is empty, returns every nanoflow in the project.
func (r *nanoflowRepo) List(moduleID model.ID) ([]*genMf.Nanoflow, error) {
	refs, err := r.r.ListUnitsByType(nanoflowTypeName)
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
	result := make([]*genMf.Nanoflow, 0, len(refs))
	for _, ref := range refs {
		// Strict $Type filter — ListUnitsByType is prefix-matched and
		// would otherwise pull in similarly-prefixed kinds.
		if ref.Type != nanoflowTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		nf, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode nanoflow %s: %w", ref.ID, err)
		}
		result = append(result, nf)
	}
	return result, nil
}

func (r *nanoflowRepo) Create(parentUUID string, containmentName string, nf *genMf.Nanoflow) error {
	if nf.ID() == "" {
		nf.SetID(element.ID(mmpr.GenerateID()))
	}
	if nf.TypeName() == "" {
		nf.SetTypeName(nanoflowTypeName)
	}
	contents, err := r.enc.Encode(nf)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(nf.ID()), parentUUID, containmentName, nf.TypeName(), contents)
}

func (r *nanoflowRepo) Update(nf *genMf.Nanoflow) error {
	contents, err := r.enc.Encode(nf)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(nf.ID()), contents)
}

func (r *nanoflowRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *nanoflowRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

// ListAll returns every nanoflow in the project (all modules).
func (r *nanoflowRepo) ListAll() ([]*genMf.Nanoflow, error) {
	refs, err := r.r.ListUnitsByType(nanoflowTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genMf.Nanoflow, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != nanoflowTypeName {
			continue
		}
		nf, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode nanoflow %s: %w", ref.ID, err)
		}
		result = append(result, nf)
	}
	return result, nil
}

// FindByQualifiedName parses "Module.Name" and returns the matching
// nanoflow (or nil, nil if not found).
func (r *nanoflowRepo) FindByQualifiedName(qn string) (*genMf.Nanoflow, error) {
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
	refs, err := r.r.ListUnitsByType(nanoflowTypeName)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if ref.Type != nanoflowTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		nf, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, err
		}
		if nf.Name() == simpleName {
			return nf, nil
		}
	}
	return nil, nil
}

// GetContainerUUID returns the parent container UUID for a nanoflow
// unit by querying the MPR Unit table directly.
func (r *nanoflowRepo) GetContainerUUID(id model.ID) (model.ID, error) {
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

var _ repos.NanoflowRepository = (*nanoflowRepo)(nil)
