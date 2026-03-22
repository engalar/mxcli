package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/tui/panels"
)

// Focus indicates which panel has keyboard focus.
type Focus int

const (
	FocusModules Focus = iota
	FocusElements
	FocusPreview
)

// CmdResultMsg carries output from any mxcli command.
type CmdResultMsg struct {
	Output string
	Err    error
}

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	mxcliPath     string
	projectPath   string
	width         int
	height        int
	focus         Focus
	modulesPanel  panels.ModulesPanel
	elementsPanel panels.ElementsPanel
	previewPanel  panels.PreviewPanel
	cmdbar        CmdBar
	showHelp      bool
}

func New(mxcliPath, projectPath string) Model {
	return Model{
		mxcliPath:     mxcliPath,
		projectPath:   projectPath,
		focus:         FocusModules,
		modulesPanel:  panels.NewModulesPanel(30, 20),
		elementsPanel: panels.NewElementsPanel(40, 20),
		previewPanel:  panels.NewPreviewPanel(50, 20),
		cmdbar:        NewCmdBar(),
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

// describeNode returns a tea.Cmd that runs DESCRIBE for a given node.
func (m Model) describeNode(node *panels.TreeNode) tea.Cmd {
	return func() tea.Msg {
		out, err := runMxcli(m.mxcliPath, "-p", m.projectPath, "-c",
			fmt.Sprintf("DESCRIBE %s %s", node.Type, node.QualifiedName))
		return panels.DescribeResultMsg{Content: out, Err: err}
	}
}

// runShowCmd executes an MDL statement via mxcli -p <path> -c "<stmt>".
func (m Model) runShowCmd(stmt string) tea.Cmd {
	return func() tea.Msg {
		out, err := runMxcli(m.mxcliPath, "-p", m.projectPath, "-c", stmt)
		return CmdResultMsg{Output: out, Err: err}
	}
}

// openDiagram generates a temp HTML file via mxcli describe --format elk
// and opens it in the system browser.
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
		return CmdResultMsg{Output: fmt.Sprintf("Opened diagram in browser: %s", tmpFile.Name())}
	}
}

