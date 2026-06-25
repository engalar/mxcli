// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.1 tests: structural skeleton + control-flow framing for
// DescribeMicroflowGenToString. Activity bodies in 3.2.1 are TODO
// placeholders — these tests verify the surrounding shape, not the
// per-activity output (that's 3.2.2's surface).

package executor

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mendixlabs/mxcli/internal/goldenfs"
	"github.com/mendixlabs/mxcli/internal/testfsutil"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// parallelOnceGuard ensures t.Parallel() is called at most once per test,
// even when openMprWriterForTest is called multiple times (e.g. via helpers).
var parallelOnceGuard sync.Map

func parallelOnce(t *testing.T) {
	t.Helper()
	if _, loaded := parallelOnceGuard.LoadOrStore(t, struct{}{}); !loaded {
		t.Cleanup(func() { parallelOnceGuard.Delete(t) })
		t.Parallel()
	}
}

// fixtureMprPath and openMprWriterForTest are defined in testopen_*_test.go
// (platform-specific: goldenfs on Linux, file copy elsewhere).

// findMicroflowByQN is a small helper that wraps FindByQualifiedName
// + nil/error guard so each test can stay focused on assertions.
func findMicroflowByQN(t *testing.T, w *mmpr.Writer, qn string) *genMf.Microflow {
	t.Helper()
	repo := mprrepos.NewMicroflowRepository(w)
	mf, err := repo.FindByQualifiedName(qn)
	if err != nil {
		t.Fatalf("FindByQualifiedName(%q): %v", qn, err)
	}
	if mf == nil {
		t.Fatalf("FindByQualifiedName(%q): not found", qn)
	}
	return mf
}

func newGenDescribeContext(t *testing.T, w *mmpr.Writer) *ExecContext {
	t.Helper()
	repoCtx := mprbackend.NewExecutorContext(w)

	// Wire up a backend so getHierarchy(ctx) can resolve module names
	// from the SQL-backed container chain (the canonical path used by
	// genMicroflowQualifiedName after BSON roundtrip strips Container()).
	// We open a second MprBackend on the same MPR file (already copied
	// into a tempdir by the helper); modernc/sqlite supports multiple
	// opens of the same file.
	path := w.ConcreteReader().Path()
	be, err := mprbackend.NewFromPath(path)
	if err != nil {
		t.Fatalf("mprbackend.NewFromPath(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	_ctx := &ExecContext{
		Backend:   be,
		ExecRepos: ExecRepos{Microflows: repoCtx.Microflows, Nanoflows: repoCtx.Nanoflows},
		ExecIO:    ExecIO{Output: io.Discard},
	}
	_ctx.initRoles()
	return _ctx
}

// TestDescribeMicroflowGenToString_StructuralSkeleton renders a trivial
// microflow (Start → End with no activities) and asserts that the
// surrounding scaffolding is present.
func TestDescribeMicroflowGenToString_StructuralSkeleton(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "MyFirstModule.MyFirstLogic")

	out, err := DescribeMicroflowGenToString(newGenDescribeContext(t, w), mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}

	mustContain(t, out,
		"create or modify microflow MyFirstModule.MyFirstLogic", // resolved module name
		"\n{\n", // body open
		"\n}",   // body close
		"\n/",   // statement terminator
	)

	// Stage 3.2.2.a: the SQL-backed module-name resolution should never
	// emit the placeholder.
	if strings.Contains(out, "<unknown>") {
		t.Errorf("module name should resolve from container chain (no <unknown>); got:\n%s", out)
	}

	// A trivial Start→End microflow should emit a bare `return;` from
	// the EndEvent emit helper — no TODO placeholder for any activity.
	if !strings.Contains(out, "  return;") {
		t.Errorf("trivial microflow should render `return;`; got:\n%s", out)
	}
	if strings.Contains(out, "TODO Stage 3.2.2:") {
		t.Errorf("MyFirstLogic has no activities; should not emit any TODO placeholders. Got:\n%s", out)
	}
}

