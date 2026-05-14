// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3 Phase D — gen-typed write path tests.

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

func TestBuildJumpToGenActivity(t *testing.T) {
	n := &ast.WorkflowJumpToNode{Target: "Approve", Caption: "back"}
	got := buildJumpToGenActivity(n)
	if got == nil {
		t.Fatal("nil result")
	}
	if got.Name() != "Approve" {
		t.Errorf("Name = %q, want Approve", got.Name())
	}
	if got.Caption() != "back" {
		t.Errorf("Caption = %q, want back", got.Caption())
	}
	if got.TargetActivityQualifiedName() != "Approve" {
		t.Errorf("Target = %q, want Approve", got.TargetActivityQualifiedName())
	}
	if got.TypeName() != "Workflows$JumpToActivity" {
		t.Errorf("TypeName = %q", got.TypeName())
	}
	if got.ID() == "" {
		t.Error("ID was not generated")
	}
}

func TestBuildJumpToGenActivity_DefaultCaption(t *testing.T) {
	n := &ast.WorkflowJumpToNode{Target: "Approve"}
	got := buildJumpToGenActivity(n)
	if got.Caption() != "Approve" {
		t.Errorf("default caption should be target name, got %q", got.Caption())
	}
}

func TestBuildWaitForTimerGenActivity(t *testing.T) {
	n := &ast.WorkflowWaitForTimerNode{DelayExpression: "dateTime(2026,1,1)", Caption: "hold"}
	got := buildWaitForTimerGenActivity(n)
	if got.Delay() != "dateTime(2026,1,1)" {
		t.Errorf("Delay = %q", got.Delay())
	}
	if got.Caption() != "hold" {
		t.Errorf("Caption = %q", got.Caption())
	}
	if got.Name() != "hold" {
		t.Errorf("Name = %q (should mirror caption)", got.Name())
	}
	if got.TypeName() != "Workflows$WaitForTimerActivity" {
		t.Errorf("TypeName = %q", got.TypeName())
	}
}

func TestBuildWaitForTimerGenActivity_DefaultCaption(t *testing.T) {
	got := buildWaitForTimerGenActivity(&ast.WorkflowWaitForTimerNode{})
	if got.Caption() != "Wait for timer" {
		t.Errorf("default caption = %q", got.Caption())
	}
}

func TestBuildWaitForNotificationGenActivity_WithBoundaryEvents(t *testing.T) {
	n := &ast.WorkflowWaitForNotificationNode{
		Caption: "wait",
		BoundaryEvents: []ast.WorkflowBoundaryEventNode{
			{EventType: "InterruptingTimer", Delay: "${PT1H}"},
		},
	}
	got := buildWaitForNotificationGenActivity(n)
	if got.Caption() != "wait" {
		t.Errorf("Caption = %q", got.Caption())
	}
	if got.TypeName() != "Workflows$WaitForNotificationActivity" {
		t.Errorf("TypeName = %q", got.TypeName())
	}
	bes := got.BoundaryEventsItems()
	if len(bes) != 1 {
		t.Fatalf("expected 1 boundary event, got %d", len(bes))
	}
	if bes[0].TypeName() != "Workflows$InterruptingTimerBoundaryEvent" {
		t.Errorf("BE TypeName = %q", bes[0].TypeName())
	}
	if be, ok := bes[0].(*genWf.InterruptingTimerBoundaryEvent); ok {
		if be.Delay() != "${PT1H}" {
			t.Errorf("Delay = %q", be.Delay())
		}
		if !be.IsInterrupting() {
			t.Error("IsInterrupting should be true for InterruptingTimer")
		}
	}
}

func TestBuildEndWorkflowGenActivity_Default(t *testing.T) {
	got := buildEndWorkflowGenActivity(&ast.WorkflowEndNode{})
	if got.Caption() != "End" {
		t.Errorf("default caption = %q", got.Caption())
	}
	if got.Name() != "End" {
		t.Errorf("Name = %q", got.Name())
	}
}

func TestBuildAnnotationActivityGen(t *testing.T) {
	got := buildAnnotationActivityGen(&ast.WorkflowAnnotationActivityNode{Text: "note"})
	if got.Description() != "note" {
		t.Errorf("Description = %q", got.Description())
	}
	if got.TypeName() != "Workflows$FloatingAnnotation" {
		t.Errorf("TypeName = %q (should be FloatingAnnotation, the inflow positioned variant)", got.TypeName())
	}
}

