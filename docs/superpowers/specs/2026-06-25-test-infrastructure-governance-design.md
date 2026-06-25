# Test Infrastructure Governance Design

**Date:** 2026-06-25
**Status:** Approved design, awaiting implementation planning

## Summary

Current state: 584 test files, 7 nominal layers (L1-L7, but L4 is empty), 3 mock families
(~59 files), 14 time.Sleep flakes, 3 build tags, 42.1% coverage.

Root cause: Test infrastructure mirrors the same SOLID violations as production code
(MockBackend: 240 Func fields, ExecContext: 80 fields, HandlerDeps: 106 fields).
Deep fixes require production+test co-evolution.

Approach: **Dual-track governance** — Track A (independent cleanup, immediately
actionable) + Track B (architecture governance, requires separate production-code SOLID
design documents).

## SOLID Diagnosis

| Principle | Production violation | Test mirror | Severity |
|-----------|---------------------|-------------|----------|
| ISP (Interface Segregation) | `FullBackend` = 30 domain interfaces, ~240 methods | `MockBackend` = 240 Func fields | ★★★ |
| SRP (Single Responsibility) | `HandlerDeps` = 106 fields | `ExecContext` = 80 fields | ★★★ |
| OCP (Open/Closed) | New method touches N files | 63 test files forced to recompile | ★★ |
| DIP (Dependency Inversion) | High-level code depends on concrete ExecContext | Tests depend on concrete MockBackend | ★★ |
| Principle 6 (Avoid God Struct) | Executor struct > 15 fields | newMockCtx creates 80-field object | ★★★ |
| Principle 3 (Domain Packages) | Subpackages exist, but bridge back to ExecContext | **Zero test files in subpackages** | ★★★ |

## Track A: Independent Governance (Immediately Actionable)

### A1: Dead Test Cleanup

**Evidence:**
- 5 doctype-tests permanently skipped in CI (`.github/workflows/ci.yml:36`) with syntax
  errors, unfixed since 2026-06-18
- `check-mdl` Makefile target (`Makefile:281-294`) — unused, would fail on the same
  4 skip-list files
- `exec_context_export_test.go` — 9 lines, compile-only, no assertions
- 10 POC/experimental test files under `modelsdk/codec/poc/` and `cmd/exprgrammar-mine/`

**Actions:**
1. Fix the 5 skipped doctype-tests or remove the skip list (prefer fix)
2. Delete `check-mdl` target or align with CI skip list
3. Delete `exec_context_export_test.go`
4. Move POC tests out of `-tags=unit` scope (e.g., `//go:build poc`)

### A2: Flaky Test Remediation

**Evidence:** 14 time.Sleep usages across 6 files. 12/14 are replaceable.

**8 agent/listener tests** (`cmd/mxcli/tui/agent_integration_test.go` + `agent_listener_test.go`):
- Current: `time.Sleep(100-200ms)` waiting for async message delivery
- Target: Channel-based `waitForMsg(timeout)` on `msgCollector`
- Risk: Medium — CI scheduling delays cause intermittent failures

**3 watcher debounce tests** (`cmd/mxcli/tui/watcher_test.go`):
- Current: `time.Sleep(700ms)` per case (total 2.1s)
- Target: Expose `testDebounceFired` channel from watcher
- Note: Low flake risk due to 200ms margin, but wastes wall time

**1 scheduler test** (`internal/testscheduler/scheduler_test.go`):
- Current: `time.Sleep(10ms)` for concurrent signal hold
- Target: `sync.WaitGroup` + atomic counter

**2 inherently sequential tests** (retain sleep, add margin):
- `internal/expr/daemon/daemon_test.go:31` — socket polling (20ms interval, 6s timeout)
- `cmd/mxcli/daemon_server_test.go:106` — idle timeout timing

**Rule applied:** `time.Sleep` is allowed only for (a) polling a known-asynchronous
resource with bounded deadline + short interval, or (b) testing time-dependent behavior
where events cannot be injected.

### A3: Build Tag & Naming Simplification

**Current state:**
- `integration`: 31 files in executor + internal/expr
- `roundtrip`: 1 unique file (`mdl/exprcheck/roundtrip_test.go`)
- `linux`: 4 goldenfs unit tests
- `linux && integration`: 5 goldenfs integration tests
- `!integration`: 1 file
- `ce1613_integration_test.go`: **no build tag** despite `integration` in name

**Actions:**
1. Remove `roundtrip` tag — merge into `integration` tag
2. Add `//go:build integration` to `ce1613_integration_test.go`
3. Verify `!integration` file still has semantic value (remove if not)
4. Document: only 3 tags remain — `integration`, `linux`, `poc`

