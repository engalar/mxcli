// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/executor/domainmodel"
	"github.com/mendixlabs/mxcli/mdl/executor/microflow"
	"github.com/mendixlabs/mxcli/mdl/executor/misc"
	"github.com/mendixlabs/mxcli/mdl/executor/page"
	"github.com/mendixlabs/mxcli/mdl/executor/query"
	"github.com/mendixlabs/mxcli/mdl/executor/security"
	"github.com/mendixlabs/mxcli/mdl/executor/workflow"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/pmezard/go-difflib/difflib"
)

// updateGolden: go test ... -update-golden -run TestHelpdeskGolden_Update
var updateGolden = flag.Bool("update-golden", false,
	"overwrite testdata/helpdesk-golden-<HELPDESK_VERSION>/ with the current MDL execution result")

// helpdeskVersion returns the Mendix version to use for golden tests.
// Controlled by the HELPDESK_VERSION env var; defaults to "11.6.6".
func helpdeskVersion() string {
	if v := os.Getenv("HELPDESK_VERSION"); v != "" {
		return v
	}
	return "11.6.6"
}

// helpdeskBlankDir returns the directory containing the blank base MPR (A).
// Uses testdata/helpdesk-clean-11.6.6: a blank 11.6.6 project with Atlas Core
// and the DataGrid2/DropdownFilter widget MPKs already present, giving the
// correct widget baseline for the helpdesk pages.
func helpdeskBlankDir(t *testing.T) string {
	t.Helper()
	v := helpdeskVersion()
	dir := filepath.Join(repoRoot(t), "testdata", "helpdesk-clean-"+v)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/helpdesk-clean-%s not found: %v", v, err)
	}
	return dir
}

// helpdeskBlankMPR returns the path to the blank base MPR file.
func helpdeskBlankMPR(t *testing.T) string {
	t.Helper()
	p := filepath.Join(helpdeskBlankDir(t), "minimal.mpr")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("testdata/helpdesk-clean-%s/minimal.mpr not found: %v", helpdeskVersion(), err)
	}
	return p
}

// helpdeskGoldenDir returns the path to the committed B1 golden directory.
func helpdeskGoldenDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "helpdesk-golden-"+helpdeskVersion())
}

// helpdeskGoldenMPR returns the MPR path inside the golden directory.
func helpdeskGoldenMPR(t *testing.T) string {
	t.Helper()
	return filepath.Join(helpdeskGoldenDir(t), "minimal.mpr")
}

// helpdeskMDLSections reads mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
// and splits it into sections at "-- MARK:" boundaries.
// Each section is run in a fresh executor to avoid the executor list-cache bug
// where newly created enumerations are not visible to subsequent entity creation
// within the same batch (same issue as in workflow_integration_test.go).
func helpdeskMDLSections(t *testing.T) []string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "mdl-examples", "use-cases", "helpdesk", "helpdesk-app.mdl")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read helpdesk-app.mdl: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	var sections []string
	var cur strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "-- MARK:") && cur.Len() > 0 {
			sections = append(sections, cur.String())
			cur.Reset()
		}
		cur.WriteString(line)
		cur.WriteByte('\n')
	}
	if cur.Len() > 0 {
		sections = append(sections, cur.String())
	}
	return sections
}

