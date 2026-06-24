# Test Resource Orchestrator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-test resource profiling, profile-based scheduling, and mxgraph acceleration to integration tests.

**Architecture:** Three new packages in `internal/` — `testresource/` (metrics capture), `testscheduler/` (lane-based scheduling), `testresourceregistry/` (composition root). Modifications to `roundtrip_helpers_test.go` for mxgraph sharing, `Makefile` for new targets, CI workflow, and TESTING_GUIDE.

**Tech Stack:** Go 1.26+, stdlib `runtime`, `/proc/self/stat` + `/proc/self/io` for resource reading, `internal/mxgraph` for graph acceleration.

## Global Constraints

- All new interfaces must be defined at the consumer side (ISP per `GO_SOLID_PRINCIPLES.md` Principle 1)
- Option pattern for injectable dependencies (Principle 2)
- Narrow interfaces only (≤2 methods) — no god interfaces (Principle 5)
- No struct exceeds 5 exported fields — avoid god structs (Principle 6)
- Profile JSON stored in `coverage/test-profiles/`
- Resource reading must fall back gracefully (zero values) when `/proc/self/stat` or `/proc/self/io` is unavailable (non-Linux)
- mxgraph sharing is best-effort (non-fatal when graph build fails)

---
### Task 1: `internal/testresource/` — Core Package

**Files:**
- Create: `internal/testresource/profile.go` — Profile types + serialization
- Create: `internal/testresource/monitor.go` — Monitor + ResourceReader
- Create: `internal/testresource/store.go` — ProfileStore + Diff
- Create: `internal/testresource/category.go` — Category classifier
- Create: `internal/testresource/monitor_test.go` — unit tests
- Create: `internal/testresource/store_test.go` — unit tests

**Interfaces:**
- Consumes: nothing from this project (stdlib only)
- Produces: `testresource.Profile`, `testresource.Monitor`, `testresource.Store`, `testresource.Category`, `testresource.ResourceReader` interface, `testresource.ProfileSaver` interface, `testresource.ProfileLoader` interface

- [ ] **Step 1: Define Profile types + resourceSnapshot**

```go
// internal/testresource/profile.go
package testresource

import "encoding/json"

type Profile struct {
    Name          string  `json:"name"`
    DurationMs    float64 `json:"duration_ms"`
    HeapDelta     int64   `json:"heap_delta_bytes"`
    CPUTimeMs     float64 `json:"cpu_time_ms"`
    ReadBytes     int64   `json:"read_bytes"`
    WriteBytes    int64   `json:"write_bytes"`
    NumGoroutines int     `json:"num_goroutines"`
}

type resourceSnapshot struct {
    heapAlloc     uint64
    cpuTicks      int64
    ioReadBytes   int64
    ioWriteBytes  int64
    numGoroutines int
}

type Category int

const (
    CategoryUncategorized Category = iota
    CategoryIOHeavy
    CategoryCPUHeavy
    CategoryMixed
)
```

- [ ] **Step 2: Write failing test for Monitor creation + Done()**

```go
// internal/testresource/monitor_test.go
package testresource

import (
    "testing"
)

type mockReader struct {
    cpuTicks    int64
    readBytes   int64
    writeBytes  int64
}

func (m *mockReader) Read() (ResourceSnapshot, error) {
    return ResourceSnapshot{
        CPUTicks:    m.cpuTicks,
        ReadBytes:   m.readBytes,
        WriteBytes:  m.writeBytes,
    }, nil
}

func TestMonitor_Done_ReturnsProfileWithName(t *testing.T) {
    m := NewMonitor(t, WithResourceReader(&mockReader{}))
    p := m.Done()
    if p.Name != t.Name() {
        t.Errorf("Name = %q, want %q", p.Name, t.Name())
    }
}

func TestMonitor_Done_CapturesHeapDelta(t *testing.T) {
    m := NewMonitor(t, WithResourceReader(&mockReader{}))
    _ = make([]byte, 1024*1024) // allocate 1MB to create a measurable delta
    p := m.Done()
    if p.HeapDelta <= 0 {
        t.Errorf("HeapDelta = %d, want > 0", p.HeapDelta)
    }
}

func TestMonitor_Done_CapturesCPUTime(t *testing.T) {
    reader := &mockReader{cpuTicks: 150, readBytes: 0, writeBytes: 0}
    m := NewMonitor(t, WithResourceReader(reader))
    reader.cpuTicks = 650 // simulate 500 ticks consumed
    p := m.Done()
    if p.CPUTimeMs <= 0 {
        t.Errorf("CPUTimeMs = %f, want > 0", p.CPUTimeMs)
    }
}
```

Run: `go test ./internal/testresource/ -v -run 'TestMonitor'`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Implement Monitor + ResourceReader**

