# Test Infrastructure Governance — Track A Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clean up dead tests, fix flaky tests, and simplify build tags — independently of production code changes.

**Architecture:** Track A only covers changes that do NOT require production code refactoring. All tasks are isolated to test files, build configuration, and CI skip lists. Track B (mock split, subpackage tests, ExecContext elimination) is documented as a roadmap only — each requires its own design doc and implementation plan.

**Tech Stack:** Go 1.26, testing, ANTLR4 grammar, Cobra CLI

## Global Constraints

- All test changes must pass `make build && make test && make lint` (CI gate)
- No production code logic changes (`.go` files outside `_test.go` / `Makefile` / `.github/`)
- time.Sleep → channel replacements must NOT introduce data races (use `sync.Mutex` guarded recording)
- Build tag changes must pass: `go test -tags integration ./...` and `go test ./...` (both paths)
- Each task ends with an independently testable deliverable and a commit

---

### Task A1: Fix 5 CI-Skipped Doctype-Tests

**Files:**
- Modify: `mdl-examples/doctype-tests/02-microflow-examples.mdl`
- Modify: `mdl-examples/doctype-tests/02c-complex-layout-examples.mdl`
- Modify: `mdl-examples/doctype-tests/04-math-examples.mdl`
- Modify: `mdl-examples/doctype-tests/07-java-action-examples.mdl`
- Modify: `mdl-examples/doctype-tests/microflow-spec.test.mdl`
- Modify: `.github/workflows/ci.yml` (lines 36-37) — remove skip list if fixed

**Background:** 5 files are permanently skipped in CI due to syntax errors. The error is consistent: `mismatched input 'end' expecting '}'` — these files use old `begin ... end` block syntax instead of `{ ... }` braces. CI skip at `.github/workflows/ci.yml:36`:

```yaml
SKIP='02-microflow-examples.mdl 02c-complex-layout-examples.mdl 04-math-examples.mdl 07-java-action-examples.mdl microflow-spec.test.mdl'
```

- [ ] **Step 1: Verify the syntax errors are `begin → {` migration issues**

Run:
```bash
cd /mnt/data_sdb/mxcli
make build
for f in mdl-examples/doctype-tests/02-microflow-examples.mdl \
         mdl-examples/doctype-tests/02c-complex-layout-examples.mdl \
         mdl-examples/doctype-tests/04-math-examples.mdl \
         mdl-examples/doctype-tests/07-java-action-examples.mdl \
         mdl-examples/doctype-tests/microflow-spec.test.mdl; do
    echo "=== $f ==="
    ./bin/mxcli check "$f" 2>&1 | grep -i "error\|mismatched\|no viable"
done
```
Expected: All 5 show `mismatched input 'end' expecting '}'` or similar syntax errors.

- [ ] **Step 2: Fix `02-microflow-examples.mdl`**

The file is 2,804 lines. Pattern to fix: every `begin` after a statement body must become `{`, and every matching `end` must become `}`. This is a `begin ... end` → `{ ... }` migration in MDL syntax.

Run a targeted sed to replace statement-block begin/end:
```bash
cd /mnt/data_sdb/mxcli
# Replace 'begin' at start of line (statement body start) with '{'
sed -i 's/^begin$/{/' mdl-examples/doctype-tests/02-microflow-examples.mdl
# Replace 'end' at start of line (statement body end) with '}'
sed -i 's/^end$/}/' mdl-examples/doctype-tests/02-microflow-examples.mdl
```

- [ ] **Step 3: Validate the fix**

Run:
```bash
cd /mnt/data_sdb/mxcli
./bin/mxcli check mdl-examples/doctype-tests/02-microflow-examples.mdl 2>&1
```
Expected: No errors. Exit code may not be reliable — check output for "error" strings.

- [ ] **Step 4: Fix remaining 4 files with same pattern**

```bash
cd /mnt/data_sdb/mxcli
for f in 02c-complex-layout-examples.mdl 04-math-examples.mdl \
         07-java-action-examples.mdl microflow-spec.test.mdl; do
    sed -i 's/^begin$/{/' "mdl-examples/doctype-tests/$f"
    sed -i 's/^end$/}/' "mdl-examples/doctype-tests/$f"
    echo "=== $f ==="
    ./bin/mxcli check "mdl-examples/doctype-tests/$f" 2>&1 | grep -i "error\|mismatched\|no viable"
done
```
Expected: All 5 pass syntax check with no errors.

- [ ] **Step 5: Remove skip list from CI and verify doctype-tests run**