// runHelpdeskMDL executes helpdesk-app.mdl against mprPath in multiple passes,
// one per "-- MARK:" section. Each pass uses a fresh executor to avoid the
// list-cache issue where newly created types are not visible in the same batch.
func runHelpdeskMDL(t *testing.T, mprPath string) {
	t.Helper()
	sections := helpdeskMDLSections(t)

	// Shared executor: create once, reuse for all sections.
	// Between sections, explicit disconnect+connect refreshes the backend
	// connection and executorCache (including graph catalog and entity name
	// maps), avoiding the stale-cache issue of a single batch.
	//
	// This saves 33 executor creations + 33×7 handler registrations vs
	// calling runMDL per section.

	e := executor.New(io.Discard)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.ConnectionBackend { return mprbackend.New() })
	deps := e.BuildHandlerDeps()
	microflow.RegisterHandlers(e.Registry(), deps)
	page.RegisterHandlers(e.Registry(), deps)
	workflow.RegisterHandlers(e.Registry(), deps)
	domainmodel.RegisterHandlers(e.Registry(), deps)
	security.RegisterHandlers(e.Registry(), deps)
	query.RegisterHandlers(e.Registry(), deps)
	misc.RegisterHandlers(e.Registry(), deps)
	e.AddReregister(func(fresh *executor.HandlerDeps) {
		microflow.RegisterHandlers(e.Registry(), fresh)
		page.RegisterHandlers(e.Registry(), fresh)
		workflow.RegisterHandlers(e.Registry(), fresh)
		domainmodel.RegisterHandlers(e.Registry(), fresh)
		security.RegisterHandlers(e.Registry(), fresh)
		query.RegisterHandlers(e.Registry(), fresh)
		misc.RegisterHandlers(e.Registry(), fresh)
	})
	defer func() {
		if err := e.Close(); err != nil {
			t.Errorf("runHelpdeskMDL: executor close: %v", err)
		}
	}()

	for i, section := range sections {
		label := fmt.Sprintf("section-%d", i+1)
		for _, line := range strings.SplitN(section, "\n", 3) {
			if strings.HasPrefix(line, "-- MARK:") {
				label = strings.TrimPrefix(line, "-- MARK: ")
				break
			}
		}
		t.Logf("Executing: %s", label)
		// Each section runs as its own connect+execute+disconnect cycle
		// within the same executor, avoiding stale caches.
		full := "connect local '" + mprPath + "';\n" + section + "\ndisconnect;\n"
		prog, errs := visitor.Build(full)
		if len(errs) > 0 {
			t.Fatalf("MDL parse error in %s: %v", label, errs)
		}
		if err := e.ExecuteProgram(prog); err != nil {
			t.Fatalf("executor error in %s: %v", label, err)
		}
	}
}

// copyDir copies src directory tree to dst, creating dst if needed.
// Overwrites existing files in dst.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
}

// TestHelpdeskGolden_Update 生成或更新 testdata/helpdesk-golden-11.6.6/。
// 只在 -update-golden flag 存在时有效；否则直接 Skip。
// 运行方式：
//
//	go test ./internal/goldenfs/ -tags linux,integration \
//	       -run TestHelpdeskGolden_Update -update-golden -v
func TestHelpdeskGolden_Update(t *testing.T) {
	if !*updateGolden {
		t.Skip("pass -update-golden to regenerate testdata/helpdesk-golden-11.6.6/")
	}

	blankDir := helpdeskBlankDir(t)
	goldenDir := helpdeskGoldenDir(t)

	// Open FUSE overlay on top of blank A.
	snap, err := Open(blankDir)
	if err != nil {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	defer func() {
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close: %v", err)
		}
	}()

	mountMPR := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Execute helpdesk-app.mdl in multiple passes (one per MARK section) to
	// avoid the executor list-cache bug where newly created enumerations are
	// not visible to subsequent entity creation in the same batch.
	runHelpdeskMDL(t, mountMPR)

	// Copy entire FUSE mount (A + dirty layer = B2) to testdata/helpdesk-golden-11.6.6/.
	// NOTE: do NOT call snap.Commit() — that would write back to blankDir (A).
	// Use best-effort removal: skip files owned by other users (e.g., .claude/
	// skills directories synced from the host). copyDir overwrites existing files.
	if err := os.RemoveAll(goldenDir); err != nil {
		t.Logf("remove old golden (partial, continuing): %v", err)
	}
	if err := copyDir(snap.MountDir(), goldenDir); err != nil {
		t.Fatalf("copy to golden: %v", err)
	}

	// Regenerate describe-snapshot.mdl for each project so the committed
	// snapshots always reflect the actual MPR content — not whatever stale
	// file happened to be in helpdesk-clean-11.6.6/ before the copy.
	//
	//  • helpdesk-golden-11.6.6/describe-snapshot.mdl — post-MDL (full app)
	//  • helpdesk-clean-11.6.6/describe-snapshot.mdl  — pre-MDL (blank base)
	goldenSnapshotPath := filepath.Join(goldenDir, "describe-snapshot.mdl")
	goldenSnapshot := describeMDLParseable(t, filepath.Join(goldenDir, "minimal.mpr"))
	if err := os.WriteFile(goldenSnapshotPath, []byte(goldenSnapshot), 0o644); err != nil {
		t.Fatalf("write golden describe-snapshot: %v", err)
	}
	t.Logf("Snapshot updated: %s", goldenSnapshotPath)

	cleanSnapshotPath := filepath.Join(blankDir, "describe-snapshot.mdl")
	cleanSnapshot := describeMDLParseableClean(t, filepath.Join(blankDir, "minimal.mpr"))
	if err := os.WriteFile(cleanSnapshotPath, []byte(cleanSnapshot), 0o644); err != nil {
		t.Fatalf("write clean describe-snapshot: %v", err)
	}
	t.Logf("Snapshot updated: %s", cleanSnapshotPath)

	t.Logf("Golden updated: %s", goldenDir)
	t.Logf("Next step: git add testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-clean-11.6.6/describe-snapshot.mdl && git commit")
}

