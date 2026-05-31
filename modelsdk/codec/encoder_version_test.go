package codec_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/property"
	"github.com/mendixlabs/mxcli/modelsdk/version"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// versionStub is a synthetic element type used only in version-encoder tests.
// It implements version.PropertyVersioner directly via a switch — no global registry,
// no heap allocation, no GC pressure on benchmarks.
type versionStub struct {
	element.Base
	vGatedProp *property.Primitive[string]
}

func (s *versionStub) PropertyVersionInfo(camelName string) (version.PropertyVersionInfo, bool) {
	switch camelName {
	case "vGatedProp":
		return version.PropertyVersionInfo{Introduced: "10.14.0"}, true
	default:
		return version.PropertyVersionInfo{}, false
	}
}

// newVersionStubElement returns a new versionStub (raw == nil) with a dirty
// "VGatedProp" primitive property.
func newVersionStubElement() *versionStub {
	s := &versionStub{}
	s.SetTypeName("TestEncoder$VersionStub")
	s.SetID("test-stub-id")
	s.vGatedProp = property.NewPrimitive[string]("VGatedProp", property.DecodeString)
	s.vGatedProp.Bind(&s.Base, 0)
	s.vGatedProp.Set("gated-value")
	s.SetProperties([]element.Property{s.vGatedProp})
	return s
}

func TestEncoder_SkipsPropertyNotYetIntroduced(t *testing.T) {
	enc := &codec.Encoder{Version: version.Parse("10.13.0")}
	data, err := enc.Encode(newVersionStubElement())
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["VGatedProp"]; ok {
		t.Error("VGatedProp should be absent for project version 10.13.0")
	}
}

func TestEncoder_EmitsPropertyWhenVersionSufficient(t *testing.T) {
	enc := &codec.Encoder{Version: version.Parse("10.14.0")}
	data, err := enc.Encode(newVersionStubElement())
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["VGatedProp"]; !ok {
		t.Error("VGatedProp should be present for project version 10.14.0")
	}
}

func TestEncoder_NoVersionGating_EmitsAll(t *testing.T) {
	enc := &codec.Encoder{} // zero version = no gating
	data, err := enc.Encode(newVersionStubElement())
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["VGatedProp"]; !ok {
		t.Error("VGatedProp should be present with zero-version encoder")
	}
}
