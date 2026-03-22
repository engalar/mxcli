package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/tui/panels"
)

// Focus indicates which panel has keyboard focus.
type Focus int

const (
	FocusModules Focus = iota
	FocusElements
	FocusPreview
)

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
}

func New(mxcliPath, projectPath string) Model {
	return Model{
		mxcliPath:     mxcliPath,
		projectPath:   projectPath,
		focus:         FocusModules,
		modulesPanel:  panels.NewModulesPanel(30, 20),
		elementsPanel: panels.NewElementsPanel(40, 20),
		previewPanel:  panels.NewPreviewPanel(50, 20),
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

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
						// Drill down into node with children
						m.elementsPanel.SetNodes(node.Children)
						return m, nil
					}
					// Leaf node → show DESCRIBE in preview
					if node.QualifiedName != "" {
						m.focus = FocusPreview
						m.previewPanel.SetLoading()
						return m, m.describeNode(node)
					}
				}
			case "j", "down", "k", "up":
				// Forward navigation, then auto-preview
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

	statusLine := fmt.Sprintf(" mxcli tui  %s  [Tab: cycle focus | :: command | q: quit]",
		m.projectPath)

	return renderLayout(
		m.width,
		m.modulesPanel.View(),
		m.elementsPanel.View(),
		m.previewPanel.View(),
		statusLine,
	)
}
