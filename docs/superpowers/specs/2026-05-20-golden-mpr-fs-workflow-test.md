# Golden MPR FS — Comprehensive Workflow Integration Test

**Date:** 2026-05-20
**Status:** Approved for implementation

---

## Goal

Add a `linux,integration`-tagged test in `internal/goldenfs/` that exercises the golden MPR FS overlay with a comprehensive Mendix Workflow object. The test creates an enumeration, a fully-typed entity, module-role security, and a workflow that uses every major activity type — then verifies the FUSE-written project passes `mx check` without FUSE-corruption signatures.

---

## Context

The existing `TestGoldenFS_ExecAndMxCheck` (in `mxcheck_test.go`) only creates a single empty microflow, exercising a narrow slice of the BSON write path. Mendix Workflow objects are significantly more complex: they contain nested BSON documents for each activity, boundary events, outcome routing, and targeting expressions. This test validates that the golden FS overlay correctly handles that complexity.

Reference documentation: `/mnt/data_sdd/gh/mendix_docs/content/en/docs/refguide/modeling/application-logic/workflows/`

---

## File

| File | Role |
|------|------|
| `internal/goldenfs/workflow_integration_test.go` | New integration test, `//go:build linux,integration` |

No changes to production code.

---

## MDL Execution Plan

The test executes 5 sequential `ExecuteProgram()` calls. Each is a separate step; failure messages name the step. After all steps succeed, `mx check` is invoked on the FUSE-mounted project path.

### Step 1 — Infrastructure (module role + stub microflows)

```sql
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
```

**Why separate step:** Security and microflow objects must exist before the entity grant and workflow can reference them.

### Step 2 — Data model (enumeration + entity)

```sql
create or modify enumeration MyFirstModule.WF_Status (
  Draft        'Draft',
  InReview     'In Review',
  Approved     'Approved',
  Rejected     'Rejected'
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
```

Covers all 10 Mendix native attribute types (string fixed, string unlimited, integer, long, decimal, boolean, datetime, binary, autonumber, enumeration).

### Step 3 — Entity access grant

```sql
grant MyFirstModule.Reviewer on MyFirstModule.WF_Item
  (create, delete, read *, write *);
```

### Step 4 — Create workflow

```sql
create or modify workflow MyFirstModule.WF_ComplexApproval
  parameter $ctx: MyFirstModule.WF_Item
  display 'Complex Approval Flow'
  description 'Comprehensive workflow integration test covering all major activity types'
begin
  user task InitialReview 'Initial Review'
    targeting users xpath '[Status = ''Draft'']'
    outcomes
      'Submit' {
        decision '$ctx/Amount > 1000'
          outcomes
            true -> {
              multi user task SeniorApproval 'Senior Approval'
                targeting groups xpath '[id != 0]'
                outcomes
                  'Approve' { }
                  'Reject'  { };
            }
            false -> {
              user task StandardApproval 'Standard Approval'
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
          outcomes
            'Sign'              { }
            'ReturnForRevision' { jump to InitialReview; }
          ;
      }
      'Cancel' { };
end workflow;
```

**Activity types covered:**

| Activity | Location |
|----------|----------|
| User Task | InitialReview, StandardApproval, FinalSignOff |
| Multi-User Task | SeniorApproval (groups xpath targeting) |
| Decision | After 'Submit', expression `$ctx/Amount > 1000` |
| Parallel Split | 2 paths after wait-for-notification |
| Call Microflow | path 1 of parallel split |
| Wait for Notification | Between decision merge and parallel split |
| Jump To | FinalSignOff 'ReturnForRevision' outcome loops back |

### Step 5 — Boundary event (ALTER WORKFLOW)

```sql
alter workflow MyFirstModule.WF_ComplexApproval
  insert boundary event on InitialReview
    non interrupting timer 'addHours([%CurrentDateTime%], 24)' {
      call microflow MyFirstModule.ACT_BoundaryHandler;
    };
```

Adds a non-interrupting 24-hour timer boundary event to `InitialReview`. Uses `ALTER WORKFLOW INSERT` to verify the alter-path write is also correctly handled by the overlay.

---

## Test Structure

