// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"flag"
	"fmt"
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
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/pmezard/go-difflib/difflib"
)

// updateGolden: go test ... -update-golden -run TestHelpdeskGolden_Update
var updateGolden = flag.Bool("update-golden", false,
	"overwrite testdata/helpdesk-golden/ with the current MDL execution result")

// helpdeskBlankDir returns the directory containing the blank base MPR (A).
// Uses testdata/expr-checker which is already committed and v2-format.
func helpdeskBlankDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "testdata", "expr-checker")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/expr-checker not found: %v", err)
	}
	return dir
}

// helpdeskBlankMPR returns the path to the blank base MPR file.
func helpdeskBlankMPR(t *testing.T) string {
	t.Helper()
	p := filepath.Join(helpdeskBlankDir(t), "minimal.mpr")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("testdata/expr-checker/minimal.mpr not found: %v", err)
	}
	return p
}

// helpdeskGoldenDir returns the path to the committed B1 golden directory.
func helpdeskGoldenDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "helpdesk-golden")
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
	for i, section := range sections {
		label := fmt.Sprintf("section-%d", i+1)
		// Extract MARK comment for logging
		for _, line := range strings.SplitN(section, "\n", 3) {
			if strings.HasPrefix(line, "-- MARK:") {
				label = strings.TrimPrefix(line, "-- MARK: ")
				break
			}
		}
		t.Logf("Executing: %s", label)
		runMDL(t, mprPath, section)
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

// TestHelpdeskGolden_Update 生成或更新 testdata/helpdesk-golden/。
// 只在 -update-golden flag 存在时有效；否则直接 Skip。
// 运行方式：
//
//	go test ./internal/goldenfs/ -tags linux,integration \
//	       -run TestHelpdeskGolden_Update -update-golden -v
func TestHelpdeskGolden_Update(t *testing.T) {
	if !*updateGolden {
		t.Skip("pass -update-golden to regenerate testdata/helpdesk-golden/")
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

	// Copy entire FUSE mount (A + dirty layer = B2) to testdata/helpdesk-golden/.
	// NOTE: do NOT call snap.Commit() — that would write back to blankDir (A).
	if err := os.RemoveAll(goldenDir); err != nil {
		t.Fatalf("remove old golden: %v", err)
	}
	if err := copyDir(snap.MountDir(), goldenDir); err != nil {
		t.Fatalf("copy to golden: %v", err)
	}

	t.Logf("Golden updated: %s", goldenDir)
	t.Logf("Next step: git add testdata/helpdesk-golden/ && git commit")
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

	// mx check: verify B2 BSON is valid to Studio Pro
	mxBin := findMxBinaryForTest()
	if mxBin == "" {
		t.Log("mx binary not available — skipping mx check (set MX_BINARY or install mxbuild)")
	} else {
		cmd := exec.Command(mxBin, "check", mountMPR)
		output, _ := cmd.CombinedOutput()
		// Use assertNoFUSECorruption for fatal BSON error signatures
		assertNoFUSECorruption(t, string(output),
			"HD.Ticket", "HD.Customer", "KB.Article",
			"HD.ACT_Ticket_Submit", "HD.WF_TicketEscalation",
		)
		// Full CE check: expect 0 [error] lines
		var ceErrors []string
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "[error]") {
				ceErrors = append(ceErrors, line)
			}
		}
		if len(ceErrors) > 0 {
			t.Errorf("mx check found %d error(s) — expected 0:\n%s",
				len(ceErrors), strings.Join(ceErrors, "\n"))
		}
	}
}