func TestBuildBoundaryEventGen_NonInterrupting(t *testing.T) {
	be := buildBoundaryEventGen(ast.WorkflowBoundaryEventNode{
		EventType: "NonInterruptingTimer",
		Delay:     "${PT5M}",
	})
	if be == nil {
		t.Fatal("nil result")
	}
	if be.TypeName() != "Workflows$NonInterruptingTimerBoundaryEvent" {
		t.Errorf("TypeName = %q", be.TypeName())
	}
	if v, ok := be.(*genWf.NonInterruptingTimerBoundaryEvent); ok {
		if v.IsInterrupting() {
			t.Error("IsInterrupting should be false for NonInterruptingTimer")
		}
	}
}

func TestBuildBoundaryEventGen_DefaultTimer(t *testing.T) {
	be := buildBoundaryEventGen(ast.WorkflowBoundaryEventNode{
		EventType: "Timer",
		Delay:     "${PT5M}",
	})
	if be.TypeName() != "Workflows$TimerBoundaryEvent" {
		t.Errorf("TypeName = %q", be.TypeName())
	}
}

func TestBuildBoundaryEventGen_WithSubFlow(t *testing.T) {
	be := buildBoundaryEventGen(ast.WorkflowBoundaryEventNode{
		EventType: "InterruptingTimer",
		Delay:     "${PT1M}",
		Activities: []ast.WorkflowActivityNode{
			&ast.WorkflowJumpToNode{Target: "X"},
		},
	})
	v, ok := be.(*genWf.InterruptingTimerBoundaryEvent)
	if !ok {
		t.Fatalf("wrong type: %T", be)
	}
	flow, ok := v.Flow().(*genWf.Flow)
	if !ok || flow == nil {
		t.Fatal("expected non-nil Flow")
	}
	if len(flow.ActivitiesItems()) != 1 {
		t.Errorf("expected 1 nested activity, got %d", len(flow.ActivitiesItems()))
	}
}

// ── D1.b — UserTask family tests ──────────────────────────────────────

func TestBuildUserTaskGenActivity_SingleByDefault(t *testing.T) {
	n := &ast.WorkflowUserTaskNode{Name: "Approve", Caption: "approve"}
	got := buildUserTaskGenActivity(n)
	if got.TypeName() != "Workflows$SingleUserTaskActivity" {
		t.Errorf("TypeName = %q, want SingleUserTaskActivity", got.TypeName())
	}
}

func TestBuildUserTaskGenActivity_MultiWhenFlagSet(t *testing.T) {
	n := &ast.WorkflowUserTaskNode{Name: "Vote", IsMultiUser: true}
	got := buildUserTaskGenActivity(n)
	if got.TypeName() != "Workflows$MultiUserTaskActivity" {
		t.Errorf("TypeName = %q, want MultiUserTaskActivity", got.TypeName())
	}
}

func TestBuildSingleUserTaskGenActivity_FieldsPropagate(t *testing.T) {
	n := &ast.WorkflowUserTaskNode{
		Name:            "Step",
		Caption:         "step caption",
		DueDate:         "[%CurrentDateTime%]",
		TaskDescription: "do this",
	}
	got := buildSingleUserTaskGenActivity(n)
	if got.Name() != "Step" {
		t.Errorf("Name = %q", got.Name())
	}
	if got.Caption() != "step caption" {
		t.Errorf("Caption = %q", got.Caption())
	}
	if got.DueDate() != "[%CurrentDateTime%]" {
		t.Errorf("DueDate = %q", got.DueDate())
	}
	if got.TaskDescription() == nil {
		t.Error("TaskDescription was not set (Texts$Text wrapper missing)")
	}
}

