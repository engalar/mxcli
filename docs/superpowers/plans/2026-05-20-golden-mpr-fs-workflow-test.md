# Golden MPR FS Workflow Integration Test — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `TestGoldenFS_WorkflowIntegration` — a `linux,integration`-tagged test that writes a comprehensive Mendix Workflow (all major activity types + boundary event) through the golden FUSE overlay and verifies `mx check` produces no FUSE-corruption signatures.

**Architecture:** Two tasks. Task 1 extracts shared test helpers from `mxcheck_test.go` into `helpers_test.go` (DRY refactor). Task 2 writes the new integration test in `workflow_integration_test.go` using those helpers. All code is in `internal/goldenfs/`, build tag `linux && integration`. No production code changes.

**Tech Stack:** Go test (`testing`), `mdl/executor`, `mdl/backend/mpr`, `mdl/visitor`, MDL (Mendix Definition Language), FUSE overlay (`goldenfs.Open`), Mendix `mx check` binary.

**Spec:** `docs/superpowers/specs/2026-05-20-golden-mpr-fs-workflow-test.md`

---

## File Map

| File | Role |
|------|------|
| `internal/goldenfs/helpers_test.go` | **Create** — shared helpers: `findMxBinaryForTest`, `assertNoFUSECorruption`, `checkFUSEIsolation` |
| `internal/goldenfs/mxcheck_test.go` | **Modify** — remove duplicated helpers, call shared ones |
| `internal/goldenfs/workflow_integration_test.go` | **Create** — `TestGoldenFS_WorkflowIntegration` with 6-step MDL execution |

All three files: `//go:build linux && integration`.

---

## Task 1: Extract shared test helpers

**Files:**
- Create: `internal/goldenfs/helpers_test.go`
- Modify: `internal/goldenfs/mxcheck_test.go`

- [ ] **Step 1: Create `helpers_test.go` with three shared helpers**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// findMxBinaryForTest returns the path to an mx binary, or "" if unavailable.
// Prefers MX_BINARY env; falls back to highest lexicographic version under
// ~/.mxcli/mxbuild/*/modeler/mx.
func findMxBinaryForTest() string {
	if p := os.Getenv("MX_BINARY"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		pattern := filepath.Join(home, ".mxcli", "mxbuild", "*", "modeler", "mx")
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			// Last in lexicographic order — set MX_BINARY to override when version
			// layout causes incorrect selection (e.g. 11.10.x < 11.9.x lexicographically).
			return matches[len(matches)-1]
		}
	}
	return ""
}

// assertNoFUSECorruption fails the test if output contains any FUSE-corruption
// signature, or if any of ourObjects appears in a flagged error line.
// Pre-existing CE6083 design-property errors are allowed through.
func assertNoFUSECorruption(t *testing.T, output string, ourObjects ...string) {
	t.Helper()
	fatalSignatures := []string{
		"StorageLoadException",  // SQLite file corruption / mprcontents desync
		"TypeCacheUnknownType",  // BSON $Type written that Mendix doesn't recognise
		"CE0066",                // Entity access out of date
		"CE0463",                // Widget definition changed (BSON shape mismatch)
		"CE1613",                // Layout no longer exists
		"Invalid file format",   // SQLite header damage
	}
	for _, sig := range fatalSignatures {
		if strings.Contains(output, sig) {
			t.Fatalf("mx check reported FUSE-corruption signature %q in output", sig)
		}
	}
	for _, obj := range ourObjects {
		if strings.Contains(output, obj) {
			t.Fatalf("our newly-created object %q is flagged by mx check", obj)
		}
	}
}