// TestDescribeMicroflowGenToString_IfElseFraming exercises the boolean
// ExclusiveSplit framing on Administration.SaveNewAccount, which has a
// password-equality check producing one if/else block.
func TestDescribeMicroflowGenToString_IfElseFraming(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "Administration.SaveNewAccount")

	out, err := DescribeMicroflowGenToString(newGenDescribeContext(t, w), mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}

	mustContain(t, out,
		"create or modify microflow Administration.SaveNewAccount", // resolved module name
		"if ", " {\n",
		"} else {\n",
		"\n}",
	)
	if strings.Contains(out, "<unknown>") {
		t.Errorf("module name should resolve from container chain (no <unknown>); got:\n%s", out)
	}

	// As of Stage 3.2.2.e the data family covers RetrieveAction, so the
	// previously-TODO retrieve line in this fixture now renders as the
	// real `retrieve $V from …` statement. Spot-check that the formatter
	// reached the body — both the retrieve and the change-with-refresh
	// surfaces should appear inside the if-branch.
	mustContain(t, out,
		"retrieve $",
		"change $Account",
	)

	// Footer: roles section.
	if !strings.Contains(out, "grant execute on microflow ") {
		t.Errorf("expected `grant execute` footer; got:\n%s", out)
	}
}

// TestDescribeMicroflowGenToString_InheritanceSplitFraming exercises the
// InheritanceSplit framing on Administration.ManageMyAccount, which
// case-splits on the current user's specialised type.
// MDL {} syntax: split type $Var { case Module.Entity { body } else { body } }
func TestDescribeMicroflowGenToString_InheritanceSplitFraming(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "Administration.ManageMyAccount")

	out, err := DescribeMicroflowGenToString(newGenDescribeContext(t, w), mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}

	mustContain(t, out,
		"create or modify microflow Administration.ManageMyAccount",
		"split type $",         // correct split header
		"case Administration.", // case branch with entity name (not "when ... then")
	)
	mustNotContain(t, out,
		" inheritance", // old wrong syntax
		"when ",        // old wrong syntax
		"end case;",    // old wrong syntax
		"end split;",   // old wrong syntax (replaced by "}")
		"<unknown>",
	)
}

// TestDescribeMicroflowGenToString_ErrorEvent verifies that an ErrorEvent
// (raise error) describes as `raise error;` and not as a TODO placeholder.
func TestDescribeMicroflowGenToString_ErrorEvent(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "MyFirstModule.ACT_TestRaiseError")

	out, err := DescribeMicroflowGenToString(newGenDescribeContext(t, w), mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}

	mustContain(t, out, "raise error;")
	mustNotContain(t, out, "// TODO", "Stage 3.2.2")
}

// TestDescribeMicroflowGenToString_ParametersAndReturn checks that the
// parameter list and return clause appear when the microflow declares
// them. Administration.SaveNewAccount has one parameter and no return.
func TestDescribeMicroflowGenToString_ParametersAndReturn(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "Administration.SaveNewAccount")

	out, err := DescribeMicroflowGenToString(newGenDescribeContext(t, w), mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}

	if !strings.Contains(out, "$AccountPasswordData:") {
		t.Errorf("expected $AccountPasswordData parameter line; got:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "/") {
		t.Errorf("expected trailing `/` statement terminator; got:\n%s", out)
	}
}

// TestDescribeMicroflowGenToString_NilGuard verifies the basic guard.
func TestDescribeMicroflowGenToString_NilGuard(t *testing.T) {
	if _, err := DescribeMicroflowGenToString(nil, nil); err == nil {
		t.Error("expected error on nil microflow, got nil")
	}
}

