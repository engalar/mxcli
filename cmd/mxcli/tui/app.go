package tui

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/chrome"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/task"
)

// chromeHeight is the vertical space consumed by tab bar (1) + hint bar (1) + status bar (1).
const chromeHeight = 3

// handledNoop is a pre-allocated no-op Msg to avoid per-call goroutine allocation.
var handledNoop tea.Msg = struct{}{}

// handledCmd is returned by handleBrowserAppKeys to signal that a key was
// consumed without producing a follow-up message.  Using a shared variable
// avoids allocating a new closure on every handled keystroke.
var handledCmd tea.Cmd = func() tea.Msg { return handledNoop }

// compareFlashClearMsg is sent 1 s after a clipboard copy in compare view.
type compareFlashClearMsg struct{}

// App is the root Bubble Tea model for the yazi-style TUI.
type App struct {
	tabs      []Tab
	activeTab int
	nextTabID int

	width     int
	height    int
	mxcliPath string

	views    ViewStack
	showHelp bool
	picker   *PickerModel // non-nil when cross-project picker is open

	// Check error navigation state (]e / [e)
	checkNavActive    bool
	checkNavIndex     int
	checkNavLocations []CheckNavLocation
	pendingKey        rune // ']' or '[' waiting for 'e', 0 if none

	tabBar        chrome.TabBar
	hintBar       chrome.HintBar
	statusBar     chrome.StatusBar
	previewEngine *PreviewEngine

	watcher      *Watcher
	checkErrors  []CheckError // nil = no check run yet, empty = pass
	checkRunning bool

	pendingSession *TUISession // session to restore after tree loads

	agentListener    *AgentListener
	agentAutoProceed bool                 // skip human confirmation for agent ops (set before tea.NewProgram)
	agentPending     *agentPendingOp      // non-nil when waiting for user confirmation
	agentCheckCh     chan<- AgentResponse // non-nil when agent check is in-flight
	agentCheckReqID  int                  // request ID for pending agent check
	agentExecCtx     *agentExecContext    // non-nil when agent-initiated exec/delete/create is in progress
}

// agentPendingOp tracks an in-flight agent operation awaiting user confirmation.
type agentPendingOp struct {
	RequestID  int
	Output     string
	Success    bool
	ResponseCh chan<- AgentResponse
}

// agentExecContext tracks an agent-initiated operation routed through UI views.
// The agent's exec/delete/create_module actions push the same views a human
// would use (ExecView/ConfirmView/InputView). This context links the UI flow
// back to the agent response channel.
type agentExecContext struct {
	RequestID  int
	ResponseCh chan<- AgentResponse
}

// NewApp creates the root App model.
func NewApp(mxcliPath, projectPath string) App {
	initTrace()
	Trace("app: NewApp mxcli=%q project=%q", mxcliPath, projectPath)

	engine := NewPreviewEngine(mxcliPath, projectPath)
	tab := NewTab(1, projectPath, engine, nil)

	browserView := NewBrowserView(&tab, mxcliPath, engine)

	app := App{
		mxcliPath:     mxcliPath,
		nextTabID:     2,
		views:         NewViewStack(browserView),
		tabBar:        chrome.NewTabBar(nil),
		statusBar:     chrome.NewStatusBar(),
		hintBar:       chrome.NewHintBar(ListBrowsingHints),
		previewEngine: engine,
	}
	app.tabs = []Tab{tab}
	app.syncTabBar()
	return app
}

// StartWatcher begins watching MPR files for external changes.
// Call after tea.NewProgram is created but before p.Run().
func (a *App) StartWatcher(prog *tea.Program) {
	tab := a.activeTabPtr()
	if tab == nil {
		return
	}
	mprPath := tab.ProjectPath
	contentsDir := ""
	dir := filepath.Dir(mprPath)
	candidate := filepath.Join(dir, "mprcontents")
	if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
		contentsDir = candidate
	}
	w, err := NewWatcher(mprPath, contentsDir, prog)
	if err != nil {
		Trace("app: failed to start watcher: %v", err)
		return
	}
	a.watcher = w
	Trace("app: watcher started for %s (contentsDir=%q)", mprPath, contentsDir)
}

// SetAgentAutoProceed configures whether agent operations skip human confirmation.
// Must be called BEFORE tea.NewProgram so the value is captured in the model copy.
func (a *App) SetAgentAutoProceed(autoProceed bool) {
	a.agentAutoProceed = autoProceed
}

// StartAgentListener begins listening on a Unix socket for agent commands.
// Call after tea.NewProgram is created, like StartWatcher.
func (a *App) StartAgentListener(prog *tea.Program, socketPath string, autoProceed bool) error {
	listener, err := NewAgentListener(socketPath, prog.Send, autoProceed)
	if err != nil {
		return err
	}
	a.agentListener = listener
	Trace("app: agent listener started on %s (autoProceed=%v)", socketPath, autoProceed)
	return nil
}

// CloseAgentListener stops the agent listener if running.
func (a *App) CloseAgentListener() {
	if a.agentListener != nil {
		a.agentListener.Close()
	}
}

func (a *App) activeTabPtr() *Tab {
	if a.activeTab >= 0 && a.activeTab < len(a.tabs) {
		return &a.tabs[a.activeTab]
	}
	return nil
}

func (a *App) activeTabProjectPath() string {
	tab := a.activeTabPtr()
	if tab != nil {
		return tab.ProjectPath
	}
	return ""
}

