# Test Resource Orchestrator — Design Spec

**Date**: 2026-06-24
**Status**: Draft

## Problem

Integration tests have three resource-related problems:
1. No visibility into per-test resource usage (memory, CPU, file I/O)
2. No adaptive scheduling — parallelism is static (85% nproc) regardless of test mix
3. mxgraph is built from scratch per test, repeating ~14s of work

## Solution: Full Orchestrator

Three independent packages, each with a single responsibility:

```
internal/
  testresource/         Resource metrics capture
  testscheduler/        Adaptive test scheduling + execution
  testresourceregistry/ Profile registry (discovery + storage)
```

Each package follows SOLID as defined in `docs/03-development/GO_SOLID_PRINCIPLES.md`.

---

## Package 1: `internal/testresource/`

Single job: capture resource usage for one test execution.

### Types

```go
type Profile struct {
    Name          string  `json:"name"`
    DurationMs    float64 `json:"duration_ms"`
    HeapDelta     int64   `json:"heap_delta_bytes"`
    CPUTimeMs     float64 `json:"cpu_time_ms"`
    ReadBytes     int64   `json:"read_bytes"`
    WriteBytes    int64   `json:"write_bytes"`
    NumGoroutines int     `json:"num_goroutines"`
}

type Category int
const (
    CategoryUncategorized Category = iota
    CategoryIOHeavy
    CategoryCPUHeavy
    CategoryMixed
)

func Classify(p Profile) Category
```

### Monitor

```go
type Monitor struct {
    start resourceSnapshot
}

func NewMonitor(t *testing.T, opts ...MonitorOption) *Monitor
func (m *Monitor) Done() Profile

type MonitorOption func(*Monitor)
func WithResourceReader(rr ResourceReader) MonitorOption

type ResourceReader interface {
    Read() (ResourceSnapshot, error)
}
```

### Store

```go
type Store struct {
    dir string
}

func NewStore(dir string) *Store
func (s *Store) Save(p Profile) error
func (s *Store) Load(name string) (Profile, bool)
func (s *Store) List() ([]Profile, error)
func (s *Store) Compare(baseline, current Profile) (Diff, error)

type Diff struct {
    Name          string
    HeapDeltaPct  float64
    CPUTimePct    float64
    ReadBytesPct  float64
    WriteBytesPct float64
}
```

### ISP Interfaces

```go
type ProfileSaver interface { Save(Profile) error }
type ProfileLoader interface { Load(name string) (Profile, bool) }
type ResourceReader interface { Read() (ResourceSnapshot, error) }
```

---

## Package 2: `internal/testscheduler/`

### Types

```go
type ResourceLimit struct {
    MaxParallelIO  int
    MaxParallelCPU int
    MaxHeapMB      int
}

type Plan struct {
    IOHeavy  []PlanSlot
    CPUHeavy []PlanSlot
    Mixed    []PlanSlot
}

type PlanSlot struct {
    Name     string
    Duration time.Duration
    Profile  Profile
}

type Schedule struct {
    Lanes  []Lane
    Limits ResourceLimit
}

type Lane struct {
    Name  string
    Tests []PlanSlot
}
```

### Planner + Strategy

```go
type Planner struct {
    store  ProfileLoader
    limits ResourceLimit
}

func NewPlanner(store ProfileLoader, limits ResourceLimit) *Planner
func (p *Planner) Plan(profiles []Profile) Schedule

type PlanningStrategy interface {
    Plan(profiles []Profile, limits ResourceLimit) Schedule
}

type DefaultStrategy struct{}
func (DefaultStrategy) Plan(profiles []Profile, limits ResourceLimit) Schedule
```

### Scheduler

```go
type Scheduler struct {
    ioToken  chan struct{}
    cpuToken chan struct{}
    reader   ResourceReader
}

func New(maxIO, maxCPU int, reader ResourceReader) *Scheduler
func (s *Scheduler) AcquireIO(ctx context.Context) error
func (s *Scheduler) AcquireCPU(ctx context.Context) error
func (s *Scheduler) ReleaseIO()
func (s *Scheduler) ReleaseCPU()
func (s *Scheduler) Adjust(limits ResourceLimit)
```

