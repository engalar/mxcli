# TUI Architecture Refactoring Design

## Context

The TUI (`cmd/mxcli/tui/`) is a bubbletea-based terminal UI for Mendix project browsing and operations. It currently consists of **62 files, ~14,900 lines of Go, all in a single `package tui`**. The two features driving this refactoring are **build** (`mxcli local build`) and **run** (`mxcli local run`) — long-running operations that need live progress display, cancel support, and process lifecycle management within the TUI.

## Problems Found

### P0 — App is a God Object

The `App` struct (`app.go:25`) has **25+ fields** spanning 6 unrelated domains: navigation, chrome (tab/status/hint bars), background IO (watcher, agent listener), result caching (check errors), async preview, and agent protocol. Every new feature adds fields, switch cases, and rendering branches to App.

### P0 — Message Switch Violates OCP

`app_update.go:14` `App.Update()` contains a single **~30-case switch** on message type. Adding build/run would require 3+ new message types and 3+ new cases, repeating the same patterns.

### P1 — Type Assertions Everywhere

~25 occurrences of `a.views.Base().(BrowserView)` / `a.views.Active().(OverlayView)` etc. across the codebase. This violates DIP — callers depend on concrete types, not interfaces.

### P1 — No Task/Operation Abstraction

Three existing long-running operations (MDL exec, mx check, project-tree load) each have unique, non-reusable implementations. Adding build/run would create a fourth and fifth. There is no shared abstraction for: progress reporting, cancellation, output streaming, lifecycle management.

### P1 — BrowserView Value Receiver Fragility

`NewBrowserView()` returns a value but stores `*Tab`. `navigateToNode()` uses a pointer receiver while other methods use value receivers. `syncBrowserView()` recreates the view on every tab switch, discarding state.

### P2 — Single Package, No Encapsulation Boundaries

All 62 files share `package tui`. No compile-time dependency direction enforcement. No way to reason about subsystem boundaries from imports.

## Proposed Architecture

### Package Layout

```
cmd/mxcli/tui/
├── app.go                   # Root App — routes to sub-models, owns chrome only
├── view.go                  # View interface + shared messages
│
├── browser/                 # Navigation context: project tree browsing
│   ├── model.go
│   ├── miller.go            # Three-column view
│   ├── column.go            # Single column
│   ├── preview.go           # Async preview with caching
│   └── tree.go              # Tree parsing
│
├── executor/                # Execution context: MDL scripts, build, run
│   ├── model.go
│   ├── exec.go              # MDL script editor
│   ├── build.go             # Build view (live output)
│   └── run.go               # Runtime process view (live output)
│
├── diagnostics/             # Diagnostics context: check, lint
│   └── check.go
│
├── overlay/                 # Modal/dialog context
│   ├── overlay.go
│   ├── confirm.go
│   ├── input.go
│   └── content.go
│
├── agent/                   # LLM agent communication
│   ├── listener.go
│   └── protocol.go
│
├── chrome/                  # UI chrome (no App coupling)
│   ├── tabbar.go
│   ├── statusbar.go
│   └── hintbar.go
│
├── task/                    # ★ NEW: Operation abstraction
│   ├── runner.go            # Task interface + lifecycle
│   ├── build.go             # Build task adapter
│   ├── run.go               # Run task adapter
│   └── progress.go          # Progress bar renderer
│
└── kernel/                  # Shared kernel (zero dependencies)
    ├── theme.go
    ├── icons.go
    └── types.go
```

### Dependency Direction

```
app → {browser, executor, diagnostics, overlay, agent, chrome} → task → kernel
```

Strictly acyclic. `kernel` imports nothing else in `tui/`. Every package depends only on `kernel` and external libraries.

### Core Abstraction: `task.Task`

