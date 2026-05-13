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
	genPr "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const folderTypeName = "Projects$Folder"

// folderRepo is the direct-mode FolderRepository.
type folderRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewFolderRepository constructs the direct-mode repository.
func NewFolderRepository(w *mmpr.Writer) repos.FolderRepository {
	return &folderRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *folderRepo) Get(id model.ID) (*genPr.Folder, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("folder not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode folder %s: %w", id, err)
	}
	f, ok := elem.(*genPr.Folder)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Folder (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return f, nil
}

func (r *folderRepo) List(moduleID model.ID) ([]*genPr.Folder, error) {
	refs, err := r.r.ListUnitsByType(folderTypeName)
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
	result := make([]*genPr.Folder, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != folderTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		f, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode folder %s: %w", ref.ID, err)
		}
		result = append(result, f)
	}
	return result, nil
}

func (r *folderRepo) Create(parentUUID string, containmentName string, f *genPr.Folder) error {
	if f.ID() == "" {
		f.SetID(element.ID(mmpr.GenerateID()))
	}
	if f.TypeName() == "" {
		f.SetTypeName(folderTypeName)
	}
	contents, err := r.enc.Encode(f)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(f.ID()), parentUUID, containmentName, f.TypeName(), contents)
}

func (r *folderRepo) Update(f *genPr.Folder) error {
	contents, err := r.enc.Encode(f)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(f.ID()), contents)
}

func (r *folderRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *folderRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.FolderRepository = (*folderRepo)(nil)
