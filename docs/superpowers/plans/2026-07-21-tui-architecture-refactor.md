# TUI Architecture Refactoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor 62-file flat `tui` package into sub-packages by bounded context, adding build/run long-running operation views.

**Architecture:** 8 incremental phases, each independently buildable+testable, moving code out of the God `tui` package and into sub-packages (`kernel`, `chrome`, `overlay`, `browser`, `executor`, `task`, `agent`). Re-add type-safer interfaces to eliminate ~25 type assertions.

**Tech Stack:** bubbletea v1.3.10, lipgloss, Go 1.22+

## Global Constraints

- Every phase must pass `make build && make test` before next phase
- No behavioral changes during extraction phases — only package moves and import rewrites
- `tui/` top-level package shrinks but continues to exist as the App+ViewStack wiring layer
- The `task/` package is the only package that imports `cmd/mxcli/docker/` (build/run adapters)
- All new packages go under `cmd/mxcli/tui/` (e.g. `cmd/mxcli/tui/kernel/`)

---

### Phase 1: Extract `kernel/` — Shared Types

**Files:**
- Create: `cmd/mxcli/tui/kernel/theme.go`
- Create: `cmd/mxcli/tui/kernel/icons.go`
- Create: `cmd/mxcli/tui/kernel/types.go`
- Modify: `cmd/mxcli/tui/theme.go` — delete, replaced by kernel import
- Modify: `cmd/mxcli/tui/*.go` — update import paths

**Interfaces:**
- Produces: `kernel.MutedColor`, `kernel.AccentColor`, etc. — same var names, just in `kernel` package
- Produces: `kernel.Hint`, `kernel.StatusInfo`, `kernel.ViewMode` — shared types

- [ ] **Step 1: Create `kernel/types.go`**

Move shared type definitions from `cmd/mxcli/tui/view.go`:
- `ViewMode` enum (all values)
- `StatusInfo` struct
- `Hint` struct
- `PushViewMsg`, `PopViewMsg`

```go
// cmd/mxcli/tui/kernel/types.go
package kernel

import tea "github.com/charmbracelet/bubbletea"

type ViewMode int

const (
	ModeBrowser ViewMode = iota
	ModeOverlay
	ModeCompare
	ModeDiff
	ModePicker
	ModeJumper
	ModeExec
	ModeCommandPalette
	ModeInput
	ModeConfirm
)

func (m ViewMode) String() string {
	switch m {
	case ModeBrowser:    return "Browse"
	case ModeOverlay:    return "Overlay"
	case ModeCompare:    return "Compare"
	case ModeDiff:       return "Diff"
	case ModePicker:     return "Picker"
	case ModeJumper:     return "Jump"
	case ModeExec:       return "Exec"
	case ModeCommandPalette: return "Palette"
	case ModeInput:      return "Input"
	case ModeConfirm:    return "Confirm"
	default:             return "Unknown"
	}
}

type StatusInfo struct {
	Breadcrumb []string
	Position   string
	Mode       string
	Extra      string
}

type Hint struct {
	Key   string
	Label string
}

type PushViewMsg struct{ View View }
type PopViewMsg struct{}
```

- [ ] **Step 2: Create `kernel/theme.go`**

Copy from `cmd/mxcli/tui/theme.go`, adding `package kernel`.

- [ ] **Step 3: Create `kernel/icons.go`**

Copy from `cmd/mxcli/tui/icons.go`, adding `package kernel`.

- [ ] **Step 4: Update all references in `tui/` package**

Replace `AccentColor` → `kernel.AccentColor`, `MutedColor` → `kernel.MutedColor`, etc. in every file in `tui/`. Use `sed` or grep to find all references:

```bash
# grep all theme/kernel references in tui/ (excluding kernel dir itself)
rg -l 'AccentColor|MutedColor|MutedColor|FocusColor|ActiveTabStyle|InactiveTabStyle|ColumnTitleStyle|SelectedItemStyle|DirectoryStyle|LeafStyle|BreadcrumbDimStyle|BreadcrumbCurrentStyle|LoadingStyle|PositionStyle|PreviewModeStyle|HintKeyStyle|HintLabelStyle|StatusBarStyle|CmdBarStyle|FocusedTitleStyle|FocusedEdgeChar|FocusedEdgeStyle|AccentStyle|CheckErrorStyle|CheckWarnStyle|CheckDeprecStyle|CheckPassStyle|CheckLocStyle|CheckHeaderStyle|CheckRunningStyle|AgentBadgeStyle|SeparatorChar|SeparatorStyle|AddedColor|RemovedColor|DiffAddedFg|DiffAddedChangedFg|DiffAddedChangedBg|DiffRemovedFg|DiffRemovedChangedFg|DiffRemovedChangedBg|DiffEqualGutter|DiffGutterAddedFg|DiffGutterRemovedFg' cmd/mxcli/tui/ | grep -v kernel/
```