```go
// task/runner.go
type State int
const (
    StatePending State = iota
    StateRunning
    StateCompleted
    StateFailed
    StateCancelled
)

type Event struct {
    State   State
    Phase   string    // "resolving mxbuild", "building PAD", ...
    Message string    // human-readable log line
    Pct     float64   // 0.0–100.0 or -1 if unknown
    Err     error
}

type Task interface {
    Start(ctx context.Context) tea.Cmd  // returns a Cmd that emits Events
    Cancel()                            // close context
    State() State
    Events() <-chan Event               // read-only channel
}
```

Concrete implementations:
- `BuildTask` — wraps `docker.Build()` with phase detection
- `RunTask` — wraps `docker.StartLocal()` with process I/O capture

### View Additions

Both new views follow the same pattern:

```
BuildView / RunView
├── Title bar (phase name)
├── Event log (scrollable, tailing)
├── Progress bar (if available)
├── Status line (Pct, elapsed time, state)
└── Hotkeys: q=close, c=cancel(if running), [/]=scroll
```

These are sub-views pushed onto the view stack, same pattern as `ExecView`.

### How Build and Run Flow Through the System

```
:build command → App routes to executor/build.go
  → Executor creates BuildTask(ctx, opts)
  → Pushes BuildView onto stack
  → BuildView subscribes to task.Events()
  → task.Events drives live rendering
  → q/c cancels → StateCancelled → view pops
  → completion → StateCompleted + final output overlay
```

No new fields on App. No new cases in the giant switch. The executor sub-model handles all build/run lifecycle internally and only sends `PushViewMsg` / `PopViewMsg` to App.

## Migration Strategy

### Phase 1: Extract `kernel/`

Move `theme.go` and `icons.go` into `tui/kernel/`. All references to `MutedColor`, `AccentStyle`, etc. are updated to `kernel.MutedColor`, etc.

**Files:** create `kernel/theme.go`, `kernel/icons.go`, `kernel/types.go`

**Test:** `make build && make test`

### Phase 2: Extract `chrome/`

Move `tabbar.go`, `statusbar.go`, `hintbar.go` into `tui/chrome/`. These components are already clean — they take styled strings, render, and return. No logic changes needed.

**Files:** move `tabbar.go`, `statusbar.go`, `hintbar.go`

**Test:** `make build && make test`

### Phase 3: Build `task/` package

Create `task/runner.go` with the `Task` interface. Implement `task/build.go` (wrapping `docker.Build()`) and `task/run.go` (wrapping `docker.StartLocal()`). These are pure Go — no bubbletea dependency.

**Key design:** `task.Build` should call `docker.Build()` in a goroutine and emit progress events as phases transition. Phase detection logic:

```
Phase 0: "resolving mxbuild"      → 5%
Phase 1: "resolving JDK"          → 10%
Phase 2: "resolving Maven JARs"   → 15%
Phase 3: "pre-build check"        → 20%
Phase 4: "building Mendix app"    → 20–80%
Phase 5: "post-processing PAD"    → 80–95%
Phase 6: "done"                   → 100%
```

**Files:** create `task/runner.go`, `task/build.go`, `task/run.go`, `task/progress.go`

**Test:** `make build && make test`

### Phase 4: Extract `overlay/`

Move `overlay.go` → `overlay/overlay.go`, `overlayview.go` → `overlay/view.go`, `confirmview.go` → `overlay/confirm.go`, `inputview.go` → `overlay/input.go`, `contentview.go` → `overlay/content.go`.

**Files:** move 5 files

**Test:** `make build && make test`

### Phase 5: Extract `browser/`

Move `miller.go`, `column.go`, `browserview.go`, `preview.go` into `tui/browser/`. This is the largest extraction — `miller.go` alone is 940 lines. Keep `miller.go` as-is but in the `browser` package; do not refactor internally during extraction.

Create `browser/tree.go` by extracting from existing code:
- `ParseTree()` (currently in browserview.go or app.go)
- `TreeNode` related helpers (`findNodePath`, `flattenQualifiedNames`)
- `LoadTreeMsg` and `LoadTreeDoneMsg` message types
- `buildDescribeCmd()` (currently in preview.go)

