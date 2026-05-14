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
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// snippetTypeName is the BSON $Type Mendix uses for snippets. Note
// the stub doc-comment in mdl/repos/snippets.go references
// "Pages$Snippet" (stale); the actual storage name is "Forms$Snippet"
// (legacy "Form" terminology, matching Page / Layout).
const snippetTypeName = "Forms$Snippet"

// snippetRepo is the direct-mode SnippetRepository.
type snippetRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewSnippetRepository constructs the direct-mode repository.
func NewSnippetRepository(w *mmpr.Writer) repos.SnippetRepository {
	return &snippetRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

func (r *snippetRepo) Get(id model.ID) (*genPg.Snippet, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("snippet not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode snippet %s: %w", id, err)
	}
	s, ok := elem.(*genPg.Snippet)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Snippet (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return s, nil
}

// List returns all snippets whose unit chains up to moduleID. When
// moduleID is empty, returns every snippet in the project.
func (r *snippetRepo) List(moduleID model.ID) ([]*genPg.Snippet, error) {
	refs, err := r.r.ListUnitsByType(snippetTypeName)
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
	result := make([]*genPg.Snippet, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != snippetTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		s, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode snippet %s: %w", ref.ID, err)
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *snippetRepo) ListAll() ([]*genPg.Snippet, error) {
	refs, err := r.r.ListUnitsByType(snippetTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genPg.Snippet, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != snippetTypeName {
			continue
		}
		s, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode snippet %s: %w", ref.ID, err)
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *snippetRepo) FindByQualifiedName(qn string) (*genPg.Snippet, error) {
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
	refs, err := r.r.ListUnitsByType(snippetTypeName)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if ref.Type != snippetTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		s, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, err
		}
		if s.Name() == simpleName {
			return s, nil
		}
	}
	return nil, nil
}

func (r *snippetRepo) GetContainerUUID(id model.ID) (model.ID, error) {
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

func (r *snippetRepo) Create(parentUUID string, containmentName string, s *genPg.Snippet) error {
	if s.ID() == "" {
		s.SetID(element.ID(mmpr.GenerateID()))
	}
	if s.TypeName() == "" {
		s.SetTypeName(snippetTypeName)
	}
	contents, err := r.enc.Encode(s)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(s.ID()), parentUUID, containmentName, s.TypeName(), contents)
}

func (r *snippetRepo) Update(s *genPg.Snippet) error {
	contents, err := r.enc.Encode(s)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(s.ID()), contents)
}

func (r *snippetRepo) Delete(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

func (r *snippetRepo) Move(id model.ID, newParentUUID string) error {
	return r.sink.UpdateUnitContainer(string(id), newParentUUID)
}

var _ repos.SnippetRepository = (*snippetRepo)(nil)
