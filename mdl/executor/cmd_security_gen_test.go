// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5a tests: cmd_security_gen.go gen-typed security read paths.

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestListAccessOnMicroflowGen_NotFound asserts the gen-typed reader
// returns a not-found error for a microflow that does not exist in the
// fixture, mirroring the legacy listAccessOnMicroflow contract.
func TestListAccessOnMicroflowGen_NotFound(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)
	ctx.Output = &bytes.Buffer{}

	err := listAccessOnMicroflowGen(ctx, &ast.QualifiedName{
		Module: "MyFirstModule",
		Name:   "DoesNotExist_Stage3_2_5a",
	})
	if err == nil {
		t.Fatalf("expected NotFound error, got nil")
	}
	if !strings.Contains(err.Error(), "microflow") {
		t.Errorf("error should mention microflow; got %q", err.Error())
	}
}

// TestListAccessOnMicroflowGen_Existing asserts that an existing
// microflow renders an output line (either "No module roles..." or
// "Allowed module roles..."). The fixture's microflows may or may not
// have allowed roles configured; we only assert the path completes
// without error and emits non-empty output.
func TestListAccessOnMicroflowGen_Existing(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)
	var buf bytes.Buffer
	ctx.Output = &buf

	err := listAccessOnMicroflowGen(ctx, &ast.QualifiedName{
		Module: "MyFirstModule",
		Name:   "MyFirstLogic",
	})
	if err != nil {
		t.Fatalf("listAccessOnMicroflowGen: %v", err)
	}

	out := buf.String()
	if out == "" {
		t.Fatalf("expected non-empty output for existing microflow")
	}
	// One of the two well-known prefixes must be present.
	if !strings.Contains(out, "No module roles granted execute access") &&
		!strings.Contains(out, "Allowed module roles for") {
		t.Errorf("unexpected output for existing microflow: %q", out)
	}
}

// TestSplitRoleQualifiedName covers the small split helper.
func TestSplitRoleQualifiedName(t *testing.T) {
	cases := []struct {
		in       string
		wantMod  string
		wantRole string
	}{
		{"MyFirstModule.User", "MyFirstModule", "User"},
		{"Mod.Sub.Role", "Mod", "Sub.Role"},
		{"BareRole", "", "BareRole"},
		{"", "", ""},
	}
	for _, c := range cases {
		mod, role := splitRoleQualifiedName(c.in)
		if mod != c.wantMod || role != c.wantRole {
			t.Errorf("splitRoleQualifiedName(%q)=(%q,%q); want (%q,%q)", c.in, mod, role, c.wantMod, c.wantRole)
		}
	}
}

// TestListProjectSecurityGen_OutputsLevel asserts that listProjectSecurityGen
// completes without error and emits a "Security Level:" line.
func TestListProjectSecurityGen_OutputsLevel(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listProjectSecurityGen(ctx); err != nil {
		t.Fatalf("listProjectSecurityGen: %v", err)
	}
	if !strings.Contains(buf.String(), "Security Level:") {
		t.Errorf("output missing Security Level line: %q", buf.String())
	}
}

// TestListModuleRolesGen_FiltersByModule asserts that listModuleRolesGen
// completes without error and emits a "Module" column header. The fixture
// module name is "MyFirstModule" (not "TestModule" from the spec template).
func TestListModuleRolesGen_FiltersByModule(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listModuleRolesGen(ctx, "MyFirstModule"); err != nil {
		t.Fatalf("listModuleRolesGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "MyFirstModule") {
		t.Errorf("expected MyFirstModule in output, got: %q", out)
	}
}

// TestListUserRolesGen_OutputsRoleNames asserts listUserRolesGen completes
// without error and emits a "Name" column header.
func TestListUserRolesGen_OutputsRoleNames(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listUserRolesGen(ctx); err != nil {
		t.Fatalf("listUserRolesGen: %v", err)
	}
	if !strings.Contains(buf.String(), "Name") {
		t.Errorf("expected column header Name, got: %q", buf.String())
	}
}

// TestListSecurityMatrixGen_Smoke asserts the matrix renderer does not
// error and emits the section headers ("## Microflow Access" etc.).
func TestListSecurityMatrixGen_Smoke(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)
	var buf bytes.Buffer
	ctx.Output = &buf

	if err := listSecurityMatrixGen(ctx, ""); err != nil {
		t.Fatalf("listSecurityMatrixGen: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"Security Matrix",
		"## Entity Access",
		"## Microflow Access",
		"## Page Access",
		"## Workflow Access",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("matrix output missing section %q; full output:\n%s", want, out)
		}
	}
}

