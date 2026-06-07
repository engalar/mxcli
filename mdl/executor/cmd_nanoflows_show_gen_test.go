// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5c tests — gen-typed DESCRIBE NANOFLOW.
//
// Mirrors cmd_microflows_show_gen_test.go. Reuses openMprWriterForTest /
// newGenDescribeContext / mustContain / fixture-copy helpers from that
// file — they live in the same package and are intentionally exported
// to test files only via lower-case names.

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// findNanoflowByQNGen walks the gen NanoflowRepository and returns the
// nanoflow matching `Module.Name`. Fails the test on miss.
func findNanoflowByQNGen(t *testing.T, repo repos.NanoflowReader, ctx *ExecContext, qn string) *genMf.Nanoflow {
	t.Helper()
	all, err := repo.List("")
	if err != nil {
		t.Fatalf("NanoflowReader.List: %v", err)
	}
	parts := strings.SplitN(qn, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("expected qualified name Module.Name, got %q", qn)
	}
	h, _ := getHierarchy(ctx)
	for _, nf := range all {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(nf.ID()))
		if modName == parts[0] && nf.Name() == parts[1] {
			return nf
		}
	}
	t.Fatalf("nanoflow %q not found in fixture", qn)
	return nil
}

// newGenNanoflowDescribeContext is the nanoflow analogue of
// newGenDescribeContext: same SQL-backed plumbing, but also wires
// ctx.Nanoflows so describeNanoflowGen* helpers can resolve the flow
// from the gen repo without tripping the nil guard.
func newGenNanoflowDescribeContext(t *testing.T, w *mmpr.Writer) *ExecContext {
	t.Helper()
	ctx := newGenDescribeContext(t, w)
	repoCtx := mprbackend.NewExecutorContext(w)
	ctx.Nanoflows = repoCtx.Nanoflows
	return ctx
}

// TestDescribeNanoflowGenToString_StructuralSkeleton renders a trivial
// nanoflow (Start→End with one or two activities) and asserts the
// surrounding scaffolding is present.
func TestDescribeNanoflowGenToString_StructuralSkeleton(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)
	repo := mprrepos.NewNanoflowRepository(w)
	nf := findNanoflowByQNGen(t, repo, ctx, "FeedbackModule.ACT_Open_Feedback_Modal")

	out, err := DescribeNanoflowGenToString(ctx, nf)
	if err != nil {
		t.Fatalf("DescribeNanoflowGenToString: %v", err)
	}

	mustContain(t, out,
		"create or modify nanoflow FeedbackModule.ACT_Open_Feedback_Modal", // resolved module name
		"\n{\n", // body open
		"\n}",   // body close
		"\n/",   // statement terminator
	)

	if strings.Contains(out, "<unknown>") {
		t.Errorf("module name should resolve from container chain (no <unknown>); got:\n%s", out)
	}
}

// TestDescribeNanoflowGenToString_IfElseFraming exercises the boolean
// ExclusiveSplit framing on a nanoflow with at least one if/else block.
func TestDescribeNanoflowGenToString_IfElseFraming(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)
	repo := mprrepos.NewNanoflowRepository(w)
	nf := findNanoflowByQNGen(t, repo, ctx, "FeedbackModule.SUB_Feedback_GetOrCreate")

	out, err := DescribeNanoflowGenToString(ctx, nf)
	if err != nil {
		t.Fatalf("DescribeNanoflowGenToString: %v", err)
	}

	mustContain(t, out,
		"create or modify nanoflow FeedbackModule.SUB_Feedback_GetOrCreate",
		"if ", " {\n",
		"} else {\n",
		"\n}",
	)
	if strings.Contains(out, "<unknown>") {
		t.Errorf("module name should resolve from container chain (no <unknown>); got:\n%s", out)
	}
}