```go
// internal/testresource/monitor.go
package testresource

import (
    "fmt"
    "os"
    "runtime"
    "strconv"
    "strings"
    "testing"
    "time"
)

type ResourceReader interface {
    Read() (ResourceSnapshot, error)
}

type ResourceSnapshot struct {
    CPUTicks   int64
    ReadBytes  int64
    WriteBytes int64
}

type MonitorOption func(*Monitor)

func WithResourceReader(rr ResourceReader) MonitorOption {
    return func(m *Monitor) {
        m.reader = rr
    }
}

type Monitor struct {
    t       *testing.T
    start   resourceSnapshot
    reader  ResourceReader
    startAt time.Time
}

func NewMonitor(t *testing.T, opts ...MonitorOption) *Monitor {
    m := &Monitor{
        t:       t,
        reader:  &procfsReader{},
        startAt: time.Now(),
    }
    for _, opt := range opts {
        opt(m)
    }
    m.captureStart()
    return m
}

func (m *Monitor) captureStart() {
    runtime.ReadMemStats(&m.start.heapAlloc, m.start.memStats)
    // Use the local variable directly instead
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)
    m.start.heapAlloc = ms.HeapAlloc
    if snap, err := m.reader.Read(); err == nil {
        m.start.cpuTicks = snap.CPUTicks
        m.start.ioReadBytes = snap.ReadBytes
        m.start.ioWriteBytes = snap.WriteBytes
    }
    m.start.numGoroutines = runtime.NumGoroutine()
}

func (m *Monitor) Done() Profile {
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)

    p := Profile{
        Name:          m.t.Name(),
        DurationMs:    float64(time.Since(m.startAt).Microseconds()) / 1000.0,
        HeapDelta:     int64(ms.HeapAlloc - m.start.heapAlloc),
        NumGoroutines: runtime.NumGoroutine() - m.start.numGoroutines,
    }

    if snap, err := m.reader.Read(); err == nil {
        const ticksPerSec = 100.0 // CLK_TCK on Linux
        cpuTicks := snap.CPUTicks - m.start.cpuTicks
        if cpuTicks < 0 {
            cpuTicks = 0
        }
        p.CPUTimeMs = float64(cpuTicks) * 1000.0 / ticksPerSec
        p.ReadBytes = snap.ReadBytes - m.start.ioReadBytes
        if p.ReadBytes < 0 {
            p.ReadBytes = 0
        }
        p.WriteBytes = snap.WriteBytes - m.start.ioWriteBytes
        if p.WriteBytes < 0 {
            p.WriteBytes = 0
        }
    }

    return p
}

type procfsReader struct{}

func (r *procfsReader) Read() (ResourceSnapshot, error) {
    var snap ResourceSnapshot

    // CPU ticks from /proc/self/stat
    if data, err := os.ReadFile("/proc/self/stat"); err == nil {
        fields := strings.Fields(string(data))
        if len(fields) > 15 {
            utime, _ := strconv.ParseInt(fields[13], 10, 64) // field 14 in /proc/self/stat
            stime, _ := strconv.ParseInt(fields[14], 10, 64) // field 15
            snap.CPUTicks = utime + stime
        }
    } else {
        return snap, fmt.Errorf("read /proc/self/stat: %w", err)
    }

    // I/O bytes from /proc/self/io
    if data, err := os.ReadFile("/proc/self/io"); err == nil {
        for _, line := range strings.Split(string(data), "\n") {
            if strings.HasPrefix(line, "read_bytes:") {
                fmt.Sscanf(line, "read_bytes: %d", &snap.ReadBytes)
            } else if strings.HasPrefix(line, "write_bytes:") {
                fmt.Sscanf(line, "write_bytes: %d", &snap.WriteBytes)
            }
        }
    }

    return snap, nil
}
```

**Note: `captureStart` has a compile error** — it was written to test for `runtime.ReadMemStats(&m.start.heapAlloc, m.start.memStats)` which doesn't exist. The actual code should just be:
```go
func (m *Monitor) captureStart() {
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)
    m.start.heapAlloc = ms.HeapAlloc
    ...
}
```

Run: `go test ./internal/testresource/ -v -run 'TestMonitor'`
Expected: PASS

- [ ] **Step 4: Write failing test for Store save/load/compare**

```go
// internal/testresource/store_test.go
package testresource

import (
    "os"
    "path/filepath"
    "testing"
)

func TestStore_SaveAndLoad(t *testing.T) {
    dir := t.TempDir()
    s := NewStore(dir)

    p := Profile{
        Name:       "TestSomething",
        HeapDelta:  42,
        CPUTimeMs:  10.5,
        ReadBytes:  1000,
        WriteBytes: 500,
    }

    if err := s.Save(p); err != nil {
        t.Fatalf("Save: %v", err)
    }

    loaded, ok := s.Load("TestSomething")
    if !ok {
        t.Fatal("Load returned false")
    }
    if loaded.HeapDelta != 42 {
        t.Errorf("HeapDelta = %d, want 42", loaded.HeapDelta)
    }
}

func TestStore_Compare_DetectsDelta(t *testing.T) {
    baseline := Profile{Name: "T", HeapDelta: 1000, ReadBytes: 50000}
    current := Profile{Name: "T", HeapDelta: 1500, ReadBytes: 60000}

    diff, err := (&Store{}).Compare(baseline, current)
    if err != nil {
        t.Fatalf("Compare: %v", err)
    }
    if diff.HeapDeltaPct < 49 || diff.HeapDeltaPct > 51 {
        t.Errorf("HeapDeltaPct = %.1f, want ~50", diff.HeapDeltaPct)
    }
}

func TestStore_List(t *testing.T) {
    dir := t.TempDir()
    s := NewStore(dir)
    s.Save(Profile{Name: "A", HeapDelta: 1})
    s.Save(Profile{Name: "B", HeapDelta: 2})

    profiles, err := s.List()
    if err != nil {
        t.Fatalf("List: %v", err)
    }
    if len(profiles) != 2 {
        t.Errorf("List returned %d profiles, want 2", len(profiles))
    }
}
```

Run: `go test ./internal/testresource/ -v -run 'TestStore'`
Expected: FAIL — Store not yet implemented

- [ ] **Step 5: Implement Store + Category**