// TestHelpdeskGolden_Regression_BSON 是主 BSON 层回归测试。
// 从空白 A（expr-checker）出发，通过 FUSE 执行 helpdesk-app.mdl 得到 B2,
// 然后与 B1 golden 做字段级 BSON 对比。
// 任何字段缺失、字段值变化、单元增减都会导致测试失败。
func TestHelpdeskGolden_Regression_BSON(t *testing.T) {
	goldenMPR := helpdeskGoldenMPR(t)
	if _, err := os.Stat(goldenMPR); err != nil {
		t.Fatalf("B1 golden not found at %s — run: make update-helpdesk-golden", goldenMPR)
	}

	snap, err := Open(helpdeskBlankDir(t))
	if err != nil {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	defer func() {
		snap.Rollback()
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close: %v", err)
		}
	}()

	mountMPR := filepath.Join(snap.MountDir(), "minimal.mpr")
	runHelpdeskMDL(t, mountMPR)

	bsoncompare.AssertEqual(t,
		goldenMPR, // A = B1 golden
		mountMPR,  // B = B2 current
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)

	// mx check: verify B2 BSON is valid to Studio Pro.
	// Use the binary matching helpdeskVersion() — not lex-last — because
	// "11.10.0" < "11.6.6" lexicographically, so the lex-last binary would
	// be 11.6.6 which cannot open an 11.10.0 project (InvalidOperationException).
	mxBin := findMxBinaryForVersion(helpdeskVersion())
	if mxBin == "" {
		t.Log("mx binary not available — skipping mx check (set MX_BINARY or install mxbuild)")
	} else {
		cmd := exec.Command(mxBin, "check", mountMPR)
		output, _ := cmd.CombinedOutput()
		// Filter out known pre-existing false positives before calling
		// assertNoFUSECorruption:
		//   CE0463 on dgTickets / fStatus — DataGrid2/DropdownFilter
		//     widget-definition mismatch (Studio Pro validates against platform-
		//     internal definition, not the project's local MPK).
		//   CE1613 on NanoflowCommons.GetCurrentLocation — JS action API changed;
		//     timeout/maximumAge/highAccuracy parameters removed in newer
		//     NanoflowCommons versions installed in the test base project.
		var filteredLines []string
		for _, line := range strings.Split(string(output), "\n") {
			if strings.Contains(line, "CE0463") &&
				(strings.Contains(line, "dgTickets") || strings.Contains(line, "fStatus")) {
				t.Logf("known CE0463 (pre-existing DataGrid2/DropdownFilter template mismatch): %s", strings.TrimSpace(line))
				continue
			}
			if strings.Contains(line, "CE1613") &&
				strings.Contains(line, "NanoflowCommons.GetCurrentLocation") {
				t.Logf("known CE1613 (pre-existing NanoflowCommons JS action API change): %s", strings.TrimSpace(line))
				continue
			}
			filteredLines = append(filteredLines, line)
		}
		filtered := strings.Join(filteredLines, "\n")

		assertNoFUSECorruption(t, filtered,
			"HD.Ticket", "HD.Customer", "KB.Article",
			"HD.ACT_Ticket_Submit", "HD.WF_TicketEscalation",
		)
		// CE0720 — placeholder index > parameter count (TextTemplate.Parameters key fix)
		// CE0091 — no member selected on validation feedback (object-only target)
		// CE0639 — no variable selected on validation feedback attribute
		for _, code := range []string{"CE0720", "CE0091", "CE0639"} {
			if strings.Contains(filtered, code) {
				t.Errorf("forbidden error %s in mx check output — BSON regression:\n%s", code, output)
			}
		}
	}
}

