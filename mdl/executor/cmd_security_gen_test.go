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
