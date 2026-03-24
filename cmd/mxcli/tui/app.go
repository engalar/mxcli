package tui

import (
	"fmt"
	"os"
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

	overlay  Overlay
	compare  CompareView
	showHelp bool
	picker   *PickerModel // non-nil when cross-project picker is open

	// Overlay switch state
	overlayQName    string
	overlayNodeType string
	overlayIsNDSL   bool

	tabBar        TabBar
	statusBar     StatusBar
	hintBar       HintBar
	previewEngine *PreviewEngine
}

// NewApp creates the root App model.
func NewApp(mxcliPath, projectPath string) App {
	initTrace()
	Trace("app: NewApp mxcli=%q project=%q", mxcliPath, projectPath)

	engine := NewPreviewEngine(mxcliPath, projectPath)
	tab := NewTab(1, projectPath, engine, nil)

	app := App{
		mxcliPath:     mxcliPath,
		nextTabID:     2,
		overlay:       NewOverlay(),
		compare:       NewCompareView(),
		tabBar:        NewTabBar(nil),
		statusBar:     NewStatusBar(),
		hintBar:       NewHintBar(ListBrowsingHints),
		previewEngine: engine,
	}
	app.tabs = []Tab{tab}
	app.syncTabBar()
	return app
}

func (a *App) activeTabPtr() *Tab {
	if a.activeTab >= 0 && a.activeTab < len(a.tabs) {
		return &a.tabs[a.activeTab]
	}
	return nil
}

func (a *App) syncTabBar() {
	infos := make([]TabInfo, len(a.tabs))
	for i, t := range a.tabs {
		infos[i] = TabInfo{ID: t.ID, Label: t.Label, Active: i == a.activeTab}
	}
	a.tabBar.SetTabs(infos)
}

func (a *App) syncStatusBar() {
	tab := a.activeTabPtr()
	if tab == nil {
		return
	}
	crumbs := tab.Miller.Breadcrumb()
	a.statusBar.SetBreadcrumb(crumbs)

	mode := "MDL"
	if tab.Miller.preview.mode == PreviewNDSL {
		mode = "NDSL"
	}
	a.statusBar.SetMode(mode)

	col := tab.Miller.current
	pos := fmt.Sprintf("%d/%d", col.cursor+1, col.ItemCount())
	a.statusBar.SetPosition(pos)
}

