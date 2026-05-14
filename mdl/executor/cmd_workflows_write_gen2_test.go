// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3 Phase D — gen-typed write path tests.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
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