func (a *App) syncTabBar() {
	infos := make([]chrome.TabInfo, len(a.tabs))
	for i, t := range a.tabs {
		infos[i] = chrome.TabInfo{ID: t.ID, Label: t.Label, Active: i == a.activeTab}
	}
	a.tabBar.SetTabs(infos)
}

func (a *App) syncBrowserView() {
	tab := a.activeTabPtr()
	if tab == nil {
		return
	}
	bv := NewBrowserView(tab, a.mxcliPath, a.previewEngine)
	bv.allNodes = tab.AllNodes
	// Ensure miller has current dimensions so scroll calculations in
	// Update() work correctly (Render operates on a value copy).
	if a.height > 0 {
		contentH := max(5, a.height-chromeHeight-1) // -1 for LLM anchor line
		bv.miller.SetSize(a.width, contentH)
	}
	a.views.SetBase(bv)
}

// --- Init ---

func (a App) Init() tea.Cmd {
	tab := a.activeTabPtr()
	if tab == nil {
		return nil
	}
	tabID := tab.ID
	mxcliPath := a.mxcliPath
	projectPath := tab.ProjectPath
	return func() tea.Msg {
		out, err := runMxcli(mxcliPath, "project-tree", "-p", projectPath)
		if err != nil {
			return LoadTreeMsg{TabID: tabID, Err: err}
		}
		nodes, parseErr := ParseTree(out)
		return LoadTreeMsg{TabID: tabID, Nodes: nodes, Err: parseErr}
	}
}

// --- Tab management ---

func (a *App) findTabByID(id int) *Tab {
	for i := range a.tabs {
		if a.tabs[i].ID == id {
			return &a.tabs[i]
		}
	}
	return nil
}

func (a *App) switchToTabByID(id int) {
	for i, t := range a.tabs {
		if t.ID == id {
			a.activeTab = i
			a.syncBrowserView()
			a.syncTabBar()
			return
		}
	}
}

// SetPendingSession stores a session to be restored after the project tree loads.
func (a *App) SetPendingSession(session *TUISession) {
	a.pendingSession = session
}

// applySessionRestore applies the pending session state to the loaded app.
// Called after LoadTreeMsg delivers nodes so navigation paths can be resolved.
// Takes *App because it's called from Update (value receiver) via &a.
func applySessionRestore(a *App) {
	session := a.pendingSession
	if session == nil {
		return
	}
	a.pendingSession = nil

	if len(session.Tabs) == 0 {
		return
	}

	// Restore the first tab's navigation (multi-tab restore: only the
	// primary tab is restored since additional tabs need separate
	// project-tree loads which are not wired yet).
	ts := session.Tabs[0]
	tab := a.activeTabPtr()
	if tab == nil || len(tab.AllNodes) == 0 {
		return
	}

	// Navigate to the selected node if available
	if ts.SelectedNode != "" {
		if bv, ok := a.views.Base().(BrowserView); ok {
			bv.allNodes = tab.AllNodes
			bv.navigateToNode(ts.SelectedNode)
			// Set preview mode after navigation (navigateToNode resets miller)
			setPreviewMode(&bv.miller, ts.PreviewMode)
			tab.Miller = bv.miller
			tab.UpdateLabel()
			a.views.SetBase(bv)
			a.syncTabBar()
			Trace("app: session restored — navigated to %q", ts.SelectedNode)
			return
		}
	}

	// Fallback: navigate the miller path breadcrumb
	if len(ts.MillerPath) > 0 {
		restoreMillerPath(a, tab, ts.MillerPath)
	}

	// Set preview mode (for path-based or no-navigation restore)
	setPreviewMode(&tab.Miller, ts.PreviewMode)
}

// setPreviewMode sets the miller preview mode from a string value.
func setPreviewMode(miller *MillerView, mode string) {
	if mode == "NDSL" {
		miller.preview.mode = PreviewNDSL
	} else {
		miller.preview.mode = PreviewMDL
	}
}

// restoreMillerPath drills the miller view through a breadcrumb path.
func restoreMillerPath(a *App, tab *Tab, millerPath []string) {
	bv, ok := a.views.Base().(BrowserView)
	if !ok {
		return
	}
	bv.allNodes = tab.AllNodes
	bv.miller.SetRootNodes(tab.AllNodes)

	for _, segment := range millerPath {
		found := false
		for j, item := range bv.miller.current.items {
			if item.Label == segment {
				bv.miller.current.SetCursor(j)
				if item.Node != nil && len(item.Node.Children) > 0 {
					bv.miller, _ = bv.miller.drillIn()
				}
				found = true
				break
			}
		}
		if !found {
			Trace("app: session restore — path segment %q not found, stopping", segment)
			break
		}
	}

	tab.Miller = bv.miller
	tab.UpdateLabel()
	a.views.SetBase(bv)
	a.syncTabBar()
	Trace("app: session restored via miller path %v", millerPath)
}

func (a App) startRun() tea.Cmd {
	projectPath := a.activeTabProjectPath()
	if projectPath == "" {
		return nil
	}
	rt := task.NewRunTask(task.RunOptions{
		PadDir:  filepath.Join(filepath.Dir(projectPath), ".docker", "build"),
		CmdHint: "-p " + projectPath,
	})
	rv := NewRunView(rt)
	a.views.Push(rv)
	return rt.Start()
}