```go
func TestGoldenFS_WorkflowIntegration(t *testing.T) {
    mxBin := findMxBinaryForTest()
    if mxBin == "" {
        t.Skip("mx binary not available (set MX_BINARY or install mxbuild)")
    }

    // Snapshot isolation: all writes go to FUSE overlay, base untouched
    snap, err := Open(exprCheckerDir(t))
    ...
    defer snap.Close()

    mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

    e := executor.New(io.Discard)
    defer e.Close()
    e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })

    run := func(label, script string) {
        t.Helper()
        prog, errs := visitor.Build(fmt.Sprintf("connect local '%s'; %s", mprPath, script))
        if len(errs) > 0 {
            t.Fatalf("[%s] parse error: %v", label, errs)
        }
        if err := e.ExecuteProgram(prog); err != nil {
            t.Fatalf("[%s] exec error: %v", label, err)
        }
    }

    run("step1-infra",     step1InfraScript)
    run("step2-datamodel", step2DataModelScript)
    run("step3-grant",     step3GrantScript)
    run("step4-workflow",  step4WorkflowScript)
    run("step5-alter",     step5AlterScript)

    if err := e.Close(); err != nil {
        t.Logf("executor close: %v", err)
    }

    // mx check on FUSE-mounted project
    cmd := exec.Command(mxBin, "check", mprPath)
    output, _ := cmd.CombinedOutput()
    assertNoFUSECorruption(t, string(output))

    // Base directory must be untouched
    for _, suffix := range []string{"-wal", "-shm", "-journal"} {
        if _, err := os.Stat(filepath.Join(exprCheckerDir(t), "minimal.mpr"+suffix)); err == nil {
            t.Fatalf("%s must not exist in base dir", suffix)
        }
    }
    snap.Rollback()
}
```

The `run` helper is defined inline; `assertNoFUSECorruption` reuses the same logic as `mxcheck_test.go` (filter fatal BSON corruption signatures, allow pre-existing CE errors).

---

## Helpers Reused from Existing Tests

| Helper | Defined in | Used for |
|--------|-----------|----------|
| `exprCheckerDir(t)` | `integration_test.go` | locates testdata/expr-checker |
| `findMxBinaryForTest()` | `mxcheck_test.go` | finds `mx` binary |
| FUSE corruption assertion | `mxcheck_test.go` | `assertNoFUSECorruption` (extract to shared helper) |

`assertNoFUSECorruption` should be extracted from `mxcheck_test.go` into a `helpers_test.go` (also `linux,integration` tagged) so both tests can use it without duplication.

---

## Assertions

**Pass conditions (mx check output):**
- Zero lines matching `StorageLoadException`, `TypeCacheUnknownTypeException`, `Invalid file format`
- Zero lines matching `CE0066`, `CE0463`, `CE1613` on objects created by this test
- `WF_ComplexApproval` and `WF_Item` do not appear alongside any of the above error codes

**Pre-existing errors:** The testdata project has ~1078 CE6083 design-property errors unrelated to this test. These are allowed through.

**Isolation:**
- `minimal.mpr` mtime unchanged after test
- No `-wal`, `-shm`, `-journal` files in `testdata/expr-checker/`
- `snap.Rollback()` called at end (Close via defer would also suffice, but explicit rollback documents intent)

---

## Out of Scope

- Creating Workflow pages (requires page BSON which adds significant complexity)
- User role + demo user creation (not required for `mx check` structural validation)
- GRANT WORKFLOW ACCESS (no dedicated MDL syntax — workflow access is controlled via entity/microflow/page grants, which Step 3 covers)
- Testing Rollback/Commit of workflow objects (covered by existing `TestSnapshot_Rollback_BaseUnchanged`)

---

## Self-Review

**Placeholder scan:** None found.

**Internal consistency:**
- Step 1 creates `ACT_Notify` and `ACT_BoundaryHandler` before they are referenced in Steps 4–5. ✅
- Step 2 creates `WF_Status` before it is used as attribute type in `WF_Item`. ✅
- Step 3 grants on `WF_Item` which exists after Step 2. ✅
- `assertNoFUSECorruption` is marked for extraction — this is a concrete refactor, not a placeholder. ✅

**Scope:** Single file, no production code changes, reuses existing test infrastructure. Appropriately scoped. ✅

**Ambiguity:** The `connect local '...'` command is prepended to each script segment via the `run` helper, ensuring the executor reconnects to the FUSE-mounted MPR on each call. This matches the pattern in `mxcheck_test.go`. ✅