```go
// internal/testresource/store.go
package testresource

import (
    "encoding/json"
    "fmt"
    "math"
    "os"
    "path/filepath"
    "strings"
)

type ProfileSaver interface {
    Save(Profile) error
}

type ProfileLoader interface {
    Load(name string) (Profile, bool)
}

type Store struct {
    dir string
}

func NewStore(dir string) *Store {
    return &Store{dir: dir}
}

func (s *Store) filePath(name string) string {
    safe := strings.ReplaceAll(name, "/", "_")
    safe = strings.ReplaceAll(safe, " ", "_")
    return filepath.Join(s.dir, safe+".json")
}

func (s *Store) Save(p Profile) error {
    if err := os.MkdirAll(s.dir, 0755); err != nil {
        return fmt.Errorf("create profile dir: %w", err)
    }
    data, err := json.MarshalIndent(p, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal profile: %w", err)
    }
    return os.WriteFile(s.filePath(p.Name), data, 0644)
}

func (s *Store) Load(name string) (Profile, bool) {
    data, err := os.ReadFile(s.filePath(name))
    if err != nil {
        return Profile{}, false
    }
    var p Profile
    if err := json.Unmarshal(data, &p); err != nil {
        return Profile{}, false
    }
    p.Name = name
    return p, true
}

func (s *Store) List() ([]Profile, error) {
    entries, err := os.ReadDir(s.dir)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, fmt.Errorf("read profile dir: %w", err)
    }
    var profiles []Profile
    for _, e := range entries {
        if !strings.HasSuffix(e.Name(), ".json") {
            continue
        }
        name := strings.TrimSuffix(e.Name(), ".json")
        if p, ok := s.Load(name); ok {
            profiles = append(profiles, p)
        }
    }
    return profiles, nil
}

type Diff struct {
    Name          string
    HeapDeltaPct  float64
    CPUTimePct    float64
    ReadBytesPct  float64
    WriteBytesPct float64
}

func (s *Store) Compare(baseline, current Profile) (Diff, error) {
    return Diff{
        Name:          baseline.Name,
        HeapDeltaPct:  pctDiff(baseline.HeapDelta, current.HeapDelta),
        CPUTimePct:    pctDiffF(baseline.CPUTimeMs, current.CPUTimeMs),
        ReadBytesPct:  pctDiff(baseline.ReadBytes, current.ReadBytes),
        WriteBytesPct: pctDiff(baseline.WriteBytes, current.WriteBytes),
    }, nil
}

func pctDiff(baseline, current int64) float64 {
    if baseline == 0 && current == 0 {
        return 0
    }
    if baseline == 0 {
        return 100.0
    }
    return math.Round(float64(current-baseline) / float64(baseline) * 100)
}

func pctDiffF(baseline, current float64) float64 {
    if baseline == 0 && current == 0 {
        return 0
    }
    if baseline == 0 {
        return 100.0
    }
    return math.Round((current - baseline) / baseline * 100)
}
```

```go
// internal/testresource/category.go
package testresource

const (
    ioReadThreshold  = 10 * 1024 * 1024  // 10MB
    ioWriteThreshold = 1 * 1024 * 1024   // 1MB
    cpuRatioThreshold = 0.5
)

func Classify(p Profile) Category {
    isIO := p.ReadBytes > ioReadThreshold || p.WriteBytes > ioWriteThreshold
    isCPU := p.DurationMs > 0 && p.CPUTimeMs/p.DurationMs > cpuRatioThreshold

    switch {
    case isIO && isCPU:
        return CategoryMixed
    case isIO:
        return CategoryIOHeavy
    case isCPU:
        return CategoryCPUHeavy
    default:
        return CategoryUncategorized
    }
}
```

Run: `go test ./internal/testresource/ -v`
Expected: All PASS

- [ ] **Step 6: Verify fmt + vet**

Run: `go fmt ./internal/testresource/ && go vet ./internal/testresource/`
Expected: Clean output

- [ ] **Step 7: Commit**

```bash
git add internal/testresource/
git commit -m "feat(testresource): add resource profiling package

- Profile struct for per-test resource snapshots (heap, CPU, I/O)
- Monitor wraps testing.T, captures start/end deltas
- Store persists/loads/compares profiles as JSON
- Category classifier (IO_Heavy/CPU_Heavy/Mixed)
- procfsReader reads /proc/self/stat + /proc/self/io
- All types use narrow interfaces (ISP), Option pattern"
```

---

### Task 2: `internal/testscheduler/` — Scheduling Package

**Files:**
- Create: `internal/testscheduler/plan.go` — Plan, Schedule, Lane, PlanSlot types
- Create: `internal/testscheduler/planner.go` — Planner + PlanningStrategy
- Create: `internal/testscheduler/scheduler.go` — Scheduler + token buckets
- Create: `internal/testscheduler/planner_test.go` — Planner unit tests
- Create: `internal/testscheduler/scheduler_test.go` — Scheduler unit tests

**Interfaces:**
- Consumes: `testresource.Profile`, `testresource.Category`, `testresource.ProfileLoader`, `testresource.ResourceReader`
- Produces: `testscheduler.Planner`, `testscheduler.PlanningStrategy`, `testscheduler.Scheduler`, `testscheduler.ResourceLimit`

- [ ] **Step 1: Write failing tests for Planner**

