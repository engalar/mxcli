# TUI mx check Enhancement Design

## Overview

Enhance the TUI mx check overlay with error grouping, navigation, expanded diagnostics, and LLM-friendly structured output.

## Part 1: Error Grouping + Deduplication

### Problem

Repeated errors (same code + element-id) fill the screen. 8 CE1613 errors render as 8 separate blocks.

### Design

Group by error code, deduplicate by element-id within each group.

```
mx check Results
● 8 errors

CE1613 — The selected association/attribute no longer exists
  MyFirstModule.P_ComboBox_Enum (Page)
    > Property 'Association' of combo box 'cmbPriority'
  MyFirstModule.P_ComboBox_Assoc (Page) (x7)
    > Property 'Attribute' of combo box 'cmbCategory'
```

### Implementation

- New types: `CheckGroup{Code, Severity, Message, Items}` and `CheckGroupItem{DocLocation, ElementName, Count}`
- `groupCheckErrors([]CheckError) []CheckGroup`: groups by Code, deduplicates by element-id, counts occurrences
- `renderCheckResults` renders grouped output instead of flat list
- `formatCheckBadge` unchanged (counts raw errors by severity)
- Group title: first entry's message (or common prefix if messages differ within a code)

### Files

- `cmd/mxcli/tui/checker.go` — add grouping types and logic, update renderCheckResults
- `cmd/mxcli/tui/checker_test.go` — test grouping, deduplication, rendering

## Part 2: Error Navigation

### Design

1. Check overlay becomes a selectable list — `j/k` moves cursor between error locations
2. `Enter` on a location → closes overlay → tree navigates to the document
3. App enters **check nav mode**: status bar shows current error + `]e`/`[e` hints
4. `]e` jumps to next error document, `[e` jumps to previous
5. `Esc` or any non-nav key exits check nav mode
6. `!` reopens overlay at any time

### Implementation

- Add `checkNavActive bool`, `checkNavIndex int`, `checkNavLocations []CheckNavLocation` to App
- `CheckNavLocation{ModuleName, DocumentName, Code, Message}` — unique documents extracted from grouped errors
- `NavigateToDocMsg{ModuleName, DocumentName}` — sent by overlay Enter, received by App
- App handles NavigateToDocMsg: search tree for matching node (by module + document name), expand path, select node
- `]e`/`[e` keys in browser mode (when checkNavActive): increment/decrement checkNavIndex, send NavigateToDocMsg
- Status bar in check nav mode: `[2/5] CE1613: MyFirstModule.P_ComboBox_Enum  ]e next  [e prev`
- Check overlay needs cursor state: `selectedIndex int`, highlight selected row, Enter emits NavigateToDocMsg

### Files

- `cmd/mxcli/tui/checker.go` — add CheckNavLocation type, extraction function
- `cmd/mxcli/tui/app.go` — add check nav state, handle NavigateToDocMsg, ]e/[e keys, status bar update
- `cmd/mxcli/tui/overlayview.go` — add selectable mode with cursor for check overlay
- `cmd/mxcli/tui/browserview.go` — add NavigateToNode method for tree navigation
- `cmd/mxcli/tui/hintbar.go` — no changes (hints come from overlay/app)
- `cmd/mxcli/tui/help.go` — document ]e/[e keys

## Part 3: Warning + Deprecation Support

### Design

Run `mx check -j -w -d` to capture all diagnostic types. Add Tab filtering in check overlay.

```
mx check Results  [All: 8E 2W 1D]

Tab cycles: All → Errors → Warnings → Deprecations → All
```

### Implementation

- `runMxCheck`: add `-w`, `-d` flags to mx command
- `mxCheckJSON`: add `Deprecations []mxCheckEntry` field
- `CheckError.Severity`: extend to three values — `ERROR`, `WARNING`, `DEPRECATION`
- Check overlay: `checkFilter` state (`all`/`error`/`warning`/`deprecation`), Tab key cycles filter
- `renderCheckResults` accepts filter parameter, only renders matching groups
- Filter indicator in overlay title bar: `[All: 8E 2W 1D]` or `[Errors: 8]`
- `formatCheckBadge`: update to show `✗ 8E 2W 1D`
- When overlay is refreshable (not switchable), Tab is repurposed for filter cycling

### Files

- `cmd/mxcli/tui/checker.go` — update runMxCheck flags, parse deprecations, add filter logic
- `cmd/mxcli/tui/checker_test.go` — test deprecation parsing, filter rendering
- `cmd/mxcli/tui/overlayview.go` — add checkFilter state, Tab handling for non-switchable overlays

## Part 4: LLM Anchor Structured Output

### Design

Embed faint structured anchors in overlay rendering for LLM consumption via screenshots or clipboard copy.

```
[mxcli:check] errors=8 warnings=2 deprecations=1
[mxcli:check:CE1613] severity=ERROR count=6 doc=MyFirstModule.P_ComboBox_Assoc type=Page element=combo_box.cmbCategory
[mxcli:check:CE1613] severity=ERROR count=1 doc=MyFirstModule.P_ComboBox_Enum type=Page element=combo_box.cmbPriority
[mxcli:check:CW0001] severity=WARNING count=2 doc=MyFirstModule.DoSomething type=Microflow element=variable.$var
```

### Implementation

- Replace current `[mxcli:overlay] mx check MDL` anchor with `[mxcli:check]` summary
- Add per-group-item anchors with key=value pairs
- Use `Faint(true)` styling — nearly invisible in terminal but preserved in copy/screenshot
- `PlainText()` (for clipboard via `y`) includes anchor text
- Anchors rendered before visible content in overlay

### Files

- `cmd/mxcli/tui/overlayview.go` — update Render() for check overlay anchor format
- `cmd/mxcli/tui/checker.go` — add `renderCheckAnchors([]CheckGroup) string` function

## Task Order

1. Part 1: Error grouping + dedup (foundation for all other parts)
2. Part 3: Warning/deprecation support (extends data model before navigation uses it)
3. Part 4: LLM anchors (uses grouped data, no interaction changes)
4. Part 2: Error navigation (most complex, depends on grouping being stable)

## Dependencies

- No new Go dependencies required
- `element-id` and `unit-id` fields need to be added to `mxCheckLocation` struct for dedup and future navigation