// TestDescribeMicroflowGenToString_ReturnTypeDisplay verifies the
// returns-clause renders correctly for each return-type shape. All
// fixtures in expr-checker/minimal.mpr have empty `ReturnType()` strings
// — the real type information lives inside the `MicroflowReturnType()`
// part element (a DataType subtype). Until Stage 3.2 this code only
// looked at the bare string and silently dropped the clause, hiding
// entity/list/primitive returns from `describe microflow` output.
//
// Parity with `genFlowReturnDisplay` (used by nanoflows since 3.2.5c).
func TestDescribeMicroflowGenToString_ReturnTypeDisplay(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	cases := []struct {
		qn      string
		want    string // exact substring expected in output
		notWant string // optional negative assertion
	}{
		// StringType — primitive
		{qn: "FeedbackModule.ConvertUUIDToURL", want: "\nreturns String\n"},
		// BooleanType — primitive
		{qn: "FeedbackModule.VAL_Feedback", want: "\nreturns Boolean\n"},
		// ObjectType — entity (short-name display)
		{qn: "FeedbackModule.SUB_Feedback_SendToServer", want: "\nreturns "},
		// VoidType — must NOT emit a returns clause
		{qn: "MyFirstModule.MyFirstLogic", notWant: "\nreturns "},
	}

	for _, tc := range cases {
		t.Run(tc.qn, func(t *testing.T) {
			mf := findMicroflowByQN(t, w, tc.qn)
			out, err := DescribeMicroflowGenToString(ctx, mf)
			if err != nil {
				t.Fatalf("DescribeMicroflowGenToString: %v", err)
			}
			if tc.want != "" && !strings.Contains(out, tc.want) {
				t.Errorf("expected substring %q; got:\n%s", tc.want, out)
			}
			if tc.notWant != "" && strings.Contains(out, tc.notWant) {
				t.Errorf("did not expect substring %q; got:\n%s", tc.notWant, out)
			}
		})
	}

	// Spot-check the entity case more thoroughly: SUB_Feedback_SendToServer
	// returns the Feedback entity. The nanoflow-side parity test already
	// asserts the same entity short-name flows out of the helper.
	t.Run("entity-return short-name", func(t *testing.T) {
		mf := findMicroflowByQN(t, w, "FeedbackModule.SUB_Feedback_SendToServer")
		out, err := DescribeMicroflowGenToString(ctx, mf)
		if err != nil {
			t.Fatalf("DescribeMicroflowGenToString: %v", err)
		}
		// Must NOT degrade to "Object" or empty — those would mean we
		// surfaced the gen TypeName instead of the resolved entity name.
		if strings.Contains(out, "\nreturns Object\n") {
			t.Errorf("entity-returning microflow should not surface bare 'Object'; got:\n%s", out)
		}
		if !strings.Contains(out, "\nreturns ") {
			t.Errorf("entity-returning microflow should emit a returns clause; got:\n%s", out)
		}
	})
}

// TestGenMicroflowParameters_EntityType guards the Bug-1 fix: when a
// MicroflowParameter carries a VariableType (DataTypes$ObjectType) child
// element the describer must return the entity qualified name, not the
// plain "Object" fallback that surfaces when VariableType is absent or
// the describer only reads the deprecated Type() string.
func TestGenMicroflowParameters_EntityType(t *testing.T) {
	// Build an ObjectType child pointing at a specific entity.
	ot := genDt.NewObjectType()
	ot.SetEntityQualifiedName("Mod.MyEntity")

	// Wire the ObjectType as the VariableType of a fresh MicroflowParameter.
	param := genMf.NewMicroflowParameter()
	param.SetName("Dto")
	param.SetParameterType(ot)

	// Wrap in a MicroflowObjectCollection so genMicroflowParameters can find it.
	oc := genMf.NewMicroflowObjectCollection()
	oc.AddObjects(param)

	mf := genMf.NewMicroflow()
	mf.SetObjectCollection(oc)

	got := genMicroflowParameters(mf)
	if len(got) != 1 {
		t.Fatalf("params = %d, want 1", len(got))
	}
	// Core assertion: entity QN must be surfaced, not the "Object" fallback.
	if got[0].declType != "Mod.MyEntity" {
		t.Errorf("declType = %q, want Mod.MyEntity (entity QN from VariableType)", got[0].declType)
	}
}

// TestGenMicroflowParameters_PrimitiveVariableType checks that a
// MicroflowParameter with a DataTypes$StringType VariableType surfaces
// "String" — not "Object".
func TestGenMicroflowParameters_PrimitiveVariableType(t *testing.T) {
	st := genDt.NewStringType()

	param := genMf.NewMicroflowParameter()
	param.SetName("Name")
	param.SetParameterType(st)

	oc := genMf.NewMicroflowObjectCollection()
	oc.AddObjects(param)

	mf := genMf.NewMicroflow()
	mf.SetObjectCollection(oc)

	got := genMicroflowParameters(mf)
	if len(got) != 1 {
		t.Fatalf("params = %d, want 1", len(got))
	}
	if got[0].declType != "String" {
		t.Errorf("declType = %q, want String", got[0].declType)
	}
}

// TestGenMicroflowParameters_FixtureEntityParam verifies that entity
// parameters stored in a real MPR (fixture has Administration.Account
// in ShowPasswordForm) surface their entity QN through the full decode
// path, not just the in-memory construction used by the unit tests above.
func TestGenMicroflowParameters_FixtureEntityParam(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "Administration.ShowPasswordForm")

	params := genMicroflowParameters(mf)
	found := false
	for _, p := range params {
		if p.name == "Account" {
			found = true
			if p.declType != "Administration.Account" {
				t.Errorf("Account param declType = %q, want Administration.Account", p.declType)
			}
		}
	}
	if !found {
		t.Error("Account parameter not found in ShowPasswordForm")
	}
}

