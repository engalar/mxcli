// internal/fkg/guidance_test.go
package fkg_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/fkg"
	_ "github.com/mendixlabs/mxcli/internal/fkg/concepts"
)

// ── Guide ──────────────────────────────────────────────────────────────────────

func TestGuide_EntityReturnsResult(t *testing.T) {
	g, ok := mustNew(t).(fkg.GuidanceQuerier)
	if !ok {
		t.Fatal("fkg.Querier does not implement GuidanceQuerier")
	}
	result, err := g.Guide("entity")
	if err != nil {
		t.Fatalf("Guide(entity): %v", err)
	}
	if result.Concept.ID != "entity" {
		t.Errorf("Concept.ID = %q, want entity", result.Concept.ID)
	}
}

func TestGuide_UnknownConceptReturnsError(t *testing.T) {
	g, ok := mustNew(t).(fkg.GuidanceQuerier)
	if !ok {
		t.Fatal("fkg.Querier does not implement GuidanceQuerier")
	}
	_, err := g.Guide("nonexistent-concept")
	if err == nil {
		t.Error("expected error for unknown concept")
	}
}

func TestGuide_EntityHasPatternsAndSyntaxRefs(t *testing.T) {
	g, ok := mustNew(t).(fkg.GuidanceQuerier)
	if !ok {
		t.Fatal("fkg.Querier does not implement GuidanceQuerier")
	}
	result, err := g.Guide("entity")
	if err != nil {
		t.Fatalf("Guide(entity): %v", err)
	}
	if len(result.Patterns) == 0 {
		t.Error("entity should have at least 1 pattern")
	}
	if len(result.SyntaxRefs) == 0 {
		t.Error("entity should have syntax references")
	}
}

// ── Plan ───────────────────────────────────────────────────────────────────────

func TestPlan_KnownModuleReturnsResult(t *testing.T) {
	c, ok := mustNew(t).(fkg.CurriculumQuerier)
	if !ok {
		t.Fatal("fkg.Querier does not implement CurriculumQuerier")
	}
	result, err := c.Plan("curriculum:academy-04-pages")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if result.Module.ID != "curriculum:academy-04-pages" {
		t.Errorf("Module.ID = %q", result.Module.ID)
	}
	if len(result.Concepts) == 0 {
		t.Error("expected concepts for academy-04-pages")
	}
}

// ── Orchestrate ─────────────────────────────────────────────────────────────────

func TestOrchestrate_EntityAndSecurityBothPresent(t *testing.T) {
	o, ok := mustNew(t).(fkg.Orchestrator)
	if !ok {
		t.Fatal("fkg.Querier does not implement Orchestrator")
	}
	plan, err := o.Orchestrate([]string{"entity", "security"})
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if len(plan.Steps) == 0 {
		t.Fatal("expected at least 1 step")
	}
	ids := map[string]bool{}
	for _, s := range plan.Steps {
		ids[s.Concept.ID] = true
	}
	if !ids["entity"] {
		t.Error("entity not found in orchestration")
	}
	if !ids["security"] {
		t.Error("security not found in orchestration")
	}
}

func TestOrchestrate_UnknownConceptReturnsError(t *testing.T) {
	o, ok := mustNew(t).(fkg.Orchestrator)
	if !ok {
		t.Fatal("fkg.Querier does not implement Orchestrator")
	}
	_, err := o.Orchestrate([]string{"entity", "nonexistent"})
	if err == nil {
		t.Error("expected error for unknown concept")
	}
}

func TestOrchestrate_PageHasPatterns(t *testing.T) {
	o, ok := mustNew(t).(fkg.Orchestrator)
	if !ok {
		t.Fatal("fkg.Querier does not implement Orchestrator")
	}
	plan, err := o.Orchestrate([]string{"page"})
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if len(plan.Steps) < 1 {
		t.Fatal("expected at least 1 step")
	}
	if len(plan.Steps[0].Patterns) == 0 {
		t.Error("page should have patterns in orchestration")
	}
}

// ── Guide Steps ─────────────────────────────────────────────────────────────────

func TestGuide_PageHasSteps(t *testing.T) {
	g, ok := mustNew(t).(fkg.GuidanceQuerier)
	if !ok {
		t.Fatal("fkg.Querier does not implement GuidanceQuerier")
	}
	result, err := g.Guide("page")
	if err != nil {
		t.Fatalf("Guide(page): %v", err)
	}
	if len(result.Steps) == 0 {
		t.Error("page should have implementation steps from patterns")
	}
	for _, s := range result.Steps {
		if s.Order == 0 {
			t.Error("step Order must be > 0")
		}
		if s.Action == "" {
			t.Error("step Action must not be empty")
		}
	}
}

func TestGuide_MicroflowHasSteps(t *testing.T) {
	g, ok := mustNew(t).(fkg.GuidanceQuerier)
	if !ok {
		t.Fatal("fkg.Querier does not implement GuidanceQuerier")
	}
	result, err := g.Guide("microflow")
	if err != nil {
		t.Fatalf("Guide(microflow): %v", err)
	}
	if len(result.Steps) == 0 {
		t.Error("microflow should have implementation steps from patterns")
	}
}

func TestGuide_WorkflowHasPatternsAndSteps(t *testing.T) {
	g, ok := mustNew(t).(fkg.GuidanceQuerier)
	if !ok {
		t.Fatal("fkg.Querier does not implement GuidanceQuerier")
	}
	result, err := g.Guide("workflow")
	if err != nil {
		t.Fatalf("Guide(workflow): %v", err)
	}
	if len(result.Patterns) == 0 {
		t.Error("workflow should have patterns")
	}
	if len(result.Steps) == 0 {
		t.Error("workflow should have implementation steps")
	}
}

func TestPlan_UnknownModuleReturnsError(t *testing.T) {
	c, ok := mustNew(t).(fkg.CurriculumQuerier)
	if !ok {
		t.Fatal("fkg.Querier does not implement CurriculumQuerier")
	}
	_, err := c.Plan("nonexistent")
	if err == nil {
		t.Error("expected error for unknown module")
	}
}