// helpdeskParseableDescribeScript returns the MDL script (no tabular SHOW
// commands) that describes every artifact in helpdesk-app.mdl. The output of
// executing this script is valid MDL and can be parsed by visitor.Build.
//
// The body is read from mdl-examples/use-cases/helpdesk/helpdesk-describe.mdl
// so that the file can be used standalone via "mxcli -p app.mpr exec helpdesk-describe.mdl".
// The test prepends "connect local '<mprPath>';" to target a temporary MPR copy.
func helpdeskParseableDescribeScript(t *testing.T, mprPath string) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "mdl-examples", "use-cases", "helpdesk", "helpdesk-describe.mdl")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read helpdesk-describe.mdl: %v", err)
	}
	return fmt.Sprintf("connect local '%s';\n", mprPath) + string(data)
}

// newDescribeExecutor creates a bare executor wired for describe/test use
// (factory backend + subpackage handlers + re-registration support).
func newDescribeExecutor(w io.Writer) *executor.Executor {
	e := executor.New(w)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.ConnectionBackend { return mprbackend.New() })
	deps := e.BuildHandlerDeps()
	microflow.RegisterHandlers(e.Registry(), deps)
	page.RegisterHandlers(e.Registry(), deps)
	workflow.RegisterHandlers(e.Registry(), deps)
	domainmodel.RegisterHandlers(e.Registry(), deps)
	security.RegisterHandlers(e.Registry(), deps)
	query.RegisterHandlers(e.Registry(), deps)
	misc.RegisterHandlers(e.Registry(), deps)
	e.AddReregister(func(fresh *executor.HandlerDeps) {
		microflow.RegisterHandlers(e.Registry(), fresh)
		page.RegisterHandlers(e.Registry(), fresh)
		workflow.RegisterHandlers(e.Registry(), fresh)
		domainmodel.RegisterHandlers(e.Registry(), fresh)
		security.RegisterHandlers(e.Registry(), fresh)
		query.RegisterHandlers(e.Registry(), fresh)
		misc.RegisterHandlers(e.Registry(), fresh)
	})
	return e
}

// helpdeskCleanDescribeScript returns the describe script for the blank
// base project (helpdesk-clean-11.6.6). It only targets MyFirstModule
// content that exists in every Mendix 11.6 starter project, never KB/HD
// which are absent from the blank base.
func helpdeskCleanDescribeScript(mprPath string) string {
	return fmt.Sprintf(`connect local '%s';
describe page MyFirstModule.Home_Web;
describe microflow MyFirstModule.MyFirstLogic;
`, mprPath)
}

