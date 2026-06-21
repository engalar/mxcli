package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/gen/images"
)

func (b *MprBackend) listImageCollectionsViaModelsdk() ([]*types.ImageCollection, error) {
	rawUnits, err := b.msdkReader.ListRawUnitsByType("Images$ImageCollection")
	if err != nil {
		return nil, err
	}
	decoder := codec.NewDecoder(codec.DefaultRegistry)
	out := make([]*types.ImageCollection, 0, len(rawUnits))
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		ic, err := decodeImageCollection(decoder, ru)
		if err != nil {
			return nil, err
		}
		out = append(out, ic)
	}
	return out, nil
}

func decodeImageCollection(decoder *codec.Decoder, ru *types.RawUnit) (*types.ImageCollection, error) {
	elem, err := decoder.Decode(bson.Raw(ru.Contents))
	if err != nil {
		return nil, fmt.Errorf("decode image collection %s: %w", ru.ID, err)
	}
	ic, ok := elem.(*images.ImageCollection)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T for image collection %s", elem, ru.ID)
	}
	return convertImageCollection(ic, string(ru.ID), string(ru.ContainerID)), nil
}

func convertImageCollection(ic *images.ImageCollection, unitID, containerID string) *types.ImageCollection {
	mdl := &types.ImageCollection{
		BaseElement: model.BaseElement{
			ID:       model.ID(unitID),
			TypeName: "Images$ImageCollection",
		},
		ContainerID:   model.ID(containerID),
		Name:          ic.Name(),
		Documentation: ic.Documentation(),
		ExportLevel:   ic.ExportLevel(),
	}
	for _, child := range ic.ImagesItems() {
		img, ok := child.(*images.Image)
		if !ok {
			continue
		}
		mdl.Images = append(mdl.Images, types.Image{
			ID:     model.ID(img.ID()),
			Name:   img.Name(),
			Format: img.ImageFormat(),
			Data:   img.ImageData(),
		})
	}
	return mdl
}