func (a *App) syncHintBar() {
	if a.overlay.IsVisible() {
		a.hintBar.SetHints(OverlayHints)
	} else if a.compare.IsVisible() {
		a.hintBar.SetHints(CompareHints)
	} else {
		tab := a.activeTabPtr()
		if tab != nil && tab.Miller.focusedColumn().IsFilterActive() {
			a.hintBar.SetHints(FilterActiveHints)
		} else {
			a.hintBar.SetHints(ListBrowsingHints)
		}
	}
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
			a.resizeAll()
			a.syncTabBar()
			a.syncStatusBar()
			a.syncHintBar()
			// Load project tree for new tab
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
		a.syncHintBar()
		return a, nil

	case OpenOverlayMsg:
		a.overlay.Show(msg.Title, msg.Content, a.width, a.height)
		a.syncHintBar()
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

	case CompareLoadMsg:
		a.compare.SetContent(msg.Side, msg.Title, msg.NodeType, msg.Content)
		return a, nil

	case ComparePickMsg:
		a.compare.SetLoading(msg.Side)
		return a, a.loadForCompare(msg.QName, msg.NodeType, msg.Side, msg.Kind)

	case CompareReloadMsg:
		var cmds []tea.Cmd
		if a.compare.left.qname != "" {
			a.compare.SetLoading(CompareFocusLeft)
			cmds = append(cmds, a.loadForCompare(a.compare.left.qname, a.compare.left.nodeType, CompareFocusLeft, msg.Kind))
		}
		if a.compare.right.qname != "" {
			a.compare.SetLoading(CompareFocusRight)
			cmds = append(cmds, a.loadForCompare(a.compare.right.qname, a.compare.right.nodeType, CompareFocusRight, msg.Kind))
		}
		return a, tea.Batch(cmds...)

	case overlayFlashClearMsg:
		a.overlay.copiedFlash = false
		return a, nil

	case compareFlashClearMsg:
		a.compare.copiedFlash = false
		return a, nil

	case tea.KeyMsg:
		Trace("app: key=%q picker=%v overlay=%v compare=%v help=%v", msg.String(), a.picker != nil, a.overlay.IsVisible(), a.compare.IsVisible(), a.showHelp)
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		// Picker modal
		if a.picker != nil {
			result, cmd := a.picker.Update(msg)
			p := result.(PickerModel)
			a.picker = &p
			return a, cmd
		}

		// Fullscreen modes
		if a.compare.IsVisible() {
			var cmd tea.Cmd
			a.compare, cmd = a.compare.Update(msg)
			if !a.compare.IsVisible() {
				a.syncHintBar()
			}
			return a, cmd
		}
		if a.overlay.IsVisible() {
			if msg.String() == "tab" && a.overlayQName != "" && !a.overlay.content.IsSearching() {
				a.overlayIsNDSL = !a.overlayIsNDSL
				if a.overlayIsNDSL {
					bsonType := inferBsonType(a.overlayNodeType)
					return a, a.runBsonOverlay(bsonType, a.overlayQName)
				}
				return a, a.runMDLOverlay(a.overlayNodeType, a.overlayQName)
			}
			var cmd tea.Cmd
			a.overlay, cmd = a.overlay.Update(msg)
			if !a.overlay.IsVisible() {
				a.syncHintBar()
			}
			return a, cmd
		}
		if a.showHelp {
			a.showHelp = false
			return a, nil
		}

		return a.updateNormalMode(msg)

	case tea.MouseMsg:
		Trace("app: mouse x=%d y=%d btn=%v action=%v", msg.X, msg.Y, msg.Button, msg.Action)
		if a.picker != nil || a.compare.IsVisible() || a.overlay.IsVisible() {
			return a, nil
		}
		// Tab bar clicks (row 0)
		if msg.Y == 0 && msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if clickMsg := a.tabBar.HandleClick(msg.X); clickMsg != nil {
				if tc, ok := clickMsg.(TabClickMsg); ok {
					a.switchToTabByID(tc.ID)
					return a, nil
				}
			}
		}
		// Forward to Miller (offset Y by -1 for tab bar)
		tab := a.activeTabPtr()
		if tab != nil {
			millerMsg := tea.MouseMsg{
				X: msg.X, Y: msg.Y - 1,
				Button: msg.Button, Action: msg.Action,
			}
			var cmd tea.Cmd
			tab.Miller, cmd = tab.Miller.Update(millerMsg)
			a.syncStatusBar()
			return a, cmd
		}

	case tea.WindowSizeMsg:
		Trace("app: resize %dx%d", msg.Width, msg.Height)
		a.width = msg.Width
		a.height = msg.Height
		a.resizeAll()
		return a, nil

	case LoadTreeMsg:
		Trace("app: LoadTreeMsg tabID=%d err=%v nodes=%d", msg.TabID, msg.Err, len(msg.Nodes))
		if msg.Err == nil && msg.Nodes != nil {
			tab := a.findTabByID(msg.TabID)
			if tab != nil {
				tab.AllNodes = msg.Nodes
				tab.Miller.SetRootNodes(msg.Nodes)
				a.compare.SetItems(flattenQualifiedNames(msg.Nodes))
				a.syncStatusBar()
				a.syncTabBar()
			}
		}

	case PreviewReadyMsg, PreviewLoadingMsg, CursorChangedMsg, animTickMsg:
		tab := a.activeTabPtr()
		if tab != nil {
			var cmd tea.Cmd
			tab.Miller, cmd = tab.Miller.Update(msg)
			a.syncStatusBar()
			return a, cmd
		}

	case CmdResultMsg:
		content := msg.Output
		if msg.Err != nil {
			content = "-- Error:\n" + msg.Output
		}
		a.overlayQName = ""
		a.overlay.switchable = false
		a.overlay.Show("Result", DetectAndHighlight(content), a.width, a.height)
		a.syncHintBar()
	}
	return a, nil
}

