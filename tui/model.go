package tui

import (
	"github.com/charmbracelet/lipgloss"

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
}

func New(mxcliPath, projectPath string) Model {
	return Model{
		mxcliPath:     mxcliPath,
		projectPath:   projectPath,
		focus:         FocusModules,
		modulesPanel:  panels.NewModulesPanel(30, 20),
		elementsPanel: panels.NewElementsPanel(40, 20),
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
				// Open selected module → show children in elements panel
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
				// Drill down into node with children
				if node := m.elementsPanel.SelectedNode(); node != nil && len(node.Children) > 0 {
					m.elementsPanel.SetNodes(node.Children)
					return m, nil
				}
			default:
				var cmd tea.Cmd
				m.elementsPanel, cmd = m.elementsPanel.Update(msg)
				return m, cmd
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentH := m.height - 2
		m.modulesPanel.SetSize(m.width/3, contentH)
		m.elementsPanel.SetSize(m.width/3, contentH)

	case panels.LoadTreeMsg:
		if msg.Err == nil && msg.Nodes != nil {
			m.modulesPanel.SetNodes(msg.Nodes)
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

	return lipgloss.JoinHorizontal(lipgloss.Top,
		m.modulesPanel.View(),
		m.elementsPanel.View(),
	)
}
