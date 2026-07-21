package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/who"
)

type WhoOverlayView struct {
	inner     who.Overlay
	width     int
	height    int
}

func NewWhoOverlayView(inner who.Overlay) WhoOverlayView {
	return WhoOverlayView{inner: inner}
}

func (w WhoOverlayView) Update(msg tea.Msg) (View, tea.Cmd) {
	i, cmd := w.inner.Update(msg)
	w.inner = i.(who.Overlay)
	return w, cmd
}

func (w WhoOverlayView) Render(width, height int) string {
	w.width = width
	w.height = height
	return w.inner.Render(width, height)
}

func (w WhoOverlayView) Hints() []Hint     { return nil }
func (w WhoOverlayView) StatusInfo() StatusInfo {
	prefix := "SPC"
	label := "Menu"
	return kernel.StatusInfo{
		Breadcrumb: []string{prefix, label},
		Mode:       "MENU",
	}
}
func (w WhoOverlayView) Mode() ViewMode { return ModeCommandPalette }