func (a App) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tab := a.activeTabPtr()

	// If filter is active, forward to miller
	if tab != nil && tab.Miller.focusedColumn().IsFilterActive() {
		var cmd tea.Cmd
		tab.Miller, cmd = tab.Miller.Update(msg)
		a.syncHintBar()
		a.syncStatusBar()
		return a, cmd
	}

	switch msg.String() {
	case "q":
		for i := range a.tabs {
			a.tabs[i].Miller.previewEngine.Cancel()
		}
		CloseTrace()
		return a, tea.Quit
	case "?":
		a.showHelp = !a.showHelp
		return a, nil

	// Tab management
	case "t":
		if tab != nil {
			newTab := tab.CloneTab(a.nextTabID, a.previewEngine)
			a.nextTabID++
			a.tabs = append(a.tabs, newTab)
			a.activeTab = len(a.tabs) - 1
			a.syncTabBar()
			a.syncStatusBar()
		}
		return a, nil
	case "T":
		p := NewEmbeddedPicker()
		p.width = a.width
		p.height = a.height
		a.picker = &p
		return a, nil
	case "W":
		if len(a.tabs) > 1 {
			a.tabs[a.activeTab].Miller.previewEngine.Cancel()
			a.tabs = append(a.tabs[:a.activeTab], a.tabs[a.activeTab+1:]...)
			if a.activeTab >= len(a.tabs) {
				a.activeTab = len(a.tabs) - 1
			}
			a.resizeAll()
			a.syncTabBar()
			a.syncStatusBar()
		}
		return a, nil
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0]-'0') - 1
		if idx >= 0 && idx < len(a.tabs) {
			a.activeTab = idx
			a.resizeAll()
			a.syncTabBar()
			a.syncStatusBar()
		}
		return a, nil
	case "[":
		if a.activeTab > 0 {
			a.activeTab--
			a.resizeAll()
			a.syncTabBar()
			a.syncStatusBar()
		}
		return a, nil
	case "]":
		if a.activeTab < len(a.tabs)-1 {
			a.activeTab++
			a.resizeAll()
			a.syncTabBar()
			a.syncStatusBar()
		}
		return a, nil

	// Actions on selected node
	case "b":
		if tab != nil {
			if node := tab.Miller.SelectedNode(); node != nil && node.QualifiedName != "" {
				if bsonType := inferBsonType(node.Type); bsonType != "" {
					a.overlayQName = node.QualifiedName
					a.overlayNodeType = node.Type
					a.overlayIsNDSL = true
					a.overlay.switchable = true
					return a, a.runBsonOverlay(bsonType, node.QualifiedName)
				}
			}
		}
	case "m":
		if tab != nil {
			if node := tab.Miller.SelectedNode(); node != nil && node.QualifiedName != "" {
				a.overlayQName = node.QualifiedName
				a.overlayNodeType = node.Type
				a.overlayIsNDSL = false
				a.overlay.switchable = true
				return a, a.runMDLOverlay(node.Type, node.QualifiedName)
			}
		}
	case "c":
		a.compare.Show(CompareNDSL, a.width, a.height)
		if tab != nil {
			a.compare.SetItems(flattenQualifiedNames(tab.AllNodes))
			if node := tab.Miller.SelectedNode(); node != nil && node.QualifiedName != "" {
				a.compare.SetLoading(CompareFocusLeft)
				a.syncHintBar()
				return a, a.loadBsonNDSL(node.QualifiedName, node.Type, CompareFocusLeft)
			}
		}
		a.syncHintBar()
		return a, nil
	case "d":
		if tab != nil {
			if node := tab.Miller.SelectedNode(); node != nil && node.QualifiedName != "" {
				return a, a.openDiagram(node.Type, node.QualifiedName)
			}
		}
	case "y":
		// Copy preview content to clipboard
		if tab != nil && tab.Miller.preview.content != "" {
			raw := stripAnsi(tab.Miller.preview.content)
			_ = writeClipboard(raw)
		}
		return a, nil
	case "r":
		return a, a.Init()
	}

	// Forward to Miller
	if tab != nil {
		var cmd tea.Cmd
		tab.Miller, cmd = tab.Miller.Update(msg)
		tab.UpdateLabel()
		a.syncTabBar()
		a.syncStatusBar()
		a.syncHintBar()
		return a, cmd
	}
	return a, nil
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
			a.resizeAll()
			a.syncTabBar()
			a.syncStatusBar()
			return
		}
	}
}

