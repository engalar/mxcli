package tui

import "github.com/charmbracelet/lipgloss"

const helpText = `
  mxcli tui — Keyboard Reference

  NAVIGATION
    j / ↓          move down / scroll
    k / ↑          move up / scroll
    l / → / Enter  drill in / expand
    h / ←          go back
    Space          fuzzy jump to object
    /              filter in list
    Esc            back / close

  ACTIONS
    b     BSON dump (overlay)
    c     compare view (side-by-side)
    d     diagram in browser
    y     copy to clipboard
    r     refresh project tree
    z     zen mode (zoom panel)
    x     execute MDL script
    Tab   switch MDL / NDSL preview
    t     new tab (same project)
    T     new tab (pick project)
    1-9   switch tab

  CHECK OVERLAY
    j/k     scroll + select error
    Enter   go to document in tree
    Tab     cycle filter (all/error/warn/depr)
    r       rerun mx check
    /       search in content
    y       copy to clipboard
    q       close

  CHECK NAVIGATION (after Enter from check overlay)
    ]e      next error document
    [e      previous error document
    !       reopen check overlay
    Esc     exit nav mode

  OVERLAY
    j/k   scroll content
    /     search in content
    y     copy to clipboard
    Tab   switch MDL / NDSL
    r     rerun (check overlay)
    q     close

  COMPARE VIEW
    h/l   navigate panes
    /     search in content
    s     toggle sync scroll
    1/2/3 NDSL|NDSL / NDSL|MDL / MDL|MDL
    d     open diff view
    q     close

  DIFF VIEW
    j/k     scroll
    Tab     cycle mode (unified/side-by-side/plain)
    ]c/[c   next/prev hunk
    /       search
    q       close

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