// checkFUSEIsolation verifies that the real MPR file was not modified and
// no SQLite auxiliary files leaked to the base directory.
func checkFUSEIsolation(t *testing.T, realMpr string, origStat os.FileInfo) {
	t.Helper()
	newStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr after: %v", err)
	}
	if !origStat.ModTime().Equal(newStat.ModTime()) {
		t.Fatalf("real minimal.mpr mtime changed (%v → %v) — overlay leaked",
			origStat.ModTime(), newStat.ModTime())
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(realMpr + suffix); err == nil {
			t.Fatalf("%s must not exist in base dir after FUSE write", suffix)
		}
	}
}
```

- [ ] **Step 2: Update `mxcheck_test.go` to use the shared helpers**

Remove `findMxBinaryForTest` (lines 22-39) from `mxcheck_test.go` — it now lives in `helpers_test.go`.

Replace the inline fatal-signatures check and base-dir check with calls to the shared helpers. The updated test body (from `// mx check...` onward) becomes:

```go
	cmd := exec.Command(mxBin, "check", mprPath)
	output, err := cmd.CombinedOutput()
	t.Logf("mx check output (%d bytes, err=%v):\n%s", len(output), err, output)

	assertNoFUSECorruption(t, string(output), "ACT_GoldenTest")
	checkFUSEIsolation(t, realMpr, origMprStat)

	snap.Rollback()
```

Also remove `"strings"` from the imports if it is no longer used in `mxcheck_test.go` after this change (the string scan now lives in the helper).

- [ ] **Step 3: Build both files together to verify no duplicate symbols**

```bash
go build -tags integration ./internal/goldenfs/...
```

Expected: no errors, no "already declared" compile errors.

- [ ] **Step 4: Run the existing integration test to confirm it still works**

```bash
go test -tags integration ./internal/goldenfs/ -run TestGoldenFS_ExecAndMxCheck -v
```

Expected: PASS (or SKIP if `mx` binary unavailable). All 20 unit tests must continue to pass without the tag:

```bash
go test ./internal/goldenfs/ -v -count=1 2>&1 | tail -3
```

Expected: `ok github.com/mendixlabs/mxcli/internal/goldenfs`.

- [ ] **Step 5: Commit**

```bash
git add internal/goldenfs/helpers_test.go internal/goldenfs/mxcheck_test.go
git commit -m "refactor(goldenfs): extract shared integration test helpers to helpers_test.go"
```

---

## Task 2: Write `TestGoldenFS_WorkflowIntegration`

**Files:**
- Create: `internal/goldenfs/workflow_integration_test.go`

The test opens a FUSE snapshot, connects the executor once, runs 6 sequential MDL scripts (each via a separate `ExecuteProgram` call with a named label), disconnects the executor, runs `mx check`, asserts no corruption, and verifies base-dir isolation.

- [ ] **Step 1: Create `workflow_integration_test.go`**

Write the complete file:

```go
// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// step scripts are package-level vars so each const stays readable.

const wfStep1aInfra = `
create module role MyFirstModule.Reviewer
  description 'Integration test reviewer role';

create or modify microflow MyFirstModule.ACT_Notify ()
  returns Nothing
  begin
    return;
  end;

create or modify microflow MyFirstModule.ACT_BoundaryHandler ()
  returns Nothing
  begin
    return;
  end;
`

const wfStep1bPages = `
create page MyFirstModule.Page_WF_InitialReview    layout Atlas_Core.Atlas_TopBar ();
create page MyFirstModule.Page_WF_StandardApproval layout Atlas_Core.Atlas_TopBar ();
create page MyFirstModule.Page_WF_SeniorApproval   layout Atlas_Core.Atlas_TopBar ();
create page MyFirstModule.Page_WF_FinalSignOff     layout Atlas_Core.Atlas_TopBar ();
`

const wfStep2DataModel = `
create or modify enumeration MyFirstModule.WF_Status (
  Draft    'Draft',
  InReview 'In Review',
  Approved 'Approved',
  Rejected 'Rejected'
);

create or modify entity MyFirstModule.WF_Item (
  Title       : string(200),
  Description : string(unlimited),
  Count       : integer,
  Total       : long,
  Amount      : decimal,
  IsActive    : boolean default true,
  DueDate     : datetime,
  Payload     : binary,
  SeqNo       : autonumber,
  Status      : enumeration(MyFirstModule.WF_Status) default MyFirstModule.WF_Status.Draft
);
`

const wfStep3Grant = `
grant MyFirstModule.Reviewer on MyFirstModule.WF_Item
  (create, delete, read *, write *);
