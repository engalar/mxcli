// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5b tests for the gen-typed security write paths.

package executor

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

// ─────────────────────────────────────────────────────────────────
// Pure helper tests — no fixture required.
// ─────────────────────────────────────────────────────────────────

func TestMergeAllowedRoles(t *testing.T) {
	cases := []struct {
		name       string
		existing   []string
		toAdd      []ast.QualifiedName
		wantMerged []string
		wantAdded  []string
	}{
		{
			name:       "all new roles added",
			existing:   nil,
			toAdd:      []ast.QualifiedName{{Module: "M", Name: "User"}, {Module: "M", Name: "Admin"}},
			wantMerged: []string{"M.User", "M.Admin"},
			wantAdded:  []string{"M.User", "M.Admin"},
		},
		{
			name:       "duplicate skipped",
			existing:   []string{"M.User"},
			toAdd:      []ast.QualifiedName{{Module: "M", Name: "User"}, {Module: "M", Name: "Admin"}},
			wantMerged: []string{"M.User", "M.Admin"},
			wantAdded:  []string{"M.Admin"},
		},
		{
			name:       "all duplicates",
			existing:   []string{"M.User", "M.Admin"},
			toAdd:      []ast.QualifiedName{{Module: "M", Name: "User"}},
			wantMerged: []string{"M.User", "M.Admin"},
			wantAdded:  nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			merged, added := mergeAllowedRoles(c.existing, c.toAdd)
			if !reflect.DeepEqual(merged, c.wantMerged) {
				t.Errorf("merged = %v; want %v", merged, c.wantMerged)
			}
			if !reflect.DeepEqual(added, c.wantAdded) {
				t.Errorf("added = %v; want %v", added, c.wantAdded)
			}
		})
	}
}

func TestFilterAllowedRoles(t *testing.T) {
	cases := []struct {
		name          string
		existing      []string
		toRemove      []ast.QualifiedName
		wantRemaining []string
		wantRemoved   []string
	}{
		{
			name:          "remove single",
			existing:      []string{"M.User", "M.Admin"},
			toRemove:      []ast.QualifiedName{{Module: "M", Name: "Admin"}},
			wantRemaining: []string{"M.User"},
			wantRemoved:   []string{"M.Admin"},
		},
		{
			name:          "remove all",
			existing:      []string{"M.User", "M.Admin"},
			toRemove:      []ast.QualifiedName{{Module: "M", Name: "User"}, {Module: "M", Name: "Admin"}},
			wantRemaining: []string{},
			wantRemoved:   []string{"M.User", "M.Admin"},
		},
		{
			name:          "no match",
			existing:      []string{"M.User"},
			toRemove:      []ast.QualifiedName{{Module: "M", Name: "Other"}},
			wantRemaining: []string{"M.User"},
			wantRemoved:   nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			remaining, removed := filterAllowedRoles(c.existing, c.toRemove)
			if !reflect.DeepEqual(remaining, c.wantRemaining) {
				t.Errorf("remaining = %v; want %v", remaining, c.wantRemaining)
			}
			if !reflect.DeepEqual(removed, c.wantRemoved) {
				t.Errorf("removed = %v; want %v", removed, c.wantRemoved)
			}
		})
	}
}

func TestRemoveRoleFromList(t *testing.T) {
	remaining, removed := removeRoleFromList([]string{"M.User", "M.Admin"}, "M.User")
	if !removed {
		t.Errorf("expected removed=true")
	}
	if !reflect.DeepEqual(remaining, []string{"M.Admin"}) {
		t.Errorf("remaining = %v; want [M.Admin]", remaining)
	}

	remaining2, removed2 := removeRoleFromList([]string{"M.Admin"}, "M.User")
	if removed2 {
		t.Errorf("expected removed=false (role not present)")
	}
	if !reflect.DeepEqual(remaining2, []string{"M.Admin"}) {
		t.Errorf("remaining = %v; want [M.Admin] unchanged", remaining2)
	}
}

// ─────────────────────────────────────────────────────────────────
// Integration tests — exercise grant/revoke against the fixture.
// ─────────────────────────────────────────────────────────────────

// TestExecGrantMicroflowAccessGen_Roundtrip grants execute access to
// the User role on MyFirstModule.MyFirstLogic, then re-reads the
// microflow via the gen repo and asserts the role is present.
func TestExecGrantMicroflowAccessGen_Roundtrip(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)
	var buf bytes.Buffer
	ctx.Output = &buf

	stmt := &ast.GrantMicroflowAccessStmt{
		Microflow: ast.QualifiedName{Module: "MyFirstModule", Name: "MyFirstLogic"},
		Roles: []ast.QualifiedName{
			{Module: "MyFirstModule", Name: "User"},
		},
	}

	if err := ExecGrantMicroflowAccessGenFn(ctx, stmt, ctx.Deps); err != nil {
		t.Fatalf("execGrantMicroflowAccessGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "MyFirstLogic") {
		t.Errorf("expected output to mention microflow; got %q", out)
	}

	// Re-read via the gen repo and verify the role is present.
	mf := findMicroflowByQN(t, w, "MyFirstModule.MyFirstLogic")
	roles := genMicroflowAllowedRoles(mf)
	hasUser := false
	for _, r := range roles {
		if r == "MyFirstModule.User" {
			hasUser = true
			break
		}
	}
	if !hasUser {
		t.Errorf("MyFirstModule.User not found in allowed roles after grant; got %v", roles)
	}
}

