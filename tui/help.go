package tui

import "github.com/charmbracelet/lipgloss"

const helpText = `
  mxcli tui — Keyboard Reference

  NAVIGATION
    h / ←       move focus left
    l / → / Enter  open / move focus right
    j / ↓       move down
    k / ↑       move up
    Tab         cycle panel focus
    /           search/filter in current column
    Esc         back / close

  COMMANDS (press : to activate)
    :run             run MDL file
    :check           check MDL syntax
    :callers         show callers of selected element
    :callees         show callees
    :context         show context
    :impact          show impact
    :refs            show references
    :diagram         open diagram in browser
    :search <kw>     full-text search

  OTHER
    d    open diagram in browser
    r    refresh project tree
    ?    show/hide this help
    q    quit
`

func renderHelp(width, height int) string {
	helpWidth := width / 2
	if helpWidth < 60 {
		helpWidth = 60
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(helpWidth).
		Padding(1, 2).
		Render(helpText)
}