// TestListDemoUsersGen_DisabledMessage asserts that listDemoUsersGen emits a
// "disabled" message when demo users are disabled.
// Note: the fixture MPR has demo users enabled; this test disables them on
// the in-memory object (without clearing cache) so the cached ps reflects
// the disabled state when listDemoUsersGen calls getProjectSecurityGen.
func TestListDemoUsersGen_DisabledMessage(t *testing.T) {
	ctx := newSecurityTestContext(t)
	// warm the cache, then flip the flag on the cached object
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		t.Fatalf("getProjectSecurityGen: %v", err)
	}
	if ps == nil {
		t.Fatal("ProjectSecurity is nil in fixture")
	}
	// If getProjectSecurityGen ever deep-copies its return value, this test will
	// silently stop exercising the disabled branch (the next call would re-read the
	// fixture's enabled state). TestListDemoUsersGen_EnabledOutputsUsers would still
	// pass, masking the regression. Re-evaluate this trick if cache semantics change.
	ps.SetEnableDemoUsers(false) // mutate cached object — cache is NOT invalidated

	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listDemoUsersGen(ctx); err != nil {
		t.Fatalf("listDemoUsersGen: %v", err)
	}
	if !strings.Contains(buf.String(), "disabled") {
		t.Errorf("expected disabled message, got: %q", buf.String())
	}
}

// TestListDemoUsersGen_EnabledOutputsUsers exercises the enabled branch:
// type-assert loop, role join, and summary. The fixture has demo users
// enabled (demo_administrator + demo_user) so no cache mutation is needed.
func TestListDemoUsersGen_EnabledOutputsUsers(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listDemoUsersGen(ctx); err != nil {
		t.Fatalf("listDemoUsersGen: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"User Name", "User Roles", "demo users)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got: %q", want, out)
		}
	}
	if !strings.Contains(out, "demo_administrator") && !strings.Contains(out, "demo_user") {
		t.Errorf("output missing any known demo user name; got: %q", out)
	}
}

// TestDescribeModuleRoleGen_OutputsCreateStatement asserts that
// describeModuleRoleGen emits a CREATE MODULE ROLE statement for an existing role.
// The fixture module role is "MyFirstModule.User" (not "TestModule.User" from the
// spec template — the fixture module is MyFirstModule per A2 fixture-name swap).
func TestDescribeModuleRoleGen_OutputsCreateStatement(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	if err := describeModuleRoleGen(ctx, ast.QualifiedName{Module: "MyFirstModule", Name: "User"}); err != nil {
		t.Fatalf("describeModuleRoleGen: %v", err)
	}
	if !strings.Contains(buf.String(), "create module role MyFirstModule.User") {
		t.Errorf("expected create statement, got: %q", buf.String())
	}
}

// TestDescribeModuleRoleGen_NotFound asserts that a non-existent module role
// returns a not-found error from describeModuleRoleGen.
func TestDescribeModuleRoleGen_NotFound(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	err := describeModuleRoleGen(ctx, ast.QualifiedName{Module: "Nope", Name: "Bogus"})
	if err == nil {
		t.Fatalf("expected NotFound error, got nil")
	}
	if !strings.Contains(err.Error(), "module role") {
		t.Errorf("error should mention module role; got %q", err.Error())
	}
}

// TestDescribeUserRoleGen_OutputsCreateStatement asserts that describeUserRoleGen
// emits a CREATE USER ROLE statement for an existing user role.
// The fixture user role "User" has 4 module roles and checkSecurity=true,
// matching the legacy describe user role 'User' output.
func TestDescribeUserRoleGen_OutputsCreateStatement(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	if err := describeUserRoleGen(ctx, ast.QualifiedName{Name: "User"}); err != nil {
		t.Fatalf("describeUserRoleGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "create user role User") {
		t.Errorf("expected create statement with role name, got: %q", out)
	}
	// module roles must be parenthesized
	if !strings.Contains(out, "(") || !strings.Contains(out, ")") {
		t.Errorf("expected parenthesized module roles, got: %q", out)
	}
	// terminator
	if !strings.Contains(out, ";") {
		t.Errorf("expected semicolon terminator, got: %q", out)
	}
	if !strings.Contains(out, "/") {
		t.Errorf("expected / terminator, got: %q", out)
	}
	// check security comment (User role has checkSecurity=true in fixture)
	if !strings.Contains(out, "-- Check security: enabled") {
		t.Errorf("expected check security comment, got: %q", out)
	}
}

// TestDescribeUserRoleGen_NotFound asserts that a non-existent user role
// returns a not-found error from describeUserRoleGen.
func TestDescribeUserRoleGen_NotFound(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	err := describeUserRoleGen(ctx, ast.QualifiedName{Name: "DoesNotExist_Stage3_3_A6"})
	if err == nil {
		t.Fatalf("expected NotFound error, got nil")
	}
	if !strings.Contains(err.Error(), "user role") {
		t.Errorf("error should mention user role; got %q", err.Error())
	}
}
