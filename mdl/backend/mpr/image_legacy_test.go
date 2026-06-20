package mprbackend

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// TestParseImageCollectionRaw_RoundTrip guards against the read-path bug where
// nested BSON documents decode as bson.D (not map[string]any) under mongo-driver
// v2, causing every image to be silently skipped during parsing. The symptom is
// ALTER ADD reporting success while describe shows zero images.
func TestParseImageCollectionRaw_RoundTrip(t *testing.T) {
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

	got, err := parseImageCollectionRaw("unit-1", "container-1", contents)
	if err != nil {
		t.Fatalf("parse: %v", err)
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