```go
// internal/testscheduler/planner_test.go
package testscheduler

import (
    "testing"
    "github.com/mendixlabs/mxcli/internal/testresource"
)

type mockProfileLoader struct {
    profiles []testresource.Profile
}

func (m *mockProfileLoader) Load(name string) (testresource.Profile, bool) {
    for _, p := range m.profiles {
        if p.Name == name {
            return p, true
        }
    }
    return testresource.Profile{}, false
}

func TestPlanner_AssignsIOHeavyToIOLane(t *testing.T) {
    profiles := []testresource.Profile{
        {Name: "IO_Test", ReadBytes: 50_000_000, WriteBytes: 0, DurationMs: 1000, CPUTimeMs: 100},
    }
    limits := ResourceLimit{MaxParallelIO: 2, MaxParallelCPU: 4}
    p := NewPlanner(nil, limits)
    schedule := p.Plan(profiles)

    if len(schedule.Lanes) == 0 {
        t.Fatal("expected at least 1 lane")
    }

    // The IO-heavy test should be in the IO lane
    var found bool
    for _, lane := range schedule.Lanes {
        for _, slot := range lane.Tests {
            if slot.Name == "IO_Test" {
                if lane.Name != "io" {
                    t.Errorf("IO_Test in lane %q, want 'io'", lane.Name)
                }
                found = true
            }
        }
    }
    if !found {
        t.Error("IO_Test not assigned to any lane")
    }
}

func TestPlanner_AssignsCPUHeavyToCPULane(t *testing.T) {
    profiles := []testresource.Profile{
        // cpu_time / duration = 800/1000 = 0.8 > 0.5 threshold
        {Name: "CPU_Test", DurationMs: 1000, CPUTimeMs: 800, ReadBytes: 0},
    }
    limits := ResourceLimit{MaxParallelIO: 2, MaxParallelCPU: 4}
    p := NewPlanner(nil, limits)
    schedule := p.Plan(profiles)

    var found bool
    for _, lane := range schedule.Lanes {
        for _, slot := range lane.Tests {
            if slot.Name == "CPU_Test" {
                if lane.Name != "cpu" {
                    t.Errorf("CPU_Test in lane %q, want 'cpu'", lane.Name)
                }
                found = true
            }
        }
    }
    if !found {
        t.Error("CPU_Test not assigned to any lane")
    }
}

func TestPlanner_StaggersByDuration(t *testing.T) {
    profiles := []testresource.Profile{
        {Name: "Long", DurationMs: 10000, ReadBytes: 50_000_000},
        {Name: "Short", DurationMs: 1000, ReadBytes: 50_000_000},
    }
    limits := ResourceLimit{MaxParallelIO: 2, MaxParallelCPU: 4}
    p := NewPlanner(nil, limits)
    schedule := p.Plan(profiles)

    for _, lane := range schedule.Lanes {
        if lane.Name != "io" {
            continue
        }
        if len(lane.Tests) < 2 {
            continue
        }
        // Long should come first, then Short (longest first for better staggering)
        if lane.Tests[0].Duration < lane.Tests[1].Duration {
            t.Errorf("expected Long before Short by duration, got Long=%v Short=%v",
                lane.Tests[0].Duration, lane.Tests[1].Duration)
        }
    }
}
```

Run: `go test ./internal/testscheduler/ -v -run 'TestPlanner'`
Expected: FAIL — package doesn't exist

- [ ] **Step 2: Implement Planner + Plan types**

```go
// internal/testscheduler/plan.go
package testscheduler

import (
    "time"
    "github.com/mendixlabs/mxcli/internal/testresource"
)

type ResourceLimit struct {
    MaxParallelIO  int
    MaxParallelCPU int
    MaxHeapMB      int
}

type PlanSlot struct {
    Name     string
    Duration time.Duration
    Profile  testresource.Profile
}

type Lane struct {
    Name  string
    Tests []PlanSlot
}

type Schedule struct {
    Lanes  []Lane
    Limits ResourceLimit
}

// internal/testscheduler/planner.go
package testscheduler

import (
    "sort"
    "time"
    "github.com/mendixlabs/mxcli/internal/testresource"
)

type PlanningStrategy interface {
    Plan(profiles []testresource.Profile, limits ResourceLimit) Schedule
}

type Planner struct {
    store    testresource.ProfileLoader
    limits   ResourceLimit
    strategy PlanningStrategy
}

func NewPlanner(store testresource.ProfileLoader, limits ResourceLimit) *Planner {
    return &Planner{
        store:    store,
        limits:   limits,
        strategy: &DefaultStrategy{},
    }
}

func (p *Planner) Plan(profiles []testresource.Profile) Schedule {
    return p.strategy.Plan(profiles, p.limits)
}

type DefaultStrategy struct{}

func (s *DefaultStrategy) Plan(profiles []testresource.Profile, limits ResourceLimit) Schedule {
    var ioHeavy, cpuHeavy, mixed, uncat []testresource.Profile

    for _, p := range profiles {
        switch testresource.Classify(p) {
        case testresource.CategoryIOHeavy:
            ioHeavy = append(ioHeavy, p)
        case testresource.CategoryCPUHeavy:
            cpuHeavy = append(cpuHeavy, p)
        case testresource.CategoryMixed:
            mixed = append(mixed, p)
        default:
            uncat = append(uncat, p)
        }
    }

    // Sort each lane by duration descending (longest first for staggering)
    sortByDurationDesc := func(profiles []testresource.Profile) {
        sort.Slice(profiles, func(i, j int) bool {
            return profiles[i].DurationMs > profiles[j].DurationMs
        })
    }
    sortByDurationDesc(ioHeavy)
    sortByDurationDesc(cpuHeavy)
    sortByDurationDesc(mixed)
    sortByDurationDesc(uncat)

    lanes := []Lane{}
    if len(ioHeavy) > 0 {
        lanes = append(lanes, laneFromProfiles("io", ioHeavy))
    }
    if len(cpuHeavy) > 0 {
        lanes = append(lanes, laneFromProfiles("cpu", cpuHeavy))
    }
    if len(mixed) > 0 {
        lanes = append(lanes, laneFromProfiles("mixed", mixed))
    }
    if len(uncat) > 0 {
        lanes = append(lanes, laneFromProfiles("uncategorized", uncat))
    }

    return Schedule{Lanes: lanes, Limits: limits}
}

func laneFromProfiles(name string, profiles []testresource.Profile) Lane {
    slots := make([]PlanSlot, len(profiles))
    for i, p := range profiles {
        slots[i] = PlanSlot{
            Name:     p.Name,
            Duration: time.Duration(p.DurationMs) * time.Millisecond,
            Profile:  p,
        }
    }
    return Lane{Name: name, Tests: slots}
}
```

