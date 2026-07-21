package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/task"
)

type RunView struct {
	task     *task.RunTask
	events   []task.Event
	logLines []string
	scroll   int
	width    int
	height   int
	running  bool
	started  time.Time
}

func NewRunView(t *task.RunTask) RunView {
	return RunView{
		task:    t,
		running: true,
		started: time.Now(),
	}
}

func (rv RunView) Mode() ViewMode { return ModeExec }

func (rv RunView) Hints() []Hint {
	hints := []kernel.Hint{
		{Key: "q", Label: "close"},
		{Key: "j/k", Label: "scroll"},
	}
	if rv.running {
		hints = append([]kernel.Hint{{Key: "c", Label: "stop"}}, hints...)
	}
	return hints
}

func (rv RunView) StatusInfo() StatusInfo {
	s := "stopped"
	if rv.running {
		s = "running"
	}
	elapsed := time.Since(rv.started).Round(time.Second)
	return StatusInfo{
		Breadcrumb: []string{"Run"},
		Position:   fmt.Sprintf("%s (%d lines)", elapsed, len(rv.logLines)),
		Mode:       s,
	}
}

func (rv RunView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return rv, func() tea.Msg { return PopViewMsg{} }
		case "c":
			if rv.running {
				rv.task.Cancel()
				rv.running = false
			}
		case "j":
			if rv.scroll < len(rv.logLines)-1 {
				rv.scroll++
			}
		case "k":
			if rv.scroll > 0 {
				rv.scroll--
			}
		}

	case task.Event:
		rv.events = append(rv.events, msg)
		rv.logLines = append(rv.logLines, fmt.Sprintf("[%s] %s", msg.Phase, msg.Message))
		rv.scroll = len(rv.logLines) - 1
		if msg.State == task.StateCompleted || msg.State == task.StateFailed || msg.State == task.StateCancelled {
			rv.running = false
			return rv, nil
		}
		if rv.task != nil {
			return rv, rv.task.StreamCmd()
		}

	case tea.WindowSizeMsg:
		rv.width = msg.Width
		rv.height = msg.Height
	}

	return rv, nil
}

func (rv RunView) Render(width, height int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(kernel.AccentColor).Render("Run")

	var sb strings.Builder
	sb.WriteString(title + "\n\n")

	visibleH := height - 4
	start := max(0, rv.scroll-visibleH+1)
	end := min(len(rv.logLines), start+visibleH)

	for i := start; i < end; i++ {
		line := rv.logLines[i]
		if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			line = kernel.CheckErrorStyle.Render(line)
		}
		sb.WriteString(line + "\n")
	}

	if rv.running {
		sb.WriteString(kernel.LoadingStyle.Render("\n⟳ Runtime running... (press c to stop)"))
	} else if len(rv.events) > 0 {
		last := rv.events[len(rv.events)-1]
		switch last.State {
		case task.StateCompleted:
			sb.WriteString(kernel.CheckPassStyle.Render("\n⏹ Runtime stopped"))
		case task.StateFailed:
			sb.WriteString(kernel.CheckErrorStyle.Render("\n✗ " + last.Message))
		case task.StateCancelled:
			sb.WriteString(kernel.CheckWarnStyle.Render("\n⊘ Cancelled"))
		}
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}