func TestBuildUserSourceGen_AllKinds(t *testing.T) {
	cases := []struct {
		name   string
		t      ast.WorkflowTargetingNode
		expect string
	}{
		{
			name:   "microflow",
			t:      ast.WorkflowTargetingNode{Kind: "microflow", Microflow: ast.QualifiedName{Module: "M", Name: "Pick"}},
			expect: "Workflows$MicroflowBasedUserSource",
		},
		{
			name:   "xpath",
			t:      ast.WorkflowTargetingNode{Kind: "xpath", XPath: "//User"},
			expect: "Workflows$XPathBasedUserSource",
		},
		{
			name:   "group_microflow",
			t:      ast.WorkflowTargetingNode{Kind: "group_microflow", Microflow: ast.QualifiedName{Module: "M", Name: "Group"}},
			expect: "Workflows$MicroflowGroupTargeting",
		},
		{
			name:   "group_xpath",
			t:      ast.WorkflowTargetingNode{Kind: "group_xpath", XPath: "//Group"},
			expect: "Workflows$XPathGroupTargeting",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := buildUserSourceGen(tc.t)
			if src == nil {
				t.Fatal("nil source")
			}
			if src.TypeName() != tc.expect {
				t.Errorf("TypeName = %q, want %q", src.TypeName(), tc.expect)
			}
		})
	}
}

func TestBuildUserSourceGen_EmptyKindReturnsNil(t *testing.T) {
	src := buildUserSourceGen(ast.WorkflowTargetingNode{Kind: ""})
	if src != nil {
		t.Errorf("expected nil for empty kind, got %v", src)
	}
}

func TestBuildUserTaskOutcomesGen_ValueMatchesCaption(t *testing.T) {
	nodes := []ast.WorkflowUserTaskOutcomeNode{
		{Caption: "Approved"},
		{Caption: "Rejected", Activities: []ast.WorkflowActivityNode{
			&ast.WorkflowJumpToNode{Target: "X"},
		}},
	}
	out := buildUserTaskOutcomesGen(nodes)
	if len(out) != 2 {
		t.Fatalf("expected 2 outcomes, got %d", len(out))
	}
	if out[0].Value() != "Approved" || out[0].Name() != "Approved" || out[0].Caption() != "Approved" {
		t.Errorf("outcome 0 fields mismatch: V=%q N=%q C=%q", out[0].Value(), out[0].Name(), out[0].Caption())
	}
	if out[0].Flow() != nil {
		t.Error("outcome with no activities should have nil Flow")
	}
	if out[1].Flow() == nil {
		t.Error("outcome with activities should have non-nil Flow")
	}
}

// ── D1.c — CallMicroflow / CallWorkflow / ParameterMapping tests ──────

func TestBuildCallMicroflowGenActivity_FullyQualifiedMappings(t *testing.T) {
	n := &ast.WorkflowCallMicroflowNode{
		Microflow: ast.QualifiedName{Module: "Demo", Name: "Action"},
		Caption:   "do thing",
		ParameterMappings: []ast.WorkflowParameterMappingNode{
			{Parameter: "X", Expression: "$WorkflowContext"},
		},
	}
	got := buildCallMicroflowGenActivity(n)
	if got.TypeName() != "Workflows$CallMicroflowActivity" {
		t.Errorf("TypeName = %q, want CallMicroflowActivity", got.TypeName())
	}
	if got.Name() != "Action" {
		t.Errorf("Name = %q", got.Name())
	}
	if got.MicroflowQualifiedName() != "Demo.Action" {
		t.Errorf("MicroflowQualifiedName = %q", got.MicroflowQualifiedName())
	}
	pms := got.ParameterMappingsItems()
	if len(pms) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(pms))
	}
	pm, _ := pms[0].(*genWf.MicroflowCallParameterMapping)
	if pm == nil {
		t.Fatalf("wrong mapping type: %T", pms[0])
	}
	if pm.ParameterQualifiedName() != "Demo.Action.X" {
		t.Errorf("Parameter = %q (BSON requires fully qualified)", pm.ParameterQualifiedName())
	}
	if pm.Expression() != "$WorkflowContext" {
		t.Errorf("Expression = %q", pm.Expression())
	}
}

func TestBuildCallMicroflowGenActivity_DefaultCaptionMirrorsName(t *testing.T) {
	n := &ast.WorkflowCallMicroflowNode{
		Microflow: ast.QualifiedName{Module: "M", Name: "DoIt"},
	}
	got := buildCallMicroflowGenActivity(n)
	if got.Caption() != "DoIt" {
		t.Errorf("default caption = %q, want microflow simple name", got.Caption())
	}
}

