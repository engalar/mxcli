package mprbackend

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// TestParseImageCollectionViaModelsdk_RoundTrip guards against the read-path bug
// where the codec decoder correctly populates Image (BinaryPrimitive) and
// Images (PartList) from raw BSON produced by SerializeImageCollection.
func TestParseImageCollectionViaModelsdk_RoundTrip(t *testing.T) {
	src := &types.ImageCollection{
		Name:        "Icons",
		ExportLevel: "Hidden",
		Images: []types.Image{
			{Name: "logo", Format: "Png", Data: []byte("PNGDATA")},
			{Name: "banner", Format: "Svg", Data: []byte("<svg/>")},
		},
	}
	contents, err := modelsdkmpr.SerializeImageCollection(src)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	decoder := codec.NewDecoder(codec.DefaultRegistry)
	got, err := decodeImageCollection(decoder, &types.RawUnit{
		ID:          "unit-1",
		ContainerID: "container-1",
		Contents:    contents,
	})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(got.Images) != 2 {
		t.Fatalf("expected 2 images after round-trip, got %d: %+v", len(got.Images), got.Images)
	}
	if got.Images[0].Name != "logo" || string(got.Images[0].Data) != "PNGDATA" {
		t.Errorf("image[0] = %+v, want name=logo data=PNGDATA", got.Images[0])
	}
	if got.Images[1].Name != "banner" || got.Images[1].Format != "Svg" {
		t.Errorf("image[1] = %+v, want name=banner format=Svg", got.Images[1])
	}
}