// TestExecGrantMicroflowAccessGen_NotFound asserts a missing
// microflow yields a NotFound error and does not touch any state.
func TestExecGrantMicroflowAccessGen_NotFound(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)
	ctx.Output = &bytes.Buffer{}

	stmt := &ast.GrantMicroflowAccessStmt{
		Microflow: ast.QualifiedName{Module: "MyFirstModule", Name: "DoesNotExist_Stage3_2_5b"},
		Roles:     []ast.QualifiedName{{Module: "MyFirstModule", Name: "User"}},
	}

	err := ExecGrantMicroflowAccessGenFn(ctx, stmt, ctx.Deps)
	if err == nil {
		t.Fatal("expected NotFound error, got nil")
	}
	if !strings.Contains(err.Error(), "microflow") {
		t.Errorf("error should mention microflow; got %q", err.Error())
	}
}

// TestExecRevokeMicroflowAccessGen_Roundtrip grants a role first
// then revokes it and asserts the role is gone.
func TestExecRevokeMicroflowAccessGen_Roundtrip(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)
	ctx.Output = &bytes.Buffer{}

	grant := &ast.GrantMicroflowAccessStmt{
		Microflow: ast.QualifiedName{Module: "MyFirstModule", Name: "MyFirstLogic"},
		Roles:     []ast.QualifiedName{{Module: "MyFirstModule", Name: "User"}},
	}
	if err := ExecGrantMicroflowAccessGenFn(ctx, grant, ctx.Deps); err != nil {
		t.Fatalf("grant setup: %v", err)
	}

	revoke := &ast.RevokeMicroflowAccessStmt{
		Microflow: ast.QualifiedName{Module: "MyFirstModule", Name: "MyFirstLogic"},
		Roles:     []ast.QualifiedName{{Module: "MyFirstModule", Name: "User"}},
	}
	var buf bytes.Buffer
	ctx.Output = &buf
	if err := ExecRevokeMicroflowAccessGenFn(ctx, revoke, ctx.Deps); err != nil {
		t.Fatalf("execRevokeMicroflowAccessGen: %v", err)
	}
	if !strings.Contains(buf.String(), "Revoked") {
		t.Errorf("expected 'Revoked' in output; got %q", buf.String())
	}

	mf := findMicroflowByQN(t, w, "MyFirstModule.MyFirstLogic")
	for _, r := range genMicroflowAllowedRoles(mf) {
		if r == "MyFirstModule.User" {
			t.Errorf("MyFirstModule.User still present after revoke")
		}
	}
}

// TestExecRevokeMicroflowAccessGen_NoMatch asserts revoking a role
// that is not currently granted produces the "None of the specified
// roles" message and does not error.
func TestExecRevokeMicroflowAccessGen_NoMatch(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)
	var buf bytes.Buffer
	ctx.Output = &buf

	stmt := &ast.RevokeMicroflowAccessStmt{
		Microflow: ast.QualifiedName{Module: "MyFirstModule", Name: "MyFirstLogic"},
		Roles:     []ast.QualifiedName{{Module: "MyFirstModule", Name: "User"}},
	}
	if err := ExecRevokeMicroflowAccessGenFn(ctx, stmt, ctx.Deps); err != nil {
		t.Fatalf("execRevokeMicroflowAccessGen: %v", err)
	}
	if !strings.Contains(buf.String(), "None of the specified roles") {
		t.Errorf("expected 'None of the specified roles' message; got %q", buf.String())
	}
}

// TestValidateModuleRole_ModuleNotFound_IsWarning asserts that when the
// referenced module does not exist, validateModuleRole prints a WARNING
// and returns (false, nil) instead of propagating a fatal BackendError.
// Regression for BUG-04 (M-0022 POC): a GRANT against a missing module
// previously aborted the entire script.
func TestValidateModuleRole_ModuleNotFound_IsWarning(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{}, nil // no modules
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))

	role := ast.QualifiedName{Module: "Customer_Lookups", Name: "User"}
	found, err := validateModuleRole(ctx, role)

	if err != nil {
		t.Fatalf("validateModuleRole returned error for missing module: %v (should be non-fatal warning)", err)
	}
	if found {
		t.Fatal("validateModuleRole returned found=true for missing module")
	}
	if !strings.Contains(buf.String(), "WARNING") {
		t.Errorf("expected WARNING in output, got: %q", buf.String())
	}
}
