// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// makeEntityParamMF 创建一个带实体类型参数的微流（用于测试）。
func makeEntityParamMF(name, entityQN string, isList bool) *genMf.Microflow {
	mf := genMf.NewMicroflow()
	mf.SetName(name)
	mf.SetID(element.ID(types.GenerateID()))

	oc := genMf.NewMicroflowObjectCollection()
	oc.SetID(element.ID(types.GenerateID()))

	param := genMf.NewMicroflowParameter()
	param.SetID(element.ID(types.GenerateID()))
	param.SetName("InputParam")

	if isList {
		dt := genDt.NewListType()
		dt.SetID(element.ID(types.GenerateID()))
		dt.SetEntityQualifiedName(entityQN)
		param.SetParameterType(dt)
	} else {
		dt := genDt.NewObjectType()
		dt.SetID(element.ID(types.GenerateID()))
		dt.SetEntityQualifiedName(entityQN)
		param.SetParameterType(dt)
	}
	oc.AddObjects(param)
	mf.SetObjectCollection(oc)
	return mf
}

func TestScanMicroflowsForEntityParam_FindsObjectParam(t *testing.T) {
	mf := makeEntityParamMF("ContractorReg.SUB_Save", "ContractorRegistration.ContractorApplication", false)
	names := map[*genMf.Microflow]string{mf: "ContractorReg.SUB_Save"}

	refs := scanMicroflowsForEntityParam(
		[]*genMf.Microflow{mf}, names,
		"ContractorRegistration.ContractorApplication",
	)
	if len(refs) != 1 {
		t.Fatalf("expected 1 result, got %d", len(refs))
	}
	if refs[0].MFName != "ContractorReg.SUB_Save" {
		t.Errorf("MFName = %q", refs[0].MFName)
	}
	if refs[0].ParamName != "InputParam" {
		t.Errorf("ParamName = %q", refs[0].ParamName)
	}
	if refs[0].IsList {
		t.Error("IsList should be false for Object param")
	}
}

func TestScanMicroflowsForEntityParam_FindsListParam(t *testing.T) {
	mf := makeEntityParamMF("M.MF", "Mod.Entity", true)
	names := map[*genMf.Microflow]string{mf: "M.MF"}

	refs := scanMicroflowsForEntityParam(
		[]*genMf.Microflow{mf}, names, "Mod.Entity",
	)
	if len(refs) != 1 {
		t.Fatalf("expected 1, got %d", len(refs))
	}
	if !refs[0].IsList {
		t.Error("IsList should be true for List param")
	}
}

func TestScanMicroflowsForEntityParam_IgnoresDifferentEntity(t *testing.T) {
	mf := makeEntityParamMF("M.MF", "OtherModule.OtherEntity", false)
	names := map[*genMf.Microflow]string{mf: "M.MF"}

	refs := scanMicroflowsForEntityParam(
		[]*genMf.Microflow{mf}, names,
		"ContractorRegistration.ContractorApplication",
	)
	if len(refs) != 0 {
		t.Fatalf("expected 0, got %d", len(refs))
	}
}

func TestScanMicroflowsForEntityParam_MultipleMFs(t *testing.T) {
	mf1 := makeEntityParamMF("M.MF1", "Mod.Entity", false)
	mf2 := makeEntityParamMF("M.MF2", "Mod.Entity", true)
	mf3 := makeEntityParamMF("M.MF3", "Mod.Other", false)
	names := map[*genMf.Microflow]string{
		mf1: "M.MF1", mf2: "M.MF2", mf3: "M.MF3",
	}

	refs := scanMicroflowsForEntityParam(
		[]*genMf.Microflow{mf1, mf2, mf3}, names, "Mod.Entity",
	)
	if len(refs) != 2 {
		t.Fatalf("expected 2, got %d", len(refs))
	}
}

func TestScanMicroflowsForEntityParam_IgnoresNonEntityParam(t *testing.T) {
	// 一个没有参数（或者只有 StringParam）的微流
	mf := makeMFWithActions("M.MF") // 无 actions，ObjectCollection 为空
	names := map[*genMf.Microflow]string{mf: "M.MF"}

	refs := scanMicroflowsForEntityParam(
		[]*genMf.Microflow{mf}, names, "Mod.Entity",
	)
	if len(refs) != 0 {
		t.Fatalf("expected 0, got %d", len(refs))
	}
}
