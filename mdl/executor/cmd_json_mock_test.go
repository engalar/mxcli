// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

func TestShowEnumerations_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	enum := mkEnumeration(mod.ID, "Status", "Active", "Inactive")
	withContainer(h, enum.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:      func() bool { return true },
		ListEnumerationsFunc: func() ([]*model.Enumeration, error) { return []*model.Enumeration{enum}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listEnumerations(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Status")
}

func TestShowConstants_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	c := mkConstant(mod.ID, "Timeout", "Integer", "30")
	withContainer(h, c.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:   func() bool { return true },
		ListConstantsFunc: func() ([]*model.Constant, error) { return []*model.Constant{c}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listConstants(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Timeout")
}

// Stage 3.2.6.3a: TestShowMicroflows_Mock_JSON / TestShowNanoflows_Mock_JSON
// removed. The new gen-path `listMicroflows` / `listNanoflows` (in
// cmd_microflows_show_list_gen.go) reads from ctx.Microflows /
// ctx.Nanoflows repos, not from the sdk-typed mock backend's
// ListMicroflowsFunc / ListNanoflowsFunc, so seeding `mf` / `nf` via
// the mock backend is never observed by the formatter. Equivalent
// JSON-output coverage will land alongside a mock repo surface for
// gen flows; in the meantime the gen path is exercised by the
// fixture-based tests in cmd_microflows_show_gen_test.go and
// cmd_nanoflows_show_gen_test.go, plus the live MPR roundtrip tests.

func TestShowPages_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	pg := mkPage(mod.ID, "Page_Home")
	withContainer(h, pg.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPagesFunc:   func() ([]*pages.Page, error) { return []*pages.Page{pg}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listPages(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Page_Home")
}

func TestShowSnippets_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	snp := mkSnippet(mod.ID, "Snippet_Header")
	withContainer(h, snp.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:  func() bool { return true },
		ListSnippetsFunc: func() ([]*pages.Snippet, error) { return []*pages.Snippet{snp}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listSnippets(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Snippet_Header")
}

func TestShowLayouts_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	lay := mkLayout(mod.ID, "Layout_Main")
	withContainer(h, lay.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListLayoutsFunc: func() ([]*pages.Layout, error) { return []*pages.Layout{lay}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listLayouts(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Layout_Main")
}

func TestShowWorkflows_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	wf := mkWorkflowGen(string(nextID("wf")), "WF_Approve")

	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	ctx.Workflows = &repostesting.RecordingWorkflowRepository{
		ListAllFunc:          func() ([]*genWf.Workflow, error) { return []*genWf.Workflow{wf}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return mod.ID, nil },
	}
	assertNoError(t, listWorkflowsGen(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "WF_Approve")
}

func TestShowODataClients_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	svc := &model.ConsumedODataService{
		BaseElement: model.BaseElement{ID: nextID("cos")},
		ContainerID: mod.ID,
		Name:        "ExtService",
	}
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:               func() bool { return true },
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) { return []*model.ConsumedODataService{svc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listODataClients(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "ExtService")
}

func TestShowODataServices_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	svc := &model.PublishedODataService{
		BaseElement: model.BaseElement{ID: nextID("pos")},
		ContainerID: mod.ID,
		Name:        "PubOData",
	}
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:                func() bool { return true },
		ListPublishedODataServicesFunc: func() ([]*model.PublishedODataService, error) { return []*model.PublishedODataService{svc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listODataServices(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "PubOData")
}

func TestShowRestClients_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	svc := &model.ConsumedRestService{
		BaseElement: model.BaseElement{ID: nextID("crs")},
		ContainerID: mod.ID,
		Name:        "RestClient1",
	}
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:              func() bool { return true },
		ListConsumedRestServicesFunc: func() ([]*model.ConsumedRestService, error) { return []*model.ConsumedRestService{svc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listRestClients(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "RestClient1")
}

func TestShowPublishedRestServices_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	svc := &model.PublishedRestService{
		BaseElement: model.BaseElement{ID: nextID("prs")},
		ContainerID: mod.ID,
		Name:        "PubRest1",
	}
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:               func() bool { return true },
		ListPublishedRestServicesFunc: func() ([]*model.PublishedRestService, error) { return []*model.PublishedRestService{svc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listPublishedRestServices(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "PubRest1")
}

func TestShowJavaActions_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	jaGen := genJA.NewJavaAction()
	jaGen.SetName("MyJavaAction")
	withContainer(h, model.ID(jaGen.ID()), mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListJavaActionsGenFunc: func() ([]*genJA.JavaAction, error) {
			return []*genJA.JavaAction{jaGen}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listJavaActionsGen(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "MyJavaAction")
}

// TestShowJavaScriptActions_Mock_JSON retired in Stage 3.3.2.C1 — the
// gen path renders via ctx.JavaScriptActions and is exercised through
// listJavaScriptActionsGen in cmd_javaactions_gen_test.go.

func TestShowDatabaseConnections_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	dc := &model.DatabaseConnection{
		BaseElement: model.BaseElement{ID: nextID("dc")},
		ContainerID: mod.ID,
		Name:        "MyDB",
	}
	withContainer(h, dc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:             func() bool { return true },
		ListDatabaseConnectionsFunc: func() ([]*model.DatabaseConnection, error) { return []*model.DatabaseConnection{dc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listDatabaseConnections(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "MyDB")
}

func TestShowImageCollections_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
	}
	withContainer(h, ic.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listImageCollections(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Icons")
}

func TestShowJsonStructures_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	js := &types.JsonStructure{
		BaseElement: model.BaseElement{ID: nextID("js")},
		ContainerID: mod.ID,
		Name:        "OrderSchema",
	}
	withContainer(h, js.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		ListJsonStructuresFunc: func() ([]*types.JsonStructure, error) { return []*types.JsonStructure{js}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listJsonStructures(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "OrderSchema")
}

func TestShowUserRoles_Mock_JSON(t *testing.T) {
	ps := genSec.NewProjectSecurity()
	ur := genSec.NewUserRole()
	ur.SetName("Administrator")
	ps.AddUserRoles(ur)

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec), withFormat(FormatJSON))
	assertNoError(t, listUserRolesGen(ctx))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Administrator")
}

func TestShowModuleRoles_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	ms := genSec.NewModuleSecurity()
	mr := genSec.NewModuleRole()
	mr.SetName("User")
	ms.AddModuleRoles(mr)

	sec := &repostesting.RecordingSecurityRepository{
		GetModuleSecFunc: func(id model.ID) (*genSec.ModuleSecurity, error) {
			if id == mod.ID {
				return ms, nil
			}
			return nil, nil
		},
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}

	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listModuleRolesGen(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "User")
}

func TestShowDemoUsers_Mock_JSON(t *testing.T) {
	ps := genSec.NewProjectSecurity()
	ps.SetEnableDemoUsers(true)
	du := genSec.NewDemoUser()
	du.SetUserName("demo_admin")
	ps.AddDemoUsers(du)

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec), withFormat(FormatJSON))
	assertNoError(t, listDemoUsersGen(ctx))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "demo_admin")
}

func TestShowBusinessEventServices_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	svc := &model.BusinessEventService{
		BaseElement: model.BaseElement{ID: nextID("bes")},
		ContainerID: mod.ID,
		Name:        "OrderEvents",
	}
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:               func() bool { return true },
		ListBusinessEventServicesFunc: func() ([]*model.BusinessEventService, error) { return []*model.BusinessEventService{svc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listBusinessEventServices(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "OrderEvents")
}

func TestShowAgentEditorModels_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	m1 := &types.Model{
		BaseElement: model.BaseElement{ID: nextID("aem")},
		ContainerID: mod.ID,
		Name:        "GPT4o",
	}
	withContainer(h, m1.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return []*types.Model{m1}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listAgentEditorModels(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "GPT4o")
}

func TestShowAgentEditorAgents_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	a1 := &types.Agent{
		BaseElement: model.BaseElement{ID: nextID("aea")},
		ContainerID: mod.ID,
		Name:        "Helper",
	}
	withContainer(h, a1.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return []*types.Agent{a1}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listAgentEditorAgents(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Helper")
}

func TestShowAgentEditorKnowledgeBases_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	kb := &types.KnowledgeBase{
		BaseElement: model.BaseElement{ID: nextID("aek")},
		ContainerID: mod.ID,
		Name:        "FAQ",
	}
	withContainer(h, kb.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:                   func() bool { return true },
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return []*types.KnowledgeBase{kb}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listAgentEditorKnowledgeBases(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "FAQ")
}

func TestShowAgentEditorMCPServices_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	svc := &types.ConsumedMCPService{
		BaseElement: model.BaseElement{ID: nextID("aes")},
		ContainerID: mod.ID,
		Name:        "ToolSvc",
	}
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:                        func() bool { return true },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) { return []*types.ConsumedMCPService{svc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listAgentEditorConsumedMCPServices(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "ToolSvc")
}

func TestListDataTransformers_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	dt := &model.DataTransformer{
		BaseElement: model.BaseElement{ID: nextID("dt")},
		ContainerID: mod.ID,
		Name:        "Transform1",
	}
	withContainer(h, dt.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		ListDataTransformersFunc: func() ([]*model.DataTransformer, error) { return []*model.DataTransformer{dt}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listDataTransformers(ctx, ""))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "Transform1")
}

// Stage 3.2.6.5: TestShowAccessOnMicroflow_Mock_JSON removed —
// `listAccessOnMicroflow` (sdk-typed) is gone; the dispatch in
// executor_query.go now calls `listAccessOnMicroflowGen` which reads
// from ctx.Microflows. Equivalent JSON-output coverage will land
// alongside a gen-typed mock repo surface.

func TestShowAccessOnPage_Mock_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	pg := mkPage(mod.ID, "Page_Home")
	pg.AllowedRoles = []model.ID{"MyModule.User"}
	withContainer(h, pg.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPagesFunc:   func() ([]*pages.Page, error) { return []*pages.Page{pg}, nil },
	}

	name := &ast.QualifiedName{Module: "MyModule", Name: "Page_Home"}
	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listAccessOnPageGen(ctx, name))
	assertValidJSON(t, buf.String())
	assertContainsStr(t, buf.String(), "User")
}

// TestShowConstants_Mock_JSON_EmptyResult verifies that an empty result still
// produces valid JSON (not the "No ... found." plain-text message).
func TestShowConstants_Mock_JSON_EmptyResult(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:   func() bool { return true },
		ListConstantsFunc: func() ([]*model.Constant, error) { return nil, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listConstants(ctx, ""))
	assertValidJSON(t, buf.String())
	assertNotContainsStr(t, buf.String(), "No constants found")
}

// TestShowPublishedRestServices_Mock_JSON_EmptyResult verifies that an empty
// result still produces valid JSON in JSON mode.
func TestShowPublishedRestServices_Mock_JSON_EmptyResult(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:               func() bool { return true },
		ListPublishedRestServicesFunc: func() ([]*model.PublishedRestService, error) { return nil, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON), withHierarchy(h))
	assertNoError(t, listPublishedRestServices(ctx, ""))
	assertValidJSON(t, buf.String())
	assertNotContainsStr(t, buf.String(), "No published rest services found")
}
