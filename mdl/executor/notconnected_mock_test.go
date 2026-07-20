// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
)

// disconnectedBackend returns a MockBackend that reports not connected.
func disconnectedBackend() *mock.MockBackend {
	return &mock.MockBackend{
		IsConnectedFunc: func() bool { return false },
	}
}

func TestShowModules_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listModules(ctx))
}

func TestShowSettings_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listSettings(ctx))
}

func TestShowVersion_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listVersion(ctx))
}

func TestShowExportMappings_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listExportMappings(ctx, ""))
}

func TestShowImportMappings_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listImportMappings(ctx, ""))
}

func TestShowBusinessEventServices_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listBusinessEventServices(ctx, ""))
}

func TestShowAgentEditorModels_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listAgentEditorModels(ctx, ""))
}

func TestShowAgentEditorAgents_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listAgentEditorAgents(ctx, ""))
}

func TestShowAgentEditorKnowledgeBases_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listAgentEditorKnowledgeBases(ctx, ""))
}

func TestShowAgentEditorMCPServices_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, listAgentEditorConsumedMCPServices(ctx, ""))
}

func TestDescribeAgentEditorModel_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describeAgentEditorModel(ctx, ast.QualifiedName{Module: "M", Name: "X"}))
}

func TestDescribeAgentEditorAgent_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describeAgentEditorAgent(ctx, ast.QualifiedName{Module: "M", Name: "X"}))
}

func TestDescribeAgentEditorKnowledgeBase_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describeAgentEditorKnowledgeBase(ctx, ast.QualifiedName{Module: "M", Name: "X"}))
}

func TestDescribeAgentEditorMCPService_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describeAgentEditorConsumedMCPService(ctx, ast.QualifiedName{Module: "M", Name: "X"}))
}

func TestDescribeMermaid_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describeMermaid(ctx, "domainmodel", "MyModule"))
}

func TestDescribeSettings_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describeSettings(ctx))
}

func TestDescribeBusinessEventService_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describeBusinessEventService(ctx, ast.QualifiedName{Module: "M", Name: "S"}))
}

func TestDescribeDataTransformer_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describeDataTransformer(ctx, ast.QualifiedName{Module: "M", Name: "D"}))
}

func TestDescribePublishedRestService_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, describePublishedRestServiceFn(ctx, ctx.Deps.Output, ctx.Deps, ast.QualifiedName{Module: "M", Name: "R"}))
}

func TestExecCreateModule_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, execCreateModule(ctx, &ast.CreateModuleStmt{Name: "M"}))
}

func TestExecCreateEnumeration_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, execCreateEnumeration(ctx, &ast.CreateEnumerationStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
	}))
}

func TestExecDropEnumeration_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, execDropEnumeration(ctx, &ast.DropEnumerationStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
	}))
}

func TestExecDropEntity_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, ExecDropEntityDeps(ctx, &ast.DropEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
	}, ctx.Deps))
}

func TestExecDropMicroflow_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, ExecDropMicroflowFn(ctx, &ast.DropMicroflowStmt{
		Name: ast.QualifiedName{Module: "M", Name: "F"},
	}, ctx.Deps))
}

func TestExecDropPage_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, ExecDropPageDeps(ctx, &ast.DropPageStmt{
		Name: ast.QualifiedName{Module: "M", Name: "P"},
	}, ctx.Deps))
}

func TestExecDropSnippet_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, ExecDropSnippetDeps(ctx, &ast.DropSnippetStmt{
		Name: ast.QualifiedName{Module: "M", Name: "S"},
	}, ctx.Deps))
}

func TestExecDropAssociation_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, execDropAssociation(ctx, &ast.DropAssociationStmt{
		Name: ast.QualifiedName{Module: "M", Name: "A"},
	}))
}

func TestExecDropJavaAction_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, execDropJavaActionGenFn(ctx, &ast.DropJavaActionStmt{
		Name: ast.QualifiedName{Module: "M", Name: "J"},
	}, ctx.Deps))
}

func TestExecDropFolder_Mock_NotConnected(t *testing.T) {
	ctx, _ := newMockCtx(t, withBackend(disconnectedBackend()))
	assertError(t, execDropFolderFn(ctx, &ast.DropFolderStmt{
		FolderPath: "Resources/Images",
		Module:     "M",
	}, ctx.Deps))
}
