# TUI Enhancement Design

**Date**: 2026-03-22
**Status**: Draft

## Context

The current TUI (`mxcli tui`) uses bubbles `list.Model` for panels with page-based scrolling, fixed 3-panel layout, no mouse support, no syntax highlighting, and no BSON/NDSL integration. This design overhauls the TUI for a more fluid, interactive experience.

## Requirements

1. **Unified ScrollList** — Replace `list.Model` with custom component using picker-style smooth scrolling + visual scrollbar
2. **Breadcrumb navigation** — Top of each panel shows navigation path, clickable
3. **Z mode** — Press `z` to zoom current panel to fullscreen
4. **Progressive expansion** — Start with 1 panel, expand as user drills in
5. **Panel 3 summary mode** — Show metadata summary, Enter opens fullscreen overlay
6. **Mouse support** — Click selection, scroll wheel, breadcrumb click
7. **MDL/SQL/NDSL syntax highlighting** — Using alecthomas/chroma
8. **BSON/NDSL commands** — Cmdbar verbs with multi-level completion

## Architecture

### New Files

| File | Lines | Responsibility |
|------|-------|----------------|
| `tui/panels/scrolllist.go` | ~350 | Reusable scrollable list: cursor, scrollOffset, scrollbar, filter, mouse |
| `tui/panels/breadcrumb.go` | ~80 | Breadcrumb path display and click-to-navigate |
| `tui/highlight.go` | ~120 | Chroma-based syntax highlighting (MDL/SQL/NDSL) |
| `tui/overlay.go` | ~150 | Fullscreen overlay with scrollable viewport |

### Modified Files

| File | Changes |
|------|---------|
| `tui/panels/modules.go` | Replace `list.Model` → `ScrollList`, add breadcrumb + nav stack |
| `tui/panels/elements.go` | Same refactor as modules |
| `tui/layout.go` | Dynamic panel widths, `PanelRect` geometry for mouse hit testing |
| `tui/model.go` | Visibility state, zen mode, mouse routing, overlay integration, BSON dispatch |
| `tui/styles.go` | Remove duplicate `typeIconMap`, add scrollbar/overlay styles |
| `tui/cmdbar.go` | Multi-level completion tree, qualified name completion |

### New Dependency

- `github.com/alecthomas/chroma/v2`

## Phase 1: ScrollList + Breadcrumb (Foundation)

### ScrollList (`tui/panels/scrolllist.go`)

```go
type ScrollListItem interface {
    Label() string
    Icon() string
    Description() string
    FilterValue() string
}

type ScrollList struct {
    items         []ScrollListItem
    filteredItems []int              // indices into items (nil = no filter)
    cursor        int
    scrollOffset  int
    filterInput   textinput.Model
    filterActive  bool
    width, height int
    focused       bool
    headerHeight  int               // reserved for breadcrumb
}
```

**Scrolling**: `scrollOffset + maxVisible` window. Cursor moves smoothly, scrollOffset follows.

**Scrollbar**: Right-side vertical track (`│`) with thumb (`█`). Position = `scrollOffset / (total - maxVisible) * trackHeight`.

**Mouse**: `MouseWheelUp/Down` adjusts scrollOffset. `MouseActionPress` computes `clickedIndex = scrollOffset + (Y - topOffset)`.

**Filter**: `/` activates textinput, real-time substring filter on `FilterValue()`. Esc exits.

### Breadcrumb (`tui/panels/breadcrumb.go`)

```go
type BreadcrumbSegment struct {
    Label string
}

type Breadcrumb struct {
    segments []BreadcrumbSegment
    width    int
}
```

Methods: `Push()`, `PopTo(level)`, `Depth()`, `View()` (renders `A > B > C`), `ClickedSegment(x int) int`.

### Panel Refactor

Each panel (modules, elements) maintains:
- `ScrollList` instead of `list.Model`
- `Breadcrumb` for navigation path
- `navigationStack [][]*TreeNode` for drill-in/back

## Phase 2: Progressive Expansion + Z Mode

### Dynamic Layout (`tui/layout.go`)

```go
type PanelVisibility int
const (
    ShowOnePanel    PanelVisibility = iota  // modules only, 100%
    ShowTwoPanels                           // modules 35% + elements 65%
    ShowThreePanels                         // 20% + 30% + 50%
    ShowZoomed                              // zoomed panel 100%
)

type PanelRect struct {
    X, Y, Width, Height int
    Visible             bool
}
```

### Visibility State Machine

- Start: `ShowOnePanel`
- Select module + right/enter → `ShowTwoPanels`
- Select element → `ShowThreePanels`
- Left from elements (empty stack) → `ShowOnePanel`
- Left from preview → `ShowTwoPanels`

### Z Mode

- `z` toggles between `ShowZoomed` and previous visibility
- Remembers `zenPrevFocus` and `zenPrevVisibility` for restore
- `Esc` also exits zen mode

## Phase 3: Mouse Support

Root model translates `tea.MouseMsg` coordinates using `PanelRect`:

```go
case tea.MouseMsg:
    for i, rect := range m.panelLayout {
        if rect.contains(msg.X, msg.Y) {
            localMsg := translateMouse(msg, rect)
            m.focus = Focus(i)
            // forward to panel
        }
    }
```

ScrollList handles translated coordinates internally. Breadcrumb click detected by checking `localMsg.Y < headerHeight`.

## Phase 4: Summary + Overlay + Highlighting

### Preview Summary Mode

Panel 3 shows compact metadata card:
```
Type:    Entity
Module:  MyModule
Name:    Customer
Attrs:   5  Assocs: 2
[Enter] view details
```

`SetContent()` stores both `summaryContent` (panel) and `fullContent` (overlay).

### Fullscreen Overlay (`tui/overlay.go`)

- Reuses `viewport.Model` for scrollable content
- `lipgloss.Place` for centering
- Title bar + content + bottom hints
- Serves: detail view, BSON dump, NDSL output

### Syntax Highlighting (`tui/highlight.go`)

- `alecthomas/chroma/v2` with `terminal256` formatter + `monokai` style
- SQL lexer as MDL base
- Custom NDSL lexer (regex: field paths, type annotations, values)
- Functions: `HighlightMDL()`, `HighlightSQL()`, `HighlightNDSL()`, `DetectAndHighlight()`

## Phase 5: BSON/NDSL Commands

### Multi-level Cmdbar

```go
type cmdDef struct {
    name     string
    children []cmdDef
}
```

Commands: `bson dump <name>`, `bson compare <name>`, `ndsl <name>`

Completion levels:
1. Command name (bson, ndsl, callers, ...)
2. Subcommand (dump, compare)
3. Qualified name (from flattened tree nodes)

Results → `HighlightNDSL()` → `OpenOverlayMsg` → fullscreen overlay

## Phase Dependency

```
Phase 1 (ScrollList + Breadcrumb)
  ├──→ Phase 2 (Layout + Z Mode)
  │       └──→ Phase 3 (Mouse)
  └──→ Phase 4 (Summary + Overlay + Highlighting)
          └──→ Phase 5 (BSON/NDSL Commands)
```

Phases 2 and 4 can proceed in parallel after Phase 1.

## Verification

After each phase:
1. `make build` — compiles cleanly
2. `./bin/mxcli tui -p /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr` — manual testing
3. Verify scrolling, mouse, breadcrumb, overlay, highlighting visually
4. Run `make test` for any unit tests added
