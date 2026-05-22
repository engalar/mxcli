// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.k — execCreateMicroflowGen end-to-end tests (TDD).
//
// Covers the entry point shape: name validation, module resolution,
// existence check, body iteration via buildFlowGraphGen,
// persist via ctx.Microflows.Create (or .Update for replace).
//
// Tests run offline (no full ExecContext / repo wiring) and exercise
// the validation / construction paths. Full integration with
// ctx.Microflows.Create is covered by the existing
// cmd_nanoflows_create_gen_test integration suite — the gen
// MicroflowRepository writer mirror that pattern.

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/types"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// TestMultipleParametersHaveDistinctPositions verifies that when a microflow
// is created with multiple parameters, each MicroflowParameter in the
// ObjectCollection gets a unique, non-empty RelativeMiddlePoint.
// Without this, all parameters overlap at the same coordinates in Studio Pro.
func TestMultipleParametersHaveDistinctPositions(t *testing.T) {
	mod := mkModule("MyModule")

	var capturedMF *genMf.Microflow
	repo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) { return nil, nil },
		CreateFunc: func(call repostesting.MicroflowCreateCall) error {
			capturedMF = call.Microflow
			return nil
		},
	}

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(mkHierarchy(mod)))
	ctx.Microflows = repo

	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "DoWork"},
		Parameters: []ast.MicroflowParam{
			{Name: "Param1", Type: ast.DataType{Kind: ast.TypeString}},
			{Name: "Param2", Type: ast.DataType{Kind: ast.TypeBoolean}},
			{Name: "Param3", Type: ast.DataType{Kind: ast.TypeInteger}},
		},
	}

	if err := execCreateMicroflowGen(ctx, stmt); err != nil {
		t.Fatalf("execCreateMicroflowGen failed: %v", err)
	}
	if capturedMF == nil {
		t.Fatal("Create was not called — no microflow captured")
	}

	oc, ok := capturedMF.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		t.Fatal("expected non-nil MicroflowObjectCollection")
	}

	type posGetter interface {
		element.Element
		RelativeMiddlePoint() string
	}

	var paramPositions []string
	for _, obj := range oc.ObjectsItems() {
		if obj == nil || obj.TypeName() != "Microflows$MicroflowParameter" {
			continue
		}
		pg, ok := obj.(posGetter)
		if !ok {
			t.Errorf("MicroflowParameter does not implement RelativeMiddlePoint()")
			continue
		}
		pos := pg.RelativeMiddlePoint()
		if pos == "" {
			t.Errorf("MicroflowParameter %q has empty RelativeMiddlePoint", obj.TypeName())
		}
		paramPositions = append(paramPositions, pos)
	}

	if len(paramPositions) != 3 {
		t.Fatalf("expected 3 parameters with positions, got %d", len(paramPositions))
	}

	seen := make(map[string]bool)
	for _, pos := range paramPositions {
		if seen[pos] {
			t.Errorf("duplicate parameter position %q — parameters must not overlap", pos)
		}
		seen[pos] = true
	}
}

func TestExecCreateMicroflowGenRejectsEmptyName(t *testing.T) {
	ctx := &ExecContext{}
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: ""},
	}
	err := execCreateMicroflowGen(ctx, stmt)
	if err == nil {
		t.Fatal("expected error for empty microflow name")
	}
}

func TestExecCreateMicroflowGenRejectsWhitespaceName(t *testing.T) {
	ctx := &ExecContext{}
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "   "},
	}
	err := execCreateMicroflowGen(ctx, stmt)
	if err == nil {
		t.Fatal("expected error for whitespace-only microflow name")
	}
}

func TestExecCreateMicroflowGenRejectsNotConnected(t *testing.T) {
	// Without ConnectedForWrite, execCreateMicroflowGen should refuse.
	ctx := &ExecContext{}
	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "NewMF"},
	}
	err := execCreateMicroflowGen(ctx, stmt)
	if err == nil {
		t.Fatal("expected error when not connected for write")
	}
	// Error should mention "not connected" or similar.
	if !strings.Contains(err.Error(), "connect") && !strings.Contains(err.Error(), "name must not be empty") {
		t.Logf("got error: %v", err)
	}
}

func TestExecCreateMicroflowGen_WarnsOnRemovedParam(t *testing.T) {
	// Setup：existing 微流有 OldParam；caller 微流引用了它；
	// CREATE OR REPLACE 只保留 NewParam → 应该打印 CE1613 警告。

	// 注意：这个测试只验证警告输出，不需要真正写入 MPR。
	// execCreateMicroflowGen 在验证和警告之后会因为缺少 backend 而返回错误，
	// 但警告已经被写入 ctx.Output，所以可以直接断言。

	existingMF := genMf.NewMicroflow()
	existingMF.SetName("GET_Message_ById")
	existingMF.SetID(element.ID(types.GenerateID()))
	existingOC := genMf.NewMicroflowObjectCollection()
	existingOC.SetID(element.ID(types.GenerateID()))
	oldParam := genMf.NewMicroflowParameter()
	oldParam.SetID(element.ID(types.GenerateID()))
	oldParam.SetName("OldParam")
	existingOC.AddObjects(oldParam)
	existingMF.SetObjectCollection(existingOC)

	brokenAction := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam",
	)
	callerMF := makeMFWithActions("Common_Utils.CallerMF", brokenAction)
	callerMF.SetID(element.ID(types.GenerateID()))

	mod := mkModule("Common_Utils")

	var buf bytes.Buffer
	repo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{existingMF, callerMF}, nil
		},
		// Return mod.ID so that hierarchy.FindModuleID can resolve "Common_Utils".
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			return mod.ID, nil
		},
	}

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(mkHierarchy(mod)))
	ctx.Microflows = repo
	ctx.Output = &buf

	stmt := &ast.CreateMicroflowStmt{
		Name:           ast.QualifiedName{Module: "Common_Utils", Name: "GET_Message_ById"},
		CreateOrModify: true,
		Parameters: []ast.MicroflowParam{
			{Name: "NewParam", Type: ast.DataType{Kind: ast.TypeString}},
			// OldParam 不再出现
		},
	}

	// 允许 error（backend 未完整配置），但警告应在 output 里
	_ = execCreateMicroflowGen(ctx, stmt)

	output := buf.String()
	if !strings.Contains(output, "OldParam") {
		t.Errorf("expected CE1613 warning about OldParam, got:\n%s", output)
	}
	if !strings.Contains(output, "CE1613") {
		t.Errorf("expected CE1613 keyword in warning, got:\n%s", output)
	}
}