// describeMDLParseableClean is like describeMDLParseable but uses the
// helpdeskCleanDescribeScript intended for the blank base project.
func describeMDLParseableClean(t *testing.T, mprPath string) string {
	t.Helper()
	var buf strings.Builder
	e := newDescribeExecutor(&buf)
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close: %v", err)
		}
	}()

	script := helpdeskCleanDescribeScript(mprPath)
	prog, errs := visitor.Build(script)
	if len(errs) > 0 {
		t.Fatalf("describeMDLParseable parse: %v", errs)
	}
	if err := e.ExecuteProgram(prog); err != nil {
		t.Logf("describeMDLParseable partial output:\n%s", buf.String())
		t.Fatalf("describeMDLParseable exec: %v", err)
	}
	return buf.String()
}

// helpdeskFullDescribeScript extends helpdeskParseableDescribeScript with
// non-parseable SHOW commands (module roles). Used for regression text
// comparison and the describe snapshot.
func helpdeskFullDescribeScript(t *testing.T, mprPath string) string {
	t.Helper()
	return helpdeskParseableDescribeScript(t, mprPath) + `show module roles in KB;
show module roles in HD;
`
}

// describeMDL 对 mprPath 执行一组 describe/show 命令，返回合并的文本输出。
func describeMDL(t *testing.T, mprPath string) string {
	t.Helper()
	var buf strings.Builder
	e := newDescribeExecutor(&buf)
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close: %v", err)
		}
	}()

	script := helpdeskFullDescribeScript(t, mprPath)

	prog, errs := visitor.Build(script)
	if len(errs) > 0 {
		t.Fatalf("describeMDL parse: %v", errs)
	}
	if err := e.ExecuteProgram(prog); err != nil {
		t.Logf("describeMDL partial output:\n%s", buf.String())
		t.Fatalf("describeMDL exec: %v", err)
	}
	return buf.String()
}

// TestHelpdeskGolden_DescribeSnapshot opens the committed B1 golden MPR
// (no MDL execution) and verifies that the parseable describe output matches
// the committed snapshot at testdata/helpdesk-golden-11.6.6/describe-snapshot.mdl.
// The snapshot is pure MDL (no tabular SHOW output) so that mxcli check can
// validate it. Run with -update-golden to regenerate the snapshot.
func TestHelpdeskGolden_DescribeSnapshot(t *testing.T) {
	goldenMPR := helpdeskGoldenMPR(t)
	if _, err := os.Stat(goldenMPR); err != nil {
		t.Fatalf("B1 golden not found at %s — run: make update-helpdesk-golden", goldenMPR)
	}

	snapshotPath := filepath.Join(helpdeskGoldenDir(t), "describe-snapshot.mdl")

	got := describeMDLParseable(t, goldenMPR)

	if *updateGolden {
		if err := os.WriteFile(snapshotPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write snapshot: %v", err)
		}
		t.Logf("Snapshot updated: %s", snapshotPath)
		return
	}

	want, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("snapshot not found at %s — run: go test ... -update-golden -run TestHelpdeskGolden_DescribeSnapshot", snapshotPath)
	}

	assertPageBodiesNonEmpty(t, string(want))

	if string(want) == got {
		return
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(want)),
		B:        difflib.SplitLines(got),
		FromFile: "snapshot (expected)",
		ToFile:   "describe output (actual)",
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	t.Errorf("describe snapshot mismatch (re-run with -update-golden to update):\n%s", text)
}

// describeMDLParseable runs a describe script that produces only MDL
// statements (no tabular SHOW output) so that the result can be fed
// directly to visitor.Build for AST-level assertions.
func describeMDLParseable(t *testing.T, mprPath string) string {
	t.Helper()
	var buf strings.Builder
	e := newDescribeExecutor(&buf)
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close: %v", err)
		}
	}()

	script := helpdeskParseableDescribeScript(t, mprPath)

	prog, errs := visitor.Build(script)
	if len(errs) > 0 {
		t.Fatalf("describeMDLParseable parse: %v", errs)
	}
	if err := e.ExecuteProgram(prog); err != nil {
		t.Logf("describeMDLParseable partial output:\n%s", buf.String())
		t.Fatalf("describeMDLParseable exec: %v", err)
	}
	return buf.String()
}

