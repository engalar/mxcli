package tui

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/chrome"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/task"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/who"
)

// chromeHeight is the vertical space consumed by tab bar (1) + hint bar (1) + status bar (1).
const chromeHeight = 3

// handledNoop is a pre-allocated no-op Msg to avoid per-call goroutine allocation.
var handledNoop tea.Msg = struct{}{}

// handledCmd is returned by handleBrowserAppKeys to signal that a key was
// consumed without producing a follow-up message.  Using a shared variable
// avoids allocating a new closure on every handled keystroke.
var handledCmd tea.Cmd = func() tea.Msg { return handledNoop }

// BuildSucceededMsg is sent when a BuildView completes with autoRun=true.
// The App handles it by popping the BuildView and starting the runtime.
type BuildSucceededMsg struct {
	ProjectPath string
}

// MockSucceededMsg is sent when a MockView reaches StateRunning with autoRun=true.
// The App handles it by starting the runtime.
type MockSucceededMsg struct{}

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
		hintBar:       chrome.NewHintBar(BrowserHints),
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

func (a *App) startRun() tea.Cmd {
	projectPath := a.activeTabProjectPath()
	if projectPath == "" {
		return nil
	}

	// Check for existing runtime lock
	projectDir := filepath.Dir(projectPath)
	if lock, err := task.ReadLock(projectDir); err == nil && lock.Alive() {
		Trace("startRun: killing existing runtime PID=%d", lock.PID)
		_ = task.KillByProject(projectDir)
	} else if err == nil {
		_ = task.RemoveLock(projectDir)
	}

	rt := task.NewRunTask(task.RunOptions{
		DeployDir:  docker.ResolveRunDir(projectDir),
		CmdHint:    "-p " + projectPath,
		ProjectDir: projectDir,
	})
	rv := NewRunView(rt)
	a.views.Push(rv)
	return rt.Start()
}

// chordTree returns the chord tree root for the leader key menu.
// Actions are methods on *App, captured as closures.
func (a *App) chordTree() who.ChordNode {
	p := func(fn func() tea.Cmd) func() tea.Cmd { return fn }
	return who.ChordNode{
		Label: "Menu",
		Children: []who.ChordNode{
			{
				Key: "b", Label: "Build & Run",
				Children: []who.ChordNode{
					{Key: "b", Label: "Build",    Action: p(a.actionBuild)},
					{Key: "r", Label: "Run",      Action: p(a.actionRun)},
					{Key: "a", Label: "All",      Action: p(a.actionBuildRun)},
				},
			},
			{
				Key: "c", Label: "Check",
				Children: []who.ChordNode{
					{Key: "c", Label: "Check results", Action: p(a.actionCheck)},
					{Key: "n", Label: "Next error",    Action: p(a.actionCheckNext)},
					{Key: "p", Label: "Prev error",    Action: p(a.actionCheckPrev)},
				},
			},
			{
				Key: "d", Label: "Diagram & Debug",
				Children: []who.ChordNode{
					{Key: "d", Label: "Open diagram", Action: p(a.actionDiagram)},
					{Key: "b", Label: "BSON dump",   Action: p(a.actionBSON)},
				},
			},
			{
				Key: "e", Label: "Execute",
				Children: []who.ChordNode{
					{Key: "e", Label: "Execute MDL",   Action: p(a.actionExec)},
					{Key: "m", Label: "Describe MDL",  Action: p(a.actionDescribe)},
				},
			},
			{
				Key: "f", Label: "Find",
				Children: []who.ChordNode{
					{Key: "f", Label: "Fuzzy jump",    Action: p(a.actionFuzzyJump)},
					{Key: "/", Label: "Filter",        Action: p(a.actionFilter)},
				},
			},
			{
				Key: "t", Label: "Tabs",
				Children: []who.ChordNode{
					{Key: "n", Label: "New tab",         Action: p(a.actionNewTab)},
					{Key: "N", Label: "New tab (pick)",  Action: p(a.actionNewTabPick)},
					{Key: "w", Label: "Close tab",       Action: p(a.actionCloseTab)},
				},
			},
			{
				Key: "v", Label: "View",
				Children: []who.ChordNode{
					{Key: "c", Label: "Compare",     Action: p(a.actionCompare)},
					{Key: "z", Label: "Zen mode",    Action: p(a.actionZen)},
					{Key: "y", Label: "Copy",        Action: p(a.actionCopy)},
				},
			},
			{
				Key: "r", Label: "Refresh",
				Children: []who.ChordNode{
					{Key: "r", Label: "Refresh tree",    Action: p(a.actionRefresh)},
					{Key: "R", Label: "Hard reload",     Action: p(a.actionHardReload)},
				},
			},
			{
				Key: "m", Label: "Mock/Dev",
				Children: []who.ChordNode{
					{Key: "m", Label: "Mock start", Action: p(a.actionMock)},
					{Key: "r", Label: "Mock + Run", Action: p(a.actionMockRun)},
				},
			},
			{
				Key: "x", Label: "Create module", Action: p(a.actionCreateModule),
			},
			{
				Key: "s", Label: "Settings",
				Children: []who.ChordNode{
					{Key: "s", Label: "Project Config", Action: p(a.actionConfigView)},
				},
			},
			{
				Key: "D", Label: "Delete", Action: p(a.actionDelete),
			},
		},
	}
}

