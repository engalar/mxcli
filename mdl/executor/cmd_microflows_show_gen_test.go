// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.1 tests: structural skeleton + control-flow framing for
// DescribeMicroflowGenToString. Activity bodies in 3.2.1 are TODO
// placeholders — these tests verify the surrounding shape, not the
// per-activity output (that's 3.2.2's surface).

package executor

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const fixtureMprPath = "../../testdata/expr-checker/minimal.mpr"

// openMprWriterForTest copies the fixture into a temp dir (Writer
// mutates the SQLite file) and returns a Writer rooted at the copy.
func openMprWriterForTest(t *testing.T) *mmpr.Writer {
	t.Helper()
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	w, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("mmpr.NewWriter(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w
}

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

	return &ExecContext{
		Backend:    be,
		Microflows: repoCtx.Microflows,
		Nanoflows:  repoCtx.Nanoflows,
		Output:     io.Discard,
	}
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
		"\nbegin\n", // body open
		"\nend;",    // body close
		"\n/",       // statement terminator
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
		"if ", " then\n",
		"\n  else\n",
		"\n  end if;",
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

// TestDescribeMicroflowGenToString_InheritanceCaseFraming exercises the
// InheritanceSplit framing on Administration.ManageMyAccount, which
// case-splits on the current user's specialised type.
func TestDescribeMicroflowGenToString_InheritanceCaseFraming(t *testing.T) {
	w := openMprWriterForTest(t)
	mf := findMicroflowByQN(t, w, "Administration.ManageMyAccount")

	out, err := DescribeMicroflowGenToString(newGenDescribeContext(t, w), mf)
	if err != nil {
		t.Fatalf("DescribeMicroflowGenToString: %v", err)
	}

	mustContain(t, out,
		"create or modify microflow Administration.ManageMyAccount", // resolved module name
		"case $", " inheritance\n", // inheritance case header
		"    when ", " then\n", // at least one when arm
		"\n  end case;\n", // case terminator
	)
	if strings.Contains(out, "<unknown>") {
		t.Errorf("module name should resolve from container chain (no <unknown>); got:\n%s", out)
	}
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
	exec.backend = be

	mdl := `create or modify microflow MyFirstModule.TestBreak () returns Nothing
begin
  while true begin
    break;
  end while;
  return;
end;`
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
	exec.backend = be

	mdl := `create or modify microflow MyFirstModule.TestContinue () returns Nothing
begin
  while true begin
    continue;
  end while;
  return;
end;`
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
// `when` arms show the entity qualified name, not the TODO placeholder.
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
	// The when arms must reference real entity names (Module.Entity pattern).
	if !strings.Contains(out, "when Administration.") {
		t.Errorf("expected `when Administration.<Entity> then` in output; got:\n%s", out)
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

// --- fixture copy helpers (parallels mdl/backend/mpr/factory_test.go) ---

func copyMPRFixture(t *testing.T, srcMPR, dstDir string) string {
	t.Helper()
	dstMPR := filepath.Join(dstDir, filepath.Base(srcMPR))
	if err := copyOneFileForTest(srcMPR, dstMPR); err != nil {
		t.Fatalf("copy mpr: %v", err)
	}
	srcContents := filepath.Join(filepath.Dir(srcMPR), "mprcontents")
	if info, err := os.Stat(srcContents); err == nil && info.IsDir() {
		dstContents := filepath.Join(dstDir, "mprcontents")
		if err := copyDirForTest(srcContents, dstContents); err != nil {
			t.Fatalf("copy mprcontents: %v", err)
		}
	}
	return dstMPR
}

func copyOneFileForTest(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDirForTest(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyOneFileForTest(p, target)
	})
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
	ex.backend = be

	// Create a microflow that exercises both the List return type and a
	// foreach loop. Administration.Account exists in the fixture MPR.
	mdl := `create or modify microflow MyFirstModule.TestListLoop (
  $Accs: List of Administration.Account
)
returns List of Administration.Account as $Result
begin
  $Result = CREATE LIST OF Administration.Account;
  LOOP $Acc IN $Accs BEGIN
    ADD $Acc TO $Result;
  END LOOP;
  RETURN $Result;
end;`

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
