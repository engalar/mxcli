# TUI Yazi-Style Refactor Design

Date: 2026-03-23

## Goal

Refactor the mxcli TUI from the current 2-panel + overlay model to a yazi-inspired UX with:
- Miller columns (parent / current / preview)
- Tab support (same-project multi-location + cross-project)
- Async preview engine (cursor-move triggers, with cancel + cache)
- Borderless minimalist visual style

## Layout

```
 [1] Main > Entities  [2] Admin                    ← Tab bar
                                                    ← separator
 Entities       │ ● Customer      │ Name : String   ← 3 columns
 Microflows     │   Order         │ Email : String
 Pages          │   Product       │ Age : Integer
 Enumerations   │   Invoice       │
                │                 │ 4 attributes
                │                 │ 2 associations
                                                    ← separator
 App > Main > Entities                    3/4  MDL  ← Status bar
```

### Column Ratios

- Default: parent 20% : current 30% : preview 50%
- Narrow terminals (< 80 cols): hide parent, degrade to 2-column (40:60)
- Zen mode (`z`): selected column 100%

### Column Separators

Single `│` character with dim styling. No box borders around panels.

## Architecture

### File Structure

```
tui/
├── app.go              # Root Model: tab management, global keys, window size
├── tab.go              # Tab struct: holds MillerView + independent nav state
├── miller.go           # Miller 3-column layout: parent/current/preview coordination
├── column.go           # Single column component: list rendering, cursor, filter
├── preview.go          # Preview engine: async loading, content cache, cancellation
├── statusbar.go        # Bottom status bar: breadcrumb + position + mode
├── tabbar.go           # Top tab bar rendering
├── keys.go             # Centralized key bindings
├── hintbar.go          # Context-sensitive key hint bar (HUD)
├── styles.go           # Global style constants (rewritten for borderless)
├── overlay.go          # Fullscreen overlay (retained, minor adjustments)
├── compare.go          # Compare view (retained)
├── contentview.go      # Scrollable text view (retained)
├── highlight.go        # Syntax highlighting (retained)
├── clipboard.go        # Clipboard (retained)
├── picker.go           # Project selector (retained)
├── history.go          # Project history (retained)
├── runner.go           # Subprocess execution (retained)
```

**Deleted:**
- `panels/` subpackage (merged into `column.go`)
- `layout.go` (replaced by Miller column layout logic in `miller.go`)
- `model.go` (split into `app.go` + `tab.go` + `miller.go`)

### Data Flow

```
Cursor move → Column sends CursorChangedMsg{node}
  → MillerView receives:
      1. Updates parent column (show siblings of current column's parent)
      2. If selected node has children → preview column shows child list (sync)
      3. If selected node is leaf → preview engine triggers async load
  → PreviewEngine:
      1. Cancel previous in-flight request
      2. Check cache → hit: render immediately
      3. Miss: start goroutine (mxcli describe/bson dump)
      4. On completion: send PreviewReadyMsg → render to preview column
```

## Core Types

### App (Root Model)

```go
type App struct {
    tabs       []Tab
    activeTab  int
    width      int
    height     int
    mxcliPath  string

    // Fullscreen modes (shared across tabs)
    overlay    Overlay
    compare    CompareView
    showHelp   bool
    picker     *PickerModel  // nil when not picking
}
```

### Tab

```go
type Tab struct {
    ID          int
    Label       string      // Display name (module name or project name)
    ProjectPath string      // MPR path
    Miller      MillerView  // Independent 3-column view
    NavStack    []NavState  // Navigation history stack
    NavIndex    int         // Current position in nav stack
    AllNodes    []*TreeNode // Flattened tree for this project
}

type NavState struct {
    Path       []string    // Breadcrumb segments
    ParentNode *TreeNode   // Node whose children fill the current column
    CursorIdx  int         // Cursor position in current column
}
```

### MillerView

```go
type MillerView struct {
    parent   Column       // Left: parent's siblings
    current  Column       // Center: current level items
    preview  PreviewPane  // Right: children or content preview
    focus    ColumnFocus  // Which column has input focus (always current)
}

type PreviewPane struct {
    // When showing children
    childColumn *Column

    // When showing content
    content     string
    highlighted string
    mode        PreviewMode  // MDL or NDSL
    loading     bool
}
```

### Column

```go
type Column struct {
    items        []ColumnItem
    cursor       int
    scrollOffset int
    filter       FilterState
    width        int
    height       int
    title        string
}

type ColumnItem struct {
    Label         string
    Icon          string
    Type          string      // Mendix node type
    QualifiedName string
    HasChildren   bool
    Node          *TreeNode
}

type FilterState struct {
    active bool
    input  textinput.Model
    query  string
    matches []int  // indices into items
}
```

### PreviewEngine

```go
type PreviewEngine struct {
    cache      map[string]PreviewResult
    cancelFunc context.CancelFunc
    mxcliPath  string
    projectPath string
}

type PreviewResult struct {
    Content       string
    HighlightType string  // "mdl" / "ndsl" / "plain"
}
```

## Tab System

### Key Bindings

| Key | Action |
|-----|--------|
| `1-9` | Switch to tab N |
| `t` | New tab (same project, current location) |
| `T` | New tab with project picker (cross-project) |
| `W` | Close current tab (min 1 tab) |
| `[` / `]` | Previous / next tab |

### Tab Bar Rendering

```
 [1] Main > Entities  [2] Admin  [3] OtherProject
   ▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔
```

- Active tab: Bold + Underline
- Inactive tabs: Dim
- Mouse click to switch

