package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/task"
)

// handleBrowserAppKeys handles keys that App intercepts when in Browser mode.
// Returns a non-nil tea.Cmd if the key was handled, nil if the key should
// be forwarded to the active view.
func (a *App) handleBrowserAppKeys(msg tea.KeyMsg) tea.Cmd {
	tab := a.activeTabPtr()

	// Handle two-key sequence: ]e / [e (check error navigation)
	if a.pendingKey != 0 {
		pending := a.pendingKey
		a.pendingKey = 0
		if msg.String() == "e" && len(a.checkErrors) > 0 {
			// Lazily initialize check nav state if not already active
			if !a.checkNavActive {
				a.checkNavActive = true
				a.checkNavLocations = extractCheckNavLocations(filterCheckErrors(a.checkErrors, "all"))
				a.checkNavIndex = -1 // will be incremented to 0 for ], or wrapped to last for [
			}
			if pending == ']' {
				a.checkNavIndex++
				if a.checkNavIndex >= len(a.checkNavLocations) {
					a.checkNavIndex = 0 // wrap around
				}
			} else {
				a.checkNavIndex--
				if a.checkNavIndex < 0 {
					a.checkNavIndex = len(a.checkNavLocations) - 1 // wrap around
				}
			}
			loc := a.checkNavLocations[a.checkNavIndex]
			qname := docNameToQualifiedName(loc.ModuleName, loc.DocumentName)
			if bv, ok := a.views.Base().(BrowserView); ok {
				cmd := bv.navigateToNode(qname)
				a.views.SetBase(bv)
				if tab := a.activeTabPtr(); tab != nil {
					tab.Miller = bv.miller
					tab.UpdateLabel()
					a.syncTabBar()
				}
				return cmd
			}
			return handledCmd
		}
		// Not 'e' — fall through to normal handling for the pending key
		// Re-process the pending key's original action
		if pending == ']' {
			if a.activeTab < len(a.tabs)-1 {
				a.activeTab++
				a.syncBrowserView()
				a.syncTabBar()
			}
		} else if pending == '[' {
			if a.activeTab > 0 {
				a.activeTab--
				a.syncBrowserView()
				a.syncTabBar()
			}
		}
		// Now process the current key normally (fall through)
	}

	// Non-nav keys exit check nav mode (preserve for ]/[/! which are nav-related)
	if a.checkNavActive {
		key := msg.String()
		if key != "]" && key != "[" && key != "!" && key != "\\!" {
			a.checkNavActive = false
		}
	}

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

	case "t":
		if tab != nil {
			newTab := tab.CloneTab(a.nextTabID, a.previewEngine)
			a.nextTabID++
			a.tabs = append(a.tabs, newTab)
			a.activeTab = len(a.tabs) - 1
			a.syncBrowserView()
			a.syncTabBar()
		}
		return handledCmd

	case "T":
		p := NewEmbeddedPicker()
		p.width = a.width
		p.height = a.height
		a.picker = &p
		return handledCmd

	case "W":
		if len(a.tabs) > 1 {
			a.tabs[a.activeTab].Miller.previewEngine.Cancel()
			a.tabs = append(a.tabs[:a.activeTab], a.tabs[a.activeTab+1:]...)
			if a.activeTab >= len(a.tabs) {
				a.activeTab = len(a.tabs) - 1
			}
			a.syncBrowserView()
			a.syncTabBar()
		}
		return handledCmd

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0]-'0') - 1
		if idx >= 0 && idx < len(a.tabs) {
			a.activeTab = idx
			a.syncBrowserView()
			a.syncTabBar()
		}
		return handledCmd

	case "[":
		if len(a.checkErrors) > 0 {
			a.pendingKey = '['
			return handledCmd
		}
		if a.activeTab > 0 {
			a.activeTab--
			a.syncBrowserView()
			a.syncTabBar()
		}
		return handledCmd

	case "]":
		if len(a.checkErrors) > 0 {
			a.pendingKey = ']'
			return handledCmd
		}
		if a.activeTab < len(a.tabs)-1 {
			a.activeTab++
			a.syncBrowserView()
			a.syncTabBar()
		}
		return handledCmd

	case "r":
		return a.Init()

	case "R":
		projectPath := a.activeTabProjectPath()
		if projectPath == "" {
			return handledCmd
		}
		cv := NewDiscardConfirmView(projectPath, a.mxcliPath)
		a.views.Push(cv)
		return handledCmd

	case " ":
		if tab != nil {
			items := flattenQualifiedNames(tab.AllNodes)
			jumper := NewJumperView(items, a.width, a.height)
			a.views.Push(jumper)
		}
		return handledCmd

	case "x":
		ev := NewExecView(a.mxcliPath, a.activeTabProjectPath(), a.width, a.height)
		a.views.Push(ev)
		return handledCmd

	case "b":
		bt := task.NewBuildTask(task.BuildOptions{
			ProjectPath: a.activeTabProjectPath(),
		})
		bv := NewBuildView(bt)
		a.views.Push(bv)
		return bt.Start()

	case "C":
		mxcliPath := a.mxcliPath
		projectPath := a.activeTabProjectPath()
		iv := NewInputView("Create Module", "Module name: ", func(name string) tea.Cmd {
			return func() tea.Msg {
				out, err := runMxcli(mxcliPath, "-p", projectPath, "-c", "CREATE MODULE "+name)
				return execShowResultMsg{Content: out, Success: err == nil}
			}
		})
		a.views.Push(iv)
		return handledCmd

	case "!", "\\!": // some terminals send "\\!" for shifted-1; accept both forms
		filter := "all"
		title := renderCheckFilterTitle(a.checkErrors, filter)
		content := renderCheckResults(a.checkErrors, filter)
		navLocs := extractCheckNavLocations(a.checkErrors)
		ov := NewOverlayView(title, content, a.width, a.height, OverlayViewOpts{
			HideLineNumbers: true,
			Refreshable:     true,
			RefreshMsg:      MxCheckRerunMsg{},
			CheckFilter:     filter,
			CheckErrors:     a.checkErrors,
			CheckNavLocs:    navLocs,
		})
		a.views.Push(ov)
		return handledCmd

	case ":":
		cp := NewCommandPaletteView(a.width, a.height)
		a.views.Push(cp)
		return handledCmd

	case "c":
		cv := NewCompareView()
		cv.mxcliPath = a.mxcliPath
		cv.projectPath = a.activeTabProjectPath()
		cv.Show(CompareNDSLMDL, a.width, a.height)
		if tab != nil {
			cv.SetItems(flattenQualifiedNames(tab.AllNodes))
			if node := tab.Miller.SelectedNode(); node != nil && node.QualifiedName != "" {
				cv.SetLoading(CompareFocusLeft)
				cv.SetLoading(CompareFocusRight)
				a.views.Push(cv)
				return tea.Batch(
					cv.loadBsonNDSL(node.QualifiedName, node.Type, CompareFocusLeft),
					cv.loadMDL(node.QualifiedName, node.Type, CompareFocusRight),
				)
			}
		}
		a.views.Push(cv)
		return handledCmd
	}

	return nil
}

// dispatchPaletteKey converts a palette command key string into a synthetic
// tea.KeyMsg and re-dispatches it through Update.
func (a App) dispatchPaletteKey(key string) tea.Cmd {
	var keyMsg tea.KeyMsg
	switch key {
	case " ":
		keyMsg = tea.KeyMsg{Type: tea.KeySpace}
	case "Tab":
		keyMsg = tea.KeyMsg{Type: tea.KeyTab}
	default:
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	return func() tea.Msg { return keyMsg }
}
