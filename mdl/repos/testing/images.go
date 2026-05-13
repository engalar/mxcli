// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genIm "github.com/mendixlabs/mxcli/modelsdk/gen/images"
)

// ImageCollectionCreateCall captures arguments to CreateCollection.
type ImageCollectionCreateCall struct {
	ParentUUID      string
	ContainmentName string
	Collection      *genIm.ImageCollection
}

// RecordingImageRepository records every call to its methods.
type RecordingImageRepository struct {
	GotImageIDs        []model.ID
	GotCollectionIDs   []model.ID
	ListedModule       []model.ID
	CreatedCollections []ImageCollectionCreateCall
	UpdatedCollections []*genIm.ImageCollection
	DeletedCollections []model.ID

	GetImageFunc         func(model.ID) (*genIm.Image, error)
	GetCollectionFunc    func(model.ID) (*genIm.ImageCollection, error)
	ListCollectionsFunc  func(model.ID) ([]*genIm.ImageCollection, error)
	CreateCollectionFunc func(ImageCollectionCreateCall) error
	UpdateCollectionFunc func(*genIm.ImageCollection) error
	DeleteCollectionFunc func(model.ID) error
}

var _ repos.ImageRepository = (*RecordingImageRepository)(nil)

func (m *RecordingImageRepository) GetImage(id model.ID) (*genIm.Image, error) {
	m.GotImageIDs = append(m.GotImageIDs, id)
	if m.GetImageFunc != nil {
		return m.GetImageFunc(id)
	}
	return nil, nil
}

func (m *RecordingImageRepository) GetCollection(id model.ID) (*genIm.ImageCollection, error) {
	m.GotCollectionIDs = append(m.GotCollectionIDs, id)
	if m.GetCollectionFunc != nil {
		return m.GetCollectionFunc(id)
	}
	return nil, nil
}

func (m *RecordingImageRepository) ListCollections(moduleID model.ID) ([]*genIm.ImageCollection, error) {
	m.ListedModule = append(m.ListedModule, moduleID)
	if m.ListCollectionsFunc != nil {
		return m.ListCollectionsFunc(moduleID)
	}
	return nil, nil
}

func (m *RecordingImageRepository) CreateCollection(parentUUID string, containmentName string, c *genIm.ImageCollection) error {
	call := ImageCollectionCreateCall{ParentUUID: parentUUID, ContainmentName: containmentName, Collection: c}
	m.CreatedCollections = append(m.CreatedCollections, call)
	if m.CreateCollectionFunc != nil {
		return m.CreateCollectionFunc(call)
	}
	return nil
}

func (m *RecordingImageRepository) UpdateCollection(c *genIm.ImageCollection) error {
	m.UpdatedCollections = append(m.UpdatedCollections, c)
	if m.UpdateCollectionFunc != nil {
		return m.UpdateCollectionFunc(c)
	}
	return nil
}

func (m *RecordingImageRepository) DeleteCollection(id model.ID) error {
	m.DeletedCollections = append(m.DeletedCollections, id)
	if m.DeleteCollectionFunc != nil {
		return m.DeleteCollectionFunc(id)
	}
	return nil
}