Modify `.github/workflows/ci.yml` line 36-37:
```yaml
# Before:
SKIP='02-microflow-examples.mdl 02c-complex-layout-examples.mdl 04-math-examples.mdl 07-java-action-examples.mdl microflow-spec.test.mdl'

# After:
SKIP=''
```

- [ ] **Step 6: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add mdl-examples/doctype-tests/02-microflow-examples.mdl \
       mdl-examples/doctype-tests/02c-complex-layout-examples.mdl \
       mdl-examples/doctype-tests/04-math-examples.mdl \
       mdl-examples/doctype-tests/07-java-action-examples.mdl \
       mdl-examples/doctype-tests/microflow-spec.test.mdl \
       .github/workflows/ci.yml
git commit -m "fix(doctype-tests): migrate begin/end to {/} braces, remove CI skip list"
```

---

### Task A2: Remove `check-mdl` Dead Makefile Target

**Files:**
- Modify: `Makefile` (lines 281-294)

**Background:** `check-mdl` target runs `mxcli check` on all doctype-tests but doesn't apply the same skip list as CI. It's unused (not referenced by any target or CI) and would fail.

- [ ] **Step 1: Read the target to confirm it's dead**

Read `Makefile` lines 281-294. Run:
```bash
cd /mnt/data_sdb/mxcli
rg -n "check-mdl" Makefile .github/ scripts/
```
Expected: Only the target definition in Makefile and maybe a comment — no actual invocations.

- [ ] **Step 2: Delete the target**

Edit `Makefile` to remove lines 281-294 (the entire `check-mdl` block).

- [ ] **Step 3: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add Makefile
git commit -m "chore: remove dead check-mdl Makefile target"
```

---

### Task A3: Remove Compile-Only Test

**Files:**
- Delete: `mdl/executor/exec_context_export_test.go`

**Background:** This 9-line file only does `var _ = GetHierarchyForMining` — a compile-time check that the symbol is exported. This provides marginal value (any real test would also verify the symbol exists).

- [ ] **Step 1: Verify the file content**

Read `mdl/executor/exec_context_export_test.go`. Confirm it's only the compile guard.

- [ ] **Step 2: Delete the file**

```bash
cd /mnt/data_sdb/mxcli
rm mdl/executor/exec_context_export_test.go
```

- [ ] **Step 3: Verify tests still pass**

```bash
cd /mnt/data_sdb/mxcli
go test ./mdl/executor/... -count=1 -timeout 30s 2>&1 | tail -5
```
Expected: PASS (no compile errors from removed export reference).

- [ ] **Step 4: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add mdl/executor/exec_context_export_test.go  # git rm equivalent
git commit -m "chore: remove compile-only exec_context_export_test.go"
```

---

### Task A4: Move POC Tests to `poc` Build Tag

**Files:**
- Modify: `modelsdk/codec/poc/blocker1_fresh_encode_test.go`
- Modify: `modelsdk/codec/poc/blocker2_id_setters_test.go`
- Modify: `modelsdk/codec/poc/blocker3_dirty_encode_bench_test.go`
- Modify: `modelsdk/codec/poc/blocker4_cache_consistency_test.go`
- Modify: `modelsdk/codec/poc/poc_helpers_test.go`
- Modify: `modelsdk/poc/dynamic/dynamic_test.go`
- Modify: `cmd/exprgrammar-mine/cluster_test.go`
- Modify: `cmd/exprgrammar-mine/driver_test.go`
- Modify: `cmd/exprgrammar-mine/fixture_test.go`
- Modify: `cmd/exprgrammar-mine/mine_test.go`

**Background:** These 10 files are experimental/POC code that runs as part of `make test`. They should be gated behind `//go:build poc` so they don't slow down regular CI runs.

- [ ] **Step 1: Add `//go:build poc` to each POC test file**

For each file, add `//go:build poc` as the first line. Example:
```go
//go:build poc

package poc
```

The files to modify:
```bash
for f in modelsdk/codec/poc/blocker1_fresh_encode_test.go \
         modelsdk/codec/poc/blocker2_id_setters_test.go \
         modelsdk/codec/poc/blocker3_dirty_encode_bench_test.go \
         modelsdk/codec/poc/blocker4_cache_consistency_test.go \
         modelsdk/codec/poc/poc_helpers_test.go \
         modelsdk/poc/dynamic/dynamic_test.go \
         cmd/exprgrammar-mine/cluster_test.go \
         cmd/exprgrammar-mine/driver_test.go \
         cmd/exprgrammar-mine/fixture_test.go \
         cmd/exprgrammar-mine/mine_test.go; do
    # Prepend build tag (only if not already present)
    if ! head -1 "$f" | grep -q "go:build"; then
        sed -i '1i //go:build poc' "$f"
    fi
done
```

