package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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
	mxcliPath   string
	projectPath string
	width       int
	height      int
	focus       Focus
}

func New(mxcliPath, projectPath string) Model {
	return Model{
		mxcliPath:   mxcliPath,
		projectPath: projectPath,
		focus:       FocusModules,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	return "mxcli tui — loading...\n\nPress q to quit"
}
