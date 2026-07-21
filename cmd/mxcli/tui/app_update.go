package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/chrome"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

// --- Update ---

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// --- ViewStack navigation ---
	case PushViewMsg:
		a.views.Push(msg.View)
		return a, nil
	case PopViewMsg:
		a.views.Pop()
		// If human cancelled a view while an agent operation was pending, reject it
		if ctx := a.agentExecCtx; ctx != nil {
			ctx.ResponseCh <- AgentResponse{
				ID: ctx.RequestID, OK: false,
				Error: "cancelled by user",
			}
			a.agentExecCtx = nil
		}
		return a, nil

	case PaletteExecMsg:
		a.views.Pop()
		if strings.HasPrefix(msg.Key, ":") {
			chord := msg.Key[1:]
			return a, a.executeChord(chord)
		}
		if msg.Key != "" {
			return a, a.dispatchPaletteKey(msg.Key)
		}
		return a, nil

	// --- View creation messages ---
	case OpenOverlayMsg:
		ov := NewOverlayView(msg.Title, msg.Content, a.width, a.height, msg.Opts)
		a.views.Push(ov)
		return a, nil

	case OpenImageOverlayMsg:
		w, h := a.width, a.height
		paths := msg.Paths
		title := msg.Title
		return a, func() tea.Msg {
			innerW := w - 4
			innerH := h - 4
			if innerW < 20 {
				innerW = 20
			}
			if innerH < 5 {
				innerH = 5
			}
			perImg := innerH / len(paths)
			if perImg < 1 {
				perImg = 1
			}
			content := renderImagesWithSize(paths, innerW, perImg)
			if content == "" {
				content = "(no image rendered — set MXCLI_IMAGE_PROTOCOL or install chafa)"
			}
			return OpenOverlayMsg{Title: title, Content: content}
		}

	case JumpToNodeMsg:
		// Pop the jumper view first
		a.views.Pop()
		// Navigate browser to the target node
		if bv, ok := a.views.Base().(BrowserView); ok {
			cmd := bv.navigateToNode(msg.QName)
			a.views.SetBase(bv)
			if tab := a.activeTabPtr(); tab != nil {
				tab.Miller = bv.miller
				tab.UpdateLabel()
				a.syncTabBar()
			}
			return a, cmd
		}
		return a, nil

	case NavigateToDocMsg:
		// Close overlay, navigate tree to document, enter check nav mode
		a.views.Pop()
		qname := docNameToQualifiedName(msg.ModuleName, msg.DocumentName)
		if bv, ok := a.views.Base().(BrowserView); ok {
			cmd := bv.navigateToNode(qname)
			a.views.SetBase(bv)
			if tab := a.activeTabPtr(); tab != nil {
				tab.Miller = bv.miller
				tab.UpdateLabel()
				a.syncTabBar()
			}
			// Enter check nav mode
			a.checkNavActive = true
			a.checkNavIndex = msg.NavIndex
			a.checkNavLocations = extractCheckNavLocations(filterCheckErrors(a.checkErrors, "all"))
			return a, cmd
		}
		return a, nil

	case DiffOpenMsg:
		dv := NewDiffView(msg, a.width, a.height)
		a.views.Push(dv)
		return a, nil

	case PickerDoneMsg:
		Trace("app: PickerDoneMsg path=%q", msg.Path)
		a.picker = nil
		if msg.Path != "" {
			SaveHistory(msg.Path)
			engine := NewPreviewEngine(a.mxcliPath, msg.Path)
			newTab := NewTab(a.nextTabID, msg.Path, engine, nil)
			a.nextTabID++
			a.tabs = append(a.tabs, newTab)
			a.activeTab = len(a.tabs) - 1
			a.syncBrowserView()
			a.syncTabBar()
			tabID := newTab.ID
			mxcliPath := a.mxcliPath
			projectPath := msg.Path
			return a, func() tea.Msg {
				out, err := runMxcli(mxcliPath, "project-tree", "-p", projectPath)
				if err != nil {
					return LoadTreeMsg{TabID: tabID, Err: err}
				}
				nodes, parseErr := ParseTree(out)
				return LoadTreeMsg{TabID: tabID, Nodes: nodes, Err: parseErr}
			}
		}
		return a, nil

	case CompareLoadMsg:
		if cv, ok := a.views.Active().(CompareView); ok {
			cv.SetContent(msg.Side, msg.Title, msg.NodeType, msg.Content)
			a.views.SetActive(cv)
		}
		return a, nil

	case ComparePickMsg:
		if cv, ok := a.views.Active().(CompareView); ok {
			cv.SetLoading(msg.Side)
			a.views.SetActive(cv)
			return a, cv.loadForCompare(msg.QName, msg.NodeType, msg.Side, msg.Kind)
		}
		return a, nil

	case CompareReloadMsg:
		if cv, ok := a.views.Active().(CompareView); ok {
			var cmds []tea.Cmd
			if cv.left.qname != "" {
				cv.SetLoading(CompareFocusLeft)
				cmds = append(cmds, cv.loadForCompare(cv.left.qname, cv.left.nodeType, CompareFocusLeft, msg.Kind))
			}
			if cv.right.qname != "" {
				cv.SetLoading(CompareFocusRight)
				cmds = append(cmds, cv.loadForCompare(cv.right.qname, cv.right.nodeType, CompareFocusRight, msg.Kind))
			}
			a.views.SetActive(cv)
			return a, tea.Batch(cmds...)
		}
		return a, nil

	case overlayFlashClearMsg:
		// Forward to active view (Overlay handles this internally)
		updated, cmd := a.views.Active().Update(msg)
		a.views.SetActive(updated)
		return a, cmd

	case compareFlashClearMsg:
		if cv, ok := a.views.Active().(CompareView); ok {
			cv.copiedFlash = false
			a.views.SetActive(cv)
		}
		return a, nil

	case overlayContentMsg:
		updated, cmd := a.views.Active().Update(msg)
		a.views.SetActive(updated)
		return a, cmd

	case CmdResultMsg:
		content := msg.Output
		if msg.Err != nil {
			content = "-- Error:\n" + msg.Output
		}
		ov := NewOverlayView("Result", DetectAndHighlight(content), a.width, a.height, OverlayViewOpts{})
		a.views.Push(ov)
		return a, nil

	case discardDoneMsg:
		// Clear preview cache so stale data isn't shown after discard
		a.previewEngine.ClearCache()
		return a, func() tea.Msg {
			return execShowResultMsg{Content: msg.Output, Success: msg.Success}
		}

	case execShowResultMsg:
		// Pop the ExecView (or ConfirmView)
		a.views.Pop()
		// Show result in overlay
		content := DetectAndHighlight(msg.Content)
		title := "Exec Result"
		if a.agentExecCtx != nil {
			title = "Agent Exec Result"
			if !msg.Success {
				title = "Agent Exec Error"
			}
		}
		ov := NewOverlayView(title, content, a.width, a.height, OverlayViewOpts{})
		a.views.Push(ov)
		// If execution succeeded, suppress watcher (self-modification) and refresh tree
		if msg.Success {
			if a.watcher != nil {
				a.watcher.Suppress(2 * time.Second)
			}
		}
		// Handle agent response if this was agent-initiated
		if ctx := a.agentExecCtx; ctx != nil {
			if a.agentAutoProceed {
				resp := AgentResponse{
					ID: ctx.RequestID, OK: msg.Success,
					Result: msg.Content, Mode: "overlay:exec-result",
				}
				if changes := agentParseChanges(msg.Content); len(changes) > 0 {
					resp.Changes, _ = json.Marshal(changes)
				}
				ctx.ResponseCh <- resp
				a.agentExecCtx = nil
				if msg.Success {
					return a, a.Init()
				}
				return a, nil
			}
			// Store pending op for user confirmation (q/esc in overlay)
			a.agentPending = &agentPendingOp{
				RequestID: ctx.RequestID, Output: msg.Content,
				Success: msg.Success, ResponseCh: ctx.ResponseCh,
			}
			a.agentExecCtx = nil
			return a, nil
		}
		if msg.Success {
			return a, a.Init()
		}
		return a, nil

	case OpenExecWithContentMsg:
		ev := NewExecViewWithContent(a.mxcliPath, a.activeTabProjectPath(), a.width, a.height, msg.Content)
		a.views.Push(ev)
		return a, nil

	// --- Agent channel messages ---
	case AgentExecMsg:
		Trace("app: AgentExecMsg id=%d mdl=%q", msg.RequestID, msg.MDL)
		// Route through ExecView — same UI path as human pressing 'x'
		a.agentExecCtx = &agentExecContext{
			RequestID:  msg.RequestID,
			ResponseCh: msg.ResponseCh,
		}
		ev := NewExecViewWithContent(a.mxcliPath, a.activeTabProjectPath(), a.width, a.height, msg.MDL)
		a.views.Push(ev)
		if a.agentAutoProceed {
			return a, func() tea.Msg { return AgentAutoExecMsg{} }
		}
		return a, nil

	case AgentStateMsg:
		Trace("app: AgentStateMsg id=%d", msg.RequestID)
		stateInfo := agentBuildState(a)
		stateJSON, _ := json.Marshal(stateInfo)
		msg.ResponseCh <- AgentResponse{
			ID: msg.RequestID, OK: true,
			Result: string(stateJSON),
			Mode:   "state",
		}
		return a, nil

	case AgentCheckMsg:
		Trace("app: AgentCheckMsg id=%d", msg.RequestID)
		if a.agentCheckCh != nil {
			msg.ResponseCh <- AgentResponse{
				ID: msg.RequestID, OK: false,
				Error: "check already in progress",
			}
			return a, nil
		}
		a.agentCheckCh = msg.ResponseCh
		a.agentCheckReqID = msg.RequestID
		projectPath := a.activeTabProjectPath()
		return a, runMxCheck(projectPath)

	case AgentNavigateMsg:
		Trace("app: AgentNavigateMsg id=%d target=%q", msg.RequestID, msg.Target)
		target := msg.Target
		if bv, ok := a.views.Base().(BrowserView); ok {
			qname := target
			if idx := strings.Index(target, ":"); idx >= 0 {
				qname = target[idx+1:]
			}
			cmd := bv.navigateToNode(qname)
			a.views.SetBase(bv)
			if tab := a.activeTabPtr(); tab != nil {
				tab.Miller = bv.miller
				tab.UpdateLabel()
				a.syncTabBar()
			}
			msg.ResponseCh <- AgentResponse{
				ID: msg.RequestID, OK: true,
				Result: fmt.Sprintf("navigated to %s", qname), Mode: "browser",
			}
			return a, cmd
		}
		msg.ResponseCh <- AgentResponse{
			ID: msg.RequestID, OK: false, Error: "not in browser mode",
		}
		return a, nil

	case AgentDeleteMsg:
		Trace("app: AgentDeleteMsg id=%d target=%q", msg.RequestID, msg.Target)
		nodeType, qname := parseTarget(msg.Target)
		dropCmd := buildDropCmd(nodeType, qname)
		if dropCmd == "" {
			msg.ResponseCh <- AgentResponse{
				ID: msg.RequestID, OK: false,
				Error: fmt.Sprintf("unsupported delete type: %q", nodeType),
			}
			return a, nil
		}
		// Route through ConfirmView — same UI path as human pressing 'D'
		a.agentExecCtx = &agentExecContext{
			RequestID:  msg.RequestID,
			ResponseCh: msg.ResponseCh,
		}
		message := buildDeleteMessage(nodeType, qname)
		cv := NewConfirmView("Delete", message, dropCmd, a.mxcliPath, a.activeTabProjectPath())
		a.views.Push(cv)
		if a.agentAutoProceed {
			// Auto-confirm (like pressing 'y')
			return a, func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
			}
		}
		return a, nil

	case AgentCreateModuleMsg:
		Trace("app: AgentCreateModuleMsg id=%d name=%q", msg.RequestID, msg.Name)
		// Route through InputView — same UI path as human pressing 'C'
		a.agentExecCtx = &agentExecContext{
			RequestID:  msg.RequestID,
			ResponseCh: msg.ResponseCh,
		}
		mxcliPath := a.mxcliPath
		projectPath := a.activeTabProjectPath()
		iv := NewInputView("Create Module", "Module name: ", func(name string) tea.Cmd {
			return func() tea.Msg {
				out, err := runMxcli(mxcliPath, "-p", projectPath, "-c", "CREATE MODULE "+name)
				return execShowResultMsg{Content: out, Success: err == nil}
			}
		})
		iv.input.SetValue(msg.Name)
		a.views.Push(iv)
		if a.agentAutoProceed {
			// Auto-submit (like pressing Enter)
			return a, func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyEnter}
			}
		}
		return a, nil

	case tea.KeyMsg:
		Trace("app: key=%q picker=%v mode=%v help=%v", msg.String(), a.picker != nil, a.views.Active().Mode(), a.showHelp)
		if msg.String() == "ctrl+c" {
			if a.watcher != nil {
				a.watcher.Close()
			}
			a.CloseAgentListener()
			return a, tea.Quit
		}

		// Agent confirmation: when overlay is dismissed while agent op is pending
		if a.agentPending != nil && a.views.Active().Mode() == ModeOverlay &&
			(msg.String() == "q" || msg.String() == "esc") {
			pending := a.agentPending
			a.agentPending = nil
			a.views.Pop()
			resp := AgentResponse{
				ID: pending.RequestID, OK: pending.Success,
				Result: pending.Output, Mode: "overlay:exec-result",
			}
			if changes := agentParseChanges(pending.Output); len(changes) > 0 {
				resp.Changes, _ = json.Marshal(changes)
			}
			pending.ResponseCh <- resp
			if pending.Success {
				return a, a.Init()
			}
			return a, nil
		}

		// Picker modal (not a View — special case)
		if a.picker != nil {
			result, cmd := a.picker.Update(msg)
			p := result.(PickerModel)
			a.picker = &p
			return a, cmd
		}

		// Help toggle (global, only in Browser mode)
		if a.showHelp {
			a.showHelp = false
			return a, nil
		}
		if msg.String() == "?" && a.views.Active().Mode() == ModeBrowser {
			a.showHelp = !a.showHelp
			return a, nil
		}

		// Tab management and app-level keys (only in Browser mode)
		if a.views.Active().Mode() == ModeBrowser {
			if cmd := a.handleBrowserAppKeys(msg); cmd != nil {
				return a, cmd
			}
		}

		// Delegate to active view
		updated, cmd := a.views.Active().Update(msg)
		a.views.SetActive(updated)

		// Sync tab label if browser view
		if a.views.Active().Mode() == ModeBrowser {
			tab := a.activeTabPtr()
			if tab != nil {
				if bv, ok := a.views.Active().(BrowserView); ok {
					tab.Miller = bv.miller
					tab.UpdateLabel()
					a.syncTabBar()
				}
			}
		}
		return a, cmd

	case tea.MouseMsg:
		Trace("app: mouse x=%d y=%d btn=%v action=%v", msg.X, msg.Y, msg.Button, msg.Action)
		if a.picker != nil {
			return a, nil
		}

		// Tab bar clicks (row 1, after LLM anchor line) — only when in browser mode
		if msg.Y == 1 && a.views.Active().Mode() == ModeBrowser &&
			msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if clickMsg := a.tabBar.HandleClick(msg.X); clickMsg != nil {
				if tc, ok := clickMsg.(chrome.TabClickMsg); ok {
					a.switchToTabByID(tc.ID)
					return a, nil
				}
			}
		}

		// Status bar clicks (last line) — breadcrumb navigation
		if msg.Y == a.height-1 && a.views.Active().Mode() == ModeBrowser &&
			msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if depth, ok := a.statusBar.HitTest(msg.X); ok {
				if bv, ok := a.views.Active().(BrowserView); ok {
					var cmd tea.Cmd
					bv.miller, cmd = bv.miller.goBackToDepth(depth)
					a.views.SetActive(bv)
					if tab := a.activeTabPtr(); tab != nil {
						tab.Miller = bv.miller
						tab.UpdateLabel()
						a.syncTabBar()
					}
					return a, cmd
				}
			}
		}

		// Offset Y by -2 (LLM anchor line + tab bar) when in browser mode
		if a.views.Active().Mode() == ModeBrowser {
			offsetMsg := tea.MouseMsg{
				X: msg.X, Y: msg.Y - 2,
				Button: msg.Button, Action: msg.Action,
			}
			updated, cmd := a.views.Active().Update(offsetMsg)
			a.views.SetActive(updated)
			return a, cmd
		}

		// Forward to active view
		updated, cmd := a.views.Active().Update(msg)
		a.views.SetActive(updated)
		return a, cmd

	case tea.WindowSizeMsg:
		Trace("app: resize %dx%d", msg.Width, msg.Height)
		a.width = msg.Width
		a.height = msg.Height
		// Propagate content dimensions to the browser view so that
		// subsequent Update() calls use correct column heights for
		// scroll calculations (Render operates on a copy and cannot
		// persist dimensions back).
		if bv, ok := a.views.Active().(BrowserView); ok {
			contentH := a.height - chromeHeight - 1 // -1 for LLM anchor line
			if contentH < 5 {
				contentH = 5
			}
			bv.miller.SetSize(a.width, contentH)
			a.views.SetActive(bv)
		}
		return a, nil

	case LoadTreeMsg:
		Trace("app: LoadTreeMsg tabID=%d err=%v nodes=%d", msg.TabID, msg.Err, len(msg.Nodes))
		if msg.Err == nil && msg.Nodes != nil {
			tab := a.findTabByID(msg.TabID)
			if tab != nil {
				tab.AllNodes = msg.Nodes
				tab.Miller.SetRootNodes(msg.Nodes)
				a.syncTabBar()
				// Update browser view if it's the base
				if bv, ok := a.views.Base().(BrowserView); ok {
					bv.allNodes = msg.Nodes
					bv.miller = tab.Miller
					if a.height > 0 {
						contentH := max(5, a.height-chromeHeight-1) // -1 for LLM anchor line
						bv.miller.SetSize(a.width, contentH)
					}
					a.views.SetBase(bv)
				}
			}

			// Apply pending session restore after tree is loaded
			if a.pendingSession != nil {
				applySessionRestore(&a)
			}
		}
		return a, nil

	case MprChangedMsg:
		Trace("app: MprChangedMsg — refreshing tree and running mx check")
		a.previewEngine.ClearCache()
		projectPath := a.activeTabProjectPath()
		return a, tea.Batch(a.Init(), runMxCheck(projectPath))

	case MxCheckRerunMsg:
		Trace("app: manual mx check rerun requested")
		projectPath := a.activeTabProjectPath()
		return a, runMxCheck(projectPath)

	case MxCheckStartMsg:
		a.checkRunning = true
		// Update check overlay content if it's currently visible
		if ov, ok := a.views.Active().(OverlayView); ok && ov.refreshable {
			ov.overlay.Show("mx check", kernel.CheckRunningStyle.Render("⟳ Running mx check..."), ov.overlay.width, ov.overlay.height)
			a.views.SetActive(ov)
		}
		return a, nil

	case MxCheckResultMsg:
		a.checkRunning = false
		if msg.Err != nil {
			Trace("app: mx check error: %v", msg.Err)
			a.checkErrors = nil
		} else {
			a.checkErrors = msg.Errors
			Trace("app: mx check done: %d diagnostics", len(msg.Errors))
		}
		// Respond to pending agent check request
		if a.agentCheckCh != nil {
			if msg.Err != nil {
				a.agentCheckCh <- AgentResponse{
					ID: a.agentCheckReqID, OK: false,
					Result: msg.Err.Error(), Mode: "check",
				}
			} else {
				result := renderCheckResultsPlain(msg.Errors)
				a.agentCheckCh <- AgentResponse{
					ID: a.agentCheckReqID, OK: true,
					Result: result, Mode: "check",
				}
			}
			a.agentCheckCh = nil
			a.agentCheckReqID = 0
		}
		// Update check overlay content if it's currently visible
		if ov, ok := a.views.Active().(OverlayView); ok && ov.refreshable {
			ov.checkErrors = a.checkErrors
			filtered := filterCheckErrors(a.checkErrors, ov.checkFilter)
			ov.checkNavLocs = extractCheckNavLocations(filtered)
			if ov.selectedIdx >= len(ov.checkNavLocs) {
				ov.selectedIdx = max(0, len(ov.checkNavLocs)-1)
			}
			if len(ov.checkNavLocs) == 0 {
				ov.selectedIdx = -1
			}
			title := renderCheckFilterTitle(a.checkErrors, ov.checkFilter)
			content := renderCheckResults(a.checkErrors, ov.checkFilter)
			ov.overlay.Show(title, content, ov.overlay.width, ov.overlay.height)
			a.views.SetActive(ov)
		}
		return a, nil

	case PreviewReadyMsg, PreviewLoadingMsg, CursorChangedMsg, animTickMsg, previewDebounceMsg:
		if a.views.Active().Mode() == ModeBrowser {
			updated, cmd := a.views.Active().Update(msg)
			a.views.SetActive(updated)
			// Sync miller back to tab
			if bv, ok := updated.(BrowserView); ok {
				tab := a.activeTabPtr()
				if tab != nil {
					tab.Miller = bv.miller
				}
			}
			return a, cmd
		}
		return a, nil

	default:
		// Forward everything else to active view
		updated, cmd := a.views.Active().Update(msg)
		a.views.SetActive(updated)
		return a, cmd
	}
}