**Note:** `PreviewEngine` currently references `PreviewEngine.mxcliPath`/`projectPath` which are also held by `App`/`BrowserView`. After extraction, `browser.Model` becomes the owner of `PreviewEngine`. `App` no longer directly calls `previewEngine.ClearCache()` — it sends a `BrowserClearCacheMsg` to the browser sub-model.

**Files:** move `miller.go`, `column.go`, `browserview.go`, `preview.go`; create `browser/tree.go`, `browser/model.go`

**Test:** `make build && make test`

### Phase 6: Extract `executor/`

Move `execview.go` → `executor/`. Create `executor/build.go` (BuildView) and `executor/run.go` (RunView). Create `executor/model.go` as the sub-model that owns execution state.

The executor sub-model replaces App's direct knowledge of exec/build/run state. App delegates to executor via messages:

```go
type ExecutorExecuteMsg struct{ MDL string }
type ExecutorBuildMsg struct{ Options docker.BuildOptions }
type ExecutorRunMsg struct{ Options docker.LocalRunOptions }
```

Executor model handles these, creates tasks, pushes views, manages lifecycle — entirely within its own `Update()`.

**Files:** create `executor/model.go`, `executor/build.go`, `executor/run.go`; move `execview.go`

**Test:** `make build && make test`

### Phase 7: Simplify `App`

After Phases 1–6, App loses ~15 fields:
- `previewEngine` → owned by `browser.Model`
- `checkErrors` / `checkNavActive` / `checkNavIndex` / `checkNavLocations` / `checkRunning` → owned by `diagnostics.Model`
- `agentListener` / `agentAutoProceed` / `agentPending` / `agentCheckCh` / `agentCheckReqID` / `agentExecCtx` → owned by `agent.Model`

App keeps only:
- `tabs []Tab`, `activeTab int`, `nextTabID int`
- `width`, `height`, `mxcliPath`
- `views ViewStack`, `showHelp`, `picker`
- `tabBar`, `hintBar`, `statusBar` (or sub-model chrome)

App.Update shrinks from 627 lines to ~150 lines — mostly delegating to sub-models.

**Files:** rewrite `app.go`, `app_update.go`, `app_view.go`, `app_keys.go`

**Test:** `make build && make test`

### Phase 8: Register `:build` and `:run` Commands

Add palette entries:
```go
{Name: "Build Project", Key: "b", Category: "Build"},  // replaces existing 'b'=BSON
{Name: "Run Project", Key: "R", Category: "Build"},     // note: 'r'=refresh, 'R'=hard reload; Run is palette-only
```

`b` key in browser mode currently opens BSON dump. Re-bind BSON to `B` (shift). Build gets `b` as a dedicated key. Run is **palette-only** (`:run`) since both `r` and `R` are taken.

**Files:** `commandpalette.go`, `app_keys.go`, `executor/build.go`/`run.go`

**Test:** `make build && make test`

### Phase 9: Polish and Edge Cases

- Port conflict detection shown in RunView before launch
- `mx check` auto-run after build, results shown in build output
- Session persistence for build/run state
- `:build --skip-check` variant
- `:run --db postgres://...` flag input via InputView

---

## In-Scope / Out-of-Scope

**In scope:**
- Package extraction with zero behavioral change
- Build and run views with live progress
- Task abstraction for all long-running operations
- Command palette integration

**Out of scope:**
- Refactoring MillerView's 940-line internals
- Changing the compare/diff view architecture
- Adding new build/run features beyond basic PAD build and local runtime start
- Performance optimization of rendering

## Risks

1. **Phase 5 (browser extraction) risk:** 940-line `miller.go` has implicit dependencies on `Tab`, `App`, and `PreviewEngine`. Need to define clean interface boundaries.
2. **Phase 7 (App simplification) risk:** Sub-model message routing must be lossless — no messages falling through cracks during migration.
3. **Build integration risk:** `docker.Build()` takes 2–5 minutes. Goroutine leak if user quits TUI during build. Need context cancellation + cleanup.

Mitigation: each phase is independently testable with `make build && make test`. The parallel `task/` package can be tested in isolation with fake progress events.