func TestBuildCallWorkflowGenActivity_AutoBindsParameterExpression(t *testing.T) {
	n := &ast.WorkflowCallWorkflowNode{
		Workflow: ast.QualifiedName{Module: "Demo", Name: "Sub"},
		Caption:  "call sub",
		ParameterMappings: []ast.WorkflowParameterMappingNode{
			{Parameter: "WorkflowContext", Expression: "$WorkflowContext"},
		},
	}
	got := buildCallWorkflowGenActivity(n)
	if got.TypeName() != "Workflows$CallWorkflowActivity" {
		t.Errorf("TypeName = %q", got.TypeName())
	}
	if got.WorkflowQualifiedName() != "Demo.Sub" {
		t.Errorf("WorkflowQualifiedName = %q", got.WorkflowQualifiedName())
	}
	if got.ParameterExpression() != "$WorkflowContext" {
		t.Errorf("ParameterExpression = %q (should auto-bind)", got.ParameterExpression())
	}
	pms := got.ParameterMappingsItems()
	if len(pms) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(pms))
	}
	pm, _ := pms[0].(*genWf.WorkflowCallParameterMapping)
	if pm == nil {
		t.Fatalf("wrong mapping type: %T", pms[0])
	}
	if pm.ParameterQualifiedName() != "Demo.Sub.WorkflowContext" {
		t.Errorf("Parameter = %q", pm.ParameterQualifiedName())
	}
}

func TestBuildConditionOutcomeGen_True(t *testing.T) {
	got := buildConditionOutcomeGen(ast.WorkflowConditionOutcomeNode{Value: "True"})
	bo, ok := got.(*genWf.BooleanConditionOutcome)
	if !ok {
		t.Fatalf("wrong type: %T", got)
	}
	if !bo.Value() {
		t.Error("Value should be true for 'True' AST")
	}
}

func TestBuildConditionOutcomeGen_False(t *testing.T) {
	got := buildConditionOutcomeGen(ast.WorkflowConditionOutcomeNode{Value: "False"})
	bo, ok := got.(*genWf.BooleanConditionOutcome)
	if !ok {
		t.Fatalf("wrong type: %T", got)
	}
	if bo.Value() {
		t.Error("Value should be false for 'False' AST")
	}
}

func TestBuildConditionOutcomeGen_Default(t *testing.T) {
	got := buildConditionOutcomeGen(ast.WorkflowConditionOutcomeNode{Value: "Default"})
	if got.TypeName() != "Workflows$VoidConditionOutcome" {
		t.Errorf("TypeName = %q, want VoidConditionOutcome", got.TypeName())
	}
}

func TestBuildConditionOutcomeGen_Enumeration(t *testing.T) {
	got := buildConditionOutcomeGen(ast.WorkflowConditionOutcomeNode{Value: "MyModule.MyEnum.OptionA"})
	ev, ok := got.(*genWf.EnumerationValueConditionOutcome)
	if !ok {
		t.Fatalf("wrong type: %T", got)
	}
	if ev.ValueQualifiedName() != "MyModule.MyEnum.OptionA" {
		t.Errorf("ValueQualifiedName = %q", ev.ValueQualifiedName())
	}
}

func TestBuildConditionOutcomeGen_WithSubFlow(t *testing.T) {
	got := buildConditionOutcomeGen(ast.WorkflowConditionOutcomeNode{
		Value: "True",
		Activities: []ast.WorkflowActivityNode{
			&ast.WorkflowJumpToNode{Target: "X"},
		},
	})
	bo, _ := got.(*genWf.BooleanConditionOutcome)
	flow, ok := bo.Flow().(*genWf.Flow)
	if !ok || flow == nil {
		t.Fatal("expected non-nil Flow")
	}
	if len(flow.ActivitiesItems()) != 1 {
		t.Errorf("nested activities = %d", len(flow.ActivitiesItems()))
	}
}

// ── D1.d — ExclusiveSplit / ParallelSplit tests ───────────────────────

func TestBuildExclusiveSplitGenActivity_BasicShape(t *testing.T) {
	n := &ast.WorkflowDecisionNode{
		Expression: "$ctx/score > 50",
		Caption:    "score check",
		Outcomes: []ast.WorkflowConditionOutcomeNode{
			{Value: "True"},
			{Value: "False"},
		},
	}
	got := buildExclusiveSplitGenActivity(n)
	if got.TypeName() != "Workflows$ExclusiveSplitActivity" {
		t.Errorf("TypeName = %q", got.TypeName())
	}
	if got.Expression() != "$ctx/score > 50" {
		t.Errorf("Expression = %q", got.Expression())
	}
	if got.Caption() != "score check" {
		t.Errorf("Caption = %q", got.Caption())
	}
	if len(got.OutcomesItems()) != 2 {
		t.Errorf("expected 2 outcomes, got %d", len(got.OutcomesItems()))
	}
}

