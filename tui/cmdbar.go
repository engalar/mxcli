package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CmdBar is the bottom command input bar activated by ":".
type CmdBar struct {
	input   textinput.Model
	visible bool
}

func NewCmdBar() CmdBar {
	ti := textinput.New()
	ti.Placeholder = "command (run, check, callers, callees, context, impact, refs, diagram, search <kw>)"
	ti.Prompt = ": "
	ti.CharLimit = 200
	return CmdBar{input: ti}
}

func (c *CmdBar) Show() {
	c.visible = true
	c.input.SetValue("")
	c.input.Focus()
}

func (c *CmdBar) Hide() {
	c.visible = false
	c.input.Blur()
}

func (c CmdBar) IsVisible() bool { return c.visible }

// Command returns the current input value split into verb + args.
func (c CmdBar) Command() (verb string, rest string) {
	parts := strings.SplitN(strings.TrimSpace(c.input.Value()), " ", 2)
	if len(parts) == 0 {
		return "", ""
	}
	verb = strings.ToLower(parts[0])
	if len(parts) > 1 {
		rest = parts[1]
	}
	return verb, rest
}

func (c CmdBar) Update(msg tea.Msg) (CmdBar, tea.Cmd) {
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

func (c CmdBar) View() string {
	if !c.visible {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(" :run  :check  :callers  :callees  :context  :impact  :refs  :diagram  :search")
	}
	return lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("214")).
		Render(c.input.View())
}
