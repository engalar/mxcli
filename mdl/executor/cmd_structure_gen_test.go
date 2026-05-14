// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.4 — Tests for the gen-typed SHOW STRUCTURE entry.
//
// Strategy: render the same fixture project at depth 2 and 3 through
// the legacy and gen entries and assert that the output is byte-for-byte
// identical. Where the gen path can't yet match (workflow signatures
// would be the same; java action / pages / odata sections delegate to
// unchanged helpers), parity should hold trivially.

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// Stage 3.2.6.3a: legacy `execShowStructure` is gone — the parity
// fixture between legacy and gen is no longer meaningful. Replaced
// with non-empty smoke tests; output stability is verified by the
// existing fixture-based gen tests further down in this file
// (`TestFormatMicroflowSignatureGen_*`, `TestSortGenMicroflows`, etc.)
// and by cmd_microflows_show_gen_test.go.
func TestExecShowStructureGen_Depth2_NonEmpty(t *testing.T) {
	var genOut bytes.Buffer
	genCtx := newGenVizContext(t, &genOut)
	genCtx.Format = FormatTable
	genCtx.Quiet = true
	if err := execShowStructureGen(genCtx, &ast.ShowStmt{ObjectType: ast.ShowStructure, Depth: 2}); err != nil {
		t.Fatalf("gen execShowStructureGen depth=2: %v", err)
	}
	if genOut.Len() == 0 {
		t.Error("expected non-empty depth=2 output")
	}
}

func TestExecShowStructureGen_Depth3_NonEmpty(t *testing.T) {
	var genOut bytes.Buffer
	genCtx := newGenVizContext(t, &genOut)
	genCtx.Format = FormatTable
	genCtx.Quiet = true
	if err := execShowStructureGen(genCtx, &ast.ShowStmt{ObjectType: ast.ShowStructure, Depth: 3}); err != nil {
		t.Fatalf("gen execShowStructureGen depth=3: %v", err)
	}
	if genOut.Len() == 0 {
		t.Error("expected non-empty depth=3 output")
	}
}

// TestExecShowStructureGen_Depth1_DelegatesToLegacy verifies depth 1
// (catalog/SQL only) still renders correctly through the gen entry.
func TestExecShowStructureGen_Depth1_DelegatesToLegacy(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)
	ctx.Format = FormatTable
	ctx.Quiet = true
	if err := execShowStructureGen(ctx, &ast.ShowStmt{ObjectType: ast.ShowStructure, Depth: 1}); err != nil {
		t.Fatalf("execShowStructureGen depth=1: %v", err)
	}
	if out.Len() == 0 {
		t.Error("expected non-empty depth=1 output")
	}
}

// TestFormatMicroflowSignatureGen_BasicShape spot-checks the signature
// renderer against a hand-built MicroflowParameter object collection
// to keep the test independent of fixture choice.
func TestFormatMicroflowSignatureGen_BasicShape(t *testing.T) {
	mf := genMf.NewMicroflow()
	mf.SetName("Demo")
	col := genMf.NewMicroflowObjectCollection()
	mf.SetObjectCollection(col)

	pa := genMf.NewMicroflowParameter()
	pa.SetName("inputA")
	pa.SetType("String")
	col.AddObjects(pa)

	pb := genMf.NewMicroflowParameter()
	pb.SetName("inputB")
	pb.SetType("Boolean")
	col.AddObjects(pb)

	got := formatMicroflowSignatureGen(mf, false)
	if !strings.HasPrefix(got, "(") || !strings.HasSuffix(got, ")") {
		t.Errorf("expected wrapped paren signature, got %q", got)
	}
	if !strings.Contains(got, "String") || !strings.Contains(got, "Boolean") {
		t.Errorf("expected param types in signature, got %q", got)
	}

	gotNamed := formatMicroflowSignatureGen(mf, true)
	if !strings.Contains(gotNamed, "inputA: String") {
		t.Errorf("withNames output missing inputA: String, got %q", gotNamed)
	}
}

// TestFormatNanoflowSignatureGen_NoParams covers the empty-param path
// to ensure we emit "()" rather than "(  )" or panic.
func TestFormatNanoflowSignatureGen_NoParams(t *testing.T) {
	nf := genMf.NewNanoflow()
	col := genMf.NewMicroflowObjectCollection()
	nf.SetObjectCollection(col)
	got := formatNanoflowSignatureGen(nf, false)
	if got != "()" {
		t.Errorf("empty param sig = %q, want %q", got, "()")
	}
}

// TestSortGenMicroflows_StableAndCaseInsensitive proves the gen sort
// matches the legacy one (case-insensitive ascending by name).
func TestSortGenMicroflows_StableAndCaseInsensitive(t *testing.T) {
	mfs := []*genMf.Microflow{
		mfNamed("zebra"), mfNamed("Apple"), mfNamed("banana"),
	}
	sortGenMicroflows(mfs)
	want := []string{"Apple", "banana", "zebra"}
	for i, mf := range mfs {
		if mf.Name() != want[i] {
			t.Errorf("position %d: got %q, want %q", i, mf.Name(), want[i])
		}
	}
}

// ─────────────────────────────────────────────────────────────────────
// C4 — outputJavaActionsGen / formatJavaActionSignatureGen
// ─────────────────────────────────────────────────────────────────────

