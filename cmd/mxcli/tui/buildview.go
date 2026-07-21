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

type BuildView struct {
	task    *task.BuildTask
	events  []task.Event
	scroll  int
	width   int
	height  int
	running bool
	started time.Time
}

func NewBuildView(t *task.BuildTask) BuildView {
	return BuildView{
		task:    t,
		running: true,
		started: time.Now(),
	}
}

func (bv BuildView) Mode() ViewMode { return ModeExec }

func (bv BuildView) Hints() []Hint {
	hints := []kernel.Hint{
		{Key: "q", Label: "close"},
		{Key: "j/k", Label: "scroll"},
	}
	if bv.running {
		hints = append([]kernel.Hint{{Key: "c", Label: "cancel"}}, hints...)
	}
	return hints
}

func (bv BuildView) StatusInfo() StatusInfo {
	s := "complete"
	if bv.running {
		s = "building"
	}
	elapsed := time.Since(bv.started).Round(time.Second)
	return StatusInfo{
		Breadcrumb: []string{"Build"},
		Position:   fmt.Sprintf("%s (%d events)", elapsed, len(bv.events)),
		Mode:       s,
	}
}

func (bv BuildView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return bv, func() tea.Msg { return PopViewMsg{} }
		case "c":
			if bv.running {
				bv.task.Cancel()
				bv.running = false
			}
		case "j":
			if bv.scroll < len(bv.events)-1 {
				bv.scroll++
			}
		case "k":
			if bv.scroll > 0 {
				bv.scroll--
			}
		}

	case task.Event:
		bv.events = append(bv.events, msg)
		bv.scroll = len(bv.events) - 1
		if msg.State == task.StateCompleted || msg.State == task.StateFailed || msg.State == task.StateCancelled {
			bv.running = false
			return bv, nil
		}
		if bv.task != nil {
			return bv, bv.task.StreamCmd()
		}

	case tea.WindowSizeMsg:
		bv.width = msg.Width
		bv.height = msg.Height
	}

	return bv, nil
}

func (bv BuildView) Render(width, height int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(kernel.AccentColor).Render("Build")

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	visibleH := height - 4
	if visibleH < 1 {
		visibleH = 1
	}
	start := max(0, bv.scroll-visibleH+1)
	end := min(len(bv.events), start+visibleH)

	for _, ev := range bv.events[start:end] {
		line := task.RenderProgress(ev, width-4)
		switch ev.State {
		case task.StateFailed:
			line = kernel.CheckErrorStyle.Render("✗ " + ev.Message)
		case task.StateCompleted:
			line = kernel.CheckPassStyle.Render("✓ " + ev.Message)
		case task.StateCancelled:
			line = kernel.CheckWarnStyle.Render("⊘ " + ev.Message)
		default:
			phase := kernel.MutedStyle.Render(ev.Phase + ":")
			line = phase + " " + ev.Message
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	if bv.running {
		sb.WriteString(kernel.LoadingStyle.Render("⟳ Building... (c to cancel)"))
	} else if len(bv.events) > 0 {
		last := bv.events[len(bv.events)-1]
		sb.WriteByte('\n')
		switch last.State {
		case task.StateCompleted:
			sb.WriteString(kernel.CheckPassStyle.Render("✓ Build complete"))
		case task.StateFailed:
			sb.WriteString(kernel.CheckErrorStyle.Render("✗ Build failed"))
		case task.StateCancelled:
			sb.WriteString(kernel.CheckWarnStyle.Render("⊘ Build cancelled"))
		}
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}
