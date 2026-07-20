// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// gitFailBackend returns an ExecContext backed by a MockBackend that reports
// MPR v2 with a non-empty ContentsDir — enough to reach findChangedMxunitFiles.
func gitFailBackend(contentsDir string) *ExecContext {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		VersionFunc:     func() types.MPRVersion { return 2 },
		ContentsDirFunc: func() string { return contentsDir },
	}
	ctx := &ExecContext{
		Context:     context.Background(),
		Backend:     mb,
		Output: &bytes.Buffer{},
		Cache: &executorCache{},
	}
	ctx.initRoles()
	return ctx
}

// TestDiffLocal_GitError_ReturnsError is a regression test for issue #424:
// when git is not available or the project is not in a git repo, DiffLocal
// must return a non-nil error so the CLI can exit with code 1.
//
// The test replaces execCommand with a stub that simulates "git not a repo"
// (exit 128) so no real git process is spawned.
func TestDiffLocal_GitError_ReturnsError(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	// Simulate `git diff` failing with exit 128 (not a git repo).
	execCommand = func(name string, args ...string) *exec.Cmd {
		// Use the "false" command (always exits 1) or a helper binary that
		// exits with a specific code. exec.Command("sh", "-c", "exit 128")
		// produces the same *ExitError that real git produces.
		return exec.Command("sh", "-c", "exit 128")
	}

	ctx := gitFailBackend("/tmp/mprcontents")
	err := diffLocal(ctx, "HEAD", DiffOptions{})
	if err == nil {
		t.Fatal("diffLocal must return a non-nil error when git fails (issue #424: exit 0 on git error)")
	}

	// Error message should mention "git diff" so callers can surface a useful message.
	if !strings.Contains(err.Error(), "git diff") {
		t.Errorf("error message should mention 'git diff', got: %q", err.Error())
	}
}

// TestDiffLocal_NoChanges_ReturnsNil verifies that when git succeeds but
// reports no changed files, DiffLocal returns nil (exit 0 is correct).
func TestDiffLocal_NoChanges_ReturnsNil(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	// Simulate `git diff` succeeding with empty output (no changes).
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}

	ctx := gitFailBackend("/tmp/mprcontents")
	err := diffLocal(ctx, "HEAD", DiffOptions{})
	if err != nil {
		t.Errorf("diffLocal must return nil when git reports no changes; got: %v", err)
	}
}

