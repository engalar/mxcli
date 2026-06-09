// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// makeCallAction constructs a MicroflowCallAction that calls targetQN and maps
// each entry in paramRefs as a MicroflowCallParameterMapping.ParameterQualifiedName.
func makeCallAction(targetQN string, paramRefs ...string) *genMf.MicroflowCallAction {
	callAction := genMf.NewMicroflowCallAction()
	callAction.SetID(element.ID(types.GenerateID()))

	call := genMf.NewMicroflowCall()
	call.SetID(element.ID(types.GenerateID()))
	call.SetMicroflowQualifiedName(targetQN)

	for _, pqn := range paramRefs {
		mapping := genMf.NewMicroflowCallParameterMapping()
		mapping.SetID(element.ID(types.GenerateID()))
		mapping.SetParameterQualifiedName(pqn)
		mapping.SetArgument("$WorkflowContext")
		call.AddParameterMappings(mapping)
	}
	callAction.SetMicroflowCall(call)
	return callAction
}

// makeMFWithActions builds a named Microflow whose ObjectCollection contains
// the given list of MicroflowCallAction elements.
func makeMFWithActions(name string, actions ...*genMf.MicroflowCallAction) *genMf.Microflow {
	mf := genMf.NewMicroflow()
	mf.SetName(name)
	mf.SetID(element.ID(types.GenerateID()))

	oc := genMf.NewMicroflowObjectCollection()
	oc.SetID(element.ID(types.GenerateID()))
	for _, a := range actions {
		oc.AddObjects(a)
	}
	mf.SetObjectCollection(oc)
	return mf
}

func TestScanBrokenCallerRefs_FindsBrokenMapping(t *testing.T) {
	action := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam",
	)
	caller := makeMFWithActions("ContractorReg.SUB_Validate", action)
	names := map[*genMf.Microflow]string{
		caller: "ContractorReg.SUB_Validate",
	}
	refs := scanBrokenCallerRefs(
		[]*genMf.Microflow{caller},
		names,
		"Common_Utils.GET_Message_ById",
		[]string{"OldParam"},
	)
	if len(refs) != 1 {
		t.Fatalf("expected 1 broken ref, got %d", len(refs))
	}
	if refs[0].CallerName != "ContractorReg.SUB_Validate" {
		t.Errorf("CallerName = %q", refs[0].CallerName)
	}
	if refs[0].TargetMF != "Common_Utils.GET_Message_ById" {
		t.Errorf("TargetMF = %q", refs[0].TargetMF)
	}
	if refs[0].BrokenParam != "Common_Utils.GET_Message_ById.OldParam" {
		t.Errorf("BrokenParam = %q", refs[0].BrokenParam)
	}
}

func TestScanBrokenCallerRefs_IgnoresValidMapping(t *testing.T) {
	action := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.MessageId",
	)
	caller := makeMFWithActions("M.Caller", action)
	names := map[*genMf.Microflow]string{caller: "M.Caller"}
	refs := scanBrokenCallerRefs(
		[]*genMf.Microflow{caller},
		names,
		"Common_Utils.GET_Message_ById",
		[]string{"OldParam"},
	)
	if len(refs) != 0 {
		t.Fatalf("expected 0 broken refs, got %d: %v", len(refs), refs)
	}
}

func TestScanBrokenCallerRefs_IgnoresCallsToOtherMF(t *testing.T) {
	action := makeCallAction(
		"Common_Utils.GET_OtherThing",
		"Common_Utils.GET_OtherThing.Param",
	)
	caller := makeMFWithActions("M.Caller", action)
	names := map[*genMf.Microflow]string{caller: "M.Caller"}
	refs := scanBrokenCallerRefs(
		[]*genMf.Microflow{caller},
		names,
		"Common_Utils.GET_Message_ById",
		[]string{"Param"},
	)
	if len(refs) != 0 {
		t.Fatalf("expected 0 refs, got %d", len(refs))
	}
}

func TestScanBrokenCallerRefs_MultipleCallersMultipleRefs(t *testing.T) {
	brokenAction := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam",
		"Common_Utils.GET_Message_ById.AnotherOld",
	)
	goodAction := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.MessageId",
	)
	caller1 := makeMFWithActions("M.Caller1", brokenAction)
	caller2 := makeMFWithActions("M.Caller2", goodAction)
	names := map[*genMf.Microflow]string{
		caller1: "M.Caller1",
		caller2: "M.Caller2",
	}
	refs := scanBrokenCallerRefs(
		[]*genMf.Microflow{caller1, caller2},
		names,
		"Common_Utils.GET_Message_ById",
		[]string{"OldParam", "AnotherOld"},
	)
	if len(refs) != 2 {
		t.Fatalf("expected 2 broken refs, got %d: %v", len(refs), refs)
	}
}

func TestScanBrokenCallerRefs_EmptyRemovedParams(t *testing.T) {
	action := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.SomeParam",
	)
	caller := makeMFWithActions("M.Caller", action)
	names := map[*genMf.Microflow]string{caller: "M.Caller"}
	refs := scanBrokenCallerRefs(
		[]*genMf.Microflow{caller},
		names,
		"Common_Utils.GET_Message_ById",
		nil,
	)
	if len(refs) != 0 {
		t.Fatalf("expected 0 refs, got %d", len(refs))
	}
}
