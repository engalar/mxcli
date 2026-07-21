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
	task      *task.BuildTask
	events    []task.Event
	phases    []phaseState
	logLines  []string
	scroll    int
	width     int
	height    int
	running   bool
	started   time.Time
	autoScroll bool
	showRaw   bool
	phaseIdx  int
}

type phaseState struct {
	Name    string
	Label   string
	Status  string // "pending" "running" "completed" "failed" "cancelled"
	Pct     float64
	Message string
}

var phaseMeta = map[string]struct {
	Label string
	Order int
}{
	"detect":  {"Detect version", 0},
	"mxbuild": {"Resolve MxBuild", 1},
	"jdk":     {"Resolve JDK 21", 2},
	"jars":    {"Resolve JAR deps", 3},
	"check":   {"Pre-build check", 4},
	"build":   {"Build PAD", 5},
	"extract": {"Post-process", 6},
	"done":    {"Complete", 7},
}

func NewBuildView(t *task.BuildTask) BuildView {
	// Pre-populate phases in order
	phases := make([]phaseState, 0, len(phaseMeta))
	for i := 0; i < len(phaseMeta); i++ {
		for name, meta := range phaseMeta {
			if meta.Order == i {
				phases = append(phases, phaseState{Name: name, Label: meta.Label, Status: "pending"})
				break
			}
		}
	}
	return BuildView{
		task:       t,
		running:    true,
		started:    time.Now(),
		phases:     phases,
		autoScroll: true,
		showRaw:    false,
	}
}

func (bv BuildView) Mode() ViewMode { return ModeExec }

