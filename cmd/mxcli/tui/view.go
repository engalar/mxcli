package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

type ViewMode = kernel.ViewMode

const (
	ModeBrowser        = kernel.ModeBrowser
	ModeOverlay        = kernel.ModeOverlay
	ModeCompare        = kernel.ModeCompare
	ModeDiff           = kernel.ModeDiff
	ModePicker         = kernel.ModePicker
	ModeJumper         = kernel.ModeJumper
	ModeExec           = kernel.ModeExec
	ModeCommandPalette = kernel.ModeCommandPalette
	ModeInput          = kernel.ModeInput
	ModeConfirm        = kernel.ModeConfirm
)

type StatusInfo = kernel.StatusInfo

type View interface {
	Update(tea.Msg) (View, tea.Cmd)
	Render(width, height int) string
	Hints() []Hint
	StatusInfo() StatusInfo
	Mode() ViewMode
}

type PushViewMsg struct{ View View }
type PopViewMsg struct{}
