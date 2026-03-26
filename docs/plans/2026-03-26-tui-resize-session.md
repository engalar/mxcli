# TUI Resize Fix + Session Restore Design

## Part 1: Window Resize Mouse Coordinate Fix

### Problem

After terminal window resize, mouse click coordinates drift — all interactive areas (tab bar, miller columns, status bar) respond to wrong positions. The terminal reports mouse positions based on stale coordinate mapping.

### Root Cause

bubbletea's mouse tracking mode is not automatically reset when the terminal window size changes. Some terminals (tmux, screen) need the mouse tracking ANSI sequences re-sent to recalibrate.

Additionally, the recently added LLM anchor line may have changed the Y-offset calculation without updating mouse coordinate translation.

### Fix

1. In `WindowSizeMsg` handler, reset mouse tracking:
```go
case tea.WindowSizeMsg:
    a.width = msg.Width
    a.height = msg.Height
    // Reset mouse tracking to recalibrate coordinates
    return a, tea.Batch(
        tea.DisableMouse,
        tea.EnableMouseCellMotion,
    )
```

2. Audit all Y-coordinate offsets in mouse handling:
   - `msg.Y - 1` offset for tab bar — verify against actual chrome height
   - If LLM anchor line is rendered, add +1 to Y offset
   - Status bar hit test Y coordinate check

### Files

- `cmd/mxcli/tui/app.go` — WindowSizeMsg handler, mouse Y-offset audit

## Part 2: Session Restore (-c flag)

### Overview

Save TUI state on exit, restore on startup with `-c` flag. Enables quick restart-verify cycles during development.

### Storage

File: `~/.mxcli/tui-session.json` (overwritten on each exit)

### Session State Schema

```json
{
  "version": 1,
  "timestamp": "2026-03-26T01:30:00Z",
  "tabs": [
    {
      "projectPath": "/path/to/App.mpr",
      "millerPath": ["Project", "MyFirstModule", "Pages"],
      "selectedNode": "P_ComboBox_Enum",
      "previewMode": "MDL"
    }
  ],
  "activeTab": 0,
  "viewStack": [
    {"type": "browser"},
    {"type": "overlay", "title": "mx check", "filter": "all"}
  ],
  "checkNavActive": true,
  "checkNavIndex": 1
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `version` | int | Schema version for forward compat (currently 1) |
| `timestamp` | string | ISO 8601 save time |
| `tabs` | []TabState | All open tabs |
| `tabs[].projectPath` | string | Absolute path to .mpr file |
| `tabs[].millerPath` | []string | Navigation breadcrumb path |
| `tabs[].selectedNode` | string | Currently selected node name |
| `tabs[].previewMode` | string | "MDL" or "NDSL" |
| `activeTab` | int | Index of active tab |
| `viewStack` | []ViewState | Stack of open views (browser at bottom) |
| `viewStack[].type` | string | "browser", "overlay", "compare", etc. |
| `viewStack[].title` | string | Overlay title (for overlay type) |
| `viewStack[].filter` | string | Check filter (for check overlay) |
| `checkNavActive` | bool | Whether check nav mode is active |
| `checkNavIndex` | int | Current nav position |

### Edge Cases

| Scenario | Handling |
|----------|----------|
| Project file deleted | Skip tab, log warning, open remaining tabs |
| Selected node deleted | Fall back to nearest existing parent in miller path |
| Miller path partially invalid | Expand to last valid level, select first child |
| Overlay data unavailable | Skip overlay restore, show browser only |
| Check results stale | Re-run check on restore, don't restore old results |
| All tabs invalid | Normal startup (no restore) |
| Session file corrupt/missing | Ignore, normal startup |
| Schema version mismatch | Ignore if version > current, attempt if <= current |

### Implementation

**New file:** `cmd/mxcli/tui/session.go`
- `TUISession` struct matching JSON schema
- `TabState`, `ViewState` sub-structs
- `SaveSession(app *App) error` — serialize current state to file
- `LoadSession() (*TUISession, error)` — read and parse session file
- `sessionFilePath() string` — returns `~/.mxcli/tui-session.json`

**Modified files:**

`cmd/mxcli/tui/app.go`:
- On quit (case "q"): call `SaveSession(a)` before returning `tea.Quit`
- New `RestoreSession(session *TUISession)` method on App
  - Opens each tab's project, navigates miller path, selects node
  - Restores view stack (overlay, etc.) with fallbacks
- Accept `*TUISession` in App constructor or Init

`cmd/mxcli/cmd_tui.go`:
- Add `-c` / `--continue` flag
- When set, call `LoadSession()` and pass to App

`cmd/mxcli/tui/browserview.go` / `cmd/mxcli/tui/miller.go`:
- May need `NavigateToPath(path []string) bool` method
- Reuse existing `navigateToNode` with path-walking

### Task Order

1. Part 1: Mouse fix (small, isolated)
2. Part 2a: Session save/load infrastructure (session.go)
3. Part 2b: App integration (save on quit, restore on start)
4. Part 2c: CLI flag + edge case testing