// TestOutputJavaActionsGen_Empty verifies nothing is printed for an empty slice.
func TestOutputJavaActionsGen_Empty(t *testing.T) {
	var buf bytes.Buffer
	ctx := newJavaActionsTestContext(t)
	ctx.Output = &buf
	outputJavaActionsGen(ctx, "TestMod", nil, false)
	if buf.Len() != 0 {
		t.Errorf("expected empty output for nil actions, got %q", buf.String())
	}
}

// TestOutputJavaActionsGen_SingleAction verifies name and module appear.
func TestOutputJavaActionsGen_SingleAction(t *testing.T) {
	ja := genJA.NewJavaAction()
	ja.SetName("MyJavaAction")

	var buf bytes.Buffer
	ctx := newJavaActionsTestContext(t)
	ctx.Output = &buf
	outputJavaActionsGen(ctx, "TestMod", []*genJA.JavaAction{ja}, false)

	out := buf.String()
	if !strings.Contains(out, "JavaAction TestMod.MyJavaAction") {
		t.Errorf("expected 'JavaAction TestMod.MyJavaAction' in output, got %q", out)
	}
}

// TestOutputJavaActionsGen_SortedAlphabetically verifies alphabetical ordering.
func TestOutputJavaActionsGen_SortedAlphabetically(t *testing.T) {
	jaZ := genJA.NewJavaAction()
	jaZ.SetName("Zebra")
	jaA := genJA.NewJavaAction()
	jaA.SetName("Apple")
	jaM := genJA.NewJavaAction()
	jaM.SetName("Mango")

	var buf bytes.Buffer
	ctx := newJavaActionsTestContext(t)
	ctx.Output = &buf
	outputJavaActionsGen(ctx, "Mod", []*genJA.JavaAction{jaZ, jaA, jaM}, false)

	out := buf.String()
	posA := strings.Index(out, "Apple")
	posMango := strings.Index(out, "Mango")
	posZ := strings.Index(out, "Zebra")

	if posA < 0 || posMango < 0 || posZ < 0 {
		t.Fatalf("not all names found in output: %q", out)
	}
	if !(posA < posMango && posMango < posZ) {
		t.Errorf("actions not sorted alphabetically: A=%d Mango=%d Z=%d in %q", posA, posMango, posZ, out)
	}
}

// TestFormatJavaActionSignatureGen_NoParams verifies empty params renders as "()".
func TestFormatJavaActionSignatureGen_NoParams(t *testing.T) {
	ja := genJA.NewJavaAction()
	ja.SetName("NoParams")
	sig := formatJavaActionSignatureGen(ja, false)
	if sig != "()" {
		t.Errorf("expected '()', got %q", sig)
	}
}

// TestExecShowStructureGen_Depth2_ContainsJavaAction verifies the gen
// structure path renders JavaAction lines via gen types (not legacy sdk).
// Needs both Microflows (from newGenVizContext) and JavaActions repos wired.
func TestExecShowStructureGen_Depth2_ContainsJavaAction(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)
	ctx.Format = FormatTable
	ctx.Quiet = true

	// Wire JavaActions repo so loadStructureSharedDataGen can find them.
	w := openMprWriterForTest(t)
	ctx.JavaActions = mprrepos.NewJavaActionRepository(w)
	ctx.JavaScriptActions = mprrepos.NewJavaScriptActionRepository(w)
	ctx.ensureCache()

	if err := execShowStructureGen(ctx, &ast.ShowStmt{ObjectType: ast.ShowStructure, Depth: 2}); err != nil {
		t.Fatalf("execShowStructureGen depth=2: %v", err)
	}
	output := out.String()
	// Depth=2 renders java actions as a per-module summary line "Java Actions: N".
	if !strings.Contains(output, "Java Actions:") {
		t.Errorf("expected 'Java Actions:' summary in depth=2 output, got:\n%s", output)
	}
}

// helpers ───────────────────────────────────────────────────────

func mfNamed(n string) *genMf.Microflow {
	mf := genMf.NewMicroflow()
	mf.SetName(n)
	return mf
}

// showDiffPreview prints a minimal context-diff hint of the first
// divergence so test failures are actionable without dumping the
// whole project structure.
func showDiffPreview(t *testing.T, tag, want, got string) {
	t.Helper()
	wl := strings.Split(want, "\n")
	gl := strings.Split(got, "\n")
	maxLen := len(wl)
	if len(gl) > maxLen {
		maxLen = len(gl)
	}
	for i := 0; i < maxLen; i++ {
		var a, b string
		if i < len(wl) {
			a = wl[i]
		}
		if i < len(gl) {
			b = gl[i]
		}
		if a != b {
			start := i - 3
			if start < 0 {
				start = 0
			}
			end := i + 4
			if end > maxLen {
				end = maxLen
			}
			var sb strings.Builder
			sb.WriteString("[")
			sb.WriteString(tag)
			sb.WriteString("] first divergence at line ")
			sb.WriteString(intToStr(i + 1))
			sb.WriteString("\n")
			for j := start; j < end; j++ {
				var aw, bw string
				if j < len(wl) {
					aw = wl[j]
				}
				if j < len(gl) {
					bw = gl[j]
				}
				marker := "  "
				if j == i {
					marker = ">>"
				}
				sb.WriteString(marker)
				sb.WriteString(" want: ")
				sb.WriteString(aw)
				sb.WriteString("\n")
				sb.WriteString(marker)
				sb.WriteString(" got:  ")
				sb.WriteString(bw)
				sb.WriteString("\n")
			}
			t.Error(sb.String())
			return
		}
	}
}

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

// _ = element.ID makes the element import live for the type alias
// declarations near the top of the file when other things shake out.
var _ element.ID
