package chrome

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

type TabClickMsg struct {
	ID int
}

type TabInfo struct {
	ID     int
	Label  string
	Active bool
}

type TabBar struct {
	tabs  []TabInfo
	width int
	zones []tabZone
}

type tabZone struct {
	start, end int
	id         int
}

func NewTabBar(tabs []TabInfo) TabBar {
	return TabBar{tabs: tabs}
}

func (t *TabBar) SetTabs(tabs []TabInfo) {
	t.tabs = tabs
}

func (t *TabBar) SetWidth(w int) {
	t.width = w
}

func (t *TabBar) HandleClick(x int) tea.Msg {
	for _, z := range t.zones {
		if x >= z.start && x < z.end {
			return TabClickMsg{ID: z.id}
		}
	}
	return nil
}

func (t *TabBar) View(width int) string {
	if width <= 0 {
		width = t.width
	}
	if len(t.tabs) == 0 {
		return ""
	}

	t.zones = t.zones[:0]
	var sb strings.Builder
	col := 1
	sb.WriteString(" ")

	for i, tab := range t.tabs {
		if i > 0 {
			sb.WriteString("  ")
			col += 2
		}

		label := fmt.Sprintf("[%d] %s", tab.ID, tab.Label)
		labelWidth := lipgloss.Width(label)

		start := col
		var rendered string
		if tab.Active {
			rendered = kernel.ActiveTabStyle.Render(label)
		} else {
			rendered = kernel.InactiveTabStyle.Render(label)
		}
		sb.WriteString(rendered)
		col += labelWidth

		t.zones = append(t.zones, tabZone{start: start, end: col, id: tab.ID})
	}

	line := sb.String()
	lineWidth := lipgloss.Width(line)
	if lineWidth < width {
		line += strings.Repeat(" ", width-lineWidth)
	}
	return line
}
