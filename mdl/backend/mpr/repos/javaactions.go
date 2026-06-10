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
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const (
	javaActionTypeName       = "JavaActions$JavaAction"
	javaScriptActionTypeName = "JavaScriptActions$JavaScriptAction"
)

// javaActionRepo is the direct-mode JavaActionRepository: reads via the
// concrete *mmpr.Reader, writes via a writeSink (no transaction).
type javaActionRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewJavaActionRepository constructs the direct-mode repository.
func NewJavaActionRepository(w *mmpr.Writer) repos.JavaActionRepository {
	return &javaActionRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *javaActionRepo) Get(id model.ID) (*genJA.JavaAction, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("java action not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode java action %s: %w", id, err)
	}
	ja, ok := elem.(*genJA.JavaAction)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a JavaAction (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return ja, nil
}

func (r *javaActionRepo) List(moduleID model.ID) ([]*genJA.JavaAction, error) {
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
	refs, err := r.r.ListUnitsByType(javaActionTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genJA.JavaAction, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != javaActionTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
			continue
		}
		for _, ja := range all {
			if string(ja.ID()) == ref.ID {
				result = append(result, ja)
				break
			}
		}
	}
	return result, nil
}

func (r *javaActionRepo) ListAll() ([]*genJA.JavaAction, error) {
	refs, err := r.r.ListUnitsByType(javaActionTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genJA.JavaAction, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != javaActionTypeName {
			continue
		}
		ja, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode java action %s: %w", ref.ID, err)
		}
		result = append(result, ja)
	}
	return result, nil
}

func (r *javaActionRepo) FindByQualifiedName(qn string) (*genJA.JavaAction, error) {
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
	refs, err := r.r.ListUnitsByType(javaActionTypeName)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if ref.Type != javaActionTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		ja, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, err
		}
		if ja.Name() == simpleName {
			return ja, nil
		}
	}
	return nil, nil
}

func (r *javaActionRepo) GetContainerUUID(id model.ID) (model.ID, error) {
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

func (r *javaActionRepo) Create(parentUUID, containmentName string, ja *genJA.JavaAction) error {
	if ja == nil {
		return fmt.Errorf("javaActionRepo.Create: nil JavaAction")
	}
	if ja.ID() == "" {
		ja.SetID(element.ID(mmpr.GenerateID()))
	}
	if ja.TypeName() == "" {
		ja.SetTypeName(javaActionTypeName)
	}
	contents, err := r.enc.Encode(ja)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(ja.ID()), parentUUID, containmentName, ja.TypeName(), contents)
}

func (r *javaActionRepo) Update(ja *genJA.JavaAction) error {
	if ja == nil {
		return fmt.Errorf("javaActionRepo.Update: nil JavaAction")
	}
	contents, err := r.enc.Encode(ja)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(ja.ID()), contents)
}

func (r *javaActionRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

var _ repos.JavaActionRepository = (*javaActionRepo)(nil)

// javaScriptActionRepo is the direct-mode read-only JavaScriptActionRepository.
// JavaScript actions have no MDL `create` surface, so there is no writer half.
type javaScriptActionRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewJavaScriptActionRepository constructs the direct-mode repository.
func NewJavaScriptActionRepository(w *mmpr.Writer) repos.JavaScriptActionRepository {
	return &javaScriptActionRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *javaScriptActionRepo) Update(jsa *genJSA.JavaScriptAction) error {
	if jsa == nil {
		return fmt.Errorf("javaScriptActionRepo.Update: nil JavaScriptAction")
	}
	contents, err := r.enc.Encode(jsa)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(jsa.ID()), contents)
}

func (r *javaScriptActionRepo) Get(id model.ID) (*genJSA.JavaScriptAction, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("javascript action not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode javascript action %s: %w", id, err)
	}
	jsa, ok := elem.(*genJSA.JavaScriptAction)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a JavaScriptAction (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return jsa, nil
}

func (r *javaScriptActionRepo) List(moduleID model.ID) ([]*genJSA.JavaScriptAction, error) {
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
	refs, err := r.r.ListUnitsByType(javaScriptActionTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genJSA.JavaScriptAction, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != javaScriptActionTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
			continue
		}
		for _, jsa := range all {
			if string(jsa.ID()) == ref.ID {
				result = append(result, jsa)
				break
			}
		}
	}
	return result, nil
}

func (r *javaScriptActionRepo) ListAll() ([]*genJSA.JavaScriptAction, error) {
	refs, err := r.r.ListUnitsByType(javaScriptActionTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genJSA.JavaScriptAction, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != javaScriptActionTypeName {
			continue
		}
		jsa, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode javascript action %s: %w", ref.ID, err)
		}
		result = append(result, jsa)
	}
	return result, nil
}

func (r *javaScriptActionRepo) FindByQualifiedName(qn string) (*genJSA.JavaScriptAction, error) {
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
	refs, err := r.r.ListUnitsByType(javaScriptActionTypeName)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if ref.Type != javaScriptActionTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		jsa, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, err
		}
		if jsa.Name() == simpleName {
			return jsa, nil
		}
	}
	return nil, nil
}

func (r *javaScriptActionRepo) GetContainerUUID(id model.ID) (model.ID, error) {
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

var _ repos.JavaScriptActionRepository = (*javaScriptActionRepo)(nil)
