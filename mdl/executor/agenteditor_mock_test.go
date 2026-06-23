// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/model"
)

// ---------------------------------------------------------------------------
// CREATE / DROP — Model
// ---------------------------------------------------------------------------

func TestCreateAgentEditorModel_Mock(t *testing.T) {
	mod := mkModule("M")
	apiKey := mkConstant(mod.ID, "APIKey", "String", "")

	h := mkHierarchy(mod)
	withContainer(h, apiKey.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListModulesFunc:           func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return nil, nil },
		ListConstantsFunc:         func() ([]*model.Constant, error) { return []*model.Constant{apiKey}, nil },
		CreateAgentEditorModelFunc: func(m *types.Model) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	key := ast.QualifiedName{Module: "M", Name: "APIKey"}
	err := execCreateAgentEditorModel(ctx, &ast.CreateModelStmt{
		Name:     ast.QualifiedName{Module: "M", Name: "GPT4"},
		Provider: "MxCloudGenAI",
		Key:      &key,
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Created model: M.GPT4")
	if !called {
		t.Fatal("CreateAgentEditorModelFunc was not called")
	}
}

func TestDropAgentEditorModel_Mock(t *testing.T) {
	mod := mkModule("M")
	m1 := &types.Model{
		BaseElement: model.BaseElement{ID: nextID("aem")},
		ContainerID: mod.ID,
		Name:        "GPT4",
	}

	h := mkHierarchy(mod)
	withContainer(h, m1.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return []*types.Model{m1}, nil },
		DeleteAgentEditorModelFunc: func(id string) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execDropAgentEditorModel(ctx, &ast.DropModelStmt{
		Name: ast.QualifiedName{Module: "M", Name: "GPT4"},
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped model: M.GPT4")
	if !called {
		t.Fatal("DeleteAgentEditorModelFunc was not called")
	}
}

// ---------------------------------------------------------------------------
// CREATE / DROP — Consumed MCP Service
// ---------------------------------------------------------------------------

func TestCreateConsumedMCPService_Mock(t *testing.T) {
	mod := mkModule("M")

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:                        func() bool { return true },
		ListModulesFunc:                        func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) { return nil, nil },
		CreateAgentEditorConsumedMCPServiceFunc: func(svc *types.ConsumedMCPService) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb))
	err := ExecCreateConsumedMCPServiceFn(ctx, &ast.CreateConsumedMCPServiceStmt{
		Name:            ast.QualifiedName{Module: "M", Name: "WebSearch"},
		ProtocolVersion: "v2025_03_26",
		Version:         "1.0",
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Created consumed mcp service: M.WebSearch")
	if !called {
		t.Fatal("CreateAgentEditorConsumedMCPServiceFunc was not called")
	}
}

func TestDropConsumedMCPService_Mock(t *testing.T) {
	mod := mkModule("M")
	svc := &types.ConsumedMCPService{
		BaseElement: model.BaseElement{ID: nextID("aemcp")},
		ContainerID: mod.ID,
		Name:        "WebSearch",
	}

	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:                        func() bool { return true },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) { return []*types.ConsumedMCPService{svc}, nil },
		DeleteAgentEditorConsumedMCPServiceFunc: func(id string) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := ExecDropConsumedMCPServiceFn(ctx, &ast.DropConsumedMCPServiceStmt{
		Name: ast.QualifiedName{Module: "M", Name: "WebSearch"},
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped consumed mcp service: M.WebSearch")
	if !called {
		t.Fatal("DeleteAgentEditorConsumedMCPServiceFunc was not called")
	}
}

// ---------------------------------------------------------------------------
// CREATE / DROP — Knowledge Base
// ---------------------------------------------------------------------------

func TestCreateKnowledgeBase_Mock(t *testing.T) {
	mod := mkModule("M")
	kbKey := mkConstant(mod.ID, "KBKey", "String", "")

	h := mkHierarchy(mod)
	withContainer(h, kbKey.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:                   func() bool { return true },
		ListModulesFunc:                   func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return nil, nil },
		ListConstantsFunc:                 func() ([]*model.Constant, error) { return []*model.Constant{kbKey}, nil },
		CreateAgentEditorKnowledgeBaseFunc: func(kb *types.KnowledgeBase) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	key := ast.QualifiedName{Module: "M", Name: "KBKey"}
	err := ExecCreateKnowledgeBaseFn(ctx, &ast.CreateKnowledgeBaseStmt{
		Name:     ast.QualifiedName{Module: "M", Name: "ProductDocs"},
		Provider: "MxCloudGenAI",
		Key:      &key,
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Created knowledge base: M.ProductDocs")
	if !called {
		t.Fatal("CreateAgentEditorKnowledgeBaseFunc was not called")
	}
}

func TestDropKnowledgeBase_Mock(t *testing.T) {
	mod := mkModule("M")
	kb := &types.KnowledgeBase{
		BaseElement: model.BaseElement{ID: nextID("aekb")},
		ContainerID: mod.ID,
		Name:        "ProductDocs",
	}

	h := mkHierarchy(mod)
	withContainer(h, kb.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:                   func() bool { return true },
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return []*types.KnowledgeBase{kb}, nil },
		DeleteAgentEditorKnowledgeBaseFunc: func(id string) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := ExecDropKnowledgeBaseFn(ctx, &ast.DropKnowledgeBaseStmt{
		Name: ast.QualifiedName{Module: "M", Name: "ProductDocs"},
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped knowledge base: M.ProductDocs")
	if !called {
		t.Fatal("DeleteAgentEditorKnowledgeBaseFunc was not called")
	}
}

// ---------------------------------------------------------------------------
// CREATE / DROP — Agent
// ---------------------------------------------------------------------------

func TestCreateAgent_Mock(t *testing.T) {
	mod := mkModule("M")
	mdl := &types.Model{
		BaseElement: model.BaseElement{ID: nextID("aem")},
		ContainerID: mod.ID,
		Name:        "GPT4",
	}

	h := mkHierarchy(mod)
	withContainer(h, mdl.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListModulesFunc:           func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return nil, nil },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return []*types.Model{mdl}, nil },
		CreateAgentEditorAgentFunc: func(a *types.Agent) error {
			called = true
			return nil
		},
	}

	modelRef := ast.QualifiedName{Module: "M", Name: "GPT4"}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := ExecCreateAgentFn(ctx, &ast.CreateAgentStmt{
		Name:         ast.QualifiedName{Module: "M", Name: "Summarizer"},
		UsageType:    "Task",
		Model:        &modelRef,
		SystemPrompt: "Summarize in 3 sentences.",
		UserPrompt:   "Enter text.",
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Created agent: M.Summarizer")
	if !called {
		t.Fatal("CreateAgentEditorAgentFunc was not called")
	}
}

func TestDropAgent_Mock(t *testing.T) {
	mod := mkModule("M")
	a := &types.Agent{
		BaseElement: model.BaseElement{ID: nextID("aea")},
		ContainerID: mod.ID,
		Name:        "Summarizer",
	}

	h := mkHierarchy(mod)
	withContainer(h, a.ContainerID, mod.ID)

	called := false
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return []*types.Agent{a}, nil },
		DeleteAgentEditorAgentFunc: func(id string) error {
			called = true
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := ExecDropAgentFn(ctx, &ast.DropAgentStmt{
		Name: ast.QualifiedName{Module: "M", Name: "Summarizer"},
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Dropped agent: M.Summarizer")
	if !called {
		t.Fatal("DeleteAgentEditorAgentFunc was not called")
	}
}

func TestShowAgentEditorModels_Mock(t *testing.T) {
	mod := mkModule("M")
	m1 := &types.Model{
		BaseElement: model.BaseElement{ID: nextID("aem")},
		ContainerID: mod.ID,
		Name:        "GPT4",
		Provider:    "MxCloudGenAI",
		DisplayName: "GPT-4 Turbo",
		Key:         &types.ConstantRef{QualifiedName: "M.APIKey"},
	}

	h := mkHierarchy(mod)
	withContainer(h, m1.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return []*types.Model{m1}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listAgentEditorModels(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "Qualified Name")
	assertContainsStr(t, out, "Module")
	assertContainsStr(t, out, "Provider")
	assertContainsStr(t, out, "Key Constant")
	assertContainsStr(t, out, "Display Name")
	assertContainsStr(t, out, "M.GPT4")
	assertContainsStr(t, out, "MxCloudGenAI")
	assertContainsStr(t, out, "M.APIKey")
	assertContainsStr(t, out, "GPT-4 Turbo")
}

func TestDescribeAgentEditorModel_Mock(t *testing.T) {
	mod := mkModule("M")
	m1 := &types.Model{
		BaseElement: model.BaseElement{ID: nextID("aem")},
		ContainerID: mod.ID,
		Name:        "GPT4",
		Provider:    "MxCloudGenAI",
		Key:         &types.ConstantRef{QualifiedName: "M.APIKey"},
	}

	h := mkHierarchy(mod)
	withContainer(h, m1.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return []*types.Model{m1}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeAgentEditorModel(ctx, ast.QualifiedName{Module: "M", Name: "GPT4"}))

	out := buf.String()
	assertContainsStr(t, out, "create model")
	assertContainsStr(t, out, "Provider")
	assertContainsStr(t, out, "Key")
}

func TestShowAgentEditorAgents_Mock(t *testing.T) {
	mod := mkModule("M")
	a1 := &types.Agent{
		BaseElement: model.BaseElement{ID: nextID("aea")},
		ContainerID: mod.ID,
		Name:        "MyAgent",
		UsageType:   "Chat",
		Model:       &types.DocRef{QualifiedName: "M.GPT4"},
		Tools:       []types.AgentTool{{ID: "t1", Enabled: true}},
		KBTools:     []types.AgentKBTool{{ID: "kb1", Enabled: true}},
	}

	h := mkHierarchy(mod)
	withContainer(h, a1.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return []*types.Agent{a1}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listAgentEditorAgents(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "Qualified Name")
	assertContainsStr(t, out, "Usage")
	assertContainsStr(t, out, "Model")
	assertContainsStr(t, out, "Tools")
	assertContainsStr(t, out, "KBs")
	assertContainsStr(t, out, "M.MyAgent")
	assertContainsStr(t, out, "Chat")
	assertContainsStr(t, out, "M.GPT4")
}

func TestDescribeAgentEditorAgent_Mock(t *testing.T) {
	mod := mkModule("M")
	a1 := &types.Agent{
		BaseElement: model.BaseElement{ID: nextID("aea")},
		ContainerID: mod.ID,
		Name:        "MyAgent",
		UsageType:   "Chat",
		Model:       &types.DocRef{QualifiedName: "M.GPT4"},
	}

	h := mkHierarchy(mod)
	withContainer(h, a1.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return []*types.Agent{a1}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeAgentEditorAgent(ctx, ast.QualifiedName{Module: "M", Name: "MyAgent"}))

	out := buf.String()
	assertContainsStr(t, out, "create agent")
	assertContainsStr(t, out, "UsageType")
	assertContainsStr(t, out, "Model")
}

func TestShowAgentEditorKnowledgeBases_Mock(t *testing.T) {
	mod := mkModule("M")
	kb := &types.KnowledgeBase{
		BaseElement:      model.BaseElement{ID: nextID("aekb")},
		ContainerID:      mod.ID,
		Name:             "MyKB",
		Provider:         "MxCloudGenAI",
		Key:              &types.ConstantRef{QualifiedName: "M.KBKey"},
		ModelDisplayName: "text-embedding-ada-002",
	}

	h := mkHierarchy(mod)
	withContainer(h, kb.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:                   func() bool { return true },
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return []*types.KnowledgeBase{kb}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listAgentEditorKnowledgeBases(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "Qualified Name")
	assertContainsStr(t, out, "Provider")
	assertContainsStr(t, out, "Key Constant")
	assertContainsStr(t, out, "Embedding Model")
	assertContainsStr(t, out, "M.MyKB")
	assertContainsStr(t, out, "MxCloudGenAI")
	assertContainsStr(t, out, "M.KBKey")
	assertContainsStr(t, out, "text-embedding-ada-002")
}

func TestDescribeAgentEditorKnowledgeBase_Mock(t *testing.T) {
	mod := mkModule("M")
	kb := &types.KnowledgeBase{
		BaseElement: model.BaseElement{ID: nextID("aekb")},
		ContainerID: mod.ID,
		Name:        "MyKB",
		Provider:    "MxCloudGenAI",
		Key:         &types.ConstantRef{QualifiedName: "M.KBKey"},
	}

	h := mkHierarchy(mod)
	withContainer(h, kb.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:                   func() bool { return true },
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return []*types.KnowledgeBase{kb}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeAgentEditorKnowledgeBase(ctx, ast.QualifiedName{Module: "M", Name: "MyKB"}))

	out := buf.String()
	assertContainsStr(t, out, "create knowledge base")
	assertContainsStr(t, out, "Provider")
}

func TestShowAgentEditorConsumedMCPServices_Mock(t *testing.T) {
	mod := mkModule("M")
	svc := &types.ConsumedMCPService{
		BaseElement:              model.BaseElement{ID: nextID("aemcp")},
		ContainerID:              mod.ID,
		Name:                     "MySvc",
		ProtocolVersion:          "2025-03-26",
		Version:                  "1.0.0",
		ConnectionTimeoutSeconds: 30,
	}

	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:                        func() bool { return true },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) { return []*types.ConsumedMCPService{svc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listAgentEditorConsumedMCPServices(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "Qualified Name")
	assertContainsStr(t, out, "Protocol")
	assertContainsStr(t, out, "Version")
	assertContainsStr(t, out, "Timeout")
	assertContainsStr(t, out, "M.MySvc")
	assertContainsStr(t, out, "2025-03-26")
	assertContainsStr(t, out, "1.0.0")
}

func TestDescribeAgentEditorConsumedMCPService_Mock(t *testing.T) {
	mod := mkModule("M")
	svc := &types.ConsumedMCPService{
		BaseElement:              model.BaseElement{ID: nextID("aemcp")},
		ContainerID:              mod.ID,
		Name:                     "MySvc",
		ProtocolVersion:          "2025-03-26",
		Version:                  "1.0.0",
		ConnectionTimeoutSeconds: 30,
	}

	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:                        func() bool { return true },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) { return []*types.ConsumedMCPService{svc}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeAgentEditorConsumedMCPService(ctx, ast.QualifiedName{Module: "M", Name: "MySvc"}))

	out := buf.String()
	assertContainsStr(t, out, "create consumed mcp service")
	// ProtocolVersion with a date value must be quoted so the output is re-parseable (issue #435)
	assertContainsStr(t, out, "ProtocolVersion: '2025-03-26'")

	// Roundtrip: DESCRIBE output must parse without errors
	_, parseErrs := visitor.Build(out)
	if len(parseErrs) > 0 {
		t.Errorf("describe output is not valid MDL: %v\nOutput:\n%s", parseErrs[0], out)
	}
}

// ---------------------------------------------------------------------------
// DESCRIBE — Not Found
// ---------------------------------------------------------------------------

func TestDescribeAgentEditorModel_Mock_NotFound(t *testing.T) {
	mod := mkModule("M")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, describeAgentEditorModel(ctx, ast.QualifiedName{Module: "M", Name: "NonExistent"}))
}

func TestDescribeAgentEditorAgent_Mock_NotFound(t *testing.T) {
	mod := mkModule("M")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, describeAgentEditorAgent(ctx, ast.QualifiedName{Module: "M", Name: "NonExistent"}))
}

func TestDescribeAgentEditorKnowledgeBase_Mock_NotFound(t *testing.T) {
	mod := mkModule("M")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:                   func() bool { return true },
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, describeAgentEditorKnowledgeBase(ctx, ast.QualifiedName{Module: "M", Name: "NonExistent"}))
}

func TestDescribeAgentEditorConsumedMCPService_Mock_NotFound(t *testing.T) {
	mod := mkModule("M")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:                        func() bool { return true },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, describeAgentEditorConsumedMCPService(ctx, ast.QualifiedName{Module: "M", Name: "NonExistent"}))
}

// ---------------------------------------------------------------------------
// DROP — Not Found
// ---------------------------------------------------------------------------

func TestDropAgentEditorModel_Mock_NotFound(t *testing.T) {
	mod := mkModule("M")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, execDropAgentEditorModel(ctx, &ast.DropModelStmt{
		Name: ast.QualifiedName{Module: "M", Name: "NonExistent"},
	}))
}

func TestDropConsumedMCPService_Mock_NotFound(t *testing.T) {
	mod := mkModule("M")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:                        func() bool { return true },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, ExecDropConsumedMCPServiceFn(ctx, &ast.DropConsumedMCPServiceStmt{
		Name: ast.QualifiedName{Module: "M", Name: "NonExistent"},
	}, execContextToDeps(ctx)))
}

func TestDropKnowledgeBase_Mock_NotFound(t *testing.T) {
	mod := mkModule("M")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:                   func() bool { return true },
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, ExecDropKnowledgeBaseFn(ctx, &ast.DropKnowledgeBaseStmt{
		Name: ast.QualifiedName{Module: "M", Name: "NonExistent"},
	}, execContextToDeps(ctx)))
}

func TestDropAgent_Mock_NotFound(t *testing.T) {
	mod := mkModule("M")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, ExecDropAgentFn(ctx, &ast.DropAgentStmt{
		Name: ast.QualifiedName{Module: "M", Name: "NonExistent"},
	}, execContextToDeps(ctx)))
}

// ---------------------------------------------------------------------------
// LIST — Filter by Module
// ---------------------------------------------------------------------------

func TestShowAgentEditorModels_Mock_FilterByModule(t *testing.T) {
	mod1 := mkModule("A")
	mod2 := mkModule("B")
	m1 := &types.Model{
		BaseElement: model.BaseElement{ID: nextID("aem")},
		ContainerID: mod1.ID,
		Name:        "M1",
	}
	m2 := &types.Model{
		BaseElement: model.BaseElement{ID: nextID("aem")},
		ContainerID: mod2.ID,
		Name:        "M2",
	}

	h := mkHierarchy(mod1, mod2)
	withContainer(h, m1.ContainerID, mod1.ID)
	withContainer(h, m2.ContainerID, mod2.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return []*types.Model{m1, m2}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listAgentEditorModels(ctx, "B"))

	out := buf.String()
	assertNotContainsStr(t, out, "A.M1")
	assertContainsStr(t, out, "B.M2")
}

func TestShowAgentEditorAgents_Mock_FilterByModule(t *testing.T) {
	mod1 := mkModule("A")
	mod2 := mkModule("B")
	a1 := &types.Agent{
		BaseElement: model.BaseElement{ID: nextID("aea")},
		ContainerID: mod1.ID,
		Name:        "Agent1",
	}
	a2 := &types.Agent{
		BaseElement: model.BaseElement{ID: nextID("aea")},
		ContainerID: mod2.ID,
		Name:        "Agent2",
	}

	h := mkHierarchy(mod1, mod2)
	withContainer(h, a1.ContainerID, mod1.ID)
	withContainer(h, a2.ContainerID, mod2.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return []*types.Agent{a1, a2}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listAgentEditorAgents(ctx, "B"))

	out := buf.String()
	assertNotContainsStr(t, out, "A.Agent1")
	assertContainsStr(t, out, "B.Agent2")
}

// ---------------------------------------------------------------------------
// CREATE OR MODIFY — preserves UUID across all agent editor types
// ---------------------------------------------------------------------------

func TestCreateAgentEditorModel_OrModify_PreservesID(t *testing.T) {
	mod := mkModule("M")
	existingID := nextID("aem")
	existing := &types.Model{
		BaseElement: model.BaseElement{ID: existingID},
		ContainerID: mod.ID,
		Name:        "GPT4",
	}
	h := mkHierarchy(mod)
	withContainer(h, existing.ContainerID, mod.ID)

	var updatedID model.ID
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListModulesFunc:           func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return []*types.Model{existing}, nil },
		ListConstantsFunc:         func() ([]*model.Constant, error) { return nil, nil },
		UpdateAgentEditorModelFunc: func(m *types.Model) error {
			updatedID = m.ID
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execCreateAgentEditorModel(ctx, &ast.CreateModelStmt{
		Name:           ast.QualifiedName{Module: "M", Name: "GPT4"},
		Provider:       "MxCloudGenAI",
		CreateOrModify: true,
	})
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Modified model")
	if updatedID != existingID {
		t.Errorf("UpdateAgentEditorModel called with ID %q, want %q", updatedID, existingID)
	}
}

func TestCreateConsumedMCPService_OrModify_PreservesID(t *testing.T) {
	mod := mkModule("M")
	existingID := nextID("aemcp")
	existing := &types.ConsumedMCPService{
		BaseElement: model.BaseElement{ID: existingID},
		ContainerID: mod.ID,
		Name:        "WebSearch",
	}
	h := mkHierarchy(mod)
	withContainer(h, existing.ContainerID, mod.ID)

	var updatedID model.ID
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) {
			return []*types.ConsumedMCPService{existing}, nil
		},
		UpdateAgentEditorConsumedMCPServiceFunc: func(c *types.ConsumedMCPService) error {
			updatedID = c.ID
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := ExecCreateConsumedMCPServiceFn(ctx, &ast.CreateConsumedMCPServiceStmt{
		Name:            ast.QualifiedName{Module: "M", Name: "WebSearch"},
		ProtocolVersion: "v2025_03_26",
		CreateOrModify:  true,
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Modified consumed mcp service")
	if updatedID != existingID {
		t.Errorf("UpdateAgentEditorConsumedMCPService called with ID %q, want %q", updatedID, existingID)
	}
}

func TestCreateKnowledgeBase_OrModify_PreservesID(t *testing.T) {
	mod := mkModule("M")
	existingID := nextID("aekb")
	existing := &types.KnowledgeBase{
		BaseElement: model.BaseElement{ID: existingID},
		ContainerID: mod.ID,
		Name:        "Docs",
	}
	h := mkHierarchy(mod)
	withContainer(h, existing.ContainerID, mod.ID)

	var updatedID model.ID
	mb := &mock.MockBackend{
		IsConnectedFunc:                    func() bool { return true },
		ListModulesFunc:                    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorKnowledgeBasesFunc:  func() ([]*types.KnowledgeBase, error) { return []*types.KnowledgeBase{existing}, nil },
		ListConstantsFunc:                  func() ([]*model.Constant, error) { return nil, nil },
		UpdateAgentEditorKnowledgeBaseFunc: func(k *types.KnowledgeBase) error { updatedID = k.ID; return nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := ExecCreateKnowledgeBaseFn(ctx, &ast.CreateKnowledgeBaseStmt{
		Name:           ast.QualifiedName{Module: "M", Name: "Docs"},
		Provider:       "MxCloudGenAI",
		CreateOrModify: true,
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Modified knowledge base")
	if updatedID != existingID {
		t.Errorf("UpdateAgentEditorKnowledgeBase called with ID %q, want %q", updatedID, existingID)
	}
}

func TestCreateAgent_OrModify_PreservesID(t *testing.T) {
	mod := mkModule("M")
	existingID := nextID("aeag")
	existing := &types.Agent{
		BaseElement: model.BaseElement{ID: existingID},
		ContainerID: mod.ID,
		Name:        "Assistant",
	}
	h := mkHierarchy(mod)
	withContainer(h, existing.ContainerID, mod.ID)

	var updatedID model.ID
	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListModulesFunc:           func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorAgentsFunc: func() ([]*types.Agent, error) { return []*types.Agent{existing}, nil },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return nil, nil },
		ListAgentEditorConsumedMCPServicesFunc: func() ([]*types.ConsumedMCPService, error) {
			return nil, nil
		},
		ListAgentEditorKnowledgeBasesFunc: func() ([]*types.KnowledgeBase, error) { return nil, nil },
		UpdateAgentEditorAgentFunc: func(a *types.Agent) error {
			updatedID = a.ID
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := ExecCreateAgentFn(ctx, &ast.CreateAgentStmt{
		Name:           ast.QualifiedName{Module: "M", Name: "Assistant"},
		UsageType:      "UserInitiated",
		CreateOrModify: true,
	}, execContextToDeps(ctx))
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Modified agent")
	if updatedID != existingID {
		t.Errorf("UpdateAgentEditorAgent called with ID %q, want %q", updatedID, existingID)
	}
}

func TestCreateAgentEditorModel_AlreadyExists_NoOrModify(t *testing.T) {
	mod := mkModule("M")
	existing := &types.Model{
		BaseElement: model.BaseElement{ID: nextID("aem")},
		ContainerID: mod.ID,
		Name:        "GPT4",
	}
	h := mkHierarchy(mod)
	withContainer(h, existing.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:           func() bool { return true },
		ListModulesFunc:           func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListAgentEditorModelsFunc: func() ([]*types.Model, error) { return []*types.Model{existing}, nil },
		ListConstantsFunc:         func() ([]*model.Constant, error) { return nil, nil },
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execCreateAgentEditorModel(ctx, &ast.CreateModelStmt{
		Name: ast.QualifiedName{Module: "M", Name: "GPT4"},
	})
	assertError(t, err)
}
