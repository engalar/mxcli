// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5c tests — gen-typed CREATE NANOFLOW write path.
//
// Covers the trivial-body surface (empty body / bare `return;`) plus
// the deferred-error path for compound bodies that Stage 3.2.3 will
// flesh out. Each test runs against a fresh fixture copy so writes
// don't leak between subtests.

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestExecCreateNanoflowGen_EmptyBodyRoundTrip creates a no-body
// nanoflow via the gen write path then describes it back via the gen
// read path — ensuring the round-trip lands on the expected header
// and that the body comes back empty (the trivial Start→End graph
// produces no MDL activity lines).
func TestExecCreateNanoflowGen_EmptyBodyRoundTrip(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	stmt := &ast.CreateNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_GenSmoke"},
	}

	if err := execCreateNanoflowGen(ctx, stmt); err != nil {
		t.Fatalf("execCreateNanoflowGen: %v", err)
	}

	rendered, _, err := describeNanoflowGenToString(ctx, stmt.Name)
	if err != nil {
		t.Fatalf("describeNanoflowGenToString after create: %v", err)
	}
	mustContain(t, rendered,
		"create or modify nanoflow MyFirstModule.NF_GenSmoke ()",
		"\nbegin\n",
		"\nend;",
	)
}

// TestExecCreateNanoflowGen_WithParametersAndDocumentation covers
// header coverage: doc block, @excluded, and a single primitive
// parameter all flow through the gen write → read pipe.
func TestExecCreateNanoflowGen_WithParametersAndDocumentation(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	stmt := &ast.CreateNanoflowStmt{
		Name:          ast.QualifiedName{Module: "MyFirstModule", Name: "NF_WithDoc"},
		Documentation: "first line\nsecond line",
		Excluded:      true,
		Parameters: []ast.MicroflowParam{
			{Name: "Username", Type: ast.DataType{Kind: ast.TypeString}},
		},
	}

	if err := execCreateNanoflowGen(ctx, stmt); err != nil {
		t.Fatalf("execCreateNanoflowGen: %v", err)
	}

	rendered, _, err := describeNanoflowGenToString(ctx, stmt.Name)
	if err != nil {
		t.Fatalf("describe after create: %v", err)
	}

	mustContain(t, rendered,
		" * first line",
		" * second line",
		"@excluded",
		"$Username:",
	)
}

// TestExecCreateNanoflowGen_ReplaceExisting walks the create-or-modify
// branch: a second create against the same qualified name without the
// CreateOrModify flag must fail; with the flag it should succeed and
// reuse the existing element ID (preserving downstream references).
func TestExecCreateNanoflowGen_ReplaceExisting(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	first := &ast.CreateNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_Replace"},
	}
	if err := execCreateNanoflowGen(ctx, first); err != nil {
		t.Fatalf("first create: %v", err)
	}

	t.Run("conflict without create-or-modify", func(t *testing.T) {
		conflict := &ast.CreateNanoflowStmt{
			Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_Replace"},
		}
		if err := execCreateNanoflowGen(ctx, conflict); err == nil {
			t.Error("expected AlreadyExists error, got nil")
		}
	})

	t.Run("succeeds with create-or-modify", func(t *testing.T) {
		replace := &ast.CreateNanoflowStmt{
			Name:           ast.QualifiedName{Module: "MyFirstModule", Name: "NF_Replace"},
			CreateOrModify: true,
			Documentation:  "updated",
		}
		if err := execCreateNanoflowGen(ctx, replace); err != nil {
			t.Errorf("create or modify: %v", err)
		}

		rendered, _, err := describeNanoflowGenToString(ctx, replace.Name)
		if err != nil {
			t.Fatalf("describe after replace: %v", err)
		}
		if !strings.Contains(rendered, " * updated") {
			t.Errorf("documentation should reflect the replace; got:\n%s", rendered)
		}
	})
}

// TestExecCreateNanoflowGen_CompoundBodyAccepted verifies the
// Stage 3.2.3 lift of the trivial-body restriction: compound bodies
// (Declare / IfStmt / etc.) are now supported via the shared
// flowBuilderGen path. The earlier "deferred to 3.2.3" rejection is
// gone — Stage 3.2.3 shipped the full flow-graph builder so
// compound nanoflow bodies route through buildFlowGraphGen.
func TestExecCreateNanoflowGen_CompoundBodyAccepted(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	// Declare-only body — was previously deferred, now must succeed.
	stmt := &ast.CreateNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_Compound"},
		Body: []ast.MicroflowStatement{
			&ast.DeclareStmt{
				Variable:     "X",
				Type:         ast.DataType{Kind: ast.TypeBoolean},
				InitialValue: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
			},
		},
	}
	if err := execCreateNanoflowGen(ctx, stmt); err != nil {
		t.Fatalf("compound body should be accepted post-Stage-3.2.3, got error: %v", err)
	}
}

// TestExecCreateNanoflowGen_ValidationFailures hits the up-front
// validators: empty name, disallowed action in body, repo unavailable.
func TestExecCreateNanoflowGen_ValidationFailures(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	t.Run("empty name", func(t *testing.T) {
		stmt := &ast.CreateNanoflowStmt{
			Name: ast.QualifiedName{Module: "MyFirstModule", Name: "  "},
		}
		if err := execCreateNanoflowGen(ctx, stmt); err == nil {
			t.Error("expected validation error for blank name")
		}
	})

	t.Run("disallowed Java action in body", func(t *testing.T) {
		stmt := &ast.CreateNanoflowStmt{
			Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_BadJava"},
			Body: []ast.MicroflowStatement{
				&ast.CallJavaActionStmt{},
			},
		}
		err := execCreateNanoflowGen(ctx, stmt)
		if err == nil {
			t.Fatal("expected nanoflow-body validation error")
		}
		if !strings.Contains(err.Error(), "Java actions cannot be called from nanoflows") {
			t.Errorf("expected Java-action-disallowed message; got: %v", err)
		}
	})

	t.Run("nil ctx.Nanoflows", func(t *testing.T) {
		bare := *ctx
		bare.Nanoflows = nil
		stmt := &ast.CreateNanoflowStmt{
			Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_NoRepo"},
		}
		if err := execCreateNanoflowGen(&bare, stmt); err == nil {
			t.Error("expected error when ctx.Nanoflows is nil")
		}
	})
}

// TestExecCreateNanoflowGen_PrintsCreatedToOutput verifies the user
// feedback line matches the legacy phrasing — downstream tooling
// (e.g. mxcli script runners) parses these prefixes.
func TestExecCreateNanoflowGen_PrintsCreatedToOutput(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	var buf bytes.Buffer
	ctx.Output = &buf

	stmt := &ast.CreateNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_PrintCheck"},
	}
	if err := execCreateNanoflowGen(ctx, stmt); err != nil {
		t.Fatalf("execCreateNanoflowGen: %v", err)
	}
	if !strings.Contains(buf.String(), "Created nanoflow: MyFirstModule.NF_PrintCheck") {
		t.Errorf("expected Created prefix; got:\n%s", buf.String())
	}
}