func (a *App) resizeAll() {
	if a.width == 0 || a.height == 0 {
		return
	}
	millerH := a.height - chromeHeight
	if millerH < 5 {
		millerH = 5
	}
	tab := a.activeTabPtr()
	if tab != nil {
		tab.Miller.SetSize(a.width, millerH)
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
	if a.compare.IsVisible() {
		return a.compare.View()
	}
	if a.overlay.IsVisible() {
		return a.overlay.View()
	}

	a.syncStatusBar()

	// Tab bar (line 1)
	tabLine := a.tabBar.View(a.width)

	// Miller columns (main area)
	millerH := a.height - chromeHeight
	if millerH < 5 {
		millerH = 5
	}
	tab := a.activeTabPtr()
	var millerView string
	if tab != nil {
		tab.Miller.SetSize(a.width, millerH)
		millerView = tab.Miller.View()
	} else {
		millerView = strings.Repeat("\n", millerH-1)
	}

	// Hint bar + Status bar (bottom 2 lines)
	hintLine := a.hintBar.View(a.width)
	statusLine := StatusBarStyle.Width(a.width).Render(a.statusBar.View(a.width))

	rendered := tabLine + "\n" + millerView + "\n" + hintLine + "\n" + statusLine

	if a.showHelp {
		helpView := renderHelp(a.width, a.height)
		rendered = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, helpView,
			lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
	}

	return rendered
}

// --- Load helpers (ported from old model.go) ---

func (a App) openDiagram(nodeType, qualifiedName string) tea.Cmd {
	tab := a.activeTabPtr()
	if tab == nil {
		return nil
	}
	mxcliPath := a.mxcliPath
	projectPath := tab.ProjectPath
	return func() tea.Msg {
		out, err := runMxcli(mxcliPath, "describe", "-p", projectPath,
			"--format", "elk", nodeType, qualifiedName)
		if err != nil {
			return CmdResultMsg{Output: out, Err: err}
		}
		htmlContent := buildDiagramHTML(out, nodeType, qualifiedName)
		tmpFile, err := os.CreateTemp("", "mxcli-diagram-*.html")
		if err != nil {
			return CmdResultMsg{Err: err}
		}
		if _, err := tmpFile.WriteString(htmlContent); err != nil {
			tmpFile.Close()
			return CmdResultMsg{Err: fmt.Errorf("writing diagram HTML: %w", err)}
		}
		tmpFile.Close()
		tmpPath := tmpFile.Name()
		openBrowser(tmpPath)
		time.AfterFunc(30*time.Second, func() { os.Remove(tmpPath) })
		return CmdResultMsg{Output: fmt.Sprintf("Opened diagram: %s", tmpPath)}
	}
}

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

func (a App) loadBsonNDSL(qname, nodeType string, side CompareFocus) tea.Cmd {
	tab := a.activeTabPtr()
	if tab == nil {
		return nil
	}
	mxcliPath := a.mxcliPath
	projectPath := tab.ProjectPath
	return func() tea.Msg {
		bsonType := inferBsonType(nodeType)
		if bsonType == "" {
			return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType,
				Content: fmt.Sprintf("Error: type %q not supported for BSON dump", nodeType),
				Err: fmt.Errorf("unsupported type")}
		}
		args := []string{"bson", "dump", "-p", projectPath, "--format", "ndsl",
			"--type", bsonType, "--object", qname}
		out, err := runMxcli(mxcliPath, args...)
		out = StripBanner(out)
		if err != nil {
			return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: "Error: " + out, Err: err}
		}
		return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: HighlightNDSL(out)}
	}
}

func (a App) loadMDL(qname, nodeType string, side CompareFocus) tea.Cmd {
	tab := a.activeTabPtr()
	if tab == nil {
		return nil
	}
	mxcliPath := a.mxcliPath
	projectPath := tab.ProjectPath
	return func() tea.Msg {
		out, err := runMxcli(mxcliPath, "-p", projectPath, "-c", buildDescribeCmd(nodeType, qname))
		out = StripBanner(out)
		if err != nil {
			return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: "Error: " + out, Err: err}
		}
		return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: DetectAndHighlight(out)}
	}
}

func (a App) loadForCompare(qname, nodeType string, side CompareFocus, kind CompareKind) tea.Cmd {
	switch kind {
	case CompareNDSL:
		return a.loadBsonNDSL(qname, nodeType, side)
	case CompareNDSLMDL:
		if side == CompareFocusLeft {
			return a.loadBsonNDSL(qname, nodeType, side)
		}
		return a.loadMDL(qname, nodeType, side)
	case CompareMDL:
		return a.loadMDL(qname, nodeType, side)
	}
	return nil
}

func (a App) runBsonOverlay(bsonType, qname string) tea.Cmd {
	tab := a.activeTabPtr()
	if tab == nil {
		return nil
	}
	mxcliPath := a.mxcliPath
	projectPath := tab.ProjectPath
	return func() tea.Msg {
		args := []string{"bson", "dump", "-p", projectPath, "--format", "ndsl",
			"--type", bsonType, "--object", qname}
		out, err := runMxcli(mxcliPath, args...)
		out = StripBanner(out)
		title := fmt.Sprintf("BSON: %s", qname)
		if err != nil {
			return OpenOverlayMsg{Title: title, Content: "Error: " + out}
		}
		return OpenOverlayMsg{Title: title, Content: HighlightNDSL(out)}
	}
}

func (a App) runMDLOverlay(nodeType, qname string) tea.Cmd {
	tab := a.activeTabPtr()
	if tab == nil {
		return nil
	}
	mxcliPath := a.mxcliPath
	projectPath := tab.ProjectPath
	return func() tea.Msg {
		out, err := runMxcli(mxcliPath, "-p", projectPath, "-c", buildDescribeCmd(nodeType, qname))
		out = StripBanner(out)
		title := fmt.Sprintf("MDL: %s", qname)
		if err != nil {
			return OpenOverlayMsg{Title: title, Content: "Error: " + out}
		}
		return OpenOverlayMsg{Title: title, Content: DetectAndHighlight(out)}
	}
}

// CmdResultMsg carries output from any mxcli command.
type CmdResultMsg struct {
	Output string
	Err    error
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
