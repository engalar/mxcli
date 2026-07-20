// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// errBackend is a sentinel used in backend-error tests.
var errBackend = fmt.Errorf("backend failure")

func TestShowEnumerations_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:      func() bool { return true },
		ListEnumerationsFunc: func() ([]*model.Enumeration, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listEnumerations(ctx, ""))
}

func TestShowConstants_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:   func() bool { return true },
		ListConstantsFunc: func() ([]*model.Constant, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listConstants(ctx, ""))
}

// Stage 3.2.6.3a: TestShowMicroflows_Mock_BackendError /
// TestShowNanoflows_Mock_BackendError removed. The new
// gen-path `listMicroflows` / `listNanoflows` (in
// cmd_microflows_show_list_gen.go) read from ctx.Microflows /
// ctx.Nanoflows repos, not from the sdk-typed mock backend's
// ListMicroflowsFunc / ListNanoflowsFunc, so the simulated
// backend error never reaches the dispatch path. Equivalent
// error coverage lives in repo-level tests once the mock repo
// surface lands; for now the gen path returns an empty list when
// the repo is nil.

func TestShowPages_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	ctx.Pages = &repostesting.RecordingPageRepository{
		ListAllFunc: func() ([]*genPg.Page, error) { return nil, errBackend },
	}
	ctx.Deps = ctx.buildDeps()
	assertError(t, listPagesGen(ctx, ""))
}

func TestShowSnippets_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	ctx.Snippets = &repostesting.RecordingSnippetRepository{
		ListAllFunc: func() ([]*genPg.Snippet, error) { return nil, errBackend },
	}
	ctx.Deps = ctx.buildDeps()
	assertError(t, listSnippetsGen(ctx, ""))
}

func TestShowLayouts_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	ctx.Layouts = &repostesting.RecordingLayoutRepository{
		ListAllFunc: func() ([]*genPg.Layout, error) { return nil, errBackend },
	}
	ctx.Deps = ctx.buildDeps()
	assertError(t, listLayoutsGen(ctx, ""))
}

func TestShowWorkflows_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	ctx.Workflows = &repostesting.RecordingWorkflowRepository{
		ListAllFunc: func() ([]*genWf.Workflow, error) { return nil, errBackend },
	}
	ctx.Deps = ctx.buildDeps()
	assertError(t, listWorkflowsGen(ctx, ""))
}

func TestShowODataClients_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:               func() bool { return true },
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listODataClients(ctx, ""))
}

func TestShowODataServices_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:                func() bool { return true },
		ListPublishedODataServicesFunc: func() ([]*model.PublishedODataService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listODataServices(ctx, ""))
}

func TestShowRestClients_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:              func() bool { return true },
		ListConsumedRestServicesFunc: func() ([]*model.ConsumedRestService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listRestClients(ctx, ""))
}

func TestShowPublishedRestServices_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:               func() bool { return true },
		ListPublishedRestServicesFunc: func() ([]*model.PublishedRestService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listPublishedRestServices(ctx, ""))
}

func TestShowJavaActions_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		ListJavaActionsGenFunc: func() ([]*genJA.JavaAction, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listJavaActionsGen(ctx, ""))
}

// TestShowJavaScriptActions_Mock_BackendError retired in Stage 3.3.2.C1
// along with listJavaScriptActions (the gen variant has its own test
// in cmd_javaactions_gen_test.go).

func TestShowDatabaseConnections_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:             func() bool { return true },
		ListDatabaseConnectionsFunc: func() ([]*model.DatabaseConnection, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listDatabaseConnections(ctx, ""))
}

func TestShowImageCollections_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listImageCollections(ctx, ""))
}

func TestShowJsonStructures_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		ListJsonStructuresFunc: func() ([]*types.JsonStructure, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listJsonStructures(ctx, ""))
}

func TestShowNavigation_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:   func() bool { return true },
		GetNavigationFunc: func() (*types.NavigationDocument, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listNavigation(ctx))
}

func TestShowProjectSecurity_Mock_BackendError(t *testing.T) {
	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return nil, errBackend },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, listProjectSecurityGen(ctx))
}

func TestShowModuleRoles_Mock_BackendError(t *testing.T) {
	// listModuleRolesGen propagates errors from ListModules (not from per-module
	// GetModuleSecurity, which is silently skipped on error).
	sec := &repostesting.RecordingSecurityRepository{}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, listModuleRolesGen(ctx, ""))
}

func TestShowUserRoles_Mock_BackendError(t *testing.T) {
	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return nil, errBackend },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, listUserRolesGen(ctx))
}

func TestShowDemoUsers_Mock_BackendError(t *testing.T) {
	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return nil, errBackend },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, listDemoUsersGen(ctx))
}

func TestShowBusinessEventServices_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:               func() bool { return true },
		ListBusinessEventServicesFunc: func() ([]*model.BusinessEventService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listBusinessEventServices(ctx, ""))
}

func TestShowAgentEditorModels_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listAgentEditorModels(ctx, ""))
}

func TestShowAgentEditorAgents_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listAgentEditorAgents(ctx, ""))
}

func TestShowAgentEditorKnowledgeBases_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:                   func() bool { return true },
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listAgentEditorKnowledgeBases(ctx, ""))
}

