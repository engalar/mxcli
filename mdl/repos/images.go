// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genIm "github.com/mendixlabs/mxcli/modelsdk/gen/images"
)

// ImageReader / ImageWriter / ImageRepository — signatures intentionally
// minimal until Stage 3 cutover. Images live inside ImageCollections;
// both sides are exposed for read, both are mutable.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// and produce an MPR implementation.
type ImageReader interface {
	GetImage(id model.ID) (*genIm.Image, error)
	GetCollection(id model.ID) (*genIm.ImageCollection, error)
	ListCollections(moduleID model.ID) ([]*genIm.ImageCollection, error)
}

type ImageWriter interface {
	CreateCollection(parentUUID string, containmentName string, c *genIm.ImageCollection) error
	UpdateCollection(c *genIm.ImageCollection) error
	DeleteCollection(id model.ID) error
}

type ImageRepository interface {
	ImageReader
	ImageWriter
}
