package tui

import tea "github.com/charmbracelet/bubbletea"

// handleBrowserAppKeys handles keys that App intercepts when in Browser mode.
// Only global keys (q, :, ?, Space) and navigation keys are handled here.
// All action keys (build, check, exec, view, etc.) go through the Space leader menu.
func (a *App) handleBrowserAppKeys(msg tea.KeyMsg) tea.Cmd {
	// Global keys — always handled
	switch msg.String() {
	case "q":
		// Save session state before quitting
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
	}

	// Navigation keys — forward to active view (miller handles them)
	if isNavigationKey(msg.String()) {
		return nil
	}

	// Leader chord routing
	if msg.String() == " " {
		return a.activateLeader()
	}
	if a.leaderState != nil {
		return a.routeLeaderKey(msg.String())
	}

	return nil
}
