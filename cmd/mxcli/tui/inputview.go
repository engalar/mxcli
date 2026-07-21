// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

// InputView is a lightweight single-line text input overlay.
type InputView struct {
	title    string
	prompt   string
	input    textinput.Model
	onSubmit func(string) tea.Cmd
	flash    string
}

// NewInputView creates a new single-line input view.
func NewInputView(title, prompt string, onSubmit func(string) tea.Cmd) *InputView {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.Focus()
	ti.CharLimit = 200
	return &InputView{
		title:    title,
		prompt:   prompt,
		input:    ti,
		onSubmit: onSubmit,
	}
}

func (v *InputView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return v, func() tea.Msg { return PopViewMsg{} }
		case "enter":
			value := strings.TrimSpace(v.input.Value())
			if value == "" {
				v.flash = "Name cannot be empty"
				return v, nil
			}
			return v, v.onSubmit(value)
		}
	}
	var cmd tea.Cmd
	v.input, cmd = v.input.Update(msg)
	return v, cmd
}

func (v *InputView) Render(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(kernel.AccentColor)
	flashStyle := lipgloss.NewStyle().Foreground(kernel.CheckErrorStyle.GetForeground())

	var b strings.Builder
	b.WriteString(titleStyle.Render(v.title))
	b.WriteString("\n\n")
	b.WriteString(v.input.View())
	if v.flash != "" {
		b.WriteString("\n")
		b.WriteString(flashStyle.Render(v.flash))
	}
	b.WriteString("\n\n")
	b.WriteString(kernel.HintLabelStyle.Render("Enter to confirm, Esc to cancel"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(kernel.AccentColor).
		Padding(1, 2).
		Width(min(60, width-4))

	box := boxStyle.Render(b.String())
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (v *InputView) Hints() []Hint {
	return []Hint{
		{Key: "Enter", Label: "confirm"},
		{Key: "Esc", Label: "cancel"},
	}
}

func (v *InputView) StatusInfo() StatusInfo {
	return StatusInfo{Mode: v.title}
}

func (v *InputView) Mode() ViewMode { return ModeInput }