// TestHelpdeskGolden_DescribeSourceMatch opens the committed B1 golden MPR
// without executing any MDL and verifies that the describe output is
// semantically consistent with helpdesk-app.mdl: system members, explicit
// access-rule member lists, XPath constraints, and workflow parameters.
func TestHelpdeskGolden_DescribeSourceMatch(t *testing.T) {
	goldenMPR := helpdeskGoldenMPR(t)
	if _, err := os.Stat(goldenMPR); err != nil {
		t.Fatalf("B1 golden not found at %s — run: make update-helpdesk-golden", goldenMPR)
	}

	desc := describeMDLParseable(t, goldenMPR)

	prog, errs := visitor.Build(desc)
	if len(errs) > 0 {
		t.Fatalf("parse describe output: %v", errs)
	}

	// Index entities, grants by entity, and workflows.
	entities := make(map[string]*ast.CreateEntityStmt)
	grantsByEntity := make(map[string][]*ast.GrantEntityAccessStmt)
	workflows := make(map[string]*ast.CreateWorkflowStmt)

	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
		case *ast.CreateEntityStmt:
			entities[s.Name.String()] = s
		case *ast.GrantEntityAccessStmt:
			key := s.Entity.String()
			grantsByEntity[key] = append(grantsByEntity[key], s)
		case *ast.CreateWorkflowStmt:
			workflows[s.Name.String()] = s
		}
	}

	// --- System members ---

	hdTicket := requireEntityInDesc(t, entities, "HD.Ticket")
	assertSystemMembersContain(t, "HD.Ticket", hdTicket.SystemMembers,
		"owner", "createdDate", "changedDate", "changedBy")

	hdCustomer := requireEntityInDesc(t, entities, "HD.Customer")
	assertSystemMembersContain(t, "HD.Customer", hdCustomer.SystemMembers, "owner")

	kbRating := requireEntityInDesc(t, entities, "KB.ArticleRating")
	assertSystemMembersContain(t, "KB.ArticleRating", kbRating.SystemMembers, "owner")

	// --- Explicit read member list: HD.CustomerRole on HD.Ticket ---

	hdTicketGrants := grantsByEntity["HD.Ticket"]
	if len(hdTicketGrants) == 0 {
		t.Fatal("no grants found for HD.Ticket in describe output")
	}
	var customerRoleGrant *ast.GrantEntityAccessStmt
	for _, g := range hdTicketGrants {
		for _, role := range g.Roles {
			if role.String() == "HD.CustomerRole" {
				customerRoleGrant = g
			}
		}
	}
	if customerRoleGrant == nil {
		t.Fatal("HD.CustomerRole grant on HD.Ticket not found in describe output")
	}
	var hasReadMembers bool
	for _, right := range customerRoleGrant.Rights {
		if right.Type == ast.EntityAccessReadMembers && len(right.Members) > 0 {
			hasReadMembers = true
		}
	}
	if !hasReadMembers {
		t.Error("HD.CustomerRole on HD.Ticket: expected explicit read (...) member list, got none")
	}

	// --- Explicit read members + XPath: KB.Reader on KB.ArticleRating ---

	kbRatingGrants := grantsByEntity["KB.ArticleRating"]
	if len(kbRatingGrants) == 0 {
		t.Fatal("no grants found for KB.ArticleRating in describe output")
	}
	var readerGrant *ast.GrantEntityAccessStmt
	for _, g := range kbRatingGrants {
		for _, role := range g.Roles {
			if role.String() == "KB.Reader" {
				readerGrant = g
			}
		}
	}
	if readerGrant == nil {
		t.Fatal("KB.Reader grant on KB.ArticleRating not found in describe output")
	}
	if readerGrant.XPathConstraint == "" {
		t.Error("KB.Reader on KB.ArticleRating: expected XPath WHERE constraint, got empty")
	}
	var kbReadMembers bool
	for _, right := range readerGrant.Rights {
		if right.Type == ast.EntityAccessReadMembers && len(right.Members) > 0 {
			kbReadMembers = true
		}
	}
	if !kbReadMembers {
		t.Error("KB.Reader on KB.ArticleRating: expected explicit read (...) member list, got none")
	}

	// --- Workflows ---

	wfEsc, ok := workflows["HD.WF_TicketEscalation"]
	if !ok {
		t.Fatal("HD.WF_TicketEscalation not found in describe output")
	}
	if got := wfEsc.ParameterEntity.String(); got != "HD.EscalationRequest" {
		t.Errorf("WF_TicketEscalation parameter entity: want HD.EscalationRequest, got %s", got)
	}

	if _, ok := workflows["HD.WF_SUB_ManagerReview"]; !ok {
		t.Fatal("HD.WF_SUB_ManagerReview not found in describe output")
	}
}