// TestDescribeMicroflowGen_BreakEvent verifies that a BreakEvent inside a
// while-true loop renders as `break;` instead of the TODO Stage 3.2.2 placeholder.
// Uses `while true` so no entity type is needed (avoids fixture dependency).
func TestDescribeMicroflowGen_BreakEvent(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	be, err := mprbackend.NewFromPath(w.ConcreteReader().Path())
	if err != nil {
		t.Fatalf("open backend: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })
	exec := New(io.Discard)
	exec.SetBackend(be)

	mdl := `create or modify microflow MyFirstModule.TestBreak () returns Nothing
{
  while true {
    break;
  }
  return;
}`
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		t.Fatalf("parse error: %v", errs[0])
	}
	if err := exec.ExecuteProgram(prog); err != nil {
		t.Fatalf("create microflow failed: %v", err)
	}

	mf := findMicroflowByQN(t, w, "MyFirstModule.TestBreak")
	out, err := DescribeMicroflowGenToString(ctx, mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}
	if strings.Contains(out, "TODO Stage 3.2.2:") {
		t.Errorf("BreakEvent rendered as TODO placeholder:\n%s", out)
	}
	if !strings.Contains(out, "break;") {
		t.Errorf("expected `break;` in output; got:\n%s", out)
	}
}

// TestDescribeMicroflowGen_ContinueEvent verifies that a ContinueEvent inside
// a while-true loop renders as `continue;`.
func TestDescribeMicroflowGen_ContinueEvent(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	be, err := mprbackend.NewFromPath(w.ConcreteReader().Path())
	if err != nil {
		t.Fatalf("open backend: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })
	exec := New(io.Discard)
	exec.SetBackend(be)

	mdl := `create or modify microflow MyFirstModule.TestContinue () returns Nothing
{
  while true {
    continue;
  }
  return;
}`
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		t.Fatalf("parse error: %v", errs[0])
	}
	if err := exec.ExecuteProgram(prog); err != nil {
		t.Fatalf("create microflow failed: %v", err)
	}

	mf := findMicroflowByQN(t, w, "MyFirstModule.TestContinue")
	out, err := DescribeMicroflowGenToString(ctx, mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}
	if strings.Contains(out, "TODO Stage 3.2.2:") {
		t.Errorf("ContinueEvent rendered as TODO placeholder:\n%s", out)
	}
	if !strings.Contains(out, "continue;") {
		t.Errorf("expected `continue;` in output; got:\n%s", out)
	}
}

// TestDescribeMicroflowGen_InheritanceCaseLabel verifies that inheritance-split
// case arms show the entity qualified name, not the TODO placeholder.
func TestDescribeMicroflowGen_InheritanceCaseLabel(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "Administration.ManageMyAccount")

	out, err := DescribeMicroflowGenToString(newGenDescribeContext(t, w), mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}
	if strings.Contains(out, "TODO Stage 3.2.2:") {
		t.Errorf("inheritance case label still a TODO placeholder:\n%s", out)
	}
	// The case arms must reference real entity names (split type $Var \n case Module.Entity).
	if !strings.Contains(out, "case Administration.") {
		t.Errorf("expected `case Administration.<Entity>` in output; got:\n%s", out)
	}
}

// mustContain reports a test failure listing every needle that's not in
// the haystack — easier to debug than rerunning to find the next miss.
func mustContain(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	var missing []string
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		t.Errorf("output missing %d expected substrings: %q\nFull output:\n%s",
			len(missing), missing, haystack)
	}
}

func mustNotContain(t *testing.T, haystack string, forbidden ...string) {
	t.Helper()
	for _, f := range forbidden {
		if strings.Contains(haystack, f) {
			t.Errorf("output must not contain %q\nFull output:\n%s", f, haystack)
		}
	}
}

// --- fixture copy helpers ---

