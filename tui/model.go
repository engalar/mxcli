package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/tui/panels"
)

// Focus indicates which panel has keyboard focus.
type Focus int

const (
	FocusModules Focus = iota
	FocusElements
)

// CmdResultMsg carries output from any mxcli command.
type CmdResultMsg struct {
	Output string
	Err    error
}

// compareFlashClearMsg is sent 1 s after a clipboard copy in compare view.
type compareFlashClearMsg struct{}

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	mxcliPath     string
	projectPath   string
	width         int
	height        int
	focus         Focus
	modulesPanel  panels.ModulesPanel
	elementsPanel panels.ElementsPanel
	showHelp      bool
	overlay       Overlay
	compare       CompareView

	// Overlay switch state: remembers the node so Tab can toggle NDSL ↔ MDL.
	overlayQName    string
	overlayNodeType string
	overlayIsNDSL   bool

	visibility        PanelVisibility
	zenMode           bool
	zenPrevFocus      Focus
	zenPrevVisibility PanelVisibility
	panelLayout       [2]PanelRect
	allNodes          []*panels.TreeNode
}

func New(mxcliPath, projectPath string) Model {
	panels.HighlightFunc = DetectAndHighlight
	return Model{
		mxcliPath:     mxcliPath,
		projectPath:   projectPath,
		focus:         FocusModules,
		visibility:    ShowOnePanel,
		modulesPanel:  panels.NewModulesPanel(30, 20),
		elementsPanel: panels.NewElementsPanel(40, 20),
		overlay:       NewOverlay(),
		compare:       NewCompareView(),
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		out, err := runMxcli(m.mxcliPath, "project-tree", "-p", m.projectPath)
		if err != nil {
			return panels.LoadTreeMsg{Err: err}
		}
		nodes, parseErr := panels.ParseTree(out)
		return panels.LoadTreeMsg{Nodes: nodes, Err: parseErr}
	}
}

func (m Model) openDiagram(nodeType, qualifiedName string) tea.Cmd {
	return func() tea.Msg {
		out, err := runMxcli(m.mxcliPath, "describe", "-p", m.projectPath,
			"--format", "elk", nodeType, qualifiedName)
		if err != nil {
			return CmdResultMsg{Output: out, Err: err}
		}
		htmlContent := buildDiagramHTML(out, nodeType, qualifiedName)
		tmpFile, err := os.CreateTemp("", "mxcli-diagram-*.html")
		if err != nil {
			return CmdResultMsg{Err: err}
		}
		defer tmpFile.Close()
		tmpFile.WriteString(htmlContent)
		openBrowser(tmpFile.Name())
		return CmdResultMsg{Output: fmt.Sprintf("Opened diagram: %s", tmpFile.Name())}
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

func (m Model) selectedNode() *panels.TreeNode {
	if node := m.elementsPanel.SelectedNode(); node != nil {
		return node
	}
	return m.modulesPanel.SelectedNode()
}

// inferBsonType maps tree node types to valid bson object types.
func inferBsonType(nodeType string) string {
	switch strings.ToLower(nodeType) {
	case "page", "microflow", "nanoflow", "workflow",
		"enumeration", "snippet", "layout", "entity":
		return strings.ToLower(nodeType)
	default:
		return ""
	}
}

// isFilterActive returns true if any panel's filter/search input is active.
func (m Model) isFilterActive() bool {
	return m.modulesPanel.IsFilterActive() || m.elementsPanel.IsFilterActive()
}

// --- Load helpers ---

func (m Model) loadBsonNDSL(qname, nodeType string, side CompareFocus) tea.Cmd {
	return func() tea.Msg {
		bsonType := inferBsonType(nodeType)
		if bsonType == "" {
			return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType,
				Content: fmt.Sprintf("Error: type %q not supported for BSON dump", nodeType),
				Err: fmt.Errorf("unsupported type")}
		}
		args := []string{"bson", "dump", "-p", m.projectPath, "--format", "ndsl",
			"--type", bsonType, "--object", qname}
		out, err := runMxcli(m.mxcliPath, args...)
		out = StripBanner(out)
		if err != nil {
			return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: "Error: " + out, Err: err}
		}
		return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: HighlightNDSL(out)}
	}
}

