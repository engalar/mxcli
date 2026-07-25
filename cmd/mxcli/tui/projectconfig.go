package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

type ConfigView struct {
	mxcliPath   string
	projectPath string
	loading     bool
	loaded      bool
	err         string
	sections    []configSection
	width       int
	height      int
	scroll      int
	contentLine string
}

type configSection struct {
	Title  string
	Fields []configField
}

type configField struct {
	Label string
	Value string
}

func NewConfigView(mxcliPath, projectPath string) ConfigView {
	return ConfigView{
		mxcliPath:   mxcliPath,
		projectPath: projectPath,
		loading:     true,
	}
}

func (cv ConfigView) Mode() ViewMode { return kernel.ModeConfigView }

func (cv ConfigView) Hints() []Hint { return ConfigViewHints }

func (cv ConfigView) StatusInfo() StatusInfo {
	return StatusInfo{
		Breadcrumb: []string{"Project Configuration"},
		Mode:       "Config",
	}
}

type configLoadedMsg struct {
	sections []configSection
	err      string
}

func (cv ConfigView) loadCmd() tea.Cmd {
	return func() tea.Msg {
		mdl := "SHOW PROJECT SECURITY\nDESCRIBE SETTINGS\n"
		tmp := "/tmp/mxcli-config.mdl"
		if err := writeFile(tmp, mdl); err != nil {
			return configLoadedMsg{err: err.Error()}
		}
		out, err := runMxcli(cv.mxcliPath, "exec", "-p", cv.projectPath, tmp)
		out = StripBanner(out)
		if err != nil {
			return configLoadedMsg{err: out}
		}
		return configLoadedMsg{sections: parseConfigOutput(out)}
	}
}

func (cv ConfigView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return cv, func() tea.Msg { return PopViewMsg{} }
		case "r":
			cv.loading = true
			cv.loaded = false
			cv.sections = nil
			cv.scroll = 0
			return cv, cv.loadCmd()
		case "j", "down":
			cv.scroll++
		case "k", "up":
			if cv.scroll > 0 {
				cv.scroll--
			}
		case "ctrl+d":
			cv.scroll += cv.contentHeight()
		case "ctrl+u":
			cv.scroll -= cv.contentHeight()
			if cv.scroll < 0 {
				cv.scroll = 0
			}
		}
	case configLoadedMsg:
		cv.loading = false
		cv.loaded = true
		cv.scroll = 0
		if msg.err != "" {
			cv.err = msg.err
		} else {
			cv.sections = msg.sections
		}
		return cv, nil
	case tea.WindowSizeMsg:
		cv.width = msg.Width
		cv.height = msg.Height
	}
	return cv, nil
}

func (cv ConfigView) contentHeight() int {
	return max(1, cv.height-4)
}

func (cv ConfigView) buildContent() string {
	padH := 2
	innerW := cv.width - padH*2
	if innerW < 40 {
		innerW = 40
	}

	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(kernel.AccentColor).Render("Project Configuration")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(padH).Render(title))
	b.WriteString("\n\n")

	for _, sec := range cv.sections {
		secStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(kernel.MutedColor).
			Padding(0, 1).
			Width(innerW)

		var secBuf strings.Builder
		secBuf.WriteString(lipgloss.NewStyle().Bold(true).Render(sec.Title))
		secBuf.WriteString("\n")

		for _, f := range sec.Fields {
			label := lipgloss.NewStyle().Foreground(kernel.MutedColor).Render(f.Label + ":")
			val := lipgloss.NewStyle().Render(f.Value)
			line := fmt.Sprintf("  %s  %s", label, val)
			if lipgloss.Width(line) > innerW-2 {
				line = line[:innerW-4] + "..."
			}
			secBuf.WriteString(line)
			secBuf.WriteString("\n")
		}

		b.WriteString(lipgloss.NewStyle().PaddingLeft(padH).Render(secStyle.Render(secBuf.String())))
		b.WriteString("\n\n")
	}

	note := lipgloss.NewStyle().
		Foreground(kernel.MutedColor).
		Italic(true).
		PaddingLeft(padH).
		Render("Powered by mxcli exec")
	b.WriteString(note)
	b.WriteString("\n")

	return b.String()
}