func TestBuildExclusiveSplitGenActivity_DropsDefaultOnBooleanDecision(t *testing.T) {
	n := &ast.WorkflowDecisionNode{
		Outcomes: []ast.WorkflowConditionOutcomeNode{
			{Value: "True"},
			{Value: "False"},
			{Value: "Default"}, // must be dropped — Mendix 11 runtime rejects VoidConditionOutcome on a boolean decision
		},
	}
	got := buildExclusiveSplitGenActivity(n)
	if len(got.OutcomesItems()) != 2 {
		t.Errorf("expected 2 outcomes (Default dropped), got %d", len(got.OutcomesItems()))
	}
	for _, oc := range got.OutcomesItems() {
		if oc.TypeName() == "Workflows$VoidConditionOutcome" {
			t.Error("Default outcome should be dropped on boolean decision")
		}
	}
}

func TestBuildExclusiveSplitGenActivity_DefaultCaption(t *testing.T) {
	got := buildExclusiveSplitGenActivity(&ast.WorkflowDecisionNode{})
	if got.Caption() != "Decision" {
		t.Errorf("default caption = %q", got.Caption())
	}
	if got.Name() != "Decision" {
		t.Errorf("default name = %q", got.Name())
	}
}

func TestBuildParallelSplitGenActivity_PathsBecomeOutcomes(t *testing.T) {
	n := &ast.WorkflowParallelSplitNode{
		Caption: "fork",
		Paths: []ast.WorkflowParallelPathNode{
			{Activities: []ast.WorkflowActivityNode{&ast.WorkflowJumpToNode{Target: "X"}}},
			{Activities: nil},
		},
	}
	got := buildParallelSplitGenActivity(n)
	if got.TypeName() != "Workflows$ParallelSplitActivity" {
		t.Errorf("TypeName = %q", got.TypeName())
	}
	if got.Caption() != "fork" {
		t.Errorf("Caption = %q", got.Caption())
	}
	outcomes := got.OutcomesItems()
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %d", len(outcomes))
	}
	if pso, ok := outcomes[0].(*genWf.ParallelSplitOutcome); ok {
		if pso.Flow() == nil {
			t.Error("first outcome should have non-nil Flow")
		}
	}
	if pso, ok := outcomes[1].(*genWf.ParallelSplitOutcome); ok {
		if pso.Flow() != nil {
			t.Error("second outcome (no activities) should have nil Flow")
		}
	}
}

func TestBuildParallelSplitGenActivity_DefaultCaption(t *testing.T) {
	got := buildParallelSplitGenActivity(&ast.WorkflowParallelSplitNode{})
	if got.Caption() != "Parallel split" {
		t.Errorf("default caption = %q", got.Caption())
	}
}

// ── End-to-end dispatcher coverage ────────────────────────────────────

func TestBuildWorkflowActivityGen_DispatchesAllTypes(t *testing.T) {
	cases := []struct {
		name string
		node ast.WorkflowActivityNode
		want string
	}{
		{"jump", &ast.WorkflowJumpToNode{Target: "T"}, "Workflows$JumpToActivity"},
		{"timer", &ast.WorkflowWaitForTimerNode{}, "Workflows$WaitForTimerActivity"},
		{"notif", &ast.WorkflowWaitForNotificationNode{}, "Workflows$WaitForNotificationActivity"},
		{"end", &ast.WorkflowEndNode{}, "Workflows$EndWorkflowActivity"},
		{"anno", &ast.WorkflowAnnotationActivityNode{Text: "n"}, "Workflows$FloatingAnnotation"},
		{"singleut", &ast.WorkflowUserTaskNode{Name: "U"}, "Workflows$SingleUserTaskActivity"},
		{"multiut", &ast.WorkflowUserTaskNode{Name: "U", IsMultiUser: true}, "Workflows$MultiUserTaskActivity"},
		{"callmf", &ast.WorkflowCallMicroflowNode{Microflow: ast.QualifiedName{Module: "M", Name: "F"}}, "Workflows$CallMicroflowActivity"},
		{"callwf", &ast.WorkflowCallWorkflowNode{Workflow: ast.QualifiedName{Module: "M", Name: "W"}}, "Workflows$CallWorkflowActivity"},
		{"decision", &ast.WorkflowDecisionNode{}, "Workflows$ExclusiveSplitActivity"},
		{"parallel", &ast.WorkflowParallelSplitNode{}, "Workflows$ParallelSplitActivity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildWorkflowActivityGen(tc.node)
			if got == nil {
				t.Fatal("nil result")
			}
			if got.TypeName() != tc.want {
				t.Errorf("TypeName = %q, want %q", got.TypeName(), tc.want)
			}
		})
	}
}

