// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/microflows"
	"github.com/mendixlabs/mxcli/sdk/mpr/version"

	"go.mongodb.org/mongo-driver/bson"
)

// bsonHasKey returns true when the top-level BSON document contains the key.
func bsonHasKey(doc bson.D, key string) bool {
	for _, e := range doc {
		if e.Key == key {
			return true
		}
	}
	return false
}

// bsonGetKey returns the value of a key or nil if absent.
func bsonGetKey(doc bson.D, key string) any {
	for _, e := range doc {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

func TestSerializeSequenceFlow_Mx9_UsesLegacyShape(t *testing.T) {
	flow := &microflows.SequenceFlow{
		BaseElement:   model.BaseElement{ID: "flow-1"},
		OriginID:      "orig-1",
		DestinationID: "dest-1",
		CaseValue:     &microflows.NoCase{BaseElement: model.BaseElement{ID: "case-1"}},
	}

	doc := serializeSequenceFlow(flow, 9)

	if !bsonHasKey(doc, "NewCaseValue") {
		t.Error("Mx 9 sequence flow must include NewCaseValue")
	}
	if bsonHasKey(doc, "CaseValues") {
		t.Error("Mx 9 sequence flow must NOT include CaseValues")
	}
	if !bsonHasKey(doc, "OriginBezierVector") || !bsonHasKey(doc, "DestinationBezierVector") {
		t.Error("Mx 9 sequence flow must include top-level {Origin,Destination}BezierVector")
	}
	if bsonHasKey(doc, "Line") {
		t.Error("Mx 9 sequence flow must NOT nest vectors under Line")
	}
}

func TestSerializeSequenceFlow_Mx10_UsesModernShape(t *testing.T) {
	flow := &microflows.SequenceFlow{
		BaseElement:   model.BaseElement{ID: "flow-1"},
		OriginID:      "orig-1",
		DestinationID: "dest-1",
		CaseValue:     &microflows.NoCase{BaseElement: model.BaseElement{ID: "case-1"}},
	}

	doc := serializeSequenceFlow(flow, 10)

	if !bsonHasKey(doc, "CaseValues") {
		t.Error("Mx 10 sequence flow must include CaseValues")
	}
	if bsonHasKey(doc, "NewCaseValue") {
		t.Error("Mx 10 sequence flow must NOT include legacy NewCaseValue")
	}
	if !bsonHasKey(doc, "Line") {
		t.Error("Mx 10 sequence flow must nest vectors under Line")
	}
	if bsonHasKey(doc, "OriginBezierVector") || bsonHasKey(doc, "DestinationBezierVector") {
		t.Error("Mx 10 sequence flow must NOT include top-level BezierVector fields")
	}
}

func TestSerializeEndEvent_EmptyReturnValueHasNoTrailingLineBreak(t *testing.T) {
	end := &microflows.EndEvent{
		BaseMicroflowObject: microflows.BaseMicroflowObject{
			BaseElement: model.BaseElement{ID: "end-empty"},
			Position:    model.Point{X: 10, Y: 20},
			Size:        model.Size{Width: 20, Height: 20},
		},
		ReturnValue: "",
	}

	doc := serializeMicroflowObject(end)
	if got := bsonGetKey(doc, "ReturnValue"); got != "" {
		t.Fatalf("ReturnValue = %q, want empty string", got)
	}
}

func TestSerializeEndEvent_NonEmptyReturnValueHasNoSyntheticLineBreak(t *testing.T) {
	end := &microflows.EndEvent{
		BaseMicroflowObject: microflows.BaseMicroflowObject{
			BaseElement: model.BaseElement{ID: "end-result"},
			Position:    model.Point{X: 10, Y: 20},
			Size:        model.Size{Width: 20, Height: 20},
		},
		ReturnValue: "$Result",
	}

	doc := serializeMicroflowObject(end)
	if got := bsonGetKey(doc, "ReturnValue"); got != "$Result" {
		t.Fatalf("ReturnValue = %q, want %q", got, "$Result")
	}
}

func TestSerializeAnnotationFlow_VersionShapes(t *testing.T) {
	af := &microflows.AnnotationFlow{
		BaseElement:   model.BaseElement{ID: "af-1"},
		OriginID:      "orig-1",
		DestinationID: "dest-1",
	}

	mx9 := serializeAnnotationFlow(af, 9)
	if !bsonHasKey(mx9, "OriginBezierVector") || !bsonHasKey(mx9, "DestinationBezierVector") {
		t.Error("Mx 9 annotation flow must use top-level BezierVector fields")
	}
	if bsonHasKey(mx9, "Line") {
		t.Error("Mx 9 annotation flow must NOT nest under Line")
	}

	mx10 := serializeAnnotationFlow(af, 10)
	if !bsonHasKey(mx10, "Line") {
		t.Error("Mx 10 annotation flow must nest vectors under Line")
	}
	if bsonHasKey(mx10, "OriginBezierVector") {
		t.Error("Mx 10 annotation flow must NOT include top-level BezierVector")
	}
}

func TestSerializeMicroflowParameter_Mx9_OmitsMx10OnlyKeys(t *testing.T) {
	p := &microflows.MicroflowParameter{
		BaseElement: model.BaseElement{ID: "p-1"},
		Name:        "Customer",
		Type:        &microflows.StringType{},
	}

	mx9 := serializeMicroflowParameter(p, 0, 9)
	if bsonHasKey(mx9, "DefaultValue") {
		t.Error("Mx 9 parameter must NOT emit DefaultValue")
	}
	if bsonHasKey(mx9, "IsRequired") {
		t.Error("Mx 9 parameter must NOT emit IsRequired")
	}

	mx10 := serializeMicroflowParameter(p, 0, 10)
	if !bsonHasKey(mx10, "DefaultValue") {
		t.Error("Mx 10 parameter must emit DefaultValue")
	}
	if !bsonHasKey(mx10, "IsRequired") {
		t.Error("Mx 10 parameter must emit IsRequired")
	}
}

func TestBuildSequenceFlowCase_NormalisesValueReceiver(t *testing.T) {
	// A value-receiver NoCase must produce the same shape as a pointer.
	fromValue := buildSequenceFlowCase(microflows.NoCase{BaseElement: model.BaseElement{ID: "x"}})
	fromPointer := buildSequenceFlowCase(&microflows.NoCase{BaseElement: model.BaseElement{ID: "x"}})

	if bsonGetKey(fromValue, "$Type") != bsonGetKey(fromPointer, "$Type") {
		t.Error("value and pointer NoCase must produce identical $Type")
	}
}

func TestBuildSequenceFlowCase_ExpressionCase_UsesEnumerationCase(t *testing.T) {
	doc := buildSequenceFlowCase(microflows.ExpressionCase{
		BaseElement: model.BaseElement{ID: "case-false"},
		Expression:  "false",
	})

	if got := bsonGetKey(doc, "$Type"); got != "Microflows$EnumerationCase" {
		t.Fatalf("$Type = %v, want Microflows$EnumerationCase", got)
	}
	if got := bsonGetKey(doc, "Value"); got != "false" {
		t.Fatalf("Value = %v, want false", got)
	}
}

// TestSerializeMicroflowStandalone_V10 verifies the package-level SerializeMicroflow
// function works without a Writer/Reader, accepting an explicit ProjectVersion.
// This is the writer-receiver-free path used by mprbackend/modelsdk-write helpers.
func TestSerializeMicroflowStandalone_V10(t *testing.T) {
	mf := &microflows.Microflow{
		BaseElement: model.BaseElement{ID: "mf-1"},
		Name:        "TestMF",
	}
	pv := &version.ProjectVersion{MajorVersion: 10, MinorVersion: 18, PatchVersion: 0}

	contents, err := SerializeMicroflow(mf, pv)
	if err != nil {
		t.Fatalf("SerializeMicroflow: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("SerializeMicroflow returned empty contents")
	}

	var doc bson.D
	if err := bson.Unmarshal(contents, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Mendix 10+ must include ReturnVariableName / StableId / Url / UrlSearchParameters.
	for _, key := range []string{"ReturnVariableName", "StableId", "Url", "UrlSearchParameters"} {
		if !bsonHasKey(doc, key) {
			t.Errorf("Mx 10 microflow must include %q", key)
		}
	}
}

// TestSerializeMicroflowStandalone_V9 verifies the package-level function omits
// Mx 10+ keys when given a Mendix 9 ProjectVersion.
func TestSerializeMicroflowStandalone_V9(t *testing.T) {
	mf := &microflows.Microflow{
		BaseElement: model.BaseElement{ID: "mf-1"},
		Name:        "TestMF",
	}
	pv := &version.ProjectVersion{MajorVersion: 9, MinorVersion: 24, PatchVersion: 0}

	contents, err := SerializeMicroflow(mf, pv)
	if err != nil {
		t.Fatalf("SerializeMicroflow: %v", err)
	}

	var doc bson.D
	if err := bson.Unmarshal(contents, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"ReturnVariableName", "StableId", "Url", "UrlSearchParameters"} {
		if bsonHasKey(doc, key) {
			t.Errorf("Mx 9 microflow must NOT include %q", key)
		}
	}
}

// TestSerializeMicroflowStandalone_NilVersion verifies a nil ProjectVersion
// falls back to the package default (currently 11.x), so callers without a
// detected version still get a serializable result.
func TestSerializeMicroflowStandalone_NilVersion(t *testing.T) {
	mf := &microflows.Microflow{
		BaseElement: model.BaseElement{ID: "mf-1"},
		Name:        "TestMF",
	}
	contents, err := SerializeMicroflow(mf, nil)
	if err != nil {
		t.Fatalf("SerializeMicroflow(nil pv): %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("expected non-empty contents")
	}
}

// TestSerializeNanoflowStandalone_V10 verifies the package-level SerializeNanoflow
// works without a Writer/Reader.
func TestSerializeNanoflowStandalone_V10(t *testing.T) {
	nf := &microflows.Nanoflow{
		BaseElement: model.BaseElement{ID: "nf-1"},
		Name:        "TestNF",
	}
	pv := &version.ProjectVersion{MajorVersion: 10, MinorVersion: 18, PatchVersion: 0}

	contents, err := SerializeNanoflow(nf, pv)
	if err != nil {
		t.Fatalf("SerializeNanoflow: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("SerializeNanoflow returned empty contents")
	}
}