// TestDescribeNanoflowGenToString_ParametersAndReturn checks that
// nanoflow parameters and the returns clause appear when declared.
// Atlas_Web_Content.ACT_Login takes one parameter and is Void.
// FeedbackModule.SUB_Feedback_GetOrCreate has no params and returns an entity.
func TestDescribeNanoflowGenToString_ParametersAndReturn(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)
	repo := mprrepos.NewNanoflowRepository(w)

	t.Run("with parameter, void return", func(t *testing.T) {
		nf := findNanoflowByQNGen(t, repo, ctx, "Atlas_Web_Content.ACT_Login")
		out, err := DescribeNanoflowGenToString(ctx, nf)
		if err != nil {
			t.Fatalf("DescribeNanoflowGenToString: %v", err)
		}
		if !strings.Contains(out, "$LoginContext:") {
			t.Errorf("expected $LoginContext parameter line; got:\n%s", out)
		}
		if strings.Contains(out, "\nreturns ") {
			t.Errorf("Void nanoflow should not emit a returns clause; got:\n%s", out)
		}
	})

	t.Run("no parameters, entity return", func(t *testing.T) {
		nf := findNanoflowByQNGen(t, repo, ctx, "FeedbackModule.SUB_Feedback_GetOrCreate")
		out, err := DescribeNanoflowGenToString(ctx, nf)
		if err != nil {
			t.Fatalf("DescribeNanoflowGenToString: %v", err)
		}
		if !strings.Contains(out, "create or modify nanoflow FeedbackModule.SUB_Feedback_GetOrCreate ()") {
			t.Errorf("expected empty parameter list `()`; got:\n%s", out)
		}
		if !strings.Contains(out, "\nreturns ") {
			t.Errorf("entity-returning nanoflow should emit a returns clause; got:\n%s", out)
		}
	})

	t.Run("trailing slash terminator", func(t *testing.T) {
		nf := findNanoflowByQNGen(t, repo, ctx, "Atlas_Web_Content.ACT_Login")
		out, err := DescribeNanoflowGenToString(ctx, nf)
		if err != nil {
			t.Fatalf("DescribeNanoflowGenToString: %v", err)
		}
		if !strings.HasSuffix(strings.TrimRight(out, "\n"), "/") {
			t.Errorf("expected trailing `/` statement terminator; got:\n%s", out)
		}
	})
}

// TestDescribeNanoflowGenToString_NilGuard verifies the basic guard.
func TestDescribeNanoflowGenToString_NilGuard(t *testing.T) {
	if _, err := DescribeNanoflowGenToString(nil, nil); err == nil {
		t.Error("expected error on nil nanoflow, got nil")
	}
}

// TestDescribeNanoflowGen_NotFound exercises the lookup failure path of
// the package-private `describeNanoflowGenToString`.
func TestDescribeNanoflowGen_NotFound(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	t.Run("missing nanoflow", func(t *testing.T) {
		_, _, err := describeNanoflowGenToString(ctx, ast.QualifiedName{Module: "FeedbackModule", Name: "DoesNotExist"})
		if err == nil {
			t.Error("expected NotFound error, got nil")
		}
	})

	t.Run("missing module", func(t *testing.T) {
		_, _, err := describeNanoflowGenToString(ctx, ast.QualifiedName{Module: "NoSuchModule", Name: "ACT_Login"})
		if err == nil {
			t.Error("expected NotFound error, got nil")
		}
	})

	t.Run("nil ctx.Nanoflows", func(t *testing.T) {
		bare := *ctx
		bare.Nanoflows = nil
		_, _, err := describeNanoflowGenToString(&bare, ast.QualifiedName{Module: "FeedbackModule", Name: "ACT_Login"})
		if err == nil {
			t.Error("expected error when ctx.Nanoflows is nil, got nil")
		}
	})
}

// TestDescribeNanoflowGen_PrintsToOutput covers the public
// describeNanoflowGen which wraps describeNanoflowGenToString and
// prints to ctx.Output.
func TestDescribeNanoflowGen_PrintsToOutput(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	var buf strings.Builder
	ctx.Output = &buf

	if err := describeNanoflowGen(ctx, ast.QualifiedName{Module: "Atlas_Web_Content", Name: "ACT_Login"}); err != nil {
		t.Fatalf("describeNanoflowGen: %v", err)
	}
	if !strings.Contains(buf.String(), "create or modify nanoflow Atlas_Web_Content.ACT_Login") {
		t.Errorf("expected header in output; got:\n%s", buf.String())
	}
}
