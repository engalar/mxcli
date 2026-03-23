package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Hint represents a single key hint (e.g. "h:back").
type Hint struct {
	Key   string
	Label string
}

// HintBar renders a context-sensitive key hint line.
type HintBar struct {
	hints []Hint
}

// NewHintBar creates a hint bar with the given hints.
func NewHintBar(hints []Hint) HintBar {
	return HintBar{hints: hints}
}

// SetHints replaces the current hints.
func (h *HintBar) SetHints(hints []Hint) {
	h.hints = hints
}

// Predefined hint sets for each context.
var (
	ListBrowsingHints = []Hint{
		{Key: "h", Label: "back"},
		{Key: "l", Label: "open"},
		{Key: "/", Label: "filter"},
		{Key: "Tab", Label: "mdl/ndsl"},
		{Key: "y", Label: "copy"},
		{Key: "c", Label: "compare"},
		{Key: "z", Label: "zen"},
		{Key: "r", Label: "refresh"},
		{Key: "t", Label: "tab"},
		{Key: "T", Label: "new project"},
		{Key: "1-9", Label: "switch tab"},
		{Key: "?", Label: "help"},
	}
	FilterActiveHints = []Hint{
		{Key: "Enter", Label: "confirm"},
		{Key: "Esc", Label: "cancel"},
	}
	OverlayHints = []Hint{
		{Key: "j/k", Label: "scroll"},
		{Key: "/", Label: "search"},
		{Key: "y", Label: "copy"},
		{Key: "Tab", Label: "mdl/ndsl"},
		{Key: "q", Label: "close"},
	}
	CompareHints = []Hint{
		{Key: "h/l", Label: "navigate"},
		{Key: "/", Label: "search"},
		{Key: "s", Label: "sync scroll"},
		{Key: "1/2/3", Label: "mode"},
		{Key: "d", Label: "diff"},
		{Key: "q", Label: "close"},
	}
)

// View renders the hint bar to fit within the given width.
// Truncates from the right if too narrow, always keeping at least 3 hints.
func (h *HintBar) View(width int) string {
	if len(h.hints) == 0 {
		return ""
	}

	separator := "  "
	sepWidth := lipgloss.Width(separator)

	// Render each hint and measure.
	type rendered struct {
		text  string
		width int
	}
	items := make([]rendered, len(h.hints))
	for i, hint := range h.hints {
		text := HintKeyStyle.Render(hint.Key) + ":" + HintLabelStyle.Render(hint.Label)
		items[i] = rendered{text: text, width: lipgloss.Width(text)}
	}

	// Determine how many hints fit. Always keep at least 3 (or all if fewer).
	minKeep := min(3, len(items))

	usable := width - 2 // 1 char padding each side
	count := 0
	total := 0
	for i, item := range items {
		needed := item.width
		if i > 0 {
			needed += sepWidth
		}
		if total+needed > usable && count >= minKeep {
			break
		}
		total += needed
		count++
	}

	var sb strings.Builder
	sb.WriteString(" ")
	for i := 0; i < count; i++ {
		if i > 0 {
			sb.WriteString(separator)
		}
		sb.WriteString(items[i].text)
	}

	line := sb.String()
	lineWidth := lipgloss.Width(line)
	if lineWidth < width {
		line += strings.Repeat(" ", width-lineWidth)
	}
	return line
}