// --- Chord action methods ---

func (a *App) actionBuild() tea.Cmd {
	projectPath := a.activeTabProjectPath()
	pv := ""
	tab := a.activeTabPtr()
	if tab != nil && len(tab.AllNodes) > 0 {
		for _, n := range tab.AllNodes {
			if n.Type == "Project" {
				pv = n.Label
				break
			}
		}
	}
	Trace("actionBuild: project=%q version=%q useDeployLayout=true", projectPath, pv)
	bt := task.NewBuildTask(task.BuildOptions{
		ProjectPath: projectPath,
	})
	bv := NewBuildView(bt)
	a.views.Push(bv)
	Trace("actionBuild: BuildView pushed, starting task")
	return bt.Start()
}

func (a *App) actionRun() tea.Cmd {
	return a.startRun()
}

func (a *App) actionMock() tea.Cmd {
	projectPath := a.activeTabProjectPath()
	if projectPath == "" {
		return nil
	}
	projectDir := filepath.Dir(projectPath)
	projectRoot := filepath.Dir(projectDir)
	specPath := filepath.Join(projectRoot, "docs", "openapi", "c01-api.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		specPath = filepath.Join(projectDir, "docs", "openapi", "c01-api.yaml")
	}
	mt := task.NewMockTask(task.MockOptions{
		SpecPath: specPath,
		Port:     4000,
		Host:     "0.0.0.0",
	})
	mv := NewMockView(mt)
	a.views.Push(mv)
	return mt.Start()
}

func (a *App) actionMockRun() tea.Cmd {
	projectPath := a.activeTabProjectPath()
	if projectPath == "" {
		return nil
	}
	projectDir := filepath.Dir(projectPath)
	projectRoot := filepath.Dir(projectDir)
	specPath := filepath.Join(projectRoot, "docs", "openapi", "c01-api.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		specPath = filepath.Join(projectDir, "docs", "openapi", "c01-api.yaml")
	}
	mt := task.NewMockTask(task.MockOptions{
		SpecPath: specPath,
		Port:     4000,
		Host:     "0.0.0.0",
	})
	mv := NewMockView(mt)
	mv.autoRun = true
	a.views.Push(mv)
	return mt.Start()
}

func (a *App) actionBuildRun() tea.Cmd {
	bt := task.NewBuildTask(task.BuildOptions{
		ProjectPath: a.activeTabProjectPath(),
	})
	bv := NewBuildView(bt)
	bv.autoRun = true
	a.views.Push(bv)
	return bt.Start()
}

func (a *App) actionCheck() tea.Cmd {
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
	return nil
}

func (a *App) actionCheckNext() tea.Cmd {
	return a.navigateCheckError(1)
}

func (a *App) actionCheckPrev() tea.Cmd {
	return a.navigateCheckError(-1)
}