// ── D2 — execCreateWorkflowGen tests ──────────────────────────────────

func TestExecCreateWorkflowGen_NewUnit_RoutesThroughCreate(t *testing.T) {
	mod := mkModule("BPModule")
	createCalled := false
	var createdWf *genWf.Workflow
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		ListWorkflowsGenFunc: func() ([]*genWf.Workflow, error) {
			return nil, nil
		},
		CreateWorkflowGenFunc: func(parentUUID, containmentName string, wf *genWf.Workflow) error {
			createCalled = true
			createdWf = wf
			if parentUUID != string(mod.ID) {
				t.Errorf("parentUUID = %q, want %q", parentUUID, mod.ID)
			}
			if containmentName != "Documents" {
				t.Errorf("containmentName = %q", containmentName)
			}
			return nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))

	stmt := &ast.CreateWorkflowStmt{
		Name:            ast.QualifiedName{Module: "BPModule", Name: "Approve"},
		Documentation:   "approval",
		ParameterEntity: ast.QualifiedName{Module: "BPModule", Name: "Order"},
		DisplayName:     "Approve order",
		Description:     "Approves the order",
		ExportLevel:     "Hidden",
		DueDate:         "[%CurrentDateTime%]",
		OverviewPage:    ast.QualifiedName{Module: "BPModule", Name: "Overview"},
		Activities: []ast.WorkflowActivityNode{
			&ast.WorkflowUserTaskNode{Name: "Step1", Caption: "first step"},
		},
	}
	if err := execCreateWorkflowGen(ctx, stmt); err != nil {
		t.Fatalf("execCreateWorkflowGen: %v", err)
	}
	if !createCalled {
		t.Fatal("CreateWorkflowGen was not called")
	}
	if !strings.Contains(buf.String(), "Created workflow: BPModule.Approve") {
		t.Errorf("missing user message: %q", buf.String())
	}
	if createdWf.Name() != "Approve" {
		t.Errorf("Name = %q", createdWf.Name())
	}
	if createdWf.Documentation() != "approval" {
		t.Errorf("Documentation = %q", createdWf.Documentation())
	}
	if createdWf.OverviewPageQualifiedName() != "BPModule.Overview" {
		t.Errorf("OverviewPageQualifiedName = %q", createdWf.OverviewPageQualifiedName())
	}
	if createdWf.ExportLevel() != "Hidden" {
		t.Errorf("ExportLevel = %q", createdWf.ExportLevel())
	}
	if createdWf.DueDate() != "[%CurrentDateTime%]" {
		t.Errorf("DueDate = %q", createdWf.DueDate())
	}
	if createdWf.Title() != "Approve order" {
		t.Errorf("Title = %q (legacy decode mirror)", createdWf.Title())
	}
	param, _ := createdWf.Parameter().(*genWf.Parameter)
	if param == nil {
		t.Fatal("Parameter was not set")
	}
	if param.EntityQualifiedName() != "BPModule.Order" {
		t.Errorf("Parameter.Entity = %q", param.EntityQualifiedName())
	}
	flow, _ := createdWf.Flow().(*genWf.Flow)
	if flow == nil {
		t.Fatal("Flow was not set")
	}
	acts := flow.ActivitiesItems()
	// Start + UserTask + End = 3 activities
	if len(acts) != 3 {
		t.Errorf("expected 3 activities (Start + UserTask + End), got %d", len(acts))
	}
	if acts[0].TypeName() != "Workflows$StartWorkflowActivity" {
		t.Errorf("first activity should be Start, got %q", acts[0].TypeName())
	}
	if acts[len(acts)-1].TypeName() != "Workflows$EndWorkflowActivity" {
		t.Errorf("last activity should be End, got %q", acts[len(acts)-1].TypeName())
	}
}