// requireEntityInDesc fails with t.Fatal if name is not in the entities map.
func requireEntityInDesc(t *testing.T, entities map[string]*ast.CreateEntityStmt, name string) *ast.CreateEntityStmt {
	t.Helper()
	e, ok := entities[name]
	if !ok {
		t.Fatalf("entity %s not found in describe output", name)
	}
	return e
}

// assertSystemMembersContain fails if any of want is absent from got.
func assertSystemMembersContain(t *testing.T, label string, got []string, want ...string) {
	t.Helper()
	have := make(map[string]bool, len(got))
	for _, s := range got {
		have[s] = true
	}
	for _, w := range want {
		if !have[w] {
			t.Errorf("%s system members: missing %q (have: %v)", label, w, got)
		}
	}
}

// TestHelpdeskGolden_Regression_DescribeMDL 对比 B1 golden 和 B2 current
// 的 describe 输出文本，捕获 describe 层可见的语义退化。
func TestHelpdeskGolden_Regression_DescribeMDL(t *testing.T) {
	goldenMPR := helpdeskGoldenMPR(t)
	if _, err := os.Stat(goldenMPR); err != nil {
		t.Fatalf("B1 golden not found at %s — run: make update-helpdesk-golden", goldenMPR)
	}

	snap, err := Open(helpdeskBlankDir(t))
	if err != nil {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	defer func() {
		snap.Rollback()
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close: %v", err)
		}
	}()

	mountMPR := filepath.Join(snap.MountDir(), "minimal.mpr")
	runHelpdeskMDL(t, mountMPR)

	b1Desc := describeMDL(t, goldenMPR)
	b2Desc := describeMDL(t, mountMPR)

	if b1Desc == b2Desc {
		return
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(b1Desc),
		B:        difflib.SplitLines(b2Desc),
		FromFile: "golden (B1)",
		ToFile:   "current (B2)",
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	t.Errorf("describe MDL roundtrip mismatch:\n%s", text)
}

// assertPageBodiesNonEmpty checks that every "create or modify page" statement
// in the snapshot has a non-empty body (i.e. contains at least one widget keyword).
func assertPageBodiesNonEmpty(t *testing.T, snapshot string) {
	t.Helper()
	widgetKeywords := []string{
		"container", "layoutgrid", "dataview", "datagrid", "gallery",
		"listview", "button", "textbox", "textarea", "datepicker",
		"tabcontainer", "groupbox", "label", "text ", "snippet",
	}
	lines := strings.Split(snapshot, "\n")
	inPage := false
	pageStart := 0
	pageHeader := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "create or modify page ") {
			inPage = true
			pageStart = i
			pageHeader = trimmed
		}
		if inPage && trimmed == "}" {
			body := strings.Join(lines[pageStart:i+1], "\n")
			hasWidget := false
			for _, kw := range widgetKeywords {
				if strings.Contains(body, kw) {
					hasWidget = true
					break
				}
			}
			if !hasWidget {
				t.Errorf("page has empty body (no widget keywords): %s (line %d)", pageHeader, pageStart+1)
			}
			inPage = false
		}
	}
}

