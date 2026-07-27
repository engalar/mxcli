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

type MockView struct {
	task     *task.MockTask
	events   []task.Event
	logLines []string
	scroll   int
	width    int
	height   int
	running  bool
	started  time.Time
	port     int
	host     string
	autoRun  bool
}

func NewMockView(t *task.MockTask) MockView {
	port := t.Opts().Port
	if port == 0 {
		port = 4000
	}
	mockHost := t.Opts().Host
	if mockHost == "" {
		mockHost = "0.0.0.0"
	}
	return MockView{
		task:    t,
		running: true,
		started: time.Now(),
		port:    port,
		host:    mockHost,
	}
}

func (mv MockView) Mode() ViewMode { return ModeExec }

func (mv MockView) Hints() []Hint {
	hints := []kernel.Hint{
		{Key: "q", Label: "close"},
	}
	if mv.running {
		hints = append([]kernel.Hint{{Key: "c", Label: "stop"}}, hints...)
	}
	return hints
}

func (mv MockView) StatusInfo() StatusInfo {
	s := "stopped"
	if mv.running {
		s = "running"
	}
	elapsed := time.Since(mv.started).Round(time.Second)
	return StatusInfo{
		Breadcrumb: []string{"Mock"},
		Position:   fmt.Sprintf("%s (%d lines)", elapsed, len(mv.logLines)),
		Mode:       s,
	}
}

func (mv MockView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if mv.running {
				mv.task.Cancel()
				mv.running = false
			}
			return mv, func() tea.Msg { return PopViewMsg{} }
		case "c":
			if mv.running {
				mv.task.Cancel()
				mv.running = false
			}
		}

	case task.Event:
		mv.events = append(mv.events, msg)
		mv.logLines = append(mv.logLines, fmt.Sprintf("[%s] %s", msg.Phase, msg.Message))
		mv.scroll = len(mv.logLines) - 1
		if msg.State == task.StateCompleted || msg.State == task.StateFailed || msg.State == task.StateCancelled {
			mv.running = false
			return mv, nil
		}
		if mv.autoRun && msg.State == task.StateRunning && msg.Phase == "running" {
			mv.autoRun = false
			return mv, tea.Sequence(
				mv.task.StreamCmd(),
				func() tea.Msg { return MockSucceededMsg{} },
			)
		}
		if mv.task != nil {
			return mv, mv.task.StreamCmd()
		}

	case tea.WindowSizeMsg:
		mv.width = msg.Width
		mv.height = msg.Height
	}

	return mv, nil
}

func (mv MockView) Render(width, height int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(kernel.AccentColor).Render("Prism Mock Server")
	elapsed := time.Since(mv.started).Round(time.Second)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("  ")
	sb.WriteString(kernel.MutedStyle.Render(elapsed.String()))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	if mv.running {
		infoStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(kernel.AccentColor).
			Padding(1, 2)

		var infoSb strings.Builder
		infoSb.WriteString(kernel.CheckPassStyle.Render("●  Mock server is running"))
		infoSb.WriteString("\n\n")
		displayHost := mv.host
		if displayHost == "0.0.0.0" {
			displayHost = "localhost (all interfaces)"
		}
		infoSb.WriteString(fmt.Sprintf("  URL:  http://%s:%d\n", displayHost, mv.port))
		infoSb.WriteString("\n")
		infoSb.WriteString(kernel.HintLabelStyle.Render("  c=stop  q=close"))

		sb.WriteString(infoStyle.Render(infoSb.String()))

		if len(mv.logLines) > 0 {
			sb.WriteByte('\n')
			visibleH := height - 12
			if visibleH > 3 {
				start := max(0, len(mv.logLines)-visibleH)
				for _, line := range mv.logLines[start:] {
					sb.WriteString(kernel.MutedStyle.Render(line))
					sb.WriteByte('\n')
				}
			}
		}
	} else {
		sb.WriteString("Mock server stopped\n")
		if len(mv.events) > 0 {
			last := mv.events[len(mv.events)-1]
			switch last.State {
			case task.StateCompleted:
				sb.WriteString(kernel.CheckPassStyle.Render("✓ " + last.Message))
			case task.StateFailed:
				sb.WriteString(kernel.CheckErrorStyle.Render("✗ " + last.Message))
			case task.StateCancelled:
				sb.WriteString(kernel.CheckWarnStyle.Render("⊘ Stopped"))
			}
		}
		if len(mv.logLines) > 0 {
			sb.WriteByte('\n')
			start := max(0, len(mv.logLines)-5)
			for _, line := range mv.logLines[start:] {
				sb.WriteString(kernel.MutedStyle.Render(line))
				sb.WriteByte('\n')
			}
		}
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}