**Test naming convention (no change from intended state):**
| Layer | Build tag | File pattern |
|-------|-----------|-------------|
| Unit | (none) | `*_test.go` |
| Mock-isolated | (none) | `*_test.go` (no special suffix) |
| Integration | `//go:build integration` | `*_integration_test.go` |

## Track B: Architecture Governance (Requires Production Code Design)

### B1: MockBackend Split (ISP)

**Requires:** `FullBackend` interface to be split into narrow interfaces.

**Principle 5 (Narrow Interfaces + Multiple Implementations):**
- Production: `MprBackend` implements 30+ narrow interfaces
- Test: Each narrow interface gets its own mock struct

**Example:**
```go
// Current: 240 Func fields in one struct
type MockBackend struct {
    ListModulesFunc  func()...
    GetModuleFunc    func()...
    // ... 238 more
}

// Target: One small mock per narrow interface
type MockModuleLister struct {
    ListModulesFunc func() ([]Module, error)
    CallRecorder[ModuleListCall]
}
var _ backend.ModuleLister = (*MockModuleLister)(nil)

// MockBackend composes them:
type MockBackend struct {
    *MockConnection
    *MockModuleLister
    *MockDomainModel
    *MockMicroflow
    *MockSecurity
    // ... compose only what tests need
}
```

**CallRecorder[T] generic type** (replaces `mdl/repos/testing/` entirely):
```go
type CallRecorder[T any] struct {
    mu    sync.Mutex
    calls []T
}

func (r *CallRecorder[T]) Record(call T) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.calls = append(r.calls, call)
}

func (r *CallRecorder[T]) Calls() []T {
    r.mu.Lock()
    defer r.mu.Unlock()
    result := make([]T, len(r.calls))
    copy(result, r.calls)
    return result
}
```

### B2: Subpackage Testing (SRP)

**Requires:** Subpackages to be truly decoupled from ExecContext.

**Current subpackage pattern** (`domainmodel/handler.go`):
- Creates narrow `DomainModelDeps` (22 fields)
- But bridges back to `executor.NewExecContext(bgCtx, deps)` for shared helpers
- **0 test files** in any subpackage

**Target:** Each subpackage owns its tests:
```
mdl/executor/domainmodel/
  handler.go          → RegisterHandlers
  deps.go             → DomainModelDeps struct
  impl.go             → ExecCreateEntityFn, ExecAlterEntityFn, etc.
  impl_test.go        → Unit tests with narrow domain mocks
  integration_test.go → Integration tests (build tag)

mdl/executor/page/
  handler.go
  deps.go
  impl.go
  impl_test.go
  integration_test.go

mdl/executor/
  ... (remaining handlers not yet extracted)
```

### B3: ExecContext Elimination (Principle 6)

**Requires:** All handler functions to accept narrow deps instead of `*ExecContext`.

**Current:**
```go
func ExecCreateEntity(ectx *ExecContext, stmt *ast.CreateEntityStmt) error {
    // 80-field object, test needs to setup all of it
}
```

**Target:**
```go
func ExecCreateEntityFn(ctx context.Context, stmt *ast.CreateEntityStmt,
    d DomainModelDeps) error {
    // 22-field narrow deps, test sets up only what's needed
}
```

### Migration Sequence

```
Phase A (independent):    P1 → P2 → P3
                          (clear dead weight, stabilize flakes, simplify tags)

Phase B1 (ISP refactor):  FullBackend split → MockBackend split → narrow mock tests
                          (design doc required: "backend interface segregation")

Phase B2 (SRP refactor):  Subpackage extraction → subpackage tests → mock_test_helpers.go removal
                          (design doc required: "executor subpackage testing")

Phase B3 (Principle 6):   ExecContext → narrow deps → newMockCtx elimination
                          (design doc required: "executor context system cleanup")
```

## Non-Goals

- Not rewriting all 1,688 test functions in this effort
- Not changing the production backend implementation logic
- Not adding new test coverage (focus is on governance)
- Not changing the CI workflow structure (only build tag alignment)

## Success Criteria

### Track A
- [ ] 0 CI-skipped doctype-tests (all pass or explicitly fixed)
- [ ] 0 `time.Sleep` in test files (except 2 documented inherently sequential cases)
- [ ] 3 build tags max (`integration`, `linux`, `poc`)
- [ ] `check-mdl` target removed or aligned
- [ ] `exec_context_export_test.go` removed
- [ ] All `_integration_test.go` files have correct build tags

### Track B
- [ ] Each B-phase has its own design doc approved
- [ ] MockBackend Func fields reduced by 80%+ (narrow mocks per interface)
- [ ] `mdl/repos/testing/` directory removed
- [ ] Each domain subpackage has `*_test.go` files
- [ ] `newMockCtx` usage trend: decreasing (target: eliminate)