`

const wfStep4Workflow = `
create or modify workflow MyFirstModule.WF_ComplexApproval
  parameter $ctx: MyFirstModule.WF_Item
  display 'Complex Approval Flow'
  description 'Comprehensive workflow integration test: user task, multi-user task, decision, parallel split, call microflow, wait for notification, jump to'
begin
  user task InitialReview 'Initial Review'
    page MyFirstModule.Page_WF_InitialReview
    targeting users xpath '[Status = ''Draft'']'
    outcomes
      'Submit' {
        decision '$ctx/Amount > 1000'
          outcomes
            true -> {
              multi user task SeniorApproval 'Senior Approval'
                page MyFirstModule.Page_WF_SeniorApproval
                targeting groups xpath '[id != 0]'
                outcomes
                  'Approve' { }
                  'Reject'  { };
            }
            false -> {
              user task StandardApproval 'Standard Approval'
                page MyFirstModule.Page_WF_StandardApproval
                targeting users xpath '[id != 0]'
                outcomes
                  'Approve' { }
                  'Reject'  { };
            }
          ;
        wait for notification comment 'AwaitExternalSignal';
        parallel split
          path 1 { call microflow MyFirstModule.ACT_Notify; }
          path 2 { }
        ;
        user task FinalSignOff 'Final Sign-off'
          page MyFirstModule.Page_WF_FinalSignOff
          outcomes
            'Sign'              { }
            'ReturnForRevision' { jump to InitialReview; }
          ;
      }
      'Cancel' { };
end workflow;
`

const wfStep5Alter = `
alter workflow MyFirstModule.WF_ComplexApproval
  insert boundary event on InitialReview
    non interrupting timer 'addHours([%CurrentDateTime%], 24)' {
      call microflow MyFirstModule.ACT_BoundaryHandler;
    };
`

// TestGoldenFS_WorkflowIntegration verifies that a comprehensive Mendix
// Workflow (user task, multi-user task, decision, parallel split, call
// microflow, wait-for-notification, jump-to, boundary event) can be written
// through the golden FUSE overlay without corrupting the project or leaking
// to the base directory.
func TestGoldenFS_WorkflowIntegration(t *testing.T) {
	mxBin := findMxBinaryForTest()
	if mxBin == "" {
		t.Skip("mx binary not available (set MX_BINARY or install mxbuild)")
	}

	realDir := exprCheckerDir(t)
	realMpr := filepath.Join(realDir, "minimal.mpr")
	origStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

	snap, err := Open(realDir)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

	e := executor.New(io.Discard)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close warning: %v", err)
		}
	}()

	run := func(label, script string) {
		t.Helper()
		prog, errs := visitor.Build(script)
		if len(errs) > 0 {
			t.Fatalf("[%s] MDL parse error: %v", label, errs)
		}
		if err := e.ExecuteProgram(prog); err != nil {
			t.Fatalf("[%s] executor error: %v", label, err)
		}
	}

	// Connect once; executor keeps the connection open across subsequent run() calls.
	run("connect", fmt.Sprintf("connect local '%s';", mprPath))
	run("step1a-infra", wfStep1aInfra)
	run("step1b-pages", wfStep1bPages)
	run("step2-datamodel", wfStep2DataModel)
	run("step3-grant", wfStep3Grant)
	run("step4-workflow", wfStep4Workflow)
	run("step5-alter", wfStep5Alter)

	// Disconnect before mx check so SQLite releases all FUSE file handles.
	if err := e.Close(); err != nil {
		t.Fatalf("close executor: %v", err)
	}

	cmd := exec.Command(mxBin, "check", mprPath)
	output, err := cmd.CombinedOutput()
	t.Logf("mx check output (%d bytes, err=%v):\n%s", len(output), err, output)

	assertNoFUSECorruption(t, string(output),
		"WF_ComplexApproval",
		"WF_Item",
		"WF_Status",
		"Page_WF_",
		"ACT_Notify",
		"ACT_BoundaryHandler",
	)
	checkFUSEIsolation(t, realMpr, origStat)

	snap.Rollback()
}
```

