package tui

import "github.com/charmbracelet/lipgloss"

const helpText = `
  mxcli tui — Keyboard Reference

  NAVIGATION
    j / ↓         move down / scroll
    k / ↑         move up / scroll
    l / → / Enter  drill in / expand
    h / ←         go back
    Tab           cycle panel focus
    /             filter in list
    Esc           back / close

  ACTIONS
    b    BSON dump (overlay)
    c    compare view (side-by-side)
    d    diagram in browser
    r    refresh project tree
    z    zen mode (zoom panel)
    Enter  full detail (in preview)

  COMPARE VIEW
    Tab   switch left/right pane
    /     fuzzy pick object
    1/2/3 NDSL|NDSL / NDSL|MDL / MDL|MDL
    s     toggle sync scroll
    j/k   scroll content
    Esc   close

  OTHER
    ?    show/hide this help
    q    quit
`

func renderHelp(width, _ int) string {
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