func buildDiagramHTML(elkJSON, nodeType, qualifiedName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>%s %s</title>
  <script src="https://cdn.jsdelivr.net/npm/elkjs@0.9.3/lib/elk.bundled.js"></script>
  <style>body{margin:0;background:#1e1e2e;color:#cdd6f4;font-family:monospace}
  svg{width:100vw;height:100vh}</style>
</head>
<body>
  <div id="diagram"></div>
  <script>
    const elkData = %s;
    const ELK = new ELKConstructor();
    ELK.layout(elkData).then(graph => {
      const svg = document.createElementNS("http://www.w3.org/2000/svg","svg");
      document.getElementById("diagram").appendChild(svg);
    });
  </script>
</body>
</html>`, nodeType, qualifiedName, elkJSON)
}

// selectedNode returns the currently selected node from the active panel.
func (m Model) selectedNode() *panels.TreeNode {
	if node := m.elementsPanel.SelectedNode(); node != nil {
		return node
	}
	return m.modulesPanel.SelectedNode()
}

// dispatchCommand dispatches a command bar verb to the appropriate action.
func (m Model) dispatchCommand(verb, rest string) tea.Cmd {
	node := m.selectedNode()
	qualifiedName := ""
	nodeType := ""
	if node != nil {
		qualifiedName = node.QualifiedName
		nodeType = node.Type
	}

	switch verb {
	case "callers":
		return m.runShowCmd(fmt.Sprintf("SHOW CALLERS OF %s %s", nodeType, qualifiedName))
	case "callees":
		return m.runShowCmd(fmt.Sprintf("SHOW CALLEES OF %s %s", nodeType, qualifiedName))
	case "context":
		return m.runShowCmd(fmt.Sprintf("SHOW CONTEXT OF %s %s", nodeType, qualifiedName))
	case "impact":
		return m.runShowCmd(fmt.Sprintf("SHOW IMPACT OF %s %s", nodeType, qualifiedName))
	case "refs":
		return m.runShowCmd(fmt.Sprintf("SHOW REFERENCES OF %s %s", nodeType, qualifiedName))
	case "diagram":
		return m.openDiagram(nodeType, qualifiedName)
	case "search":
		return m.runShowCmd(fmt.Sprintf("SEARCH '%s'", rest))
	case "run":
		if rest != "" {
			return m.runShowCmd(rest)
		}
	case "check":
		if rest != "" {
			return func() tea.Msg {
				out, err := runMxcli(m.mxcliPath, "check", rest, "-p", m.projectPath)
				return CmdResultMsg{Output: out, Err: err}
			}
		}
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Always allow ctrl+c
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Help overlay intercepts all keys
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Command bar mode
		if m.cmdbar.IsVisible() {
			switch msg.String() {
			case "esc":
				m.cmdbar.Hide()
				return m, nil
			case "enter":
				verb, rest := m.cmdbar.Command()
				m.cmdbar.Hide()
				return m, m.dispatchCommand(verb, rest)
			default:
				var cmd tea.Cmd
				m.cmdbar, cmd = m.cmdbar.Update(msg)
				return m, cmd
			}
		}

		// Global shortcuts
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case ":":
			m.cmdbar.Show()
			return m, nil
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "tab":
			// Cycle focus: modules → elements → preview → modules
			switch m.focus {
			case FocusModules:
				m.focus = FocusElements
			case FocusElements:
				m.focus = FocusPreview
			case FocusPreview:
				m.focus = FocusModules
			}
			return m, nil
		case "d":
			if node := m.selectedNode(); node != nil && node.QualifiedName != "" {
				return m, m.openDiagram(node.Type, node.QualifiedName)
			}
		case "r":
			// Refresh project tree
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
					return m, nil
				}
			default:
				var cmd tea.Cmd
				m.modulesPanel, cmd = m.modulesPanel.Update(msg)
				return m, cmd
			}

		case FocusElements:
			switch msg.String() {
			case "h", "left":
				m.focus = FocusModules
				return m, nil
			case "l", "right", "enter":
				if node := m.elementsPanel.SelectedNode(); node != nil {
					if len(node.Children) > 0 {
						m.elementsPanel.SetNodes(node.Children)
						return m, nil
					}
					if node.QualifiedName != "" {
						m.focus = FocusPreview
						m.previewPanel.SetLoading()
						return m, m.describeNode(node)
					}
				}
			case "j", "down", "k", "up":
				var cmd tea.Cmd
				m.elementsPanel, cmd = m.elementsPanel.Update(msg)
				if node := m.elementsPanel.SelectedNode(); node != nil && node.QualifiedName != "" {
					m.previewPanel.SetLoading()
					return m, tea.Batch(cmd, m.describeNode(node))
				}
				return m, cmd
			default:
				var cmd tea.Cmd
				m.elementsPanel, cmd = m.elementsPanel.Update(msg)
				return m, cmd
			}

		case FocusPreview:
			switch msg.String() {
			case "h", "left", "esc":
				m.focus = FocusElements
				return m, nil
			default:
				var cmd tea.Cmd
				m.previewPanel, cmd = m.previewPanel.Update(msg)
				return m, cmd
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		mW, eW, pW := columnWidths(m.width)
		contentH := m.height - 2
		m.modulesPanel.SetSize(mW, contentH)
		m.elementsPanel.SetSize(eW, contentH)
		m.previewPanel.SetSize(pW, contentH)

	case panels.LoadTreeMsg:
		if msg.Err == nil && msg.Nodes != nil {
			m.modulesPanel.SetNodes(msg.Nodes)
		}

	case panels.DescribeResultMsg:
		if msg.Err != nil {
			m.previewPanel.SetContent("-- Error:\n" + msg.Content)
		} else {
			m.previewPanel.SetContent(msg.Content)
		}

	case CmdResultMsg:
		if msg.Err != nil {
			m.previewPanel.SetContent("-- Error:\n" + msg.Output)
		} else {
			m.previewPanel.SetContent(msg.Output)
		}
		m.focus = FocusPreview
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "mxcli tui — loading...\n\nPress q to quit"
	}

	m.modulesPanel.SetFocused(m.focus == FocusModules)
	m.elementsPanel.SetFocused(m.focus == FocusElements)
	m.previewPanel.SetFocused(m.focus == FocusPreview)

	mW, eW, pW := columnWidths(m.width)
	contentH := m.height - 2
	m.modulesPanel.SetSize(mW, contentH)
	m.elementsPanel.SetSize(eW, contentH)
	m.previewPanel.SetSize(pW, contentH)

	statusLine := fmt.Sprintf(" mxcli tui  %s  [Tab: cycle focus | :: command | ?: help | q: quit]",
		m.projectPath)

	layout := renderLayout(
		m.width,
		m.modulesPanel.View(),
		m.elementsPanel.View(),
		m.previewPanel.View(),
		statusLine,
	)

	// Help overlay
	if m.showHelp {
		helpView := renderHelp(m.width, m.height)
		layout = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpView,
			lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
	}

	// Command bar at the very bottom
	if m.cmdbar.IsVisible() {
		lines := lipgloss.Height(layout)
		if lines > m.height-1 {
			// Truncate to make room for cmdbar
		}
		layout += "\n" + m.cmdbar.View()
	}

	return layout
}
