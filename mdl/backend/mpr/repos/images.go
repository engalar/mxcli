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
	genIm "github.com/mendixlabs/mxcli/modelsdk/gen/images"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const imageCollectionTypeName = "Images$ImageCollection"

// imagesRepo is the direct-mode ImageRepository.
//
// The fixture probe shows Mendix stores ImageCollection as a unit but
// individual Image entries live as embedded children inside each
// collection (Images$Image has 0 exact unit matches; the prefix=7
// figure was just the collection prefix overlap). GetImage therefore
// walks every collection's ImagesItems() looking for the matching ID.
type imagesRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder
	sink writeSink
}

// NewImageRepository constructs the direct-mode repository.
func NewImageRepository(w *mmpr.Writer) repos.ImageRepository {
	return &imagesRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),
		sink: newWriterSink(w),
	}
}

// GetImage finds an embedded Image by ID by scanning every
// ImageCollection. Returns an error if no match.
func (r *imagesRepo) GetImage(id model.ID) (*genIm.Image, error) {
	cols, err := r.ListCollections("")
	if err != nil {
		return nil, err
	}
	for _, col := range cols {
		for _, child := range col.ImagesItems() {
			if child == nil {
				continue
			}
			if child.ID() == element.ID(string(id)) {
				img, ok := child.(*genIm.Image)
				if !ok {
					return nil, fmt.Errorf("image child %s is not *Image (got %T)", id, child)
				}
				return img, nil
			}
		}
	}
	return nil, fmt.Errorf("image not found: %s", id)
}

func (r *imagesRepo) GetCollection(id model.ID) (*genIm.ImageCollection, error) {
	bytes, err := r.r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("image collection not found: %s", id)
	}
	elem, err := r.dec.Decode(bytes)
	if err != nil {
		return nil, fmt.Errorf("decode image collection %s: %w", id, err)
	}
	col, ok := elem.(*genIm.ImageCollection)
	if !ok {
		return nil, fmt.Errorf("unit %s is not an ImageCollection (got %T, type=%q)", id, elem, elem.TypeName())
	}
	return col, nil
}

func (r *imagesRepo) ListCollections(moduleID model.ID) ([]*genIm.ImageCollection, error) {
	refs, err := r.r.ListUnitsByType(imageCollectionTypeName)
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
	result := make([]*genIm.ImageCollection, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != imageCollectionTypeName {
			continue
		}
		if moduleID != "" {
			if mmpr.ResolveModuleName(ref.ContainerID, moduleMap, parents) != wantName {
				continue
			}
		}
		col, err := r.GetCollection(model.ID(ref.ID))
		if err != nil {
			return nil, fmt.Errorf("decode image collection %s: %w", ref.ID, err)
		}
		result = append(result, col)
	}
	return result, nil
}

func (r *imagesRepo) CreateCollection(parentUUID string, containmentName string, c *genIm.ImageCollection) error {
	if c.ID() == "" {
		c.SetID(element.ID(mmpr.GenerateID()))
	}
	if c.TypeName() == "" {
		c.SetTypeName(imageCollectionTypeName)
	}
	contents, err := r.enc.Encode(c)
	if err != nil {
		return err
	}
	return r.sink.InsertUnit(string(c.ID()), parentUUID, containmentName, c.TypeName(), contents)
}

func (r *imagesRepo) UpdateCollection(c *genIm.ImageCollection) error {
	contents, err := r.enc.Encode(c)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(c.ID()), contents)
}

func (r *imagesRepo) DeleteCollection(id model.ID) error {
	return r.sink.DeleteUnit(string(id))
}

var _ repos.ImageRepository = (*imagesRepo)(nil)