func (m Model) loadMDL(qname, nodeType string, side CompareFocus) tea.Cmd {
	return func() tea.Msg {
		out, err := runMxcli(m.mxcliPath, "-p", m.projectPath, "-c",
			fmt.Sprintf("DESCRIBE %s %s", strings.ToUpper(nodeType), qname))
		out = StripBanner(out)
		if err != nil {
			return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: "Error: " + out, Err: err}
		}
		return CompareLoadMsg{Side: side, Title: qname, NodeType: nodeType, Content: DetectAndHighlight(out)}
	}
}

func (m Model) loadForCompare(qname, nodeType string, side CompareFocus, kind CompareKind) tea.Cmd {
	switch kind {
	case CompareNDSL:
		return m.loadBsonNDSL(qname, nodeType, side)
	case CompareNDSLMDL:
		if side == CompareFocusLeft {
			return m.loadBsonNDSL(qname, nodeType, side)
		}
		return m.loadMDL(qname, nodeType, side)
	case CompareMDL:
		return m.loadMDL(qname, nodeType, side)
	}
	return nil
}

func (m Model) runBsonOverlay(bsonType, qname string) tea.Cmd {
	return func() tea.Msg {
		args := []string{"bson", "dump", "-p", m.projectPath, "--format", "ndsl",
			"--type", bsonType, "--object", qname}
		out, err := runMxcli(m.mxcliPath, args...)
		out = StripBanner(out)
		title := fmt.Sprintf("BSON: %s", qname)
		if err != nil {
			return panels.OpenOverlayMsg{Title: title, Content: "Error: " + out}
		}
		return panels.OpenOverlayMsg{Title: title, Content: HighlightNDSL(out)}
	}
}

func (m Model) runMDLOverlay(nodeType, qname string) tea.Cmd {
	return func() tea.Msg {
		out, err := runMxcli(m.mxcliPath, "-p", m.projectPath, "-c",
			fmt.Sprintf("DESCRIBE %s %s", strings.ToUpper(nodeType), qname))
		out = StripBanner(out)
		title := fmt.Sprintf("MDL: %s", qname)
		if err != nil {
			return panels.OpenOverlayMsg{Title: title, Content: "Error: " + out}
		}
		return panels.OpenOverlayMsg{Title: title, Content: DetectAndHighlight(out)}
	}
}

// --- Update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case panels.OpenOverlayMsg:
		m.overlay.Show(msg.Title, msg.Content, m.width, m.height)
		return m, nil

	case CompareLoadMsg:
		m.compare.SetContent(msg.Side, msg.Title, msg.NodeType, msg.Content)
		return m, nil

	case ComparePickMsg:
		m.compare.SetLoading(msg.Side)
		return m, m.loadForCompare(msg.QName, msg.NodeType, msg.Side, msg.Kind)

	case CompareReloadMsg:
		var cmds []tea.Cmd
		if m.compare.left.qname != "" {
			m.compare.SetLoading(CompareFocusLeft)
			cmds = append(cmds, m.loadForCompare(m.compare.left.qname, m.compare.left.nodeType, CompareFocusLeft, msg.Kind))
		}
		if m.compare.right.qname != "" {
			m.compare.SetLoading(CompareFocusRight)
			cmds = append(cmds, m.loadForCompare(m.compare.right.qname, m.compare.right.nodeType, CompareFocusRight, msg.Kind))
		}
		return m, tea.Batch(cmds...)

	case overlayFlashClearMsg:
		m.overlay.copiedFlash = false
		return m, nil

	case compareFlashClearMsg:
		m.compare.copiedFlash = false
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Fullscreen modes intercept all keys
		if m.compare.IsVisible() {
			var cmd tea.Cmd
			m.compare, cmd = m.compare.Update(msg)
			return m, cmd
		}
		if m.overlay.IsVisible() {
			// Tab switches between NDSL and MDL when the overlay was opened via b/m.
			if msg.String() == "tab" && m.overlayQName != "" && !m.overlay.content.IsSearching() {
				m.overlayIsNDSL = !m.overlayIsNDSL
				if m.overlayIsNDSL {
					bsonType := inferBsonType(m.overlayNodeType)
					return m, m.runBsonOverlay(bsonType, m.overlayQName)
				}
				return m, m.runMDLOverlay(m.overlayNodeType, m.overlayQName)
			}
			var cmd tea.Cmd
			m.overlay, cmd = m.overlay.Update(msg)
			return m, cmd
		}
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// When filter is active, only allow filter-related keys at global level
		if m.isFilterActive() {
			return m.updateFilterMode(msg)
		}

		return m.updateNormalMode(msg)

	case tea.MouseMsg:
		if m.compare.IsVisible() || m.overlay.IsVisible() {
			return m, nil
		}
		return m.updateMouse(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizePanels()

	case panels.LoadTreeMsg:
		if msg.Err == nil && msg.Nodes != nil {
			m.allNodes = msg.Nodes
			m.modulesPanel.SetNodes(msg.Nodes)
			m.compare.SetItems(flattenQualifiedNames(msg.Nodes))
		}

	case CmdResultMsg:
		content := msg.Output
		if msg.Err != nil {
			content = "-- Error:\n" + msg.Output
		}
		m.overlayQName = ""
		m.overlay.switchable = false
		m.overlay.Show("Result", DetectAndHighlight(content), m.width, m.height)
	}
	return m, nil
}

