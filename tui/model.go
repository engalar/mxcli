package tui

import (
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
	mxcliPath    string
	projectPath  string
	width        int
	height       int
	focus        Focus
	modulesPanel panels.ModulesPanel
}

func New(mxcliPath, projectPath string) Model {
	return Model{
		mxcliPath:    mxcliPath,
		projectPath:  projectPath,
		focus:        FocusModules,
		modulesPanel: panels.NewModulesPanel(30, 20),
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
		if m.focus == FocusModules {
			var cmd tea.Cmd
			m.modulesPanel, cmd = m.modulesPanel.Update(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.modulesPanel.SetSize(m.width/3, m.height-2)
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
	return m.modulesPanel.View()
}