Run: `go test ./internal/testscheduler/ -v -run 'TestPlanner'`
Expected: PASS

- [ ] **Step 3: Write failing tests for Scheduler**

```go
// internal/testscheduler/scheduler_test.go
package testscheduler

import (
    "context"
    "sync"
    "testing"
    "time"
)

type fakeResourceReader struct{}

func (f *fakeResourceReader) Read() (testresource.ResourceSnapshot, error) {
    return testresource.ResourceSnapshot{}, nil
}

func TestScheduler_AcquireIO_BlocksWhenFull(t *testing.T) {
    s := New(1, 2, &fakeResourceReader{})

    // Acquire the single token
    ctx := context.Background()
    if err := s.AcquireIO(ctx); err != nil {
        t.Fatalf("AcquireIO: %v", err)
    }

    // Second acquire should block
    done := make(chan bool, 1)
    go func() {
        err := s.AcquireIO(ctx)
        done <- (err == nil)
    }()

    select {
    case <-done:
        t.Fatal("AcquireIO should have blocked, but returned immediately")
    case <-time.After(50 * time.Millisecond):
        // Expected: blocked
    }

    s.ReleaseIO()

    select {
    case ok := <-done:
        if !ok {
            t.Fatal("AcquireIO failed after release")
        }
    case <-time.After(100 * time.Millisecond):
        t.Fatal("AcquireIO should have unblocked after release")
    }
}

func TestScheduler_ConcurrentIOAndCPU(t *testing.T) {
    s := New(2, 2, &fakeResourceReader{})
    ctx := context.Background()

    var wg sync.WaitGroup
    for i := 0; i < 4; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            if id%2 == 0 {
                s.AcquireIO(ctx)
                defer s.ReleaseIO()
            } else {
                s.AcquireCPU(ctx)
                defer s.ReleaseCPU()
            }
            time.Sleep(10 * time.Millisecond)
        }(i)
    }
    wg.Wait()
}

func TestScheduler_Adjust_ChangesTokenCounts(t *testing.T) {
    s := New(4, 4, &fakeResourceReader{})

    // Adjust to lower limits
    s.Adjust(ResourceLimit{MaxParallelIO: 1, MaxParallelCPU: 1})

    ctx := context.Background()
    if err := s.AcquireIO(ctx); err != nil {
        t.Fatalf("AcquireIO: %v", err)
    }

    done := make(chan bool, 1)
    go func() {
        err := s.AcquireIO(ctx)
        done <- (err == nil)
    }()

    select {
    case <-done:
        t.Fatal("second AcquireIO should have blocked after adjust to 1")
    case <-time.After(50 * time.Millisecond):
        // Expected
    }
}
```

Run: `go test ./internal/testscheduler/ -v -run 'TestScheduler'`
Expected: FAIL — Scheduler not yet implemented

- [ ] **Step 4: Implement Scheduler**

```go
// internal/testscheduler/scheduler.go
package testscheduler

import (
    "context"
    "github.com/mendixlabs/mxcli/internal/testresource"
)

type Scheduler struct {
    ioToken  chan struct{}
    cpuToken chan struct{}
    reader   testresource.ResourceReader
}

func New(maxIO, maxCPU int, reader testresource.ResourceReader) *Scheduler {
    s := &Scheduler{
        ioToken:  make(chan struct{}, maxIO),
        cpuToken: make(chan struct{}, maxCPU),
        reader:   reader,
    }
    // Fill token buckets
    for i := 0; i < maxIO; i++ {
        s.ioToken <- struct{}{}
    }
    for i := 0; i < maxCPU; i++ {
        s.cpuToken <- struct{}{}
    }
    return s
}

func (s *Scheduler) AcquireIO(ctx context.Context) error {
    select {
    case <-s.ioToken:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (s *Scheduler) AcquireCPU(ctx context.Context) error {
    select {
    case <-s.cpuToken:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (s *Scheduler) ReleaseIO() {
    s.ioToken <- struct{}{}
}

func (s *Scheduler) ReleaseCPU() {
    s.cpuToken <- struct{}{}
}

func (s *Scheduler) Adjust(limits ResourceLimit) {
    // Drain and recreate token channels with new sizes
    newIO := make(chan struct{}, limits.MaxParallelIO)
    newCPU := make(chan struct{}, limits.MaxParallelCPU)
    for i := 0; i < limits.MaxParallelIO; i++ {
        newIO <- struct{}{}
    }
    for i := 0; i < limits.MaxParallelCPU; i++ {
        newCPU <- struct{}{}
    }
    s.ioToken = newIO
    s.cpuToken = newCPU
}
```

