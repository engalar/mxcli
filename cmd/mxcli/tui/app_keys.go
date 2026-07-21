package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/who"
)

func (a *App) handleBrowserAppKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q":
		if session := ExtractSession(a); session != nil {
			_ = SaveSession(session)
		}
		if a.watcher != nil {
			a.watcher.Close()
		}
		for i := range a.tabs {
			a.tabs[i].Miller.previewEngine.Cancel()
		}
		CloseTrace()
		return tea.Quit

	case ":":
		cp := NewCommandPaletteView(a.width, a.height)
		a.views.Push(cp)
		return handledCmd

	case " ":
		root := a.chordTree()
		wov := who.NewOverlay(root)
		ov := NewWhoOverlayView(wov)
		a.views.Push(ov)
		return handledCmd
	}

	if isNavigationKey(msg.String()) {
		return nil
	}
	return nil
}