For each file, add `"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"` import and prefix all references with `kernel.`.

- [ ] **Step 5: Update `view.go` — delegate to kernel types**

```go
// cmd/mxcli/tui/view.go
package tui

import (
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
	tea "github.com/charmbracelet/bubbletea"
)

// Re-export kernel types for backward compatibility during migration.
type ViewMode = kernel.ViewMode
const (
	ModeBrowser         = kernel.ModeBrowser
	ModeOverlay         = kernel.ModeOverlay
	ModeCompare         = kernel.ModeCompare
	ModeDiff            = kernel.ModeDiff
	ModePicker          = kernel.ModePicker
	ModeJumper          = kernel.ModeJumper
	ModeExec            = kernel.ModeExec
	ModeCommandPalette  = kernel.ModeCommandPalette
	ModeInput           = kernel.ModeInput
	ModeConfirm         = kernel.ModeConfirm
)

type StatusInfo = kernel.StatusInfo
type Hint = kernel.Hint
```

Also remove the `View` interface definition and the `PushViewMsg`/`PopViewMsg` from `view.go` since they'll be in `kernel/types.go`. Keep the `View` interface in `tui` package:

```go
// View is the interface that all TUI views must implement.
type View interface {
	Update(tea.Msg) (View, tea.Cmd)
	Render(width, height int) string
	Hints() []kernel.Hint
	StatusInfo() kernel.StatusInfo
	Mode() kernel.ViewMode
}
```

- [ ] **Step 6: Build and test**

```bash
make build && make test
```

Expected: PASS. All theme references now go through `kernel.` prefix.

---

### Phase 2: Extract `chrome/` — StatusBar, HintBar, TabBar

**Files:**
- Create: `cmd/mxcli/tui/chrome/statusbar.go`
- Create: `cmd/mxcli/tui/chrome/hintbar.go`
- Create: `cmd/mxcli/tui/chrome/tabbar.go`
- Modify: remove `cmd/mxcli/tui/statusbar.go`, `hintbar.go`, `tabbar.go`
- Modify: all files referencing these types

**Interfaces:**
- Consumes: `kernel.MutedColor`, `kernel.Hint`, `kernel.StatusInfo`, etc.
- Produces: `chrome.StatusBar`, `chrome.HintBar`, `chrome.TabBar`

- [ ] **Step 1: Create `chrome/statusbar.go`**

Copy `cmd/mxcli/tui/statusbar.go` into `chrome/statusbar.go`, change package to `chrome`, add `kernel.` prefix to all types. The `TabInfo` type currently defined in `tabbar.go` — keep it there.

```go
package chrome

import (
	"strings"
	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

type breadcrumbZone struct {
	startX, endX int
	depth        int
}

type StatusBar struct {
	breadcrumb []string
	position   string
	mode       string
	checkBadge string
	agentBadge string
	viewDepth  int
	viewModes  []string
	zones      []breadcrumbZone
}
```

Copy all methods directly — keep `HitTest`, `View`, `SetBreadcrumb`, etc. unchanged logic.

- [ ] **Step 2: Create `chrome/hintbar.go`**

Copy `cmd/mxcli/tui/hintbar.go` into `chrome/hintbar.go`, change to `package chrome`. Replace `Hint` references with `kernel.Hint`. Keep hint set variables (`ListBrowsingHints`, `OverlayHints`, etc.) — they belong in the view code, not in chrome. Move them to `execview.go`/`browserview.go` etc.:

```go
// In chrome/hintbar.go
package chrome

type HintBar struct {
	hints []kernel.Hint
}
```

