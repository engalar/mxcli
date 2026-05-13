// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5a tests: cmd_diff_gen.go gen-typed microflow diff.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestDiffMicroflowGen_ExistingMicroflow asserts the gen-typed diff
// detects an existing microflow as not-new and produces a non-empty
// Current MDL block (sourced from describeMicroflow). The Proposed
// side is the empty-body skeleton from a parameterless Create stmt.
func TestDiffMicroflowGen_ExistingMicroflow(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "MyFirstLogic"},
	}

	got, err := diffMicroflowGen(ctx, stmt)
	if err != nil {
		t.Fatalf("diffMicroflowGen: %v", err)
	}
	if got == nil {
		t.Fatalf("diffMicroflowGen returned nil")
	}
	if got.IsNew {
		t.Errorf("expected existing microflow MyFirstModule.MyFirstLogic to be detected, got IsNew=true")
	}
	if got.Current == "" {
		t.Errorf("expected Current MDL to be populated for existing microflow; got empty")
	}
	if got.ObjectType != "Microflow" {
		t.Errorf("ObjectType: got %q want %q", got.ObjectType, "Microflow")
	}
}

// TestDiffMicroflowGen_NewMicroflow asserts a non-existent microflow
// is reported as IsNew with no Current MDL.
func TestDiffMicroflowGen_NewMicroflow(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	stmt := &ast.CreateMicroflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "DoesNotExist_Stage3_2_5a"},
	}

	got, err := diffMicroflowGen(ctx, stmt)
	if err != nil {
		t.Fatalf("diffMicroflowGen: %v", err)
	}
	if got == nil {
		t.Fatalf("diffMicroflowGen returned nil")
	}
	if !got.IsNew {
		t.Errorf("expected IsNew=true for nonexistent microflow; got false")
	}
	if got.Current != "" {
		t.Errorf("expected Current to be empty for new microflow; got %q", got.Current)
	}
}

// TestDiffNanoflowGen_NewNanoflow asserts the nanoflow diff handles
// the not-found case gracefully when the fixture has no nanoflows
// or the queried name does not exist.
func TestDiffNanoflowGen_NewNanoflow(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)
	if ctx.Nanoflows == nil {
		t.Skip("describe context has no NanoflowRepository wired")
	}

	stmt := &ast.CreateNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "DoesNotExist_NF_Stage3_2_5a"},
	}

	got, err := diffNanoflowGen(ctx, stmt)
	if err != nil {
		t.Fatalf("diffNanoflowGen: %v", err)
	}
	if got == nil {
		t.Fatalf("diffNanoflowGen returned nil")
	}
	if !got.IsNew {
		t.Errorf("expected IsNew=true for nonexistent nanoflow; got false")
	}
	if got.ObjectType != "Nanoflow" {
		t.Errorf("ObjectType: got %q want %q", got.ObjectType, "Nanoflow")
	}
}
