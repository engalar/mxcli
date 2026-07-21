package task

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

func RenderProgress(ev Event, width int) string {
	if ev.Pct < 0 {
		return kernel.MutedStyle.Render("⟳ " + ev.Phase + ": " + ev.Message)
	}

	barWidth := width - 20
	if barWidth < 10 {
		barWidth = 10
	}

	filled := int(float64(barWidth) * ev.Pct / 100.0)
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	pct := fmt.Sprintf("%3.0f%%", ev.Pct)
	return fmt.Sprintf("%s %s %s", bar, pct, kernel.MutedStyle.Render(ev.Message))
}