Run: `go test ./internal/testscheduler/ -v`
Expected: All PASS

- [ ] **Step 5: Verify fmt + vet**

Run: `go fmt ./internal/testscheduler/ && go vet ./internal/testscheduler/`
Expected: Clean output

- [ ] **Step 6: Commit**

```bash
git add internal/testscheduler/
git commit -m "feat(testscheduler): add profile-based test scheduling

- Planner assigns tests to IO/CPU/Mixed lanes by historical profile
- DefaultStrategy puts longest tests first in each lane
- Scheduler uses token buckets for IO and CPU concurrency
- Adjust() allows dynamic token count changes
- PlanningStrategy interface (OCP), ResourceReader interface (ISP)"
```

---

### Task 3: `internal/testresourceregistry/` — Composition Root

**Files:**
- Create: `internal/testresourceregistry/registry.go`
- Create: `internal/testresourceregistry/registry_test.go`

**Interfaces:**
- Consumes: `testresource.Store`, `testresource.ProfileSaver`, `testresource.ProfileLoader`, `testscheduler.Planner`, `testscheduler.Scheduler`
- Produces: `Registry` — concrete type for TestMain composition

- [ ] **Step 1: Write failing tests**

```go
// internal/testresourceregistry/registry_test.go
package testresourceregistry

import (
    "testing"
    "github.com/mendixlabs/mxcli/internal/testresource"
)

func TestRegistry_RecordAndBuildSchedule(t *testing.T) {
    dir := t.TempDir()
    r := New(dir, 2, 4, 500)

    r.Record(testresource.Profile{
        Name: "TestA", ReadBytes: 50_000_000, DurationMs: 1000, CPUTimeMs: 100,
    })
    r.Record(testresource.Profile{
        Name: "TestB", DurationMs: 1000, CPUTimeMs: 800,
    })

    schedule := r.BuildSchedule()
    if len(schedule.Lanes) == 0 {
        t.Fatal("expected at least 1 lane")
    }
}

func TestRegistry_CheckRegressions_ReturnsDiffs(t *testing.T) {
    dir := t.TempDir()
    r := New(dir, 2, 4, 500)

    r.Record(testresource.Profile{
        Name: "TestA", HeapDelta: 1000, DurationMs: 100, CPUTimeMs: 50,
    })

    // Overwrite with different values
    r.Record(testresource.Profile{
        Name: "TestA", HeapDelta: 2000, DurationMs: 100, CPUTimeMs: 50,
    })

    diffs, err := r.CheckRegressions()
    if err != nil {
        t.Fatalf("CheckRegressions: %v", err)
    }
    if len(diffs) == 0 {
        t.Fatal("expected at least 1 diff")
    }
    // Should detect ~100% heap increase
    if diffs[0].HeapDeltaPct < 90 || diffs[0].HeapDeltaPct > 110 {
        t.Errorf("HeapDeltaPct = %.1f, want ~100", diffs[0].HeapDeltaPct)
    }
}
```

Run: `go test ./internal/testresourceregistry/ -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 2: Implement Registry**

```go
// internal/testresourceregistry/registry.go
package testresourceregistry

import (
    "github.com/mendixlabs/mxcli/internal/testresource"
    "github.com/mendixlabs/mxcli/internal/testscheduler"
)

type Registry struct {
    store     *testresource.Store
    planner   *testscheduler.Planner
    scheduler *testscheduler.Scheduler
}

func New(profileDir string, maxIO, maxCPU, maxHeapMB int) *Registry {
    store := testresource.NewStore(profileDir)
    limits := testscheduler.ResourceLimit{
        MaxParallelIO:  maxIO,
        MaxParallelCPU: maxCPU,
        MaxHeapMB:      maxHeapMB,
    }
    planner := testscheduler.NewPlanner(store, limits)
    reader := &testresource.ProcfsReader{}
    scheduler := testscheduler.New(maxIO, maxCPU, reader)
    return &Registry{
        store:     store,
        planner:   planner,
        scheduler: scheduler,
    }
}

func (r *Registry) Record(p testresource.Profile) error {
    return r.store.Save(p)
}

func (r *Registry) BuildSchedule() testscheduler.Schedule {
    profiles, err := r.store.List()
    if err != nil || len(profiles) == 0 {
        return testscheduler.Schedule{}
    }
    return r.planner.Plan(profiles)
}

func (r *Registry) CheckRegressions() ([]testresource.Diff, error) {
    profiles, err := r.store.List()
    if err != nil {
        return nil, err
    }
    var diffs []testresource.Diff
    for _, current := range profiles {
        baseline, ok := r.store.Load(current.Name)
        if !ok {
            continue
        }
        diff, _ := r.store.Compare(baseline, current)
        diffs = append(diffs, diff)
    }
    return diffs, nil
}

func (r *Registry) Scheduler() *testscheduler.Scheduler {
    return r.scheduler
}
```

**Note:** The `ProcfsReader` type in `testresource` needs to be exported. Add to `monitor.go`:
```go
// Change procfsReader to ProcfsReader
type ProcfsReader struct{}
func (r *ProcfsReader) Read() (ResourceSnapshot, error) { ... }
```

Also need to update the `Monitor` to use `ProcfsReader` as well:
```go
reader: &ProcfsReader{},
```

Run: `go test ./internal/testresourceregistry/ -v`
Expected: Pass (may need to fix the import cycle — ProcfsReader must be exported)

- [ ] **Step 3: Update monitor.go to export ProcfsReader**

Edit `internal/testresource/monitor.go`:
- Rename `procfsReader` to `ProcfsReader`
- Update the `NewMonitor` default: `reader: &ProcfsReader{}`

Run: `go test ./internal/testresource/... ./internal/testscheduler/... ./internal/testresourceregistry/... -v`
Expected: All PASS

- [ ] **Step 4: Verify fmt + vet**

Run: `go fmt ./internal/testresourceregistry/ && go vet ./internal/testresourceregistry/`
Expected: Clean

- [ ] **Step 5: Commit**

```bash
git add internal/testresourceregistry/ internal/testresource/monitor.go
git commit -m "feat(testresourceregistry): add registry composition root