- [ ] **Step 2: Compile the new file (expect success)**

```bash
go build -tags integration ./internal/goldenfs/...
```

Expected: no errors. If you get "undefined: findMxBinaryForTest", check that `helpers_test.go` has the `linux && integration` build tag (not just `linux`).

- [ ] **Step 3: Run the new test**

```bash
go test -tags integration ./internal/goldenfs/ -run TestGoldenFS_WorkflowIntegration -v
```

Expected: PASS (or SKIP if `mx` binary unavailable).

**If a step fails with an MDL parse error**, the error message will include the step label (e.g. `[step4-workflow] MDL parse error`). Check the MDL syntax in the corresponding `wfStep*` constant:
- Pages: if `create page ... layout ... ();` fails, check `.claude/skills/create-page.md` for the exact syntax and adjust (may need at least one widget or a different closing syntax)
- Workflow: verify the `decision ... outcomes true -> { ... } false -> { ... }` nesting against `mdl-examples/doctype-tests/24-workflow-examples.mdl`
- Boundary event: verify `non interrupting timer` syntax against the same file

**If mx check fails** with one of the asserted object names, the executor wrote incorrect BSON. Isolate by removing later steps and running with `mx check` after each step to find the first bad write.

- [ ] **Step 4: Confirm unit tests are unaffected**

```bash
go test ./internal/goldenfs/ -v -count=1 2>&1 | tail -3
```

Expected: `ok github.com/mendixlabs/mxcli/internal/goldenfs` — 20 tests, 0 failures.

- [ ] **Step 5: Commit**

```bash
git add internal/goldenfs/workflow_integration_test.go
git commit -m "test(goldenfs): comprehensive workflow integration test (all activity types + boundary event)"
```

---

## Self-Review

**Spec coverage:**
- ✅ Step 1a: module role + ACT_Notify + ACT_BoundaryHandler
- ✅ Step 1b: 4 workflow task pages (Atlas_Core.Atlas_TopBar layout — confirmed in testdata)
- ✅ Step 2: WF_Status enum (4 values) + WF_Item entity (10 attribute types incl. binary, enum)
- ✅ Step 3: entity access grant (create/delete/read*/write*)
- ✅ Step 4: workflow with User Task, Multi-User Task, Decision, Parallel Split, Call Microflow, Wait for Notification, Jump To
- ✅ Step 5: ALTER WORKFLOW — non-interrupting timer boundary event
- ✅ Assertions: `assertNoFUSECorruption` + `checkFUSEIsolation`
- ✅ `assertNoFUSECorruption` extracted to `helpers_test.go` (no duplication)

**Placeholder scan:** No TBD, TODO, or vague steps. All code blocks are complete.

**Type consistency:**
- `findMxBinaryForTest` defined in Task 1 `helpers_test.go`, used in Task 2 test function ✅
- `assertNoFUSECorruption(t, output string, ourObjects ...string)` — variadic signature used correctly in both tests ✅
- `checkFUSEIsolation(t, realMpr string, origStat os.FileInfo)` — `os.Stat` returns `(os.FileInfo, error)`, stored as `origStat os.FileInfo` ✅
- `wfStep*` constants are all untyped string consts, passed as `string` to `run(label, script string)` ✅
- `exprCheckerDir(t)` defined in `integration_test.go` (same package, same build tag) ✅

**Executor reconnect:** The `run("connect", ...)` call connects once. Subsequent `run()` calls send additional statements to the same connected executor. This matches the existing `mxcheck_test.go` pattern and is correct — the executor is stateful. ✅