### Tab Lifecycle

1. **New tab (same project)**: Clone current tab's project path + tree, start at same location
2. **New tab (cross-project)**: Open picker, load new project tree on selection
3. **Close tab**: Remove from list, activate adjacent tab
4. **Tab label**: Auto-derived from current breadcrumb (deepest 2 levels)

## Async Preview Engine

### Strategy

Immediate trigger on cursor change, cancel-previous pattern:

1. `CursorChangedMsg` arrives
2. Check node type:
   - **Directory node** (has children) → sync render child list as Column items
   - **Leaf node** → check cache:
     - Hit → render cached content
     - Miss → cancel previous context → create new context → spawn goroutine
3. Goroutine runs `mxcli describe` (MDL) or `mxcli bson dump --format ndsl` (NDSL)
4. On completion, if context not cancelled → send `PreviewReadyMsg`
5. `PreviewReadyMsg` updates preview pane + adds to cache

### Cache Policy

- Key: `"{projectPath}:{type}:{qualifiedName}:{mode}"`
- No TTL (cache lives until `r` refresh or tab close)
- Refresh (`r`) clears all cache entries for the active tab's project

### Loading State

While loading, preview pane shows:
```
 Loading...
```
(Italic + dim styling)

## Visual Style

### Color Scheme (Terminal-Theme Adaptive)

| Element | Style |
|---------|-------|
| Active tab | Bold + Underline |
| Inactive tab | Dim |
| Column title | Bold |
| Selected item | Reverse video |
| Directory node | Bold + type icon |
| Leaf node | Normal + type icon |
| Column separator `│` | Dim |
| Breadcrumb | Dim, current segment Normal |
| Loading indicator | Italic + Dim |
| Position info | Dim |
| Preview mode (MDL/NDSL) | Bold |

### Status Bar

```
 App > Main > Entities                    3/4  MDL
 └── breadcrumb (dim)                     └── position + mode
```

## Key Bindings (Complete)

| Key | Context | Action |
|-----|---------|--------|
| `j/k` `↑/↓` | List | Move cursor up/down |
| `h/←` | List | Go back (nav stack pop) |
| `l/→` Enter | List | Drill in / open leaf fullscreen |
| `g/G` | List | Jump to first/last |
| `/` | List | Start filter |
| `Esc` | Filter | Exit filter |
| `Tab` | Preview | Toggle MDL ↔ NDSL |
| `z` | Global | Zen mode (current column fullscreen) |
| `c` | Global | Open compare view |
| `d` | Global | Open diagram in browser |
| `y` | Global | Copy preview content to clipboard |
| `r` | Global | Refresh project tree + clear cache |
| `?` | Global | Toggle help |
| `q` | Global | Quit |
| `1-9` | Global | Switch to tab N |
| `t` | Global | New tab (same project) |
| `T` | Global | New tab (pick project) |
| `W` | Global | Close tab |
| `[` `]` | Global | Previous/next tab |

## Key Hint Bar (HUD)

A context-sensitive hint bar sits above the status bar, showing available actions for the current context:

```
 h:back l:open /:filter z:zen c:compare Tab:mdl/ndsl
 App > Main > Entities                    3/4  MDL
```

### Context-Sensitive Hints

The hint bar adapts to the current state:

| Context | Hints Shown |
|---------|-------------|
| List browsing | `h:back l:open /:filter z:zen c:compare y:copy ?:help` |
| Filter active | `Esc:cancel Enter:confirm` |
| Overlay open | `j/k:scroll /:search y:copy Tab:mdl/ndsl q:close` |
| Compare view | `1/2/3:mode s:sync /:search q:close` |
| Zen mode | `z:exit zen` (prepended) |

### Styling

- Key character: Bold
- Description: Dim
- Separator between groups: `  ` (double space)
- Truncate from right if terminal is too narrow, always keep first 3 hints

### Implementation

New file `tui/hintbar.go`:

```go
type HintBar struct {
    hints []Hint
    width int
}

type Hint struct {
    Key   string  // "h", "l", "/", "Tab"
    Label string  // "back", "open", "filter"
}
```

The `App` model sets the hint bar content whenever the context changes (focus change, overlay open/close, filter toggle).

## Overlay & Compare (Retained)

The existing overlay and compare view functionality is retained with minor adjustments:

- **Overlay**: Triggered by `l`/Enter on a leaf node (instead of `b`/`m`). Shows full content with syntax highlighting, search, clipboard. Tab switches MDL↔NDSL.
- **Compare**: Triggered by `c`. Retained as-is with fuzzy picker and sync scroll.

## Responsive Behavior

| Terminal Width | Layout |
|----------------|--------|
| ≥ 120 cols | Full 3-column (20:30:50) |
| 80-119 cols | 3-column (15:35:50) |
| < 80 cols | 2-column, hide parent (40:60) |
| < 50 cols | 1-column, preview in overlay only |

## Migration Strategy

1. Build new components (`column.go`, `miller.go`, `tab.go`, `app.go`) alongside existing code
2. Wire up the new `App` model as the root
3. Integrate existing `overlay.go`, `compare.go`, `contentview.go`, `highlight.go` into new structure
4. Delete old `model.go`, `panels/`, `layout.go`
5. Update `cmd/mxcli/tui.go` to use new `App` model

## Non-Goals

- Plugin system (yazi has plugins, we don't need them)
- Image preview (not relevant for Mendix projects)
- File operations (rename, delete, move — mxcli TUI is read-only browser)
- Bulk selection / marks