func TestExecCreateWorkflowGen_ExistingWithoutModify_Errors(t *testing.T) {
	mod := mkModule("BPModule")
	wfGen := mkWorkflowGen("WF1", "Approve")
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Workflows = makeWorkflowsRepo([]*genWf.Workflow{wfGen}, mod.ID)

	stmt := &ast.CreateWorkflowStmt{
		Name: ast.QualifiedName{Module: "BPModule", Name: "Approve"},
	}
	err := execCreateWorkflowGen(ctx, stmt)
	if err == nil {
		t.Fatal("expected error for existing workflow without create-or-modify")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected already-exists error, got %v", err)
	}
}

func TestExecCreateWorkflowGen_ExistingWithModify_RoutesThroughUpdate(t *testing.T) {
	mod := mkModule("BPModule")
	wfGen := mkWorkflowGen("WF1", "Approve")
	updateCalled := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		UpdateWorkflowGenFunc: func(wf *genWf.Workflow) error {
			updateCalled = true
			if wf.ID() != "WF1" {
				t.Errorf("expected to preserve UnitID WF1, got %q", wf.ID())
			}
			return nil
		},
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Workflows = makeWorkflowsRepo([]*genWf.Workflow{wfGen}, mod.ID)

	stmt := &ast.CreateWorkflowStmt{
		Name:           ast.QualifiedName{Module: "BPModule", Name: "Approve"},
		CreateOrModify: true,
	}
	if err := execCreateWorkflowGen(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Error("UpdateWorkflowGen was not called")
	}
}

// ── D3 — execDropWorkflowGen tests ────────────────────────────────────

func TestExecDropWorkflowGen_DeletesByID(t *testing.T) {
	mod := mkModule("BPModule")
	wfGen := mkWorkflowGen("WF1", "Approve")
	deleted := false
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
		DeleteWorkflowFunc: func(id model.ID) error {
			deleted = true
			if string(id) != "WF1" {
				t.Errorf("DeleteWorkflow id = %q, want WF1", id)
			}
			return nil
		},
	}
	h := mkHierarchy(mod)
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.Workflows = makeWorkflowsRepo([]*genWf.Workflow{wfGen}, mod.ID)

	stmt := &ast.DropWorkflowStmt{Name: ast.QualifiedName{Module: "BPModule", Name: "Approve"}}
	if err := execDropWorkflowGen(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("DeleteWorkflow was not called")
	}
	if !strings.Contains(buf.String(), "Dropped workflow: BPModule.Approve") {
		t.Errorf("missing user message: %q", buf.String())
	}
}

func TestExecDropWorkflowGen_NotFound(t *testing.T) {
	mod := mkModule("BPModule")
	mb := &mock.MockBackend{
		IsConnectedFunc:      func() bool { return true },
		ListModulesFunc:      func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:      func() ([]*types.FolderInfo, error) { return nil, nil },
		ListWorkflowsGenFunc: func() ([]*genWf.Workflow, error) { return nil, nil },
	}
	h := mkHierarchy(mod)
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))

	err := execDropWorkflowGen(ctx, &ast.DropWorkflowStmt{
		Name: ast.QualifiedName{Module: "BPModule", Name: "Missing"},
	})
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestBuildWorkflowActivitiesGen_DispatchesLeafTypes(t *testing.T) {
	nodes := []ast.WorkflowActivityNode{
		&ast.WorkflowJumpToNode{Target: "A"},
		&ast.WorkflowWaitForTimerNode{Caption: "T"},
		&ast.WorkflowWaitForNotificationNode{Caption: "N"},
		&ast.WorkflowEndNode{Caption: "E"},
		&ast.WorkflowAnnotationActivityNode{Text: "note"},
	}
	out := buildWorkflowActivitiesGen(nodes)
	if len(out) != 5 {
		t.Fatalf("expected 5 elements, got %d", len(out))
	}
	wantTypes := []string{
		"Workflows$JumpToActivity",
		"Workflows$WaitForTimerActivity",
		"Workflows$WaitForNotificationActivity",
		"Workflows$EndWorkflowActivity",
		"Workflows$FloatingAnnotation",
	}
	for i, want := range wantTypes {
		if out[i].TypeName() != want {
			t.Errorf("[%d] TypeName = %q, want %q", i, out[i].TypeName(), want)
		}
	}
}