func TestShowAgentEditorMCPServices_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:                        func() bool { return true },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listAgentEditorConsumedMCPServices(ctx, ""))
}

func TestListDataTransformers_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListDataTransformersFunc: func() ([]*model.DataTransformer, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listDataTransformers(ctx, ""))
}

func TestShowExportMappings_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		ListExportMappingsFunc: func() ([]*model.ExportMapping, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listExportMappings(ctx, ""))
}

func TestShowImportMappings_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		ListImportMappingsFunc: func() ([]*model.ImportMapping, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listImportMappings(ctx, ""))
}

func TestShowSettings_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listSettings(ctx))
}

// Describe handler backend errors

func TestDescribeEnumeration_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:      func() bool { return true },
		ListEnumerationsFunc: func() ([]*model.Enumeration, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeEnumeration(ctx, ast.QualifiedName{Module: "M", Name: "E"}))
}

func TestDescribeConstant_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:   func() bool { return true },
		ListConstantsFunc: func() ([]*model.Constant, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeConstant(ctx, ast.QualifiedName{Module: "M", Name: "C"}))
}

// Stage 3.2.6.3a: TestDescribeMicroflow_Mock_BackendError removed.
// `describeMicroflow` (legacy sdk/microflows) is being deleted; the
// gen-typed `describeMicroflowGen` reads from ctx.Microflows (modelsdk
// repository) rather than ctx.Backend.ListMicroflowsFunc, so the error
// path tested here is no longer reachable through this mock surface.
// Equivalent error-path coverage lives in cmd_microflows_show_gen_test.go.

func TestDescribeWorkflow_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	ctx.Workflows = &repostesting.RecordingWorkflowRepository{
		ListAllFunc: func() ([]*genWf.Workflow, error) { return nil, errBackend },
	}
	ctx.Deps = ctx.buildDeps()
	assertError(t, describeWorkflowGen(ctx, ast.QualifiedName{Module: "M", Name: "W"}))
}

func TestDescribeNavigation_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:   func() bool { return true },
		GetNavigationFunc: func() (*types.NavigationDocument, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeNavigation(ctx, ast.QualifiedName{Module: "M", Name: "N"}))
}

func TestDescribeODataClient_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:               func() bool { return true },
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeODataClient(ctx, ast.QualifiedName{Module: "M", Name: "C"}))
}

func TestDescribeODataService_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:                func() bool { return true },
		ListPublishedODataServicesFunc: func() ([]*model.PublishedODataService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeODataService(ctx, ast.QualifiedName{Module: "M", Name: "S"}))
}

func TestDescribeRestClient_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:              func() bool { return true },
		ListConsumedRestServicesFunc: func() ([]*model.ConsumedRestService, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeRestClient(ctx, ast.QualifiedName{Module: "M", Name: "R"}))
}

func TestDescribeImageCollection_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeImageCollection(ctx, ast.QualifiedName{Module: "M", Name: "I"}))
}

func TestDescribeDatabaseConnection_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:             func() bool { return true },
		ListDatabaseConnectionsFunc: func() ([]*model.DatabaseConnection, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeDatabaseConnection(ctx, ast.QualifiedName{Module: "M", Name: "D"}))
}

func TestDescribeModuleRole_Mock_BackendError(t *testing.T) {
	// describeModuleRoleGen propagates errors from ListModules (not from per-module
	// GetModuleSecurity, which is silently skipped on error).
	sec := &repostesting.RecordingSecurityRepository{}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, describeModuleRoleGen(ctx, ast.QualifiedName{Module: "M", Name: "R"}))
}

func TestDescribeUserRole_Mock_BackendError(t *testing.T) {
	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return nil, errBackend },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, describeUserRoleGen(ctx, ast.QualifiedName{Module: "", Name: "Admin"}))
}

func TestDescribeDemoUser_Mock_BackendError(t *testing.T) {
	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return nil, errBackend },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, describeDemoUserGen(ctx, "demo"))
}

// Write handler backend errors

func TestExecCreateModule_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, execCreateModule(ctx, &ast.CreateModuleStmt{Name: "M"}))
}

func TestExecCreateEnumeration_Mock_BackendError(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc:      func() bool { return true },
		ListEnumerationsFunc: func() ([]*model.Enumeration, error) { return nil, errBackend },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, execCreateEnumeration(ctx, &ast.CreateEnumerationStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
	}))
}

// Stage 3.2.6.5: TestExecDropMicroflow_Mock_BackendError removed —
// `execDropMicroflow` now reads from ctx.Microflows (modelsdk repo)
// instead of `ctx.Backend.ListMicroflowsFunc`, so the simulated
// sdk-typed backend error never reaches the dispatch path. Equivalent
// error coverage will land alongside a mock repo surface for
// gen flows.

// Stage 3.3.5.E0.error: TestExecDropPage_Mock_BackendError and
// TestExecDropSnippet_Mock_BackendError removed — ExecDropPage /
// ExecDropSnippet now read from ctx.Pages / ctx.Snippets repos (gen
// path) instead of ctx.Backend.ListPages / ListSnippets, so the
// simulated sdk-typed backend error never reaches the dispatch path.
// Error coverage for the gen path lives in the repo-level tests.
