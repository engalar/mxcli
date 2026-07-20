// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestShowFragments_Mock(t *testing.T) {
	ctx, buf := newMockCtx(t)
	ctx.Fragments = map[string]*ast.DefineFragmentStmt{
		"myFrag": {Name: "myFrag"},
	}
	rebuildDeps(ctx)

	assertNoError(t, listFragments(ctx))

	out := buf.String()
	assertContainsStr(t, out, "myFrag")
}

func TestShowFragments_Empty_Mock(t *testing.T) {
	ctx, buf := newMockCtx(t)
	ctx.Fragments = map[string]*ast.DefineFragmentStmt{}
	rebuildDeps(ctx)

	assertNoError(t, listFragments(ctx))

	out := buf.String()
	assertContainsStr(t, out, "No fragments defined.")
}

func TestDescribeFragment_Mock(t *testing.T) {
	ctx, buf := newMockCtx(t)
	ctx.Fragments = map[string]*ast.DefineFragmentStmt{
		"myFrag": {
			Name: "myFrag",
			Widgets: []*ast.WidgetV3{
				{Type: "Button", Name: "btnSave"},
			},
		},
	}
	rebuildDeps(ctx)

	assertNoError(t, describeFragment(ctx, ast.QualifiedName{Name: "myFrag"}))

	out := buf.String()
	assertContainsStr(t, out, "define fragment myFrag")
	assertContainsStr(t, out, "Button btnSave")
}

func TestDescribeFragment_NotFound(t *testing.T) {
	ctx, _ := newMockCtx(t)
	ctx.Fragments = map[string]*ast.DefineFragmentStmt{}
	rebuildDeps(ctx)

	err := describeFragment(ctx, ast.QualifiedName{Name: "noSuchFrag"})
	assertError(t, err)
}

func TestDescribeFragment_NilFragments(t *testing.T) {
	ctx, _ := newMockCtx(t)
	// ctx.Fragments is nil by default

	err := describeFragment(ctx, ast.QualifiedName{Name: "noSuchFrag"})
	assertError(t, err)
}