---

## Package 3: `internal/testresourceregistry/`

```go
type Registry struct {
    store     *testresource.Store
    planner   *testscheduler.Planner
    scheduler *testscheduler.Scheduler
}

func New(profileDir string, limits testscheduler.ResourceLimit) *Registry
func (r *Registry) Record(p testresource.Profile) error
func (r *Registry) BuildSchedule() (testscheduler.Schedule, error)
func (r *Registry) CheckRegressions() ([]testresource.Diff, error)
```

---

## mxgraph Sharing

Each `setupTestEnv()` creates a fresh `MprBackend`, triggering `buildProjectGraph()` (~14s). Fix: build once in TestMain, inject via `SetProjectGraph()`.

```go
var sharedProjectGraph *graphcatalog.ProjectGraph

func buildSharedGraph(mprPath string) *graphcatalog.ProjectGraph {
    m, err := modelsdk.Open(mprPath)
    if err != nil { return nil }
    defer m.Close()
    mgr := mxgraph.NewIndexManager()
    // register adapters (backend_lifecycle.go:119-147)
    if err := mgr.BuildAll(context.Background(), mgr); err != nil { return nil }
    return graphcatalog.NewProjectGraph(mgr)
}

func setupTestEnv(t *testing.T) *testEnv {
    env := setupTestEnvNoGraph(t)
    if sharedProjectGraph != nil {
        if mprB, ok := env.executor.Backend().(*mprbackend.MprBackend); ok {
            mprB.SetProjectGraph(sharedProjectGraph)
        }
    }
    return env
}
```

---

## Makefile Targets

```makefile
test-integration-profiled: build install-daemon
    CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m \
        -resource-profile -resource-schedule ./...

test-profile-check:
    go test -tags integration -count=1 -run '^TestProfileCheck$$' \
        -resource-check ./...

test-profile-record:
    @mkdir -p coverage/test-profiles
    CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m \
        -resource-record ./...
```

## CI Changes

```yaml
- name: Integration tests with profiling
  run: make test-integration-profiled
  timeout-minutes: 35
- name: Upload resource profiles
  uses: actions/upload-artifact@v4
  if: always()
  with:
    name: test-profiles-${{ github.sha }}
    path: coverage/test-profiles/
    retention-days: 30
- name: Check resource regressions
  run: make test-profile-check
  continue-on-error: true
```

## SOLID Compliance

| Principle | Applied |
|-----------|---------|
| SRP | One job per package |
| OCP | PlanningStrategy interface |
| LSP | ProcfsReader / MockReader interchangeable via ResourceReader |
| ISP | 4 interfaces, each ≤2 methods |
| DIP | TestMain is composition root |
| Anti-God-Struct | Max 3 fields per struct |

## Files Changed

| File | Change |
|------|--------|
| `internal/testresource/monitor.go` | New |
| `internal/testresource/profile.go` | New |
| `internal/testresource/store.go` | New |
| `internal/testresource/category.go` | New |
| `internal/testresource/monitor_test.go` | New |
| `internal/testscheduler/planner.go` | New |
| `internal/testscheduler/scheduler.go` | New |
| `internal/testscheduler/plan.go` | New |
| `internal/testscheduler/scheduler_test.go` | New |
| `internal/testresourceregistry/registry.go` | New |
| `mdl/executor/roundtrip_helpers_test.go` | Modify — shared mxgraph |
| `Makefile` | Modify — 3 new targets |
| `.github/workflows/ci.yml` | Modify — profiling steps |
| `docs/03-development/TESTING_GUIDE.md` | Modify — L7 profiling layer |

## Verifications

1. `make test` — no regressions
2. `make test-integration` — existing tests pass with shared mxgraph
3. `make test-integration-profiled` — profiles in `coverage/test-profiles/`
4. `make test-profile-record && make test-profile-check` — regression detection works
5. `go vet ./internal/testresource/... ./internal/testscheduler/... ./internal/testresourceregistry/...` — clean
