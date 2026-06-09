// SPDX-License-Identifier: Apache-2.0

// image_compat.go — Image collection BSON parsing.
//
// gen/images.ImageCollection exposes typed accessors but its Image child stores
// the binary payload under a "ImageData" string property which does not align
// with the actual Mendix BSON key "Image" (a binary blob). Until the codegen
// gains BSON-binary-aware properties, we parse Image collections directly
// from the raw bytes returned by modelsdk/mpr.Reader.

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

const imageCollectionBsonType = "Images$ImageCollection"

func (b *MprBackend) listImageCollectionsFromRaw() ([]*types.ImageCollection, error) {
	rawUnits, err := b.msdkReader.ListRawUnitsByType(imageCollectionBsonType)
	if err != nil {
		return nil, err
	}
	out := make([]*types.ImageCollection, 0, len(rawUnits))
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		ic, err := parseImageCollectionRaw(string(ru.ID), string(ru.ContainerID), ru.Contents)
		if err != nil {
			return nil, fmt.Errorf("parse image collection %s: %w", ru.ID, err)
		}
		out = append(out, ic)
	}
	return out, nil
}

// parseImageCollectionRaw decodes a Images$ImageCollection BSON document into a
// types.ImageCollection. Mirrors the logic in the retired sdk/mpr parser.
func parseImageCollectionRaw(unitID, containerID string, contents []byte) (*types.ImageCollection, error) {
	if len(contents) < 4 {
		return nil, fmt.Errorf("contents too short (%d bytes) for unit %s", len(contents), unitID)
	}
	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}

	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{
			ID:       model.ID(unitID),
			TypeName: imageCollectionBsonType,
		},
		ContainerID: model.ID(containerID),
	}
	if name, ok := raw["Name"].(string); ok {
		ic.Name = name
	}
	if doc, ok := raw["Documentation"].(string); ok {
		ic.Documentation = doc
	}
	if exp, ok := raw["ExportLevel"].(string); ok {
		ic.ExportLevel = exp
	}

	images, ok := raw["Images"].(bson.A)
	if !ok {
		return ic, nil
	}
	for _, item := range images {
		imgMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		image := types.Image{}
		if id := extractBsonID(imgMap["$ID"]); id != "" {
			image.ID = model.ID(id)
		}
		if name, ok := imgMap["Name"].(string); ok {
			image.Name = name
		}
		if format, ok := imgMap["ImageFormat"].(string); ok {
			image.Format = format
		}
		switch data := imgMap["Image"].(type) {
		case bson.Binary:
			image.Data = data.Data
		case []byte:
			image.Data = data
		}
		ic.Images = append(ic.Images, image)
	}
	return ic, nil
}