Remove the predefined hint sets (they'll move to individual view files later in Phase 6/7). For now, leave them duplicated in `tui/` package but also available in `chrome/`.

- [ ] **Step 3: Create `chrome/tabbar.go`**

Copy `tabbar.go` → `chrome/tabbar.go`. Similar treatment.

- [ ] **Step 4: Update imports in `app.go`, `app_view.go`, etc.**

In all `tui/` files that reference `TabBar`, `StatusBar`, `HintBar`:
- Add import for `chrome`
- Prefix types with `chrome.`

- [ ] **Step 5: Build and test**

```bash
make build && make test
```

---

### Phase 3: Build `task/` Package

**Files:**
- Create: `cmd/mxcli/tui/task/runner.go` — Task interface + state types + streaming cmd
- Create: `cmd/mxcli/tui/task/build.go` — build task wrapping `docker.Build()`
- Create: `cmd/mxcli/tui/task/run.go` — run task wrapping `docker.StartLocal()`
- Create: `cmd/mxcli/tui/task/progress.go` — progress bar renderer
- Create: `cmd/mxcli/tui/task/runner_test.go`

**Interfaces:**
- Produces: `task.State` enum, `task.Event` struct, `task.BuildTask`, `task.RunTask`, `task.StreamCmd()` helper
- Consumes: `docker.BuildOptions`, `docker.LocalRunOptions` (from `cmd/mxcli/docker/`)

**Design note — streaming pattern:** Docker build/run are blocking calls. Phase events are emitted to a channel **before and after** the blocking call. The task exposes a `StreamCmd()` that returns a `tea.Cmd` reading one event from the channel. The view calls `StreamCmd()` in its `Update()` after each event to continue the stream.

```go
// Streaming flow:
// User starts task → StreamCmd() reads event 1 from channel → Update() processes event
//   → re-issues StreamCmd() → reads event 2 → ... → channel closes → stop
```

- [ ] **Step 1: Write tests for `task/runner.go`**

```go
// cmd/mxcli/tui/task/runner_test.go
package task

import (
	"testing"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		s    State
		want string
	}{
		{StatePending, "pending"},
		{StateRunning, "running"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
		{StateCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestBuildTaskDryRun(t *testing.T) {
	task := NewBuildTask(BuildOptions{
		ProjectPath: "/fake/path.mpr",
		DryRun:      true,
	})
	if task.State() != StatePending {
		t.Fatalf("expected StatePending, got %v", task.State())
	}

	// Start sends events then closes channel
	cmd := task.Start()
	msg := cmd()
	ev, ok := msg.(Event)
	if !ok {
		t.Fatalf("expected Event, got %T", msg)
	}
	if ev.State != StateFailed {
		t.Fatalf("expected StateFailed for dry run, got %v", ev.State)
	}
	if task.State() != StateFailed {
		t.Fatalf("expected StateFailed, got %v", task.State())
	}
}

func TestRunTaskDryRun(t *testing.T) {
	task := NewRunTask(RunOptions{
		CmdHint: "test-run",
		DryRun:  true,
	})
	if task.State() != StatePending {
		t.Fatalf("expected StatePending, got %v", task.State())
	}

	cmd := task.Start()
	msg := cmd()
	ev, ok := msg.(Event)
	if !ok {
		t.Fatalf("expected Event, got %T", msg)
	}
	if ev.State != StateFailed {
		t.Fatalf("expected StateFailed for dry run, got %v", ev.State)
	}
}

func TestBuildTaskStreaming(t *testing.T) {
	task := NewBuildTask(BuildOptions{
		ProjectPath: "/fake/path.mpr",
		DryRun:      true,
	})

	// Start returns a Cmd; the first call emits the initial event
	cmd := task.Start()
	msg1 := cmd()
	ev1, ok := msg1.(Event)
	if !ok {
		t.Fatalf("expected Event, got %T", msg1)
	}
	_ = ev1

	// Second call (from StreamCmd) should be nil (channel closed after dry-run fail)
	msg2 := task.StreamCmd()()
	if msg2 != nil {
		t.Fatalf("expected nil after stream end, got %T: %v", msg2, msg2)
	}
	if task.State() != StateFailed {
		t.Fatalf("expected StateFailed, got %v", task.State())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/mxcli/tui/task/ -run TestStateString -v
```

Expected: compilation error (package doesn't exist yet).

- [ ] **Step 3: Create `task/runner.go`**

```go
package task

import tea "github.com/charmbracelet/bubbletea"

type State int

const (
	StatePending State = iota
	StateRunning
	StateCompleted
	StateFailed
	StateCancelled
)

func (s State) String() string {
	switch s {
	case StatePending:   return "pending"
	case StateRunning:   return "running"
	case StateCompleted: return "completed"
	case StateFailed:    return "failed"
	case StateCancelled: return "cancelled"
	default:             return "unknown"
	}
}

type Event struct {
	State   State
	Phase   string
	Message string
	Pct     float64 // 0.0–100.0 or -1 if unknown
	Err     error
}

// StreamCmd returns a tea.Cmd that reads the next event from the event channel.
// Returns nil when the channel is closed (no more events).
// Call StreamCmd() again in Update() after processing each event to continue streaming.
func StreamCmd(ch <-chan Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return ev
	}
}
```

- [ ] **Step 4: Create `task/build.go`**

```go
package task

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

type BuildOptions struct {
	ProjectPath string
	SkipCheck   bool
	DryRun      bool
}

type BuildTask struct {
	opts   BuildOptions
	state  State
	events chan Event
}

func NewBuildTask(opts BuildOptions) *BuildTask {
	return &BuildTask{
		opts:   opts,
		state:  StatePending,
		events: make(chan Event, 10),
	}
}

func (t *BuildTask) State() State { return t.state }
func (t *BuildTask) Emit(ev Event) {
	t.state = ev.State
	select {
	case t.events <- ev:
	default:
	}
}

// Start returns a tea.Cmd that blocks until build completes.
// Call once; the first message contains the completion Event.
func (t *BuildTask) Start() tea.Cmd {
	return func() tea.Msg {
		defer close(t.events)

		if t.opts.DryRun {
			t.Emit(Event{State: StateFailed, Phase: "dry-run", Message: "Dry run — not executing build"})
			return nil
		}

		t.Emit(Event{State: StateRunning, Phase: "mxbuild", Message: "Resolving MxBuild...", Pct: 5})
		t.Emit(Event{State: StateRunning, Phase: "building", Message: "Building PAD package...", Pct: 50})

		err := docker.Build(docker.BuildOptions{
			ProjectPath:     t.opts.ProjectPath,
			SkipCheck:       t.opts.SkipCheck,
			UseDeployLayout: true,
		})

		if err != nil {
			t.Emit(Event{State: StateFailed, Phase: "error", Message: fmt.Sprintf("Build failed: %v", err), Err: err})
			return nil
		}

		t.Emit(Event{State: StateCompleted, Phase: "done", Message: "Build complete", Pct: 100})
		return nil
	}
}

// StreamCmd reads the next buffered event. The view calls this in Update()
// after processing each event until it returns nil.
func (t *BuildTask) StreamCmd() tea.Cmd {
	return StreamCmd(t.events)
}
```

- [ ] **Step 5: Create `task/run.go`**

```go
package task

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

type RunOptions struct {
	PadDir        string
	DB            string
	AdminPassword string
	AppPort       int
	AdminPort     int
	CmdHint       string
	DryRun        bool
}

type RunTask struct {
	opts   RunOptions
	state  State
	events chan Event
}

func NewRunTask(opts RunOptions) *RunTask {
	return &RunTask{
		opts:   opts,
		state:  StatePending,
		events: make(chan Event, 10),
	}
}

func (t *RunTask) State() State { return t.state }
func (t *RunTask) Emit(ev Event) {
	t.state = ev.State
	select {
	case t.events <- ev:
	default:
	}
}

func (t *RunTask) Start() tea.Cmd {
	return func() tea.Msg {
		defer close(t.events)

		if t.opts.DryRun {
			t.Emit(Event{State: StateFailed, Phase: "dry-run", Message: "Dry run — not executing run"})
			return nil
		}

		t.Emit(Event{State: StateRunning, Phase: "startup", Message: "Starting Mendix Runtime...", Pct: 10})

		err := docker.StartLocal(docker.LocalRunOptions{
			PadDir:        t.opts.PadDir,
			DB:            t.opts.DB,
			AdminPassword: t.opts.AdminPassword,
			AppPort:       t.opts.AppPort,
			AdminPort:     t.opts.AdminPort,
			CmdHint:       t.opts.CmdHint,
		})

		if err != nil {
			t.Emit(Event{State: StateFailed, Phase: "error", Message: fmt.Sprintf("Runtime error: %v", err), Err: err})
			return nil
		}

		t.Emit(Event{State: StateCompleted, Phase: "done", Message: "Runtime stopped", Pct: 100})
		return nil
	}
}

func (t *RunTask) StreamCmd() tea.Cmd {
	return StreamCmd(t.events)
}
```

- [ ] **Step 6: Create `task/progress.go`**

```go
package task

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	MutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "245", Dark: "243"})
	AccentStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "214", Dark: "214"})
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "196"})
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "114"})
)

func RenderProgress(ev Event, width int) string {
	if ev.Pct < 0 {
		return MutedStyle.Render("⟳ " + ev.Phase + ": " + ev.Message)
	}
	barWidth := width - 20
	if barWidth < 10 {
		barWidth = 10
	}
	filled := int(float64(barWidth) * ev.Pct / 100.0)
	empty := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	pct := fmt.Sprintf("%3.0f%%", ev.Pct)
	return fmt.Sprintf("%s %s %s", bar, pct, MutedStyle.Render(ev.Message))
}
```

- [ ] **Step 7: Build and test**

```bash
make build && make test
```

Expected: PASS. `task/` package compiles and tests pass.

---

### Phase 4: Extract `overlay/`

**Files:**
- Create: `cmd/mxcli/tui/overlay/overlay.go`
- Create: `cmd/mxcli/tui/overlay/view.go`
- Create: `cmd/mxcli/tui/overlay/confirm.go`
- Create: `cmd/mxcli/tui/overlay/input.go`
- Create: `cmd/mxcli/tui/overlay/content.go`
- Modify: remove original `overlay.go`, `overlayview.go`, `confirmview.go`, `inputview.go`, `contentview.go` from `tui/`

- [ ] **Step 1: Create overlay files**

For each of the 5 files, copy the original content, change to `package overlay`, add `kernel.` prefix to all types (`MutedColor` → `kernel.MutedColor`, etc.), and replace `View` interface references with `tui.View` (import from `tui` package).

- [ ] **Step 2: Update `app_update.go` references**

Replace `NewOverlayView(...)` → `overlay.NewView(...)`, `NewConfirmView(...)` → `overlay.NewConfirmView(...)`, etc.

- [ ] **Step 3: Build and test**

```bash
make build && make test
```

---

### Phase 5: Extract `browser/`

**Files:**
- Create: `cmd/mxcli/tui/browser/miller.go`
- Create: `cmd/mxcli/tui/browser/column.go`
- Create: `cmd/mxcli/tui/browser/browserview.go`
- Create: `cmd/mxcli/tui/browser/preview.go`
- Create: `cmd/mxcli/tui/browser/tree.go`
- Create: `cmd/mxcli/tui/browser/model.go`
- Modify: remove original files from `tui/`

- [ ] **Step 1: Create `browser/tree.go`**

Extract from existing code: `ParseTree()`, `TreeNode`, `findNodePath()`, `flattenQualifiedNames()`, `LoadTreeMsg`.

The `TreeNode` struct and `ParseTree()` are currently in `browserview.go` (lines ~296–310 for `findNodePath`) and `app.go` `Init()` (lines ~193–209 for tree loading):

```go
package browser

import (
	"encoding/json"
	"fmt"
	"strings"
)

type TreeNode struct {
	Label         string      `json:"label"`
	Type          string      `json:"type"`
	QualifiedName string      `json:"qualifiedName"`
	Children      []*TreeNode `json:"children"`
	ElementCount  int         `json:"elementCount"`
}

type flattenedItem struct {
	Label          string
	QualifiedName  string
	Type           string
}

type LoadTreeMsg struct {
	TabID int
	Nodes []*TreeNode
	Err   error
}

func ParseTree(output string) ([]*TreeNode, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("empty tree output")
	}
	var nodes []*TreeNode
	if err := json.Unmarshal([]byte(output), &nodes); err != nil {
		return nil, fmt.Errorf("parse tree JSON: %w", err)
	}
	return nodes, nil
}

func findNodePath(nodes []*TreeNode, qname string) []*TreeNode {
	for _, n := range nodes {
		if n.QualifiedName == qname {
			return []*TreeNode{n}
		}
		if len(n.Children) > 0 {
			if sub := findNodePath(n.Children, qname); sub != nil {
				return append([]*TreeNode{n}, sub...)
			}
		}
	}
	return nil
}

func flattenQualifiedNames(nodes []*TreeNode) []flattenedItem {
	var items []flattenedItem
	for _, n := range nodes {
		qname := n.QualifiedName
		if qname == "" {
			qname = n.Label
		}
		items = append(items, flattenedItem{Label: n.Label, QualifiedName: qname, Type: n.Type})
		items = append(items, flattenQualifiedNames(n.Children)...)
	}
	return items
}
```

- [ ] **Step 2: Create `browser/model.go`**

The browser sub-model that owns preview engine, miller state, and tree data:

```go
package browser

import tea "github.com/charmbracelet/bubbletea"

type Model struct {
	Miller        MillerView
	AllNodes      []*TreeNode
	PreviewEngine *PreviewEngine
	MxcliPath     string
	ProjectPath   string
}

func NewModel(mxcliPath, projectPath string) Model { ... }
func (m Model) Init() tea.Cmd { ... }
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) { ... }
func (m Model) View(width, height int) string { ... }
```

- [ ] **Step 3: Create `browser/` copies**

Copy `miller.go`, `column.go`, `browserview.go`, `preview.go` → `browser/` with package change. Add `kernel.` prefix to all theme/types references.

- [ ] **Step 4: Update wiring**

In `app.go`, replace direct `BrowserView`/`MillerView` usage with `browser.Model`.

- [ ] **Step 5: Build and test**

```bash
make build && make test
```

---

### Phase 6: Extract `executor/` — MDL + Build + Run Views

**Files:**
- Create: `cmd/mxcli/tui/executor/model.go`
- Create: `cmd/mxcli/tui/executor/exec.go` (from execview.go)
- Create: `cmd/mxcli/tui/executor/build.go` (new — BuildView)
- Create: `cmd/mxcli/tui/executor/run.go` (new — RunView)
- Create: `cmd/mxcli/tui/executor/exec_test.go`
- Modify: remove `execview.go` from `tui/`

- [ ] **Step 1: Create `executor/model.go`**

The executor sub-model that owns execution state:

```go
package executor

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/task"
)

type Model struct {
	buildTask *task.BuildTask
	runTask   *task.RunTask
	execView  ExecViewState
}

type ExecuteMsg struct{ MDL string }
type BuildMsg struct{ ... }
type RunMsg struct{ ... }
```

- [ ] **Step 2: Create `executor/build.go`**

BuildView — shows live build progress:

```go
package executor

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/task"
)

type BuildView struct {
	task    *task.BuildTask
	events  []task.Event
	scroll  int
	width   int
	height  int
	running bool
}

func NewBuildView(t *task.BuildTask) BuildView {
	return BuildView{task: t, running: true}
}

func (bv BuildView) Mode() kernel.ViewMode { return kernel.ModeExec }

func (bv BuildView) Hints() []kernel.Hint {
	if bv.running {
		return []kernel.Hint{
			{Key: "c", Label: "cancel"},
			{Key: "j/k", Label: "scroll"},
		}
	}
	return []kernel.Hint{
		{Key: "q", Label: "close"},
		{Key: "j/k", Label: "scroll"},
	}
}

func (bv BuildView) StatusInfo() kernel.StatusInfo {
	state := "complete"
	if bv.running {
		state = "building"
	}
	return kernel.StatusInfo{
		Breadcrumb: []string{"Build"},
		Mode:       state,
	}
}

func (bv BuildView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return bv, func() tea.Msg { return kernel.PopViewMsg{} }
		case "c":
			if bv.running {
				bv.task.Cancel()
				bv.running = false
			}
		case "j":
			if bv.scroll < len(bv.events)-1 {
				bv.scroll++
			}
		case "k":
			if bv.scroll > 0 {
				bv.scroll--
			}
		}
	case task.Event:
		bv.events = append(bv.events, msg)
		bv.scroll = len(bv.events) // auto-scroll to latest
		if msg.State == task.StateCompleted || msg.State == task.StateFailed || msg.State == task.StateCancelled {
			bv.running = false
		}
	}
	return bv, nil
}

func (bv BuildView) Render(width, height int) string {
	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(kernel.AccentColor).Render("Build")
	sb.WriteString(title + "\n\n")

	// Show events (scroll window)
	visibleH := height - 4
	start := max(0, bv.scroll-visibleH+1)
	end := min(len(bv.events), start+visibleH)

	for _, ev := range bv.events[start:end] {
		line := task.RenderProgress(ev, width-4)
		if ev.State == task.StateFailed {
			line = kernel.CheckErrorStyle.Render("✗ " + ev.Message)
		} else if ev.State == task.StateCompleted {
			line = kernel.CheckPassStyle.Render("✓ " + ev.Message)
		}
		sb.WriteString(line + "\n")
	}

	if bv.running {
		sb.WriteString(kernel.LoadingStyle.Render("⟳ Building... (press c to cancel)"))
	} else if len(bv.events) > 0 {
		last := bv.events[len(bv.events)-1]
		if last.State == task.StateCompleted {
			sb.WriteString(kernel.CheckPassStyle.Render("\n✓ Build complete"))
		} else if last.State == task.StateFailed {
			sb.WriteString(kernel.CheckErrorStyle.Render("\n✗ Build failed"))
		}
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}
```

- [ ] **Step 3: Create `executor/run.go`**

RunView — shows live runtime output below:

```go
package executor

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/task"
)

type RunView struct {
	task     *task.RunTask
	events   []task.Event
	logLines []string
	scroll   int
	width    int
	height   int
	running  bool
}

func NewRunView(t *task.RunTask) RunView {
	return RunView{task: t, running: true}
}

func (rv RunView) Mode() kernel.ViewMode { return kernel.ModeExec }

func (rv RunView) Hints() []kernel.Hint {
	if rv.running {
		return []kernel.Hint{
			{Key: "c", Label: "stop"},
			{Key: "j/k", Label: "scroll"},
		}
	}
	return []kernel.Hint{
		{Key: "q", Label: "close"},
		{Key: "j/k", Label: "scroll"},
	}
}

func (rv RunView) StatusInfo() kernel.StatusInfo {
	s := "stopped"
	if rv.running {
		s = fmt.Sprintf("running (%d lines)", len(rv.logLines))
	}
	return kernel.StatusInfo{
		Breadcrumb: []string{"Run"},
		Position:   s,
		Mode:       "RUN",
	}
}

func (rv RunView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if !rv.running {
				return rv, func() tea.Msg { return kernel.PopViewMsg{} }
			}
		case "c":
			if rv.running {
				rv.task.Cancel()
				rv.running = false
			}
		case "j":
			if rv.scroll < len(rv.logLines)-1 {
				rv.scroll++
			}
		case "k":
			if rv.scroll > 0 {
				rv.scroll--
			}
		}
	case task.Event:
		rv.events = append(rv.events, msg)
		rv.logLines = append(rv.logLines, fmt.Sprintf("[%s] %s", msg.Phase, msg.Message))
		rv.scroll = len(rv.logLines) - 1
		if msg.State == task.StateCompleted || msg.State == task.StateFailed || msg.State == task.StateCancelled {
			rv.running = false
		}
	}
	return rv, nil
}

func (rv RunView) Render(width, height int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(kernel.AccentColor).Render("Run")

	var sb strings.Builder
	sb.WriteString(title + "\n\n")

	visibleH := height - 4
	start := max(0, rv.scroll-visibleH+1)
	end := min(len(rv.logLines), start+visibleH)

	for i := start; i < end; i++ {
		line := rv.logLines[i]
		if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			line = kernel.CheckErrorStyle.Render(line)
		}
		sb.WriteString(line + "\n")
	}

	if rv.running {
		sb.WriteString(kernel.LoadingStyle.Render("\n⟳ Runtime running... (press c to stop)"))
	} else if len(rv.events) > 0 {
		last := rv.events[len(rv.events)-1]
		switch last.State {
		case task.StateCompleted:
			sb.WriteString(kernel.CheckPassStyle.Render("\n⏹ Runtime stopped"))
		case task.StateFailed:
			sb.WriteString(kernel.CheckErrorStyle.Render("\n✗ " + last.Message))
		case task.StateCancelled:
			sb.WriteString(kernel.CheckWarnStyle.Render("\n⏸ Cancelled"))
		}
	}

	elapsed := time.Duration(len(rv.events)) * time.Second / 2 // rough estimate
	if elapsed > 0 {
		sb.WriteString(kernel.MutedStyle.Render(fmt.Sprintf("\nUptime: ~%s", elapsed.Round(time.Second))))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}
```

- [ ] **Step 4: Move execview.go → executor/exec.go**

Copy `execview.go` → `executor/exec.go`, change to `package executor`.

- [ ] **Step 5: Update wiring in app.go**

```go
// In app.go — remove direct exec/build/run state
// Replace with:
type ExecutorModel struct {
    model executor.Model
}
```

- [ ] **Step 6: Build and test**

```bash
make build && make test
```

---

### Phase 7: Simplify App

**Files:**
- Modify: `cmd/mxcli/tui/app.go`
- Modify: `cmd/mxcli/tui/app_update.go`
- Modify: `cmd/mxcli/tui/app_view.go`
- Modify: `cmd/mxcli/tui/app_keys.go`

- [ ] **Step 1: Remove extracted fields from App struct**

Remove these fields from `App`:
- `previewEngine` — owned by `browser.Model`
- `checkErrors`, `checkNavActive`, `checkNavIndex`, `checkNavLocations`, `checkRunning` — owned by `diagnostics.Model`
- `agentListener`, `agentAutoProceed`, `agentPending`, `agentCheckCh`, `agentCheckReqID`, `agentExecCtx` — owned by `agent.Model`

Replace with sub-models:

```go
type App struct {
	// Existing
	tabs      []Tab
	activeTab int
	nextTabID int
	width     int
	height    int
	mxcliPath string
	views     ViewStack
	showHelp  bool
	picker    *PickerModel

	// Sub-models
	browserModel  browser.Model
	executorModel executor.Model
	agentModel    agent.Model
	diagModel     diagnostics.Model

	// Chrome
	tabBar    chrome.TabBar
	hintBar   chrome.HintBar
	statusBar chrome.StatusBar
}
```

- [ ] **Step 2: Simplify `app_update.go`**

Replace the giant switch with delegation to sub-models:

```go
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case kernel.PushViewMsg:
		a.views.Push(msg.View)
		return a, nil
	case kernel.PopViewMsg:
		a.views.Pop()
		return a, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		// Delegate to sub-models
		var cmd tea.Cmd
		updated, cmd := a.views.Active().Update(msg)
		a.views.SetActive(updated)
		return a, cmd
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	}
	// Delegate unknown messages to active view
	updated, cmd := a.views.Active().Update(msg)
	a.views.SetActive(updated)
	return a, cmd
}
```

- [ ] **Step 3: Simplify `app_view.go`**

Remove knowledge of check badges, agent badges from View(). These are pushed from sub-models as state.

- [ ] **Step 4: Build and test**

```bash
make build && make test
```

---

### Phase 8: Register `:build` and `:run` Commands

**Files:**
- Modify: `cmd/mxcli/tui/commandpalette.go`
- Modify: `cmd/mxcli/tui/app_keys.go`

- [ ] **Step 1: Add palette entries**

```go
{Name: "Build Project", Key: "b", Category: "Build"},
{Name: "Run Project",   Key: "R", Category: "Build"}, // palette-only
```

- [ ] **Step 2: Re-bind BSON dump from `b` to `B`**

In `browserview.go` `handleKey()`, change:
```go
case "b": → case "B":
```

- [ ] **Step 3: Wire `b` key to BuildView**

In `app_keys.go`, add case for `"b"`:

```go
case "b":
	bt := task.NewBuildTask(context.Background(), task.BuildOptions{
		ProjectPath: a.activeTabProjectPath(),
	})
	bv := executor.NewBuildView(bt)
	a.views.Push(bv)
	return bt.Start(context.Background())
```

- [ ] **Step 4: Wire `:run` to RunView**

In `app_update.go`, handle `ExecutorRunMsg`:

```go
case executor.RunMsg:
	rt := task.NewRunTask(context.Background(), task.RunOptions{
		PadDir:  filepath.Join(filepath.Dir(a.activeTabProjectPath()), ".docker", "build"),
		CmdHint: "-p " + a.activeTabProjectPath(),
	})
	rv := executor.NewRunView(rt)
	a.views.Push(rv)
	return rt.Start(context.Background())
```

- [ ] **Step 5: Build and test**

```bash
make build && make test
```

---

### Phase 9: Final Validation

- [ ] **Step 1: Full build**

```bash
make build
```

- [ ] **Step 2: Full test**

```bash
make test
```

- [ ] **Step 3: Lint check**

```bash
make vet
```

- [ ] **Step 4: Manual smoke test**

```bash
# Launch TUI with a test project
./bin/mxcli tui -p testdata/corpus-b/app.mpr
```

Verify:
- Tree loads and navigates correctly
- `x` opens ExecView
- `:build` shows build view
- `:run` shows run view (will fail gracefully without PAD)
- `q` quits cleanly
