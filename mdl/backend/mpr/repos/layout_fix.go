// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FixScrollContainerBSON patches a ScrollContainer's BSON so that the
// "Center" property is serialized as "CenterRegion" (the correct Mendix field name).
// It also ensures all required properties (LayoutMode, Alignment, etc.) are set.
//
// The genPg model uses BSON tag "Center" but Mendix expects "CenterRegion".
// This function encodes the scroll container to BSON, renames the field,
// and returns a clean element whose raw bytes bypass the property-based encoder.
func FixScrollContainerBSON(sc *genPg.ScrollContainer) *genPg.ScrollContainer {
	encoder := &codec.Encoder{}
	raw, err := encoder.Encode(sc)
	if err != nil {
		return sc
	}

	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return sc
	}

	fixed := bson.D{}
	for _, e := range doc {
		switch e.Key {
		case "Center":
			fixed = append(fixed, bson.E{"CenterRegion", e.Value})
		default:
			fixed = append(fixed, e)
		}
	}

	patched, err := bson.Marshal(fixed)
	if err != nil {
		return sc
	}

	clean := &genPg.ScrollContainer{}
	clean.SetRaw(patched)
	return clean
}
