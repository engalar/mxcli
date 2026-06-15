// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/linter"
)

func TestNoEntityAccessRulesRule_NoViolation(t *testing.T) {
	specs := []entitySpec{
		{id: "id1", name: "Customer", module: "MyModule", persistent: true, accessRules: 2},
	}
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewNoEntityAccessRulesRule().Check(ctx)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestNoEntityAccessRulesRule_DetectsMissing(t *testing.T) {
	specs := []entitySpec{
		{id: "id1", name: "Customer", module: "MyModule", persistent: true, accessRules: 0},
		{id: "id2", name: "Order", module: "MyModule", persistent: true, accessRules: 1},
	}
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewNoEntityAccessRulesRule().Check(ctx)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "SEC001" {
		t.Errorf("expected rule ID SEC001, got %s", violations[0].RuleID)
	}
	if violations[0].Location.DocumentName != "Customer" {
		t.Errorf("expected document Customer, got %s", violations[0].Location.DocumentName)
	}
}

func TestNoEntityAccessRulesRule_NonPersistentIgnored(t *testing.T) {
	specs := []entitySpec{
		{id: "id1", name: "TempObj", module: "MyModule", persistent: false},
	}
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewNoEntityAccessRulesRule().Check(ctx)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for non-persistent entity, got %d", len(violations))
	}
}

func TestNoEntityAccessRulesRule_ExternalIgnored(t *testing.T) {
	specs := []entitySpec{
		{id: "id1", name: "ExtEntity", module: "MyModule", persistent: true, external: true, accessRules: 0},
	}
	ctx := newGraphContext(buildEntityContext(specs), buildEntityReader(specs))
	violations := NewNoEntityAccessRulesRule().Check(ctx)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for external entity, got %d", len(violations))
	}
}

func TestNoEntityAccessRulesRule_Metadata(t *testing.T) {
	r := NewNoEntityAccessRulesRule()
	if r.ID() != "SEC001" {
		t.Errorf("ID = %q, want SEC001", r.ID())
	}
	if r.Category() != "security" {
		t.Errorf("Category = %q, want security", r.Category())
	}
}

// NOTE: SEC002 and SEC003 require ctx.Reader() → *mpr.Reader to call GetProjectSecurity().
// Without a real MPR file, we can only test the nil-reader early return and metadata.
// Full behavioral coverage requires integration tests with a real .mpr project.

func TestWeakPasswordPolicyRule_NilReader(t *testing.T) {
	ctx := linter.NewLintContext(nil, nil)
	rule := NewWeakPasswordPolicyRule()
	violations := rule.Check(ctx)

	if violations != nil {
		t.Errorf("expected nil with nil reader, got %v", violations)
	}
}

// SEC003: Demo users still active in production
// Without a real MPR file, we can only test the nil-reader early return and metadata.

func TestDemoUsersActiveRule_NilReader(t *testing.T) {
	r := NewDemoUsersActiveRule()
	ctx := linter.NewLintContext(nil, nil)
	violations := r.Check(ctx)
	if violations != nil {
		t.Errorf("expected nil with nil reader, got %v", violations)
	}
}

func TestDemoUsersActiveRule_Metadata(t *testing.T) {
	r := NewDemoUsersActiveRule()
	if r.ID() != "SEC003" {
		t.Errorf("ID = %q, want SEC003", r.ID())
	}
	if r.Category() != "security" {
		t.Errorf("Category = %q, want security", r.Category())
	}
}