func copyMPRFixture(t *testing.T, srcMPR string) string {
	t.Helper()
	snap, err := goldenfs.Open(filepath.Dir(srcMPR))
	if err == nil {
		t.Cleanup(func() { snap.Close() })
		return filepath.Join(snap.MountDir(), filepath.Base(srcMPR))
	}
	if !errors.Is(err, goldenfs.ErrNotSupported) {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	dstDir := testfsutil.SameFSTempDir(t)
	dstMPR := filepath.Join(dstDir, filepath.Base(srcMPR))
	if err := testfsutil.CopyFile(srcMPR, dstMPR); err != nil {
		t.Fatalf("copy mpr: %v", err)
	}
	srcContents := filepath.Join(filepath.Dir(srcMPR), "mprcontents")
	if info, err := os.Stat(srcContents); err == nil && info.IsDir() {
		dstContents := filepath.Join(dstDir, "mprcontents")
		if err := testfsutil.HardLinkDir(srcContents, dstContents); err != nil {
			t.Fatalf("hard-link mprcontents: %v", err)
		}
	}
	return dstMPR
}

// TestDescribeMicroflowGen_ListReturnAndLoop is a regression test that
// ensures the DESCRIBE output for a microflow with a List return type and
// a foreach loop produces valid MDL that passes the parser without errors.
//
// Before the fix:
//   - ListType was rendered as `List<ShortName>` (invalid syntax)
//   - LoopedActivity header was rendered as bare `loop` (missing $var IN $list)
func TestDescribeMicroflowGen_ListReturnAndLoop(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	be, err := mprbackend.NewFromPath(w.ConcreteReader().Path())
	if err != nil {
		t.Fatalf("open backend: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })
	ex := New(io.Discard)
	ex.SetBackend(be)

	// Create a microflow that exercises both the List return type and a
	// foreach loop. Administration.Account exists in the fixture MPR.
	mdl := `create or modify microflow MyFirstModule.TestListLoop (
  $Accs: List of Administration.Account
)
returns List of Administration.Account as $Result
{
  $Result = CREATE LIST OF Administration.Account;
  LOOP $Acc IN $Accs {
    ADD $Acc TO $Result;
  }
  RETURN $Result;
}`

	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		t.Fatalf("test MDL parse error: %v", errs[0])
	}
	if err := ex.ExecuteProgram(prog); err != nil {
		t.Fatalf("create microflow failed: %v", err)
	}

	mf := findMicroflowByQN(t, w, "MyFirstModule.TestListLoop")
	out, err := DescribeMicroflowGenToString(ctx, mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}

	// The DESCRIBE output must be re-parseable without errors.
	_, parseErrs := visitor.Build(out)
	if len(parseErrs) > 0 {
		t.Errorf("DESCRIBE output failed to parse (%d error(s)); output:\n%s\nfirst error: %v",
			len(parseErrs), out, parseErrs[0])
	}

	// Spot-check: correct list return syntax (not List<...>).
	if !strings.Contains(out, "returns List of Administration.Account") {
		t.Errorf("expected `returns List of Administration.Account` in output; got:\n%s", out)
	}

	// Spot-check: correct loop header with variable names.
	if !strings.Contains(out, "loop $Acc in $Accs") {
		t.Errorf("expected `loop $Acc in $Accs` in output; got:\n%s", out)
	}
}

// TestQuoteIfReserved_Reserved guards the Issue-003 fix: parameter names
// that collide with MDL reserved words must be backtick-quoted in DESCRIBE
// MICROFLOW output so the rendered MDL roundtrips through the parser.
func TestQuoteIfReserved_Reserved(t *testing.T) {
	for _, kw := range []string{"Template", "Attribute", "Column", "List", "Row", "Item"} {
		got := quoteIfReserved(kw)
		want := "`" + kw + "`"
		if got != want {
			t.Errorf("quoteIfReserved(%q) = %q, want %q", kw, got, want)
		}
	}
}

// TestQuoteIfReserved_Plain ensures ordinary identifiers are left untouched.
func TestQuoteIfReserved_Plain(t *testing.T) {
	for _, plain := range []string{"ImportData", "FileContent", "OrderLine"} {
		got := quoteIfReserved(plain)
		if got != plain {
			t.Errorf("quoteIfReserved(%q) = %q, want %q", plain, got, plain)
		}
	}
}

// TestDescribeMicroflowGenToString_NestedLoopBodyComplete is a regression
// test for the nested-loop DESCRIBE truncation bug: when an inner
// LoopedActivity contains more than one body activity, all activities
// after the first were silently dropped because emitLoopedActivityGen
// received the outer loop's filtered flow map (innerByOrigin) rather than
// the root-level Microflow.FlowsItems() map. The inner-inner SequenceFlows
// live at the microflow level, so filtering against the outer body's flows
// always produced an empty edge set for the inner body.
//
// Structure (all SequenceFlows at top level, matching Mendix BSON layout):
//
//	StartEvent → outerLoop → EndEvent
//	outerLoop.ObjectCollection = [ innerLoop ]
//	innerLoop.ObjectCollection = [ sv1, sv2 ]
//	Flows: S→OL, OL→E, sv1→sv2   ← sv1→sv2 is at top level
func TestDescribeMicroflowGenToString_NestedLoopBodyComplete(t *testing.T) {
	startID := element.ID("start-nested-1")
	outerID := element.ID("outer-loop-nested-1")
	endID := element.ID("end-nested-1")
	innerID := element.ID("inner-loop-nested-1")
	sv1ID := element.ID("set-var-nested-1")
	sv2ID := element.ID("set-var-nested-2")

	// Inner loop body: two ActionActivity nodes wrapping ChangeVariableAction.
	// formatActivityGen requires an ActionActivity wrapper — bare actions are
	// rendered as TODO placeholders.
	makeSetVar := func(id element.ID, name, val string) *genMf.ActionActivity {
		cva := genMf.NewChangeVariableAction()
		cva.SetChangeVariableName(name)
		cva.SetValue(val)
		aa := genMf.NewActionActivity()
		aa.SetID(id)
		aa.SetAction(cva)
		return aa
	}

	sv1 := makeSetVar(sv1ID, "shouldSkip", "true")
	sv2 := makeSetVar(sv2ID, "line1", "''")

	innerOC := genMf.NewMicroflowObjectCollection()
	innerOC.AddObjects(sv1)
	innerOC.AddObjects(sv2)

	innerSrc := genMf.NewIterableList()
	innerSrc.SetListVariableName("list2")
	innerSrc.SetVariableName("b")

	innerLoop := genMf.NewLoopedActivity()
	innerLoop.SetID(innerID)
	innerLoop.SetLoopSource(innerSrc)
	innerLoop.SetObjectCollection(innerOC)

	// Outer loop body: just the inner loop
	outerOC := genMf.NewMicroflowObjectCollection()
	outerOC.AddObjects(innerLoop)

	outerSrc := genMf.NewIterableList()
	outerSrc.SetListVariableName("list1")
	outerSrc.SetVariableName("a")

	outerLoop := genMf.NewLoopedActivity()
	outerLoop.SetID(outerID)
	outerLoop.SetLoopSource(outerSrc)
	outerLoop.SetObjectCollection(outerOC)

	// Top-level activities
	start := genMf.NewStartEvent()
	start.SetID(startID)
	end := genMf.NewEndEvent()
	end.SetID(endID)

	topOC := genMf.NewMicroflowObjectCollection()
	topOC.AddObjects(start)
	topOC.AddObjects(outerLoop)
	topOC.AddObjects(end)

	makeFlow := func(from, to element.ID) *genMf.SequenceFlow {
		sf := genMf.NewSequenceFlow()
		sf.SetOriginID(from)
		sf.SetDestinationID(to)
		return sf
	}

	mf := genMf.NewMicroflow()
	mf.SetName("NestedLoopBug")
	mf.SetObjectCollection(topOC)
	// Top-level flows: the microflow spine
	mf.AddFlows(makeFlow(startID, outerID))
	mf.AddFlows(makeFlow(outerID, endID))
	// Inner loop body flows — at top level, as Mendix BSON layout requires.
	// The bug: these flows are filtered out when building innerByOrigin,
	// so the second body activity (sv2) is never visited.
	mf.AddFlows(makeFlow(sv1ID, sv2ID))

	ctx := &ExecContext{
		ExecIO: ExecIO{Output: io.Discard},
	}
	ctx.initRoles()
	out, err := DescribeMicroflowGenToString(ctx, mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}

	// Both inner loop body statements must appear in the output.
	if !strings.Contains(out, "$shouldSkip") {
		t.Errorf("missing first inner loop body activity ($shouldSkip) in DESCRIBE output:\n%s", out)
	}
	if !strings.Contains(out, "$line1") {
		t.Errorf("missing second inner loop body activity ($line1) in DESCRIBE output — nested loop truncation bug:\n%s", out)
	}
}