func (cv ConfigView) Render(width, height int) string {
	cv.width = width
	cv.height = height

	if cv.loading {
		return lipgloss.Place(width, height,
			lipgloss.Center, lipgloss.Center,
			kernel.LoadingStyle.Render("Loading project configuration..."))
	}

	if cv.err != "" {
		es := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "196"})
		return lipgloss.Place(width, height,
			lipgloss.Center, lipgloss.Center,
			es.Render("Error:\n"+cv.err))
	}

	content := cv.buildContent()
	allLines := strings.Split(content, "\n")
	totalLines := len(allLines)
	maxVis := cv.contentHeight()

	// Clamp scroll
	if cv.scroll > totalLines-maxVis {
		cv.scroll = max(0, totalLines-maxVis)
	}

	visible := allLines[cv.scroll:]
	if len(visible) > maxVis {
		visible = visible[:maxVis]
	}
	// Pad to fill viewport
	for len(visible) < maxVis {
		visible = append(visible, "")
	}

	// Scrollbar
	scrollbarWidth := 1
	if totalLines > maxVis {
		thumbSize := max(1, maxVis*maxVis/totalLines)
		maxOffset := totalLines - maxVis
		thumbPos := 0
		if maxOffset > 0 {
			thumbPos = cv.scroll * (maxVis - thumbSize) / maxOffset
		}
		scrollCol := lipgloss.NewStyle().Width(scrollbarWidth).Render

		for i := range visible {
			if i >= thumbPos && i < thumbPos+thumbSize {
				visible[i] = visible[i] + scrollCol(kernel.AccentStyle.Render("│"))
			} else {
				visible[i] = visible[i] + scrollCol(lipgloss.NewStyle().Foreground(kernel.MutedColor).Render("·"))
			}
		}
	}

	return strings.Join(visible, "\n")
}

func parseConfigOutput(out string) []configSection {
	var sections []configSection
	var current *configSection

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		raw := line
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		indented := len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\t')

		if strings.HasPrefix(trimmed, "Security Level:") {
			current = &configSection{Title: "Security"}
			sections = append(sections, *current)
			current = &sections[len(sections)-1]
			parts := strings.SplitN(trimmed, ": ", 2)
			if len(parts) == 2 {
				current.Fields = append(current.Fields, configField{Label: "Level", Value: parts[1]})
			}
			continue
		}

		if current != nil && current.Title == "Security" && !indented {
			if strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "Password") {
				parts := strings.SplitN(trimmed, ": ", 2)
				if len(parts) == 2 {
					current.Fields = append(current.Fields, configField{Label: parts[0], Value: parts[1]})
				}
				continue
			}
			if strings.HasPrefix(trimmed, "Password") || strings.HasPrefix(trimmed, "User Roles") {
				continue
			}
			current = nil
		}

		if strings.Contains(trimmed, "alter settings model") {
			current = &configSection{Title: "Runtime Settings"}
			sections = append(sections, *current)
			current = &sections[len(sections)-1]
			continue
		}
		if strings.Contains(trimmed, "alter settings configuration") {
			current = &configSection{Title: "Configuration 'Default'"}
			sections = append(sections, *current)
			current = &sections[len(sections)-1]
			continue
		}
		if strings.Contains(trimmed, "alter settings language") ||
			strings.Contains(trimmed, "alter settings workflows") {
			current = nil
			continue
		}

		if current != nil && indented && strings.Contains(trimmed, "=") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				label := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, "'\";,")
				if label != "" {
					current.Fields = append(current.Fields, configField{Label: label, Value: val})
				}
			}
		}
	}

	return sections
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
