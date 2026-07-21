# TUI Leader Key Architecture Design

## Context

The TUI key dispatch system is spread across three layers (`app_keys.go` 18 cases, `browserview.go` 7+10 cases, `app_update.go` implicit routing) with no single source of truth. Adding new features requires competing for scarce single-key slots. The lazyvim-style leader key system solves this by organizing all actions under `Space`-prefixed chords: `Space b b` = Build, `Space c c` = Show check results.

## Architecture

```
tui/who/ (Which-Key Domain)
├── chords.go       — ChordNode, ChordRegistry, ChordRouter
├── overlay.go      — WhichKeyOverlay view
├── action.go       — Action interface + concrete adapters
└── chords_test.go  — tests
```

### ChordNode — entity
```go
type ChordNode struct {
    Key      string     // "b"
    Label    string     // "Build & Run"
    Children []ChordNode 
    Action   Action     
}
```

### Action — port (DIP)
```go
type Action interface {
    Label() string
    Execute(ctx ActionContext) tea.Cmd
    IsEnabled(ctx ActionContext) bool
}
```

### State machine: Idle → Level1 → LevelN → Execute → Idle

App.leaderState tracks current chord path. All action keys route through chord tree. Navigation keys (j/k/h/l/g/G/enter/tab) remain direct.

### File impact
- Create: 4 files in tui/who/
- Modify: app.go, app_keys.go, app_update.go, app_view.go, browserview.go, commandpalette.go
- Delete: ListBrowsingHints from hintbar.go (replaced by dynamic leaderHints())