- Registry wires Store + Planner + Scheduler together
- Record() saves profiles, BuildSchedule() plans lanes
- CheckRegressions() compares current vs baseline profiles
- Exported ProcfsReader from testresource package
- DIP: Registry is the only place that knows concrete implementations"
```

---

### Task 4: mxgraph Sharing in Roundtrip Tests

**Files:**
- Modify: `mdl/executor/roundtrip_helpers_test.go`
- Reference: `mdl/backend/mpr/backend_lifecycle.go` (adapter registration)

**Interfaces:**
- Consumes: `modelsdk`, `internal/mxgraph`, `internal/mxgraph/adapter/mpr`, `internal/mxgraph/adapter/themescss`, `internal/mxgraph/adapter/designdprops`, `mdl/graphcatalog`
- Produces: `sharedProjectGraph` package var, injected via `MprBackend.SetProjectGraph()`

- [ ] **Step 1: Read the current roundtrip_helpers_test.go to plan modifications**

Run: `wc -l mdl/executor/roundtrip_helpers_test.go`

```go
// Key insertion points:
// 1. Add imports for mxgraph types (after line 30)
// 2. Add sharedProjectGraph variable (after line 43)
// 3. Add buildSharedGraph function (near TestMain)
// 4. Modify TestMain to build graph (before os.Exit)
// 5. Modify setupTestEnv to inject graph (after connect)
```

- [ ] **Step 2: Add imports for mxgraph types**

Edit `mdl/executor/roundtrip_helpers_test.go` — add these imports after line 30 (`"github.com/pmezard/go-difflib/difflib"`):

```go
    "context"

    "github.com/mendixlabs/mxcli/internal/mxgraph"
    mpradapter "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/mpr"
    designdprops "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/designdprops"
    themescss "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/themescss"
    "github.com/mendixlabs/mxcli/mdl/graphcatalog"
    "github.com/mendixlabs/mxcli/modelsdk"
    mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
```

- [ ] **Step 3: Add sharedProjectGraph variable after line 44**

Edit — after `var sharedSourceMPR string` (around line 46), add:

```go
// sharedProjectGraph is built once in TestMain and shared across all
// integration tests to avoid rebuilding mxgraph per test (~14s each).
var sharedProjectGraph *graphcatalog.ProjectGraph
```

- [ ] **Step 4: Add buildSharedGraph function**

Add before `TestMain` (before line 50):

```go
// buildSharedGraph constructs the mxgraph index from the source project.
// Returns nil on any error — tests work without graph acceleration.
func buildSharedGraph(mprPath string) *graphcatalog.ProjectGraph {
    m, err := modelsdk.Open(mprPath)
    if err != nil {
        return nil
    }
    defer m.Close()

    mgr := mxgraph.NewIndexManager()
    mgr.RegisterAdapter(&mpradapter.DomainModelAdapter{Model: m})
    mgr.RegisterAdapter(&mpradapter.MicroflowAdapter{Model: m})
    mgr.RegisterAdapter(&mpradapter.PageAdapter{Model: m})
    mgr.RegisterAdapter(&mpradapter.SecurityAdapter{Model: m})
    mgr.RegisterAdapter(&mpradapter.EnumerationAdapter{Model: m})
    mgr.RegisterAdapter(&mpradapter.WorkflowAdapter{Model: m})
    docCache := mpradapter.NewBsonDocCache()
    mgr.RegisterAdapter(&mpradapter.AccessRuleAdapter{Model: m})
    mgr.RegisterAdapter(&mpradapter.DocumentGrantAdapter{Model: m})
    mgr.RegisterAdapter(&mpradapter.PageRefAdapter{Model: m, DocCache: docCache})
    mgr.RegisterAdapter(&mpradapter.NavigationAdapter{
        Source: &mpradapter.ModelsdkUnitSource{Model: m},
    })
    mgr.RegisterAdapter(&mpradapter.DataContainerAdapter{
        Source:   &mpradapter.ModelsdkUnitSource{Model: m},
        Model:    m,
        DocCache: docCache,
    })

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := mgr.BuildAll(ctx, mgr); err != nil {
        return nil
    }

    return graphcatalog.NewProjectGraph(mgr)
}
```

- [ ] **Step 5: Modify TestMain to build shared graph**

In `TestMain` (around line 93), after `sharedSourceMPR = "App.mpr"` and before the `os.Exit` block, add:

```go
    // Build shared mxgraph index from the source project
    mprPath := filepath.Join(sharedSourceProject, sharedSourceMPR)
    sharedProjectGraph = buildSharedGraph(mprPath)
    if sharedProjectGraph == nil {
        fmt.Fprintln(os.Stderr, "Note: mxgraph not available — tests will run without graph acceleration")
    } else {
        fmt.Fprintln(os.Stderr, "Info: shared mxgraph built successfully")
    }
```

Similarly for the committed source project path (around line 57), after `sharedSourceMPR = sourceProjectMPR`:

```go
    sharedProjectGraph = buildSharedGraph(filepath.Join(sharedSourceProject, sharedSourceMPR))