// runMDLLenient executes a MDL script statement-by-statement, logging (not fataling)
// on individual statement errors. Used for the first pass of idempotency tests where
// forward-dependency failures are expected (e.g. creating a module role before the
// module implicitly created by entity creation exists yet). Later statements continue
// executing even when earlier ones fail.
func runMDLLenient(t *testing.T, mprPath, script string) {
	t.Helper()
	e := executor.New(io.Discard)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.ConnectionBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("runMDLLenient: executor close: %v", err)
		}
	}()
	full := "connect local '" + mprPath + "';\n" + script
	prog, errs := visitor.Build(full)
	if len(errs) > 0 {
		t.Fatalf("MDL parse error: %v", errs)
	}
	for _, stmt := range prog.Statements {
		if err := e.Execute(stmt); err != nil {
			t.Logf("runMDLLenient (expected forward-dep error): %v", err)
		}
	}
}

// TestHelpdeskGolden_DescribeSnapshot_Idempotent verifies that describe-snapshot.mdl
// is self-consistent: executing it on the clean MPR and re-describing must produce
// the same output as the original snapshot.
//
// This catches lossy describe output that drops information the builder writes.
//
// Run: go test ./internal/goldenfs/ -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot_Idempotent -v
func TestHelpdeskGolden_DescribeSnapshot_Idempotent(t *testing.T) {
	snapshotPath := filepath.Join(helpdeskGoldenDir(t), "describe-snapshot.mdl")
	snapshotBytes, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("describe-snapshot.mdl not found at %s — run: make update-snapshots", snapshotPath)
	}

	snap, err := Open(helpdeskBlankDir(t))
	if err != nil {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	defer func() {
		snap.Rollback()
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close: %v", err)
		}
	}()

	mountMPR := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Execute describe-snapshot.mdl in two passes (same pattern as runHelpdeskMDL).
	//
	// Pass 1: creates entities (implicitly creating modules), enumerations, and
	//         module roles. Some statements may fail on the first pass because of
	//         forward dependencies (e.g. grants referencing roles not yet created);
	//         those are retried in pass 2.
	// Pass 2: all prerequisites exist; grants and cross-references now succeed.
	//
	// runMDL is defined in bsoncompare_integration_test.go (same package).
	runMDLLenient(t, mountMPR, string(snapshotBytes))
	runMDL(t, mountMPR, string(snapshotBytes))

	// Re-describe: this must produce the same output as the original snapshot.
	got := describeMDLParseable(t, mountMPR)

	// Normalize: strip describe-version-specific output before comparison.
	// The golden snapshot may differ from current describe output in
	// non-functional ways (returns Nothing style, role-inclusion comments).
	want := stripDescribeIrrelevant(string(snapshotBytes))
	gotNorm := stripDescribeIrrelevant(got)

	if want == gotNorm {
		return
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(want),
		B:        difflib.SplitLines(gotNorm),
		FromFile: "describe-snapshot.mdl (expected, normalized)",
		ToFile:   "re-describe after execute on clean MPR (actual, normalized)",
		Context:  5,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	t.Errorf("describe-snapshot.mdl is not idempotent — re-describe after executing on clean MPR differs:\n%s", text)
}

// stripDescribeIrrelevant normalizes describe output by removing lines that
// are not functionally part of the MDL contract, so the idempotent test can
// pass when the golden snapshot is slightly stale (e.g. different describe
// version output or role-inclusion annotations).
//
// Stripped lines:
//   - "returns Nothing" — describe versions differ on emitting it for microflows
//     ending with "return;". Both forms are semantically equivalent.
//   - "-- Included in user roles: …" — describe annotations that list which user
//     roles include a module role. These can drift across mxbuild versions or
//     project setup order without affecting functional correctness.
func stripDescribeIrrelevant(s string) string {
	lines := strings.Split(s, "\n")
	filtered := make([]string, 0, len(lines))
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "returns Nothing" {
			continue
		}
		if strings.HasPrefix(trimmed, "-- Included in user roles:") {
			continue
		}
		filtered = append(filtered, l)
	}
	return strings.Join(filtered, "\n")
}
