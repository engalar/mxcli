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
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
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
	// We open a second sdk/mpr.Writer on the same MPR file (already
	// copied into a tempdir by the helper) and Wrap it; modernc/sqlite
	// supports multiple opens of the same file.
	path := w.ConcreteReader().Path()
	sdkW, err := sdkmpr.NewWriter(path)
	if err != nil {
		t.Fatalf("sdkmpr.NewWriter(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = sdkW.Close() })
	be := mprbackend.Wrap(sdkW, path)

	return &ExecContext{
		Backend:    be,
		Microflows: repoCtx.Microflows,
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
