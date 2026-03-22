package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// columnWidths returns [modulesW, elementsW, previewW] given total terminal width.
func columnWidths(totalW int) (int, int, int) {
	// Reserve 2 chars per border (left+right) × 3 columns = 6
	available := totalW - 6
	if available < 30 {
		available = 30
	}
	modulesW := available * 20 / 100
	elementsW := available * 30 / 100
	previewW := available - modulesW - elementsW
	return modulesW, elementsW, previewW
}

// renderLayout assembles the three panels + status bar into the full screen.
func renderLayout(
	totalW int,
	modulesView, elementsView, previewView string,
	statusLine string,
) string {
	status := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Width(totalW).
		Render(statusLine)

	cols := lipgloss.JoinHorizontal(lipgloss.Top, modulesView, elementsView, previewView)

	return fmt.Sprintf("%s\n%s", cols, status)
}