func (bv BuildView) Hints() []Hint {
	hints := []kernel.Hint{
		{Key: "q", Label: "close"},
		{Key: "j/k", Label: "scroll"},
		{Key: "Tab", Label: "toggle raw"},
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
	mode := s
	if bv.showRaw {
		mode = "raw"
	}
	return StatusInfo{
		Breadcrumb: []string{"Build"},
		Position:   fmt.Sprintf("%s (%d log lines)", elapsed, len(bv.logLines)),
		Mode:       mode,
	}
}

func (bv *BuildView) updatePhase(name, status string, pct float64, msg string) {
	for i := range bv.phases {
		if bv.phases[i].Name == name {
			bv.phases[i].Status = status
			bv.phases[i].Pct = pct
			if msg != "" {
				bv.phases[i].Message = msg
			}
			if status == "running" {
				bv.phaseIdx = i
			}
			return
		}
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
			if bv.scroll < len(bv.logLines)-1 {
				bv.scroll++
				bv.autoScroll = false
			}
		case "k":
			if bv.scroll > 0 {
				bv.scroll--
				bv.autoScroll = false
			}
		case "tab":
			bv.showRaw = !bv.showRaw
		case "g":
			bv.scroll = 0
		case "G":
			bv.scroll = max(0, len(bv.logLines)-1)
			bv.autoScroll = true
		}

	case task.Event:
		bv.events = append(bv.events, msg)
		if msg.Type == task.EventPhaseChange {
			bv.updatePhase(msg.Phase, msg.State.String(), msg.Pct, msg.Message)
			if msg.State == task.StateRunning {
				bv.logLines = append(bv.logLines, fmt.Sprintf("── %s ──", msg.Message))
			} else if msg.State == task.StateCompleted {
				bv.logLines = append(bv.logLines, fmt.Sprintf("✓ %s", msg.Message))
			} else if msg.State == task.StateFailed {
				bv.logLines = append(bv.logLines, fmt.Sprintf("✗ %s", msg.Message))
			}
		} else if msg.Type == task.EventLogLine {
			bv.logLines = append(bv.logLines, msg.Line)
		}
		if bv.autoScroll {
			bv.scroll = len(bv.logLines) - 1
		}
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
	// Title bar
	elapsed := time.Since(bv.started).Round(time.Second)
	title := fmt.Sprintf("Build: %s", bv.taskName())
	titleBar := lipgloss.NewStyle().Bold(true).Foreground(kernel.AccentColor).Render(title)
	timeStr := kernel.MutedStyle.Render(elapsed.String())

	var sb strings.Builder
	sb.WriteString(titleBar)
	sb.WriteString("  ")
	sb.WriteString(timeStr)
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Phase panel (top)
	phaseH := len(bv.phases) + 1
	for _, p := range bv.phases {
		sb.WriteString(renderPhase(&p, width-4))
		sb.WriteByte('\n')
	}

	sb.WriteByte('\n')

	// Raw log panel (bottom)
	rawH := height - phaseH - 6
	if rawH < 3 {
		rawH = 3
	}
	visibleStart := max(0, bv.scroll-rawH+1)
	visibleEnd := min(len(bv.logLines), visibleStart+rawH)

	if bv.showRaw || bv.running {
		sb.WriteString(kernel.MutedStyle.Render("── Raw output ──"))
		sb.WriteByte('\n')

		if visibleStart > 0 {
			sb.WriteString(kernel.MutedStyle.Render(fmt.Sprintf("  ↑ %d more lines", visibleStart)))
			sb.WriteByte('\n')
		}
		for _, line := range bv.logLines[visibleStart:visibleEnd] {
			sb.WriteString(kernel.MutedStyle.Render(line))
			sb.WriteByte('\n')
		}
		if visibleEnd < len(bv.logLines) {
			sb.WriteString(kernel.MutedStyle.Render(fmt.Sprintf("  ↓ %d more lines", len(bv.logLines)-visibleEnd)))
			sb.WriteByte('\n')
		}
	}

	if bv.running && !bv.showRaw {
		sb.WriteString(kernel.LoadingStyle.Render("  ⟳ Building... (Tab=raw, c=cancel, q=close)"))
	} else if !bv.running {
		sb.WriteByte('\n')
		last := bv.lastPhase()
		if last != nil {
			switch last.Status {
			case "completed":
				sb.WriteString(kernel.CheckPassStyle.Render("✓ " + last.Message))
			case "failed":
				sb.WriteString(kernel.CheckErrorStyle.Render("✗ " + last.Message))
			case "cancelled":
				sb.WriteString(kernel.CheckWarnStyle.Render("⊘ Build cancelled"))
			}
		}
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}

func (bv BuildView) taskName() string {
	if bv.task == nil {
		return "?"
	}
	return "PAD build"
}

func (bv BuildView) lastPhase() *phaseState {
	for i := len(bv.phases) - 1; i >= 0; i-- {
		if bv.phases[i].Status != "pending" {
			return &bv.phases[i]
		}
	}
	return nil
}

func renderPhase(p *phaseState, width int) string {
	statusChar := "○"
	statusStyle := lipgloss.NewStyle().Foreground(kernel.MutedColor)
	labelStyle := kernel.MutedStyle

	switch p.Status {
	case "running":
		statusChar = "⟳"
		statusStyle = lipgloss.NewStyle().Foreground(kernel.AccentColor)
		labelStyle = lipgloss.NewStyle().Foreground(kernel.AccentColor)
	case "completed":
		statusChar = "✓"
		statusStyle = kernel.CheckPassStyle
	case "failed":
		statusChar = "✗"
		statusStyle = kernel.CheckErrorStyle
	case "cancelled":
		statusChar = "⊘"
		statusStyle = kernel.CheckWarnStyle
	}

	label := p.Label
	if p.Message != "" && p.Status == "running" {
		label = p.Message
	}

	line := fmt.Sprintf(" %s %s", statusStyle.Render(statusChar), labelStyle.Render(label))

	if p.Status == "running" && p.Pct > 0 {
		barWidth := width - lipgloss.Width(line) - 10
		if barWidth < 5 {
			barWidth = 5
		}
		filled := int(float64(barWidth) * p.Pct / 100.0)
		empty := barWidth - filled
		bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
		pct := fmt.Sprintf(" %3.0f%%", p.Pct)
		line += "  " + kernel.AccentStyle.Render(bar) + kernel.MutedStyle.Render(pct)
	}

	return line
}
