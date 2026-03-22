// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PickerModel lets the user select from recent projects or type a new path.
type PickerModel struct {
	history   []string
	cursor    int
	input     textinput.Model
	inputMode bool
	chosen    string
	done      bool
	width     int
	height    int
}

// NewPickerModel creates the picker model with loaded history.
func NewPickerModel() PickerModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/App.mpr"
	ti.Prompt = "  Path: "
	ti.CharLimit = 500

	return PickerModel{
		history: LoadHistory(),
		input:   ti,
	}
}

// Chosen returns the selected project path (empty if cancelled).
func (m PickerModel) Chosen() string {
	return m.chosen
}

func (m PickerModel) Init() tea.Cmd {
	return nil
}

func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.inputMode {
			switch msg.String() {
			case "esc":
				m.inputMode = false
				m.input.Blur()
				return m, nil
			case "enter":
				val := strings.TrimSpace(m.input.Value())
				if val != "" {
					m.chosen = val
					m.done = true
					return m, tea.Quit
				}
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		} else {
			switch msg.String() {
			case "ctrl+c", "q":
				m.done = true
				return m, tea.Quit
			case "j", "down":
				if m.cursor < len(m.history)-1 {
					m.cursor++
				}
			case "k", "up":
				if m.cursor > 0 {
					m.cursor--
				}
			case "enter":
				if len(m.history) > 0 {
					m.chosen = m.history[m.cursor]
					m.done = true
					return m, tea.Quit
				}
			case "n":
				m.inputMode = true
				m.input.SetValue("")
				m.input.Focus()
				return m, nil
			}
		}
	}
	return m, nil
}

func (m PickerModel) View() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(60)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Select Mendix Project") + "\n\n")

	if len(m.history) == 0 && !m.inputMode {
		sb.WriteString(dimStyle.Render("No recent projects.") + "\n\n")
	} else if !m.inputMode {
		sb.WriteString(dimStyle.Render("Recent projects:") + "\n")
		for i, path := range m.history {
			prefix := "  "
			var line string
			if i == m.cursor {
				prefix = "> "
				line = selectedStyle.Render(prefix + path)
			} else {
				line = normalStyle.Render(prefix + path)
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	if m.inputMode {
		sb.WriteString(m.input.View() + "\n\n")
		sb.WriteString(dimStyle.Render("[Enter] confirm  [Esc] back") + "\n")
	} else {
		hint := "[n] new path"
		if len(m.history) > 0 {
			hint = "[j/k] navigate  [Enter] open  [n] new path  [q] quit"
		}
		sb.WriteString(dimStyle.Render(hint) + "\n")
	}

	content := boxStyle.Render(sb.String())

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}