func (a *App) navigateCheckError(dir int) tea.Cmd {
	if len(a.checkErrors) == 0 {
		return nil
	}
	if !a.checkNavActive {
		a.checkNavActive = true
		a.checkNavLocations = extractCheckNavLocations(filterCheckErrors(a.checkErrors, "all"))
		a.checkNavIndex = -1
	}
	a.checkNavIndex += dir
	if a.checkNavIndex >= len(a.checkNavLocations) {
		a.checkNavIndex = 0
	} else if a.checkNavIndex < 0 {
		a.checkNavIndex = len(a.checkNavLocations) - 1
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
	return nil
}

func (a *App) actionDiagram() tea.Cmd {
	if bv, ok := a.views.Base().(BrowserView); ok {
		if node := bv.miller.SelectedNode(); node != nil && node.QualifiedName != "" {
			return bv.openDiagram(node.Type, node.QualifiedName)
		}
	}
	return nil
}

func (a *App) actionBSON() tea.Cmd {
	if bv, ok := a.views.Base().(BrowserView); ok {
		if node := bv.miller.SelectedNode(); node != nil && node.QualifiedName != "" {
			if bsonType := inferBsonType(node.Type); bsonType != "" {
				return bv.runBsonOverlay(bsonType, node.QualifiedName, node.Type)
			}
		}
	}
	return nil
}

func (a *App) actionExec() tea.Cmd {
	ev := NewExecView(a.mxcliPath, a.activeTabProjectPath(), a.width, a.height)
	a.views.Push(ev)
	return nil
}

func (a *App) actionDescribe() tea.Cmd {
	if bv, ok := a.views.Base().(BrowserView); ok {
		if node := bv.miller.SelectedNode(); node != nil && node.QualifiedName != "" {
			if bv.miller.preview.content != "" {
				raw := stripAnsi(bv.miller.preview.content)
				return func() tea.Msg { return OpenExecWithContentMsg{Content: raw} }
			}
			mdlCmd := buildDescribeCmd(node.Type, node.QualifiedName)
			if mdlCmd == "" {
				return nil
			}
			return func() tea.Msg {
				out, _ := runMxcli(a.mxcliPath, "-p", a.activeTabProjectPath(), "-c", mdlCmd)
				out = StripBanner(out)
				return OpenExecWithContentMsg{Content: out}
			}
		}
	}
	return nil
}

func (a *App) actionFuzzyJump() tea.Cmd {
	tab := a.activeTabPtr()
	if tab == nil {
		return nil
	}
	items := flattenQualifiedNames(tab.AllNodes)
	jumper := NewJumperView(items, a.width, a.height)
	a.views.Push(jumper)
	return nil
}

func (a *App) actionFilter() tea.Cmd {
	return nil // forwarded to miller as /
}

func (a *App) actionNewTab() tea.Cmd {
	tab := a.activeTabPtr()
	if tab == nil {
		return nil
	}
	newTab := tab.CloneTab(a.nextTabID, a.previewEngine)
	a.nextTabID++
	a.tabs = append(a.tabs, newTab)
	a.activeTab = len(a.tabs) - 1
	a.syncBrowserView()
	a.syncTabBar()
	return nil
}

func (a *App) actionNewTabPick() tea.Cmd {
	p := NewEmbeddedPicker()
	p.width = a.width
	p.height = a.height
	a.picker = &p
	return nil
}

func (a *App) actionCloseTab() tea.Cmd {
	if len(a.tabs) > 1 {
		a.tabs[a.activeTab].Miller.previewEngine.Cancel()
		a.tabs = append(a.tabs[:a.activeTab], a.tabs[a.activeTab+1:]...)
		if a.activeTab >= len(a.tabs) {
			a.activeTab = len(a.tabs) - 1
		}
		a.syncBrowserView()
		a.syncTabBar()
	}
	return nil
}

func (a *App) actionCompare() tea.Cmd {
	cv := NewCompareView()
	cv.mxcliPath = a.mxcliPath
	cv.projectPath = a.activeTabProjectPath()
	cv.Show(CompareNDSLMDL, a.width, a.height)
	tab := a.activeTabPtr()
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
	return nil
}

func (a *App) actionZen() tea.Cmd {
	if bv, ok := a.views.Base().(BrowserView); ok {
		bv.miller.zenMode = !bv.miller.zenMode
		bv.miller.relayout()
		a.views.SetBase(bv)
	}
	return nil
}

func (a *App) actionCopy() tea.Cmd {
	if bv, ok := a.views.Base().(BrowserView); ok {
		if bv.miller.preview.content != "" {
			raw := stripAnsi(bv.miller.preview.content)
			_ = writeClipboard(raw)
		}
	}
	return nil
}

func (a *App) actionRefresh() tea.Cmd {
	return a.Init()
}

func (a *App) actionHardReload() tea.Cmd {
	projectPath := a.activeTabProjectPath()
	if projectPath == "" {
		return nil
	}
	cv := NewDiscardConfirmView(projectPath, a.mxcliPath)
	a.views.Push(cv)
	return nil
}

func (a *App) actionConfigView() tea.Cmd {
	mxcliPath := a.mxcliPath
	projectPath := a.activeTabProjectPath()
	if projectPath == "" {
		return nil
	}
	cv := NewConfigView(mxcliPath, projectPath)
	a.views.Push(cv)
	return cv.loadCmd()
}

func (a *App) actionCreateModule() tea.Cmd {
	mxcliPath := a.mxcliPath
	projectPath := a.activeTabProjectPath()
	iv := NewInputView("Create Module", "Module name: ", func(name string) tea.Cmd {
		return func() tea.Msg {
			out, err := runMxcli(mxcliPath, "-p", projectPath, "-c", "CREATE MODULE "+name)
			return execShowResultMsg{Content: out, Success: err == nil}
		}
	})
	a.views.Push(iv)
	return nil
}

func (a *App) actionDelete() tea.Cmd {
	if bv, ok := a.views.Base().(BrowserView); ok {
		if node := bv.miller.SelectedNode(); node != nil && node.QualifiedName != "" {
			dropCmd := buildDropCmd(node.Type, node.QualifiedName)
			if dropCmd == "" {
				return nil
			}
			msg := buildDeleteMessage(node.Type, node.QualifiedName)
			cv := NewConfirmView("Delete", msg, dropCmd, bv.mxcliPath, bv.projectPath)
			return func() tea.Msg { return PushViewMsg{View: cv} }
		}
	}
	return nil
}

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

func isNavigationKey(key string) bool {
	switch key {
	case "j", "k", "g", "G", "h", "l", "left", "right", "up", "down",
		"enter", "tab", "/", "n", "N":
		return true
	}
	return false
}

func (a *App) executeChord(chord string) tea.Cmd {
	root := a.chordTree()
	flat := who.BuildFlatIndex(root, "")
	node, ok := flat[chord]
	if !ok || node.Action == nil {
		Trace("executeChord: chord=%q not found or no action", chord)
		return nil
	}
	Trace("executeChord: chord=%q action=%s", chord, node.Label)
	return node.Action()
}