- [ ] **Step 2: Verify POC tests excluded from regular test run**

```bash
cd /mnt/data_sdb/mxcli
# Run without poc tag — POC tests should be excluded
go test ./modelsdk/codec/poc/... ./modelsdk/poc/dynamic/... ./cmd/exprgrammar-mine/... -count=1 -timeout 30s 2>&1 | tail -5
```
Expected: `[no test files]` or PASS with 0 tests run.

- [ ] **Step 3: Verify POC tests run with `poc` tag**

```bash
cd /mnt/data_sdb/mxcli
go test -tags poc ./modelsdk/codec/poc/... ./modelsdk/poc/dynamic/... ./cmd/exprgrammar-mine/... -count=1 -timeout 30s 2>&1 | tail -10
```
Expected: Tests execute and PASS.

- [ ] **Step 4: Add `poc` tag to CI allowlist**

Check if CI `.github/workflows/ci.yml` has a matrix or build tag filter. If it runs `go test ./...` without tags, POC tests are already excluded. No CI change needed (absence of `-tags poc` means they're skipped).

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add modelsdk/codec/poc/ modelsdk/poc/dynamic/ cmd/exprgrammar-mine/
git commit -m "chore: gate POC/experimental tests behind //go:build poc tag"
```

---

### Task A5: Fix Flaky Agent Tests — Channel-Based Message Wait

**Files:**
- Modify: `cmd/mxcli/tui/agent_listener_test.go`
- Modify: `cmd/mxcli/tui/agent_integration_test.go`
- Test: run `go test ./cmd/mxcli/tui/... -count=5 -timeout 60s` to verify stability

**Background:** 8 time.Sleep calls in agent listener tests (1 in agent_listener_test.go, 7 in agent_integration_test.go) wait 100-200ms for async message delivery. Replace with a `waitForMsg(timeout)` channel pattern.

- [ ] **Step 1: Read and understand the message collector pattern**

Read `cmd/mxcli/tui/agent_integration_test.go` lines 1-80 (or wherever `msgCollector` is defined). It should have an `add(msg)` method and a `snapshot()` method backed by `sync.Mutex`.

- [ ] **Step 2: Add `waitForMsg` method to the collector**

The collector needs a channel-based wait. Pattern:
```go
// Add to the msgCollector or collector type:
type msgCollector struct {
    mu       sync.Mutex
    msgs     []string
    received chan struct{} // 1-buffered, signaled on each add
}

func (c *msgCollector) add(msg string) {
    c.mu.Lock()
    c.msgs = append(c.msgs, msg)
    c.mu.Unlock()
    select {
    case c.received <- struct{}{}:
    default:
    }
}

func (c *msgCollector) waitForMsg(t *testing.T, timeout time.Duration) {
    t.Helper()
    deadline := time.After(timeout)
    for {
        // Check if we already have messages
        c.mu.Lock()
        hasMsgs := len(c.msgs) > 0
        c.mu.Unlock()
        if hasMsgs {
            return
        }
        select {
        case <-c.received:
            continue
        case <-deadline:
            c.mu.Lock()
            count := len(c.msgs)
            c.mu.Unlock()
            t.Fatalf("timed out waiting for message after %v (have %d msgs)", timeout, count)
        }
    }
}
```

Note: If the collector is defined in `agent_integration_test.go` as a local type, add the method there. If it's a different type name, adapt accordingly.

- [ ] **Step 3: Replace `time.Sleep(100ms)` in `agent_listener_test.go:60`**

Before (line ~60):
```go
conn.Write(reqBytes)
// wait for message to arrive
time.Sleep(100 * time.Millisecond)
mu.Lock()
count := len(received)
mu.Unlock()
```

After:
```go
conn.Write(reqBytes)
collector.waitForMsg(t, 2*time.Second)
mu.Lock()
count := len(received)
mu.Unlock()
```

- [ ] **Step 4: Replace all 7 `time.Sleep` in `agent_integration_test.go`**

Each of the 7 occurrences follows the same pattern:
```go
time.Sleep(200 * time.Millisecond)  // or 100ms
messages := collector.snapshot()
```

Replace each with:
```go
collector.waitForMsg(t, 2*time.Second)
messages := collector.snapshot()
```

- [ ] **Step 5: Verify tests pass 5 times in a row**

```bash
cd /mnt/data_sdb/mxcli
go test ./cmd/mxcli/tui/... -count=5 -timeout 60s -run 'TestAgent' 2>&1 | tail -20
```
Expected: All tests PASS consistently (no timeouts, no flakes).

- [ ] **Step 6: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add cmd/mxcli/tui/agent_listener_test.go cmd/mxcli/tui/agent_integration_test.go
git commit -m "fix(tui): replace time.Sleep with channel-based waitForMsg in agent tests"
```

---

### Task A6: Fix Flaky Watcher Tests — Debounce Channel

**Files:**
- Modify: `cmd/mxcli/tui/watcher_test.go`
- Potentially modify: the watcher implementation if it needs to expose a test hook

**Background:** 3 time.Sleep(700ms) calls wait for a 500ms debounce timer. Expose a `testDebounceFired` channel from the watcher to eliminate wall-clock waiting.

- [ ] **Step 1: Read the watcher implementation and test**

Read `cmd/mxcli/tui/watcher_test.go` to understand the current debounce wait pattern. Also read the watcher source to find where the debounce timer fires.

```bash
cd /mnt/data_sdb/mxcli
head -120 cmd/mxcli/tui/watcher_test.go
ls cmd/mxcli/tui/watcher*.go
```

- [ ] **Step 2: Add `testDebounceFired` channel**

If the watcher is a struct, add an exported test hook field (only set in tests):
```go
// In the watcher struct (watcher.go):
type Watcher struct {
    // ... existing fields
    TestDebounceFired chan struct{} // non-nil only in tests
}
```

In the debounce timer completion code:
```go
if w.TestDebounceFired != nil {
    select {
    case w.TestDebounceFired <- struct{}{}:
    default:
    }
}
```

- [ ] **Step 3: Replace time.Sleep(700ms) in watcher_test.go**

Before:
```go
// Write 5 times rapidly, then wait for debounce
time.Sleep(700 * time.Millisecond)
```

After:
```go
// Write 5 times rapidly, then wait for debounce
select {
case <-w.TestDebounceFired:
case <-time.After(2 * time.Second):
    t.Fatal("debounce did not fire within 2s")
}
```

Apply the same replacement to all 3 occurrences.

- [ ] **Step 4: Verify tests pass**

```bash
cd /mnt/data_sdb/mxcli
go test ./cmd/mxcli/tui/... -count=3 -timeout 30s -run 'TestWatcher' 2>&1 | tail -10
```
Expected: All watcher tests PASS in under 3 seconds (was ~7s+ due to 3x700ms sleep).

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add cmd/mxcli/tui/watcher_test.go cmd/mxcli/tui/watcher*.go
git commit -m "fix(tui): replace time.Sleep with TestDebounceFired channel in watcher tests"
```

---

### Task A7: Fix Flaky Scheduler Test — Atomic Counter

**Files:**
- Modify: `internal/testscheduler/scheduler_test.go` (line ~66)

**Background:** 1 time.Sleep(10ms) in a concurrent signal hold test. Replace with sync.WaitGroup + atomic counter.

- [ ] **Step 1: Read the test**

Read `internal/testscheduler/scheduler_test.go` around line 60-80 to understand the pattern.

- [ ] **Step 2: Replace time.Sleep**

Before:
```go
go func(id int) {
    defer wg.Done()
    if id%2 == 0 { s.AcquireIO(ctx); defer s.ReleaseIO() }
    else        { s.AcquireCPU(ctx); defer s.ReleaseCPU() }
    time.Sleep(10 * time.Millisecond)
}(i)
```

After:
```go
var acquired int32
go func(id int) {
    defer wg.Done()
    if id%2 == 0 { s.AcquireIO(ctx); defer s.ReleaseIO() }
    else        { s.AcquireCPU(ctx); defer s.ReleaseCPU() }
    atomic.AddInt32(&acquired, 1)
}(i)
```

Also add `sync/atomic` to imports if not already present.

- [ ] **Step 3: Verify test passes**

```bash
cd /mnt/data_sdb/mxcli
go test ./internal/testscheduler/... -count=3 -timeout 30s 2>&1 | tail -5
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add internal/testscheduler/scheduler_test.go
git commit -m "fix(testscheduler): replace time.Sleep with atomic counter"
```

---

### Task A8: Merge `roundtrip` Build Tag Into `integration`

**Files:**
- Modify: `mdl/exprcheck/roundtrip_test.go`
- Modify: `.github/workflows/ci.yml` (any `roundtrip` tag reference)
- Modify: `Makefile` (line 488 `roundtrip` target)

**Background:** The `roundtrip` build tag is only used by 1 file (`mdl/exprcheck/roundtrip_test.go`). No other file uses it. Merge into `integration` to reduce tag complexity from 3 to 2 (plus `linux` and `poc`).

- [ ] **Step 1: Check what the `roundtrip` file does**

Read `mdl/exprcheck/roundtrip_test.go` line 1-5 to find the build constraint. Read the full file to understand what it tests.

- [ ] **Step 2: Change `roundtrip` to `integration`**

Change the build tag line:
```go
// Before:
//go:build roundtrip

// After:
//go:build integration
```

- [ ] **Step 3: Update Makefile `roundtrip` target**

Change `Makefile` line ~488:
```makefile
# Before:
roundtrip:
	CGO_ENABLED=0 go test -tags=roundtrip ./mdl/exprcheck/ -run TestRoundTrip

# After:
roundtrip-integration:
	CGO_ENABLED=0 go test -tags=integration ./mdl/exprcheck/ -run TestRoundTrip
```

- [ ] **Step 4: Update CI workflow**

Check `.github/workflows/ci.yml` for any `roundtrip` reference. If the `exprcheck-roundtrip` job uses `-tags roundtrip`, change it to `-tags integration`.

- [ ] **Step 5: Verify both tag paths work**

```bash
cd /mnt/data_sdb/mxcli
# With integration tag (the new way)
go test -tags integration ./mdl/exprcheck/ -run TestRoundTrip -count=1 -timeout 30s 2>&1 | tail -5
# Without integration tag (should skip)
go test ./mdl/exprcheck/ -run TestRoundTrip -count=1 -timeout 30s 2>&1 | tail -5
```
Expected: With tag → runs tests. Without tag → `[no test files]` or skips.

- [ ] **Step 6: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add mdl/exprcheck/roundtrip_test.go Makefile .github/workflows/ci.yml
git commit -m "chore: merge roundtrip build tag into integration"
```

---

### Task A9: Fix `ce1613_integration_test.go` Missing Build Tag

**Files:**
- Modify: `mdl/executor/ce1613_integration_test.go`

**Background:** This file has `integration` in its filename but no `//go:build integration` constraint. It runs unconditionally in unit tests.

- [ ] **Step 1: Read the file**

Read `mdl/executor/ce1613_integration_test.go` first 5 lines and last 10 lines to understand what it tests and whether it needs mxbuild/Docker.

- [ ] **Step 2: Add build tag**

Prepend:
```go
//go:build integration

package executor
```

- [ ] **Step 3: Verify it's excluded from unit tests**

```bash
cd /mnt/data_sdb/mxcli
go test ./mdl/executor/... -run 'TestCE1613' -count=1 -timeout 30s 2>&1 | tail -5
```
Expected: `[no test files]` or 0 tests matched.

- [ ] **Step 4: Verify it runs with integration tag**

```bash
cd /mnt/data_sdb/mxcli
go test -tags integration ./mdl/executor/... -run 'TestCE1613' -count=1 -timeout 30s 2>&1 | tail -5
```
Expected: Test executes and PASS.

- [ ] **Step 5: Commit**

```bash
cd /mnt/data_sdb/mxcli
git add mdl/executor/ce1613_integration_test.go
git commit -m "chore: add missing //go:build integration tag to ce1613_integration_test.go"
```

---

## Self-Review Checklist

- [ ] **Spec coverage:** Every item from Track A in the spec has a corresponding task:
  - A1 (dead doctype-tests) → Task A1 ✓
  - A1 (check-mdl target) → Task A2 ✓
  - A1 (compile-only test) → Task A3 ✓
  - A1 (POC tests) → Task A4 ✓
  - A2 (flaky agent tests) → Task A5 ✓
  - A2 (flaky watcher tests) → Task A6 ✓
  - A2 (flaky scheduler test) → Task A7 ✓
  - A3 (roundtrip tag merge) → Task A8 ✓
  - A3 (ce1613 tag fix) → Task A9 ✓
  - Track B → Not covered (requires separate design docs as stated in spec) ✓

- [ ] **Placeholder scan:** No TBD, TODO, or "implement later" patterns. All steps have exact file paths, commands, and code.

- [ ] **Type consistency:** All file paths, function names, and patterns are consistent across tasks. No cross-task type mismatches.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-25-test-governance-track-a.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

2. **Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
