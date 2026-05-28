// SPDX-License-Identifier: Apache-2.0

// All write paths rely on mmpr.Writer's automatic cache invalidation
// (addendum Blocker 4). Repos do NOT call ReaderCache.Invalidate
// after each write.

package mprrepos

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const microflowTypeName = "Microflows$Microflow"
const ruleTypeName = "Microflows$Rule"

// microflowRepo is the direct-mode MicroflowRepository: reads via the
// concrete *mmpr.Reader, writes via a writerSink (no transaction).
// UoW callers obtain a sink-aware writer separately via
// newMicroflowWriterWithSink.
type microflowRepo struct {
	w   *mmpr.Writer
	r   *mmpr.Reader
	dec *decoder
	*microflowWriter
}

// NewMicroflowRepository constructs the direct-mode repository.
func NewMicroflowRepository(w *mmpr.Writer) repos.MicroflowRepository {
	enc := newEncoder()
	return &microflowRepo{
		w:               w,
		r:               w.ConcreteReader(),
		dec:             newDecoder(),
		microflowWriter: newMicroflowWriterWithSink(newWriterSink(w), enc),
	}
}

func (r *microflowRepo) Get(id model.ID) (*genMf.Microflow, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("microflow not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode microflow %s: %w", id, err)
	}
	mf, ok := elem.(*genMf.Microflow)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Microflow (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return mf, nil
}

// List returns all microflows whose unit chains up to moduleID. Folder
// nesting is supported via mmpr.ResolveModuleName + BuildContainerParent.
func (r *microflowRepo) List(moduleID model.ID) ([]*genMf.Microflow, error) {
	all, err := r.ListAll()
	if err != nil {
		return nil, err
	}
	if moduleID == "" {
		return all, nil
	}
	mods, err := r.r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string, len(mods))
	wantName := ""
	for _, m := range mods {
		moduleMap[m.ID] = m.Name
		if m.ID == string(moduleID) {
			wantName = m.Name
		}
	}
	if wantName == "" {
		return nil, fmt.Errorf("module not found: %s", moduleID)
	}
	parents, err := r.r.BuildContainerParent()
	if err != nil {
		return nil, err
	}
	refs, err := r.r.ListUnitsByType(microflowTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genMf.Microflow, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != microflowTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
			continue
		}
		for _, mf := range all {
			if string(mf.ID()) == ref.ID {
				result = append(result, mf)
				break
			}
		}
	}
	return result, nil
}

func (r *microflowRepo) ListAll() ([]*genMf.Microflow, error) {
	refs, err := r.r.ListUnitsByType(microflowTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genMf.Microflow, 0, len(refs))
	for _, ref := range refs {
		// ListUnitsByType is prefix-matched; filter to exact $Type to
		// avoid pulling in similarly-prefixed unit kinds.
		if ref.Type != microflowTypeName {
			continue
		}
		mf, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode microflow %s: %w", ref.ID, err)
		}
		result = append(result, mf)
	}
	return result, nil
}

// FindByQualifiedName parses "Module.Name" and returns the matching
// microflow (or nil, error).
func (r *microflowRepo) FindByQualifiedName(qn string) (*genMf.Microflow, error) {
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
	refs, err := r.r.ListUnitsByType(microflowTypeName)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if ref.Type != microflowTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		mf, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, err
		}
		if mf.Name() == simpleName {
			return mf, nil
		}
	}
	return nil, nil
}

// GetContainerUUID returns the parent container UUID for a microflow
// unit by querying the MPR Unit table directly. Codec-decoded gen
// objects shed their container linkage during BSON roundtrip, so module
// resolution callers retrieve the parent ID from SQLite rather than
// walking Container().
func (r *microflowRepo) GetContainerUUID(id model.ID) (model.ID, error) {
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

// IsRule reports whether the qualified name refers to a Microflows$Rule
// (a microflow stored under the Rule subtype). Ports the legacy
// sdk/mpr.Reader.IsRule shape onto modelsdk/mpr APIs.
func (r *microflowRepo) IsRule(qn string) (bool, error) {
	if qn == "" {
		return false, nil
	}
	moduleName, simpleName, ok := splitQN(qn)
	if !ok {
		return false, nil
	}
	refs, err := r.r.ListUnitsByType(ruleTypeName)
	if err != nil {
		return false, err
	}
	if len(refs) == 0 {
		return false, nil
	}
	mods, err := r.r.ListModules()
	if err != nil {
		return false, err
	}
	moduleMap := make(map[string]string, len(mods))
	for _, m := range mods {
		moduleMap[m.ID] = m.Name
	}
	parents, err := r.r.BuildContainerParent()
	if err != nil {
		return false, err
	}
	for _, ref := range refs {
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		var raw map[string]any
		bts, err := r.r.GetRawUnitBytes(ref.ID)
		if err != nil {
			continue
		}
		if err := bson.Unmarshal(bts, &raw); err != nil {
			continue
		}
		if name, _ := raw["Name"].(string); name == simpleName {
			return true, nil
		}
	}
	return false, nil
}

// splitQN splits "Module.Name" into ("Module", "Name", true).
// "Module.Sub.Name" is treated as ("Module", "Sub.Name", true) to match
// the legacy single-dot semantics.
func splitQN(qn string) (module, name string, ok bool) {
	idx := strings.Index(qn, ".")
	if idx <= 0 || idx == len(qn)-1 {
		return "", "", false
	}
	return qn[:idx], qn[idx+1:], true
}
