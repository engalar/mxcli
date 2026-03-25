package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// chromeHeight is the vertical space consumed by tab bar (1) + hint bar (1) + status bar (1).
const chromeHeight = 3

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

	tabBar        TabBar
	hintBar       HintBar
	statusBar     StatusBar
	previewEngine *PreviewEngine

	watcher       *Watcher
	checkErrors   []CheckError // nil = no check run yet, empty = pass
	checkRunning  bool
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
		tabBar:        NewTabBar(nil),
		statusBar:     NewStatusBar(),
		hintBar:       NewHintBar(ListBrowsingHints),
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
	infos := make([]TabInfo, len(a.tabs))
	for i, t := range a.tabs {
		infos[i] = TabInfo{ID: t.ID, Label: t.Label, Active: i == a.activeTab}
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
	bv.compareItems = flattenQualifiedNames(tab.AllNodes)
	// Ensure miller has current dimensions so scroll calculations in
	// Update() work correctly (Render operates on a value copy).
	if a.height > 0 {
		contentH := max(5, a.height-chromeHeight)
		bv.miller.SetSize(a.width, contentH)
	}
	a.views.base = bv
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

// --- Update ---

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// --- ViewStack navigation ---
	case PushViewMsg:
		a.views.Push(msg.View)
		return a, nil
	case PopViewMsg:
		a.views.Pop()
		return a, nil

	// --- View creation messages ---
	case OpenOverlayMsg:
		ov := NewOverlayView(msg.Title, msg.Content, a.width, a.height, OverlayViewOpts{})
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
		if bv, ok := a.views.base.(BrowserView); ok {
			cmd := bv.navigateToNode(msg.QName)
			a.views.base = bv
			if tab := a.activeTabPtr(); tab != nil {
				tab.Miller = bv.miller
				tab.UpdateLabel()
				a.syncTabBar()
			}
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

	case execShowResultMsg:
		// Pop the ExecView
		a.views.Pop()
		// Show result in overlay
		content := DetectAndHighlight(msg.Content)
		ov := NewOverlayView("Exec Result", content, a.width, a.height, OverlayViewOpts{})
		a.views.Push(ov)
		// If execution succeeded, suppress watcher (self-modification) and refresh tree
		if msg.Success {
			if a.watcher != nil {
				a.watcher.Suppress(2 * time.Second)
			}
			return a, a.Init()
		}
		return a, nil

	case tea.KeyMsg:
		Trace("app: key=%q picker=%v mode=%v help=%v", msg.String(), a.picker != nil, a.views.Active().Mode(), a.showHelp)
		if msg.String() == "ctrl+c" {
			if a.watcher != nil {
				a.watcher.Close()
			}
			return a, tea.Quit
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

		// Tab bar clicks (row 0) — only when in browser mode
		if msg.Y == 0 && a.views.Active().Mode() == ModeBrowser &&
			msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if clickMsg := a.tabBar.HandleClick(msg.X); clickMsg != nil {
				if tc, ok := clickMsg.(TabClickMsg); ok {
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

		// Offset Y by -1 for tab bar when in browser mode
		if a.views.Active().Mode() == ModeBrowser {
			offsetMsg := tea.MouseMsg{
				X: msg.X, Y: msg.Y - 1,
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
			contentH := a.height - chromeHeight
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
				if bv, ok := a.views.base.(BrowserView); ok {
					bv.allNodes = msg.Nodes
					bv.compareItems = flattenQualifiedNames(msg.Nodes)
					bv.miller = tab.Miller
					if a.height > 0 {
						contentH := max(5, a.height-chromeHeight)
						bv.miller.SetSize(a.width, contentH)
					}
					a.views.base = bv
				}
			}
		}
		return a, nil

	case MprChangedMsg:
		Trace("app: MprChangedMsg — refreshing tree and running mx check")
		a.previewEngine.ClearCache()
		projectPath := a.activeTabProjectPath()
		return a, tea.Batch(a.Init(), runMxCheck(projectPath))

	case MxCheckStartMsg:
		a.checkRunning = true
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

// handleBrowserAppKeys handles keys that App intercepts when in Browser mode.
// Returns a non-nil tea.Cmd if the key was handled, nil if the key should
// be forwarded to the active view.
func (a *App) handleBrowserAppKeys(msg tea.KeyMsg) tea.Cmd {
	tab := a.activeTabPtr()

	switch msg.String() {
	case "q":
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
		return func() tea.Msg { return nil }

	case "T":
		p := NewEmbeddedPicker()
		p.width = a.width
		p.height = a.height
		a.picker = &p
		return func() tea.Msg { return nil }

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
		return func() tea.Msg { return nil }

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0]-'0') - 1
		if idx >= 0 && idx < len(a.tabs) {
			a.activeTab = idx
			a.syncBrowserView()
			a.syncTabBar()
		}
		return func() tea.Msg { return nil }

	case "[":
		if a.activeTab > 0 {
			a.activeTab--
			a.syncBrowserView()
			a.syncTabBar()
		}
		return func() tea.Msg { return nil }

	case "]":
		if a.activeTab < len(a.tabs)-1 {
			a.activeTab++
			a.syncBrowserView()
			a.syncTabBar()
		}
		return func() tea.Msg { return nil }

	case "r":
		return a.Init()

	case " ":
		if tab != nil {
			items := flattenQualifiedNames(tab.AllNodes)
			jumper := NewJumperView(items, a.width, a.height)
			a.views.Push(jumper)
		}
		return func() tea.Msg { return nil }

	case "x":
		ev := NewExecView(a.mxcliPath, a.activeTabProjectPath(), a.width, a.height)
		a.views.Push(ev)
		return func() tea.Msg { return nil }

	case "!", "\\!":
		content := renderCheckResults(a.checkErrors)
		ov := NewOverlayView("mx check", content, a.width, a.height, OverlayViewOpts{HideLineNumbers: true})
		a.views.Push(ov)
		return func() tea.Msg { return nil }

	case "c":
		cv := NewCompareView()
		cv.mxcliPath = a.mxcliPath
		cv.projectPath = a.activeTabProjectPath()
		cv.Show(CompareNDSL, a.width, a.height)
		if tab != nil {
			cv.SetItems(flattenQualifiedNames(tab.AllNodes))
			if node := tab.Miller.SelectedNode(); node != nil && node.QualifiedName != "" {
				cv.SetLoading(CompareFocusLeft)
				a.views.Push(cv)
				return cv.loadBsonNDSL(node.QualifiedName, node.Type, CompareFocusLeft)
			}
		}
		a.views.Push(cv)
		return func() tea.Msg { return nil }
	}

	return nil
}

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

// --- View ---

func (a App) View() string {
	if a.width == 0 {
		return "mxcli tui — loading...\n\nPress q to quit"
	}

	if a.picker != nil {
		return a.picker.View()
	}

	active := a.views.Active()

	// For non-browser views, delegate rendering entirely
	if active.Mode() != ModeBrowser {
		contentH := a.height
		content := active.Render(a.width, contentH)

		if a.showHelp {
			helpView := renderHelp(a.width, a.height)
			content = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, helpView,
				lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
		}

		return content
	}

	// Browser mode: App renders chrome (tab bar, hint bar, status bar)

	// Tab bar (line 1) with mode badge + context summary on the right
	tabLine := a.tabBar.View(a.width)
	tab := a.activeTabPtr()
	if tab != nil {
		modeBadge := AccentStyle.Render(active.Mode().String())
		summary := renderContextSummary(tab.AllNodes)
		rightSide := modeBadge
		if summary != "" {
			rightSide += BreadcrumbDimStyle.Render(" │ ") + BreadcrumbDimStyle.Render(summary)
		}
		rightWidth := lipgloss.Width(rightSide) + 1 // 1 char right padding
		tabWidth := lipgloss.Width(tabLine)
		if tabWidth+rightWidth <= a.width {
			// Replace trailing spaces with gap + right side
			trimmed := strings.TrimRight(tabLine, " ")
			trimmedWidth := lipgloss.Width(trimmed)
			gap := a.width - trimmedWidth - rightWidth
			if gap < 2 {
				gap = 2
			}
			tabLine = trimmed + strings.Repeat(" ", gap) + rightSide + " "
		}
	}

	// Content area
	contentH := a.height - chromeHeight
	if contentH < 5 {
		contentH = 5
	}
	content := active.Render(a.width, contentH)

	// Hint bar — declarative from active view
	a.hintBar.SetHints(active.Hints())
	hintLine := a.hintBar.View(a.width)

	// Status bar — declarative from active view
	info := active.StatusInfo()
	a.statusBar.SetBreadcrumb(info.Breadcrumb)
	a.statusBar.SetPosition(info.Position)
	a.statusBar.SetMode(info.Mode)
	a.statusBar.SetCheckBadge(formatCheckBadge(a.checkErrors, a.checkRunning))
	viewModeNames := a.collectViewModeNames()
	a.statusBar.SetViewDepth(a.views.Depth(), viewModeNames)
	statusLine := StatusBarStyle.Width(a.width).Render(a.statusBar.View(a.width))

	rendered := tabLine + "\n" + content + "\n" + hintLine + "\n" + statusLine

	if a.showHelp {
		helpView := renderHelp(a.width, a.height)
		rendered = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, helpView,
			lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
	}

	return rendered
}

// --- Load helpers ---

func buildDiagramHTML(elkJSON, nodeType, qualifiedName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><title>%s %s</title>
<script src="https://cdn.jsdelivr.net/npm/elkjs@0.9.3/lib/elk.bundled.js"></script>
<style>body{margin:0;background:#1e1e2e;color:#cdd6f4;font-family:monospace}svg{width:100vw;height:100vh}</style>
</head><body><div id="diagram"></div><script>
const elkData = %s;
const ELK = new ELKConstructor();
ELK.layout(elkData).then(graph=>{
  const svg=document.createElementNS("http://www.w3.org/2000/svg","svg");
  document.getElementById("diagram").appendChild(svg);
});
</script></body></html>`, nodeType, qualifiedName, elkJSON)
}

// CmdResultMsg carries output from any mxcli command.
type CmdResultMsg struct {
	Output string
	Err    error
}

// renderContextSummary counts top-level node types and returns a compact summary.
func renderContextSummary(nodes []*TreeNode) string {
	if len(nodes) == 0 {
		return ""
	}
	counts := map[string]int{}
	for _, n := range nodes {
		counts[n.Type]++
	}
	// Display in a predictable order
	order := []struct {
		key    string
		plural string
	}{
		{"Module", "modules"},
		{"Entity", "entities"},
		{"Microflow", "microflows"},
		{"Page", "pages"},
		{"Nanoflow", "nanoflows"},
		{"Enumeration", "enumerations"},
	}
	var parts []string
	used := map[string]bool{}
	for _, o := range order {
		if c, ok := counts[o.key]; ok {
			parts = append(parts, fmt.Sprintf("%d %s", c, o.plural))
			used[o.key] = true
		}
	}
	// Add remaining types not in the predefined order
	for k, c := range counts {
		if !used[k] {
			parts = append(parts, fmt.Sprintf("%d %s", c, strings.ToLower(k)+"s"))
		}
	}
	if len(parts) > 3 {
		parts = parts[:3]
	}
	return strings.Join(parts, ", ")
}

// collectViewModeNames returns the mode names for all views in the stack.
func (a App) collectViewModeNames() []string {
	names := []string{a.views.base.Mode().String()}
	for _, v := range a.views.stack {
		names = append(names, v.Mode().String())
	}
	return names
}

// inferBsonType maps tree node types to valid bson object types.
func inferBsonType(nodeType string) string {
	switch strings.ToLower(nodeType) {
	case "page", "microflow", "nanoflow", "workflow",
		"enumeration", "snippet", "layout", "entity", "association",
		"imagecollection", "javaaction":
		return strings.ToLower(nodeType)
	default:
		return ""
	}
}
