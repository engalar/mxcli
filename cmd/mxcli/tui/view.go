package tui

import tea "github.com/charmbracelet/bubbletea"

// ViewMode identifies the type of view currently active.
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
)

// String returns a human-readable label for the view mode.
func (m ViewMode) String() string {
	switch m {
	case ModeBrowser:
		return "Browse"
	case ModeOverlay:
		return "Overlay"
	case ModeCompare:
		return "Compare"
	case ModeDiff:
		return "Diff"
	case ModePicker:
		return "Picker"
	case ModeJumper:
		return "Jump"
	case ModeExec:
		return "Exec"
	case ModeCommandPalette:
		return "Palette"
	default:
		return "Unknown"
	}
}

// StatusInfo carries display data for the status bar.
type StatusInfo struct {
	Breadcrumb []string
	Position   string
	Mode       string
	Extra      string
}

// View is the interface that all TUI views must implement.
type View interface {
	Update(tea.Msg) (View, tea.Cmd)
	Render(width, height int) string
	Hints() []Hint
	StatusInfo() StatusInfo
	Mode() ViewMode
}

// PushViewMsg requests that App push a new view onto the ViewStack.
type PushViewMsg struct{ View View }

// PopViewMsg requests that App pop the current view from the ViewStack.
type PopViewMsg struct{}
