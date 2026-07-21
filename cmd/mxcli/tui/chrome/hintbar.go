package chrome

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

type HintBar struct {
	hints []kernel.Hint
}

func NewHintBar(hints []kernel.Hint) HintBar {
	return HintBar{hints: hints}
}

func (h *HintBar) SetHints(hints []kernel.Hint) {
	h.hints = hints
}

func (h *HintBar) View(width int) string {
	if len(h.hints) == 0 {
		return ""
	}

	separator := "  "
	sepWidth := lipgloss.Width(separator)

	type rendered struct {
		text  string
		width int
	}
	items := make([]rendered, len(h.hints))
	for i, hint := range h.hints {
		text := kernel.HintKeyStyle.Render(hint.Key) + " " + kernel.HintLabelStyle.Render(hint.Label)
		items[i] = rendered{text: text, width: lipgloss.Width(text)}
	}

	minKeep := min(3, len(items))

	usable := width - 2
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
