// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestSystemMembersClause_AllFour(t *testing.T) {
	input := `create persistent entity HD.Ticket (
  Subject: string(500) not null,
  IsOverSLA: boolean default false
)
system members (owner, createdDate, changedDate, changedBy)
index (Subject);`

	prog, errs := Build(input)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	stmt, ok := prog.Statements[0].(*ast.CreateEntityStmt)
	if !ok {
		t.Fatalf("expected CreateEntityStmt, got %T", prog.Statements[0])
	}

	// SystemMembers should carry the 4 flags, not pseudo-attributes
	if len(stmt.SystemMembers) != 4 {
		t.Errorf("SystemMembers len = %d, want 4; got %v", len(stmt.SystemMembers), stmt.SystemMembers)
	}

	want := map[string]bool{"owner": true, "createdDate": true, "changedDate": true, "changedBy": true}
	for _, m := range stmt.SystemMembers {
		if !want[m] {
			t.Errorf("unexpected system member %q", m)
		}
		delete(want, m)
	}
	for m := range want {
		t.Errorf("missing system member %q", m)
	}

	// Business attributes must NOT include owner/createdDate etc.
	for _, a := range stmt.Attributes {
		if a.Name == "owner" || a.Name == "createdDate" || a.Name == "changedDate" || a.Name == "changedBy" {
			t.Errorf("attribute %q should not appear in Attributes list", a.Name)
		}
	}
}

func TestSystemMembersClause_Subset(t *testing.T) {
	input := `create persistent entity HD.Customer (
  Name: string(200) not null
)
system members (owner);`

	prog, errs := Build(input)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	stmt := prog.Statements[0].(*ast.CreateEntityStmt)
	if len(stmt.SystemMembers) != 1 || stmt.SystemMembers[0] != "owner" {
		t.Errorf("SystemMembers = %v, want [owner]", stmt.SystemMembers)
	}
}

func TestSystemMembersClause_None(t *testing.T) {
	input := `create persistent entity KB.Article (
  Title: string(500) not null
);`

	prog, errs := Build(input)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	stmt := prog.Statements[0].(*ast.CreateEntityStmt)
	if len(stmt.SystemMembers) != 0 {
		t.Errorf("SystemMembers = %v, want empty", stmt.SystemMembers)
	}
}

func TestSystemMembersClause_OldSyntaxRemoved(t *testing.T) {
	// owner: autoowner in attribute list should no longer be valid syntax.
	input := `create persistent entity M.E (owner: autoowner);`
	_, errs := Build(input)
	if len(errs) == 0 {
		t.Error("expected parse error for old 'owner: autoowner' syntax, got none")
	}
}

func TestAlterEntitySetSystemMembers(t *testing.T) {
	input := `alter entity HD.Ticket set system members (owner, changedBy);`
	prog, errs := Build(input)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	stmt, ok := prog.Statements[0].(*ast.AlterEntityStmt)
	if !ok {
		t.Fatalf("expected AlterEntityStmt, got %T", prog.Statements[0])
	}

	if stmt.Operation != ast.AlterEntitySetSystemMembers {
		t.Errorf("Operation = %v, want AlterEntitySetSystemMembers", stmt.Operation)
	}

	if len(stmt.SystemMembers) != 2 {
		t.Errorf("SystemMembers = %v, want [owner changedBy]", stmt.SystemMembers)
	}
}