// updateFilterMode handles keys when a panel's filter input is active.
// All keys go to the focused panel — no global shortcuts.
func (m Model) updateFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.focus {
	case FocusModules:
		var cmd tea.Cmd
		m.modulesPanel, cmd = m.modulesPanel.Update(msg)
		return m, cmd
	case FocusElements:
		var cmd tea.Cmd
		m.elementsPanel, cmd = m.elementsPanel.Update(msg)
		return m, cmd
	}
	return m, nil
}

// updateNormalMode handles keys when no filter is active.
func (m Model) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global shortcuts
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "tab":
		if m.focus == FocusModules {
			m.focus = FocusElements
		} else {
			m.focus = FocusModules
		}
		return m, nil
	case "z":
		if m.zenMode {
			m.zenMode = false
			m.visibility = m.zenPrevVisibility
			m.focus = m.zenPrevFocus
		} else {
			m.zenMode = true
			m.zenPrevFocus = m.focus
			m.zenPrevVisibility = m.visibility
			m.visibility = ShowZoomed
		}
		m.resizePanels()
		return m, nil

	case "b":
		if node := m.selectedNode(); node != nil && node.QualifiedName != "" {
			if bsonType := inferBsonType(node.Type); bsonType != "" {
				m.overlayQName = node.QualifiedName
				m.overlayNodeType = node.Type
				m.overlayIsNDSL = true
				m.overlay.switchable = true
				return m, m.runBsonOverlay(bsonType, node.QualifiedName)
			}
		}
	case "m":
		if node := m.selectedNode(); node != nil && node.QualifiedName != "" {
			m.overlayQName = node.QualifiedName
			m.overlayNodeType = node.Type
			m.overlayIsNDSL = false
			m.overlay.switchable = true
			return m, m.runMDLOverlay(node.Type, node.QualifiedName)
		}
	case "c":
		m.compare.Show(CompareNDSL, m.width, m.height)
		m.compare.SetItems(flattenQualifiedNames(m.allNodes))
		if node := m.selectedNode(); node != nil && node.QualifiedName != "" {
			m.compare.SetLoading(CompareFocusLeft)
			return m, m.loadBsonNDSL(node.QualifiedName, node.Type, CompareFocusLeft)
		}
		return m, nil
	case "d":
		if node := m.selectedNode(); node != nil && node.QualifiedName != "" {
			return m, m.openDiagram(node.Type, node.QualifiedName)
		}
	case "r":
		return m, m.Init()
	}

	// Panel-specific keys
	switch m.focus {
	case FocusModules:
		switch msg.String() {
		case "l", "right", "enter":
			if node := m.modulesPanel.SelectedNode(); node != nil && len(node.Children) > 0 {
				m.elementsPanel.SetNodes(node.Children)
				m.focus = FocusElements
				m.setVisibility(ShowTwoPanels)
				return m, nil
			}
		case "j", "down", "k", "up":
			var cmd tea.Cmd
			m.modulesPanel, cmd = m.modulesPanel.Update(msg)
			if node := m.modulesPanel.SelectedNode(); node != nil && len(node.Children) > 0 {
				m.elementsPanel.SetNodes(node.Children)
				m.setVisibility(ShowTwoPanels)
			}
			return m, cmd
		default:
			var cmd tea.Cmd
			m.modulesPanel, cmd = m.modulesPanel.Update(msg)
			return m, cmd
		}

	case FocusElements:
		switch msg.String() {
		case "h", "left":
			m.focus = FocusModules
			m.setVisibility(ShowOnePanel)
			return m, nil
		case "l", "right", "enter":
			if node := m.elementsPanel.SelectedNode(); node != nil {
				if len(node.Children) > 0 {
					m.elementsPanel.SetNodes(node.Children)
					return m, nil
				}
				// Open MDL describe in overlay
				if node.QualifiedName != "" {
					return m, m.runMDLOverlay(node.Type, node.QualifiedName)
				}
			}
		case "j", "down", "k", "up":
			var cmd tea.Cmd
			m.elementsPanel, cmd = m.elementsPanel.Update(msg)
			return m, cmd
		default:
			var cmd tea.Cmd
			m.elementsPanel, cmd = m.elementsPanel.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	for i, rect := range m.panelLayout {
		if !rect.Visible {
			continue
		}
		if msg.X >= rect.X && msg.X < rect.X+rect.Width &&
			msg.Y >= rect.Y && msg.Y < rect.Y+rect.Height {
			localMsg := tea.MouseMsg{
				X: msg.X - rect.X - 1, Y: msg.Y - rect.Y - 1,
				Button: msg.Button, Action: msg.Action,
			}
			m.focus = Focus(i)
			switch Focus(i) {
			case FocusModules:
				var cmd tea.Cmd
				m.modulesPanel, cmd = m.modulesPanel.Update(localMsg)
				return m, cmd
			case FocusElements:
				var cmd tea.Cmd
				m.elementsPanel, cmd = m.elementsPanel.Update(localMsg)
				return m, cmd
			}
			break
		}
	}
	return m, nil
}

func (m *Model) setVisibility(vis PanelVisibility) {
	if m.zenMode {
		return
	}
	m.visibility = vis
	m.resizePanels()
}

func (m *Model) resizePanels() {
	mW, eW := panelWidths2(m.width, m.visibility, m.focus)
	contentH := m.height - 2
	if mW > 0 {
		m.modulesPanel.SetSize(mW, contentH)
	}
	if eW > 0 {
		m.elementsPanel.SetSize(eW, contentH)
	}

	widths := [2]int{mW, eW}
	x := 0
	for i, w := range widths {
		if w > 0 {
			m.panelLayout[i] = PanelRect{
				X: x, Y: 0, Width: w + 2, Height: contentH + 2, Visible: true,
			}
			x += w + 2
		} else {
			m.panelLayout[i] = PanelRect{}
		}
	}
}

// --- View ---

func (m Model) View() string {
	if m.width == 0 {
		return "mxcli tui — loading...\n\nPress q to quit"
	}

	if m.compare.IsVisible() {
		return m.compare.View()
	}
	if m.overlay.IsVisible() {
		return m.overlay.View()
	}

	m.modulesPanel.SetFocused(m.focus == FocusModules)
	m.elementsPanel.SetFocused(m.focus == FocusElements)

	mW, eW := panelWidths2(m.width, m.visibility, m.focus)
	contentH := m.height - 2
	if mW > 0 {
		m.modulesPanel.SetSize(mW, contentH)
	}
	if eW > 0 {
		m.elementsPanel.SetSize(eW, contentH)
	}

	statusLine := m.renderStatusBar()

	var visiblePanels []string
	if mW > 0 {
		visiblePanels = append(visiblePanels, m.modulesPanel.View())
	}
	if eW > 0 {
		visiblePanels = append(visiblePanels, m.elementsPanel.View())
	}

	cols := lipgloss.JoinHorizontal(lipgloss.Top, visiblePanels...)
	status := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Width(m.width).
		Render(statusLine)

	rendered := cols + "\n" + status

	if m.showHelp {
		helpView := renderHelp(m.width, m.height)
		rendered = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpView,
			lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
	}

	return rendered
}

func (m Model) renderStatusBar() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	active := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	var parts []string

	if node := m.selectedNode(); node != nil && node.QualifiedName != "" {
		if inferBsonType(node.Type) != "" {
			parts = append(parts, key.Render("b")+" "+dim.Render("bson"))
		}
		parts = append(parts, key.Render("m")+" "+dim.Render("mdl"))
		parts = append(parts, key.Render("c")+" "+dim.Render("compare"))
		parts = append(parts, key.Render("d")+" "+dim.Render("diagram"))
	}

	parts = append(parts, key.Render("/")+" "+dim.Render("filter"))
	parts = append(parts, key.Render("r")+" "+dim.Render("refresh"))

	if m.zenMode {
		parts = append(parts, active.Render("z ZEN"))
	} else {
		parts = append(parts, key.Render("z")+" "+dim.Render("zen"))
	}

	parts = append(parts, key.Render("?")+" "+dim.Render("help"))
	parts = append(parts, key.Render("q")+" "+dim.Render("quit"))

	return " " + strings.Join(parts, "  ")
}
