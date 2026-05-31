package codec_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"github.com/mendixlabs/mxcli/modelsdk/version"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestEncoder_SkipsPropertyNotYetIntroduced(t *testing.T) {
	// boundaryEvents introduced in 10.14.0; project is 10.13.0 → must be absent
	act := genWf.NewCallMicroflowTask()
	act.SetID("test-id")
	act.SetName("MyMF")
	be := genWf.NewTimerBoundaryEvent()
	be.SetID("be-id")
	act.AddBoundaryEvents(be) // mark dirty

	enc := &codec.Encoder{Version: version.Parse("10.13.0")}
	data, err := enc.Encode(act)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["BoundaryEvents"]; ok {
		t.Error("BoundaryEvents should be absent for project version 10.13.0")
	}
}

func TestEncoder_EmitsPropertyWhenVersionSufficient(t *testing.T) {
	act := genWf.NewCallMicroflowTask()
	act.SetID("test-id")
	act.SetName("MyMF")
	be := genWf.NewTimerBoundaryEvent()
	be.SetID("be-id")
	act.AddBoundaryEvents(be)

	enc := &codec.Encoder{Version: version.Parse("10.14.0")}
	data, err := enc.Encode(act)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["BoundaryEvents"]; !ok {
		t.Error("BoundaryEvents should be present for project version 10.14.0")
	}
}

func TestEncoder_NoVersionGating_EmitsAll(t *testing.T) {
	// Zero version = no gating; all dirty properties emitted
	act := genWf.NewCallMicroflowTask()
	act.SetID("test-id")
	act.SetName("MyMF")
	be := genWf.NewTimerBoundaryEvent()
	be.SetID("be-id")
	act.AddBoundaryEvents(be)

	enc := &codec.Encoder{} // zero version
	data, err := enc.Encode(act)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["BoundaryEvents"]; !ok {
		t.Error("BoundaryEvents should be present with zero-version encoder")
	}
}