```

- [ ] **Step 6: Modify setupTestEnv to inject shared graph**

In `setupTestEnv` (around line 253), after `env.ensureTestModule()`, add:

```go
    // Inject shared mxgraph if available
    if sharedProjectGraph != nil {
        if mprB, ok := exec.Backend().(*mprbackend.MprBackend); ok {
            mprB.SetProjectGraph(sharedProjectGraph)
        }
    }
```

- [ ] **Step 7: Verify the test compiles**

Run: `go build ./mdl/executor/`
Expected: Build succeeds (mprbackend.MprBackend is used in the same file already on line 234)

- [ ] **Step 8: Commit**

```bash
git add mdl/executor/roundtrip_helpers_test.go
git commit -m "perf: share mxgraph across roundtrip integration tests

- Build mxgraph index once in TestMain
- Inject via MprBackend.SetProjectGraph() in setupTestEnv
- Avoids ~14s per-test graph rebuild
- Non-fatal: tests work without graph acceleration"
```

---

### Task 5: Makefile + CI Integration

**Files:**
- Modify: `Makefile` — add 3 new targets
- Modify: `.github/workflows/ci.yml` — add profiling step

- [ ] **Step 1: Add Makefile targets**

After the `test-integration` target (around line 301), add:

```makefile
# Run integration tests with resource profiling and profile-based scheduling.
# Profiles are saved to coverage/test-profiles/ for later analysis.
# Scheduling uses historical profiles to assign tests to IO/CPU/Mixed lanes.
test-integration-profiled: build install-daemon
	@mkdir -p coverage/test-profiles
	CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m \
		-resource-profile -resource-schedule ./...

# Check resource profiles against baselines.
# Fails if any test exceeds its baseline by >20% in any metric.
test-profile-check:
	go test -tags integration -count=1 -run '^TestProfileCheck$$' \
		-resource-check ./...

# Record new resource profile baselines (replaces all existing profiles).
# Run after intentional performance changes to silence the regression gate.
test-profile-record: build
	@mkdir -p coverage/test-profiles
	CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m \
		-p 1 -resource-record ./...
	@echo "Profiles recorded in coverage/test-profiles/"
```

- [ ] **Step 2: Add `test-integration-profiled` to `.PHONY` list**

Find `.PHONY:` line (around line 57) and add:
```
test-integration-profiled test-profile-check test-profile-record
```

- [ ] **Step 3: Add CI profiling step to `.github/workflows/ci.yml`**

Read the current CI file first:
```bash
cat .github/workflows/ci.yml
```

Edit to add after the `make test-integration` step:
```yaml
      - name: Integration tests with profiling
        run: make test-integration
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

- [ ] **Step 4: Verify Makefile syntax**

Run: `make -n test-integration-profiled 2>&1 | head -5`
Expected: Shows the go test command without errors

- [ ] **Step 5: Commit**

```bash
git add Makefile .github/workflows/ci.yml
git commit -m "ci: add test profiling and regression check targets

- test-integration-profiled: runs with profiling + scheduling
- test-profile-check: compares profiles against baselines
- test-profile-record: replaces all profiles with new baseline
- CI uploads profiles as artifacts, checks regressions"
```

---

### Task 6: TESTING_GUIDE.md Update

**Files:**
- Modify: `docs/03-development/TESTING_GUIDE.md`

- [ ] **Step 1: Add L7 layer to the test pipeline table**

Edit the layer table (around line 16-24) to add:
```markdown
| L7 | Resource Profile | `coverage/test-profiles/*.json` | 任意包 | 需要 `-resource-record` flag |
```

- [ ] **Step 2: Add L7 section after Benchmark section (around line 176)**

```markdown
### L7 — Resource Profile（集成测试资源分析）

记录每个集成测试的资源使用量，用于调度和回归检测。

```go
func TestRoundtrip_Microflow_Basic(t *testing.T) {
    monitor := testresource.NewMonitor(t)
    defer monitor.Done() // profile captured at test end

    env := setupTestEnv(t)
    defer env.teardown()
    // ... test logic ...
}
```

**资源分类阈值：**

| 指标 | IO Heavy | CPU Heavy |
|------|----------|-----------|
| ReadBytes | > 10MB | — |
| WriteBytes | > 1MB | — |
| CPUTime/Duration | — | > 50% |

**调度规则：** IO Heavy 测试跑在 IO lane（默认 2 并发），CPU Heavy 跑在 CPU lane（默认 nproc 并发）。Mixed 测试跑在单独 lane。每个 lane 按 Duration 降序排列（长测试先启动）。
```

- [ ] **Step 3: Add PR checklist item**

Add to the PR requirements list (around line 191-198):
```markdown
[ ] L7: 如新增集成测试（roundtrip_*），须包含 `testresource.NewMonitor(t)`（或用 `make test-profile-record` 重新记录 profile）
```

- [ ] **Step 4: Commit**

```bash
git add docs/03-development/TESTING_GUIDE.md
git commit -m "docs: add L7 resource profile layer to testing guide"
```

---

### Verification

- [ ] **Run `make test`** — unit tests pass (no regressions)
- [ ] **Run `make test-integration`** — integration tests pass with shared mxgraph
- [ ] **Run `go test ./internal/testresource/... ./internal/testscheduler/... ./internal/testresourceregistry/... -v`** — all new package tests pass
- [ ] **Run `go vet ./internal/...`** — clean
- [ ] **Run `go fmt ./internal/...`** — clean
