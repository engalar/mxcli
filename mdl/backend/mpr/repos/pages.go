// SPDX-License-Identifier: Apache-2.0

// All write paths rely on mmpr.Writer's automatic cache invalidation
// (addendum Blocker 4). Repos do NOT call ReaderCache.Invalidate
// after each write.

package mprrepos

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// pageTypeName is the BSON $Type prefix Mendix uses for pages
// ("Forms$Page" — legacy "Form" terminology). Matches sdk/mpr/writer_pages.go.
const pageTypeName = "Forms$Page"

// pageRepo is the direct-mode PageRepository: reads via the concrete
// *mmpr.Reader, writes via a writerSink (no transaction). UoW callers
// obtain a sink-aware writer separately via newPageWriterWithSink.
type pageRepo struct {
	w   *mmpr.Writer
	r   *mmpr.Reader
	dec *decoder
	*pageWriter
}

// NewPageRepository constructs the direct-mode repository.
func NewPageRepository(w *mmpr.Writer) repos.PageRepository {
	enc := newEncoder()
	return &pageRepo{
		w:          w,
		r:          w.ConcreteReader(),
		dec:        newDecoder(),
		pageWriter: newPageWriterWithSink(newWriterSink(w), enc),
	}
}

func (r *pageRepo) Get(id model.ID) (*genPg.Page, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("page not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode page %s: %w", id, err)
	}
	page, ok := elem.(*genPg.Page)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Page (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return page, nil
}

func (r *pageRepo) List(moduleID model.ID) ([]*genPg.Page, error) {
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
	refs, err := r.r.ListUnitsByType(pageTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genPg.Page, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != pageTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
			continue
		}
		for _, p := range all {
			if string(p.ID()) == ref.ID {
				result = append(result, p)
				break
			}
		}
	}
	return result, nil
}

func (r *pageRepo) ListAll() ([]*genPg.Page, error) {
	refs, err := r.r.ListUnitsByType(pageTypeName)
	if err != nil {
		return nil, err
	}
	result := make([]*genPg.Page, 0, len(refs))
	for _, ref := range refs {
		// ListUnitsByType is prefix-matched; Forms$Page is also a prefix
		// of Forms$PageTemplate. Filter to exact match.
		if ref.Type != pageTypeName {
			continue
		}
		page, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode page %s: %w", ref.ID, err)
		}
		result = append(result, page)
	}
	return result, nil
}

func (r *pageRepo) FindByQualifiedName(qn string) (*genPg.Page, error) {
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
	refs, err := r.r.ListUnitsByType(pageTypeName)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if ref.Type != pageTypeName {
			continue
		}
		if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != moduleName {
			continue
		}
		page, err := r.Get(model.ID(ref.ID))
		if err != nil {
			return nil, err
		}
		if page.Name() == simpleName {
			return page, nil
		}
	}
	return nil, nil
}

func (r *pageRepo) GetContainerUUID(id model.ID) (model.ID, error) {
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

// OpenForMutation opens a Page for incremental editing. Stage 2 uses
// Option A (decode-edit-encode); Commit re-encodes and writes via
// repo.Update. Stage 2.5 may swap for direct raw-BSON edits if
// throughput on > 100 KB pages proves insufficient.
func (r *pageRepo) OpenForMutation(pageID model.ID) (repos.PageMutator, error) {
	page, err := r.Get(pageID)
	if err != nil {
		return nil, fmt.Errorf("OpenForMutation(%s): %w", pageID, err)
	}
	return &pageMutator{repo: r, page: page}, nil
}
