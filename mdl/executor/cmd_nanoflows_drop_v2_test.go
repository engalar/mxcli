// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5c tests — gen-typed DROP NANOFLOW.

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestExecDropNanoflowGen_DropExisting creates a nanoflow via the gen
// write path then drops it via the gen drop path; describing it
// afterwards must surface NotFound.
func TestExecDropNanoflowGen_DropExisting(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	create := &ast.CreateNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_DropMe"},
	}
	if err := ExecCreateNanoflowGenFn(ctx, create, ctx.Deps); err != nil {
		t.Fatalf("create: %v", err)
	}

	drop := &ast.DropNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_DropMe"},
	}
	if err := ExecDropNanoflowGenFn(ctx, drop, ctx.Deps); err != nil {
		t.Fatalf("drop: %v", err)
	}

	if _, _, err := describeNanoflowGenToString(ctx, drop.Name); err == nil {
		t.Error("expected NotFound after drop, got nil")
	}
}

// TestExecDropNanoflowGen_DropNonExistent ensures the not-found path
// returns a clean error rather than panicking on the empty list.
func TestExecDropNanoflowGen_DropNonExistent(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	drop := &ast.DropNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_NeverExisted"},
	}
	err := ExecDropNanoflowGenFn(ctx, drop, ctx.Deps)
	if err == nil {
		t.Fatal("expected NotFound error, got nil")
	}
	if !strings.Contains(err.Error(), "NF_NeverExisted") {
		t.Errorf("error should mention nanoflow name; got: %v", err)
	}
}

// TestExecDropNanoflowGen_PrintsDroppedToOutput verifies the user
// feedback line matches the legacy phrasing.
func TestExecDropNanoflowGen_PrintsDroppedToOutput(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	create := &ast.CreateNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_DropOutput"},
	}
	if err := ExecCreateNanoflowGenFn(ctx, create, ctx.Deps); err != nil {
		t.Fatalf("create: %v", err)
	}

	var buf bytes.Buffer
	ctx.Output = &buf
	rebuildDeps(ctx)

	drop := &ast.DropNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_DropOutput"},
	}
	if err := ExecDropNanoflowGenFn(ctx, drop, ctx.Deps); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if !strings.Contains(buf.String(), "Dropped nanoflow: MyFirstModule.NF_DropOutput") {
		t.Errorf("expected Dropped prefix; got:\n%s", buf.String())
	}
}

// TestExecDropNanoflowGen_NilRepoGuard exercises the up-front guards.
func TestExecDropNanoflowGen_NilRepoGuard(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	bare := *ctx
	bare.Nanoflows = nil
	bare.Deps = bare.buildDeps(); bare.Cache.initLoadFns(bare.Deps)

	drop := &ast.DropNanoflowStmt{
		Name: ast.QualifiedName{Module: "MyFirstModule", Name: "NF_Anything"},
	}
	if err := ExecDropNanoflowGenFn(&bare, drop, bare.Deps); err == nil {
		t.Error("expected error when ctx.Nanoflows is nil")
	}
}
