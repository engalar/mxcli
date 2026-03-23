package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TabClickMsg is sent when a tab is clicked.
type TabClickMsg struct {
	ID int
}

// TabInfo describes a single tab.
type TabInfo struct {
	ID     int
	Label  string
	Active bool
}

// TabBar renders a horizontal tab bar.
type TabBar struct {
	tabs  []TabInfo
	width int
	// zones tracks click regions: [startCol, endCol) per tab index.
	zones []tabZone
}

type tabZone struct {
	start, end int
	id         int
}

// NewTabBar creates a tab bar with the given tabs.
func NewTabBar(tabs []TabInfo) TabBar {
	return TabBar{tabs: tabs}
}

// SetTabs replaces the tab list and recalculates click zones.
func (t *TabBar) SetTabs(tabs []TabInfo) {
	t.tabs = tabs
	t.rebuildZones()
}

func (t *TabBar) rebuildZones() {
	t.zones = t.zones[:0]
	col := 1 // 1 char left padding
	for i, tab := range t.tabs {
		if i > 0 {
			col += 2 // "  " separator
		}
		label := fmt.Sprintf("[%d] %s", tab.ID, tab.Label)
		labelWidth := lipgloss.Width(label)
		t.zones = append(t.zones, tabZone{start: col, end: col + labelWidth, id: tab.ID})
		col += labelWidth
	}
}

// SetWidth sets the available rendering width.
func (t *TabBar) SetWidth(w int) {
	t.width = w
}

// HandleClick checks if a mouse click at column x hits a tab zone.
func (t *TabBar) HandleClick(x int) tea.Msg {
	for _, z := range t.zones {
		if x >= z.start && x < z.end {
			return TabClickMsg{ID: z.id}
		}
	}
	return nil
}

// View renders the tab bar to fit within the given width.
func (t *TabBar) View(width int) string {
	if width <= 0 {
		width = t.width
	}
	if len(t.tabs) == 0 {
		return ""
	}

	t.zones = t.zones[:0]
	var sb strings.Builder
	col := 1 // start with 1 char left padding
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
			rendered = ActiveTabStyle.Render(label)
		} else {
			rendered = InactiveTabStyle.Render(label)
		}
		sb.WriteString(rendered)
		col += labelWidth

		t.zones = append(t.zones, tabZone{start: start, end: col, id: tab.ID})
	}

	line := sb.String()
	// Pad or truncate to width
	lineWidth := lipgloss.Width(line)
	if lineWidth < width {
		line += strings.Repeat(" ", width-lineWidth)
	}
	return line
}