// helpdeskParseableDescribeScript returns the MDL script (no tabular SHOW
// commands) that describes every artifact in helpdesk-app.mdl. The output of
// executing this script is valid MDL and can be parsed by visitor.Build.
func helpdeskParseableDescribeScript(mprPath string) string {
	return fmt.Sprintf(`connect local '%s';
-- KB entities
describe entity KB.Category;
describe entity KB.Tag;
describe entity KB.Article;
describe entity KB.ArticleTag;
describe entity KB.ArticleRating;
-- HD entities
describe entity HD.Customer;
describe entity HD.Agent;
describe entity HD.Ticket;
describe entity HD.TicketComment;
describe entity HD.EscalationRequest;
describe entity HD.TicketSearch;
-- KB associations
describe association KB.Category_Parent;
describe association KB.Article_Category;
describe association KB.ArticleTag_Article;
describe association KB.ArticleTag_Tag;
describe association KB.ArticleRating_Article;
-- HD associations
describe association HD.Ticket_Customer;
describe association HD.Ticket_Agent;
describe association HD.TicketComment_Ticket;
describe association HD.EscalationRequest_Ticket;
describe association HD.Ticket_KBArticle;
-- Enumerations
describe enumeration KB.ArticleStatus;
describe enumeration HD.TicketStatus;
describe enumeration HD.TicketPriority;
-- KB microflows
describe microflow KB.ACT_Article_Publish;
describe microflow KB.ACT_Article_Archive;
describe microflow KB.SUB_Article_TruncateContent;
-- KB nanoflows
describe nanoflow KB.NF_Article_FormatPreview;
-- HD ticket microflows
describe microflow HD.ACT_Ticket_Submit;
describe microflow HD.ACT_Ticket_Assign;
describe microflow HD.ACT_Ticket_Resolve;
describe microflow HD.ACT_Ticket_Reopen;
describe microflow HD.ACT_Ticket_Close;
describe microflow HD.ACT_Ticket_SafeCommit;
describe microflow HD.ACT_Ticket_MarkCommentsRead;
describe microflow HD.ACT_EscalationRequest_Cleanup;
describe microflow HD.DS_OverdueTicketCount;
-- HD nanoflows
describe nanoflow HD.NF_Ticket_QuickCreate;
describe nanoflow HD.NF_TicketSearch_Apply;
describe nanoflow HD.NF_Priority_GetLabel;
-- HD escalation microflows
describe microflow HD.WFA_GetManagerAssignees;
describe microflow HD.WFS_SendReminder;
describe microflow HD.WFS_Approve;
describe microflow HD.WFS_Reject;
describe microflow HD.WFS_Escalation_Initialize;
describe microflow HD.WFS_AutoReject;
describe microflow HD.WFS_UpdateTicketPriority;
describe microflow HD.WFS_NotifyAgent;
describe microflow HD.WFC_EscalationRequest_OnCreate;
describe microflow HD.ACT_StartEscalation;
-- HD workflow admin microflows (all 13 activities)
describe microflow HD.ACT_Workflow_ChangeState;
describe microflow HD.ACT_Workflow_CompleteTask;
describe microflow HD.ACT_Workflow_GenerateJumpTo;
describe microflow HD.ACT_Workflow_ApplyJumpTo;
describe microflow HD.ACT_Workflow_GetHistory;
describe microflow HD.ACT_Workflow_GetContext;
describe microflow HD.DS_WorkflowInstances;
describe microflow HD.ACT_Workflow_ShowTaskPage;
describe microflow HD.ACT_Workflow_ShowAdminPage;
describe microflow HD.ACT_Workflow_Lock;
describe microflow HD.ACT_Workflow_Unlock;
describe microflow HD.ACT_Workflow_Notify;
-- Workflows
describe workflow HD.WF_SUB_ManagerReview;
describe workflow HD.WF_TicketEscalation;
`, mprPath)
}

// helpdeskFullDescribeScript extends helpdeskParseableDescribeScript with
// non-parseable SHOW commands (module roles). Used for regression text
// comparison and the describe snapshot.
func helpdeskFullDescribeScript(mprPath string) string {
	return helpdeskParseableDescribeScript(mprPath) + `show module roles in KB;
show module roles in HD;
`
}

// describeMDL 对 mprPath 执行一组 describe/show 命令，返回合并的文本输出。
func describeMDL(t *testing.T, mprPath string) string {
	t.Helper()
	var buf strings.Builder
	e := executor.New(&buf)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close: %v", err)
		}
	}()

	script := helpdeskFullDescribeScript(mprPath)

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
// (no MDL execution) and verifies that the full describe output matches
// the committed snapshot at testdata/helpdesk-golden/describe-snapshot.txt.
// Run with -update-golden to regenerate the snapshot.
func TestHelpdeskGolden_DescribeSnapshot(t *testing.T) {
	goldenMPR := helpdeskGoldenMPR(t)
	if _, err := os.Stat(goldenMPR); err != nil {
		t.Fatalf("B1 golden not found at %s — run: make update-helpdesk-golden", goldenMPR)
	}

	snapshotPath := filepath.Join(helpdeskGoldenDir(t), "describe-snapshot.txt")

	got := describeMDL(t, goldenMPR)

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
	e := executor.New(&buf)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close: %v", err)
		}
	}()

	script := helpdeskParseableDescribeScript(mprPath)

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