// readMxunitForTest loads a `.mxunit` file from the expr-checker fixture
// and returns its raw BSON bytes. Fixture paths embed the unit UUID,
// shared with the SQLite `Unit` table — that's how mxcli's own loader
// resolves units in v2 mode.
func readMxunitForTest(t *testing.T, unitUUID string) []byte {
	t.Helper()
	// Layout: testdata/expr-checker/mprcontents/<aa>/<bb>/<uuid>.mxunit
	if len(unitUUID) < 4 {
		t.Fatalf("unit UUID too short: %q", unitUUID)
	}
	d1 := unitUUID[0:2]
	d2 := unitUUID[2:4]
	root := filepath.Join("..", "..", "testdata", "expr-checker", "mprcontents")
	path := filepath.Join(root, d1, d2, unitUUID+".mxunit")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// TestMicroflowBsonToMDL_DecodesViaCodec verifies that microflowBsonToMDL
// now decodes raw BSON via codec.Decoder + DescribeMicroflowGenToString
// (Followup B), replacing the Stage 3.2.6.3a header-only placeholder.
//
// We feed the same SUB_Feedback_SendToServer fixture used by the
// describe tests — its raw .mxunit bytes are checked into the repo at
// testdata/expr-checker/mprcontents/e6/50/<uuid>.mxunit.
func TestMicroflowBsonToMDL_DecodesViaCodec(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	// SUB_Feedback_SendToServer — entity-returning microflow, exercises
	// header + parameters + returns + body activities.
	const unitUUID = "e650b806-b4f4-47b9-81ae-5ee9d1cf7914"
	content := readMxunitForTest(t, unitUUID)

	out := microflowBsonToMDL(ctx, content, "FeedbackModule.SUB_Feedback_SendToServer")

	// Reject the legacy header-only stub.
	if strings.Contains(out, "diff-local body rendering pending") {
		t.Fatalf("expected real body, got legacy stub:\n%s", out)
	}
	// Reject the new fallback markers — they only appear if codec/cast/describe failed.
	if strings.Contains(out, "diff-local: ") {
		t.Fatalf("expected successful render, got fallback:\n%s", out)
	}

	mustContain(t, out,
		"create or modify microflow FeedbackModule.SUB_Feedback_SendToServer", // header
		"$Feedback: FeedbackModule.Feedback",                                  // parameter with resolved entity QN
		"\nreturns ",                                                          // returns clause now appears (entity return)
		"\n{\n",                                                               // body open
		"\n}",                                                                 // body close
		"\n/",                                                                 // statement terminator
		"call microflow ",                                                     // body actually rendered (not stub)
	)
}

// TestNanoflowBsonToMDL_DecodesViaCodec mirrors the microflow test for
// nanoflows. SUB_Feedback_GetOrCreate has an entity return + body.
func TestNanoflowBsonToMDL_DecodesViaCodec(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenNanoflowDescribeContext(t, w)

	// We don't hard-code the nanoflow UUID — discover it via the gen
	// repo so the test is robust to fixture re-extraction.
	repo := mprrepos.NewNanoflowRepository(w)
	nf := findNanoflowByQNGen(t, repo, ctx, "FeedbackModule.SUB_Feedback_GetOrCreate")
	unitUUID := string(nf.ID())
	content := readMxunitForTest(t, unitUUID)

	out := nanoflowBsonToMDL(ctx, content, "FeedbackModule.SUB_Feedback_GetOrCreate")

	if strings.Contains(out, "diff-local: ") {
		t.Fatalf("expected successful render, got fallback:\n%s", out)
	}

	mustContain(t, out,
		"create or modify nanoflow FeedbackModule.SUB_Feedback_GetOrCreate",
		"\nreturns ",
		"\n{\n",
		"\n}",
		"\n/",
	)
}

// TestMicroflowBsonToMDL_FallbackOnGarbage exercises the fallback path
// — empty bytes, then non-microflow BSON. We don't want a panic and we
// want a recognisable diagnostic line for diff-local consumers.
func TestMicroflowBsonToMDL_FallbackOnGarbage(t *testing.T) {
	w := openMprWriterForTest(t)
	ctx := newGenDescribeContext(t, w)

	t.Run("empty content", func(t *testing.T) {
		out := microflowBsonToMDL(ctx, nil, "Mod.Foo")
		if !strings.Contains(out, "diff-local:") {
			t.Errorf("expected fallback diagnostic; got:\n%s", out)
		}
		if !strings.Contains(out, "Mod.Foo") {
			t.Errorf("fallback must echo qualified name; got:\n%s", out)
		}
	})

	t.Run("not a microflow", func(t *testing.T) {
		// Feed an entity .mxunit instead of a microflow one. Find any
		// entity in the fixture by walking mprcontents — pick the first
		// file that decodes to anything other than a microflow.
		root := filepath.Join("..", "..", "testdata", "expr-checker", "mprcontents")
		var content []byte
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".mxunit") {
				return nil
			}
			data, _ := os.ReadFile(p)
			// Cheap $Type peek: grep the bytes for "Microflows$Microflow".
			if !bytes.Contains(data, []byte("Microflows$Microflow\x00")) &&
				!bytes.Contains(data, []byte("Microflows$Nanoflow\x00")) {
				content = data
				return filepath.SkipAll
			}
			return nil
		})
		if len(content) == 0 {
			t.Skip("no non-microflow .mxunit file found in fixture")
		}
		out := microflowBsonToMDL(ctx, content, "Mod.Bar")
		if !strings.Contains(out, "not a Microflow") {
			t.Errorf("expected `not a Microflow` diagnostic; got:\n%s", out)
		}
	})
}
