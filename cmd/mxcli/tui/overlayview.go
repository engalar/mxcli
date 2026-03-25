package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// overlayContentMsg carries reloaded content for an OverlayView after Tab switch.
type overlayContentMsg struct {
	Title   string
	Content string
}

// OverlayViewOpts holds optional configuration for an OverlayView.
type OverlayViewOpts struct {
	QName           string
	NodeType        string
	IsNDSL          bool
	Switchable      bool
	MxcliPath       string
	ProjectPath     string
	HideLineNumbers bool
}

// OverlayView wraps an Overlay to satisfy the View interface,
// adding BSON/MDL switching and self-contained content reload.
type OverlayView struct {
	overlay     Overlay
	qname       string
	nodeType    string
	isNDSL      bool
	switchable  bool
	mxcliPath   string
	projectPath string
}

// NewOverlayView creates an OverlayView with the given title, content, dimensions, and options.
func NewOverlayView(title, content string, width, height int, opts OverlayViewOpts) OverlayView {
	ov := OverlayView{
		qname:       opts.QName,
		nodeType:    opts.NodeType,
		isNDSL:      opts.IsNDSL,
		switchable:  opts.Switchable,
		mxcliPath:   opts.MxcliPath,
		projectPath: opts.ProjectPath,
	}
	ov.overlay = NewOverlay()
	ov.overlay.switchable = opts.Switchable
	ov.overlay.Show(title, content, width, height)
	if opts.HideLineNumbers {
		ov.overlay.content.hideLineNumbers = true
	}
	return ov
}

// Update handles input and internal messages.
func (ov OverlayView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case overlayContentMsg:
		ov.overlay.Show(msg.Title, msg.Content, ov.overlay.width, ov.overlay.height)
		return ov, nil

	case tea.KeyMsg:
		if ov.overlay.content.IsSearching() {
			var cmd tea.Cmd
			ov.overlay, cmd = ov.overlay.Update(msg)
			return ov, cmd
		}
		switch msg.String() {
		case "esc", "q":
			return ov, func() tea.Msg { return PopViewMsg{} }
		case "tab":
			if ov.switchable && ov.qname != "" {
				ov.isNDSL = !ov.isNDSL
				return ov, ov.reloadContent()
			}
		}
	}

	var cmd tea.Cmd
	ov.overlay, cmd = ov.overlay.Update(msg)
	return ov, cmd
}

// Render returns the overlay's rendered string at the given dimensions with an LLM anchor prefix.
func (ov OverlayView) Render(width, height int) string {
	if ov.overlay.width != width || ov.overlay.height != height {
		ov.overlay.width = width
		ov.overlay.height = height
	}
	rendered := ov.overlay.View()

	// Embed LLM anchor as muted prefix on the first line
	info := ov.StatusInfo()
	anchor := fmt.Sprintf("[mxcli:overlay] %s  %s", ov.overlay.title, info.Mode)
	anchorStr := lipgloss.NewStyle().Foreground(MutedColor).Faint(true).Render(anchor)

	if idx := strings.IndexByte(rendered, '\n'); idx >= 0 {
		rendered = anchorStr + rendered[idx:]
	} else {
		rendered = anchorStr
	}
	return rendered
}

// Hints returns context-sensitive key hints for this overlay.
func (ov OverlayView) Hints() []Hint {
	hints := []Hint{
		{Key: "j/k", Label: "scroll"},
		{Key: "/", Label: "search"},
		{Key: "y", Label: "copy"},
	}
	if ov.switchable {
		hints = append(hints, Hint{Key: "Tab", Label: "mdl/ndsl"})
	}
	hints = append(hints, Hint{Key: "q", Label: "close"})
	return hints
}

// StatusInfo returns display data for the status bar.
func (ov OverlayView) StatusInfo() StatusInfo {
	modeLabel := "MDL"
	if ov.isNDSL {
		modeLabel = "NDSL"
	}
	position := fmt.Sprintf("L%d/%d", ov.overlay.content.YOffset()+1, ov.overlay.content.TotalLines())
	return StatusInfo{
		Breadcrumb: []string{ov.overlay.title},
		Position:   position,
		Mode:       modeLabel,
	}
}

// Mode returns ModeOverlay.
func (ov OverlayView) Mode() ViewMode {
	return ModeOverlay
}

// reloadContent returns a tea.Cmd that fetches new content based on isNDSL state.
func (ov OverlayView) reloadContent() tea.Cmd {
	if ov.isNDSL {
		return ov.runBsonReload()
	}
	return ov.runMDLReload()
}

func (ov OverlayView) runBsonReload() tea.Cmd {
	bsonType := inferBsonType(ov.nodeType)
	if bsonType == "" {
		return nil
	}
	mxcliPath := ov.mxcliPath
	projectPath := ov.projectPath
	qname := ov.qname
	return func() tea.Msg {
		args := []string{"bson", "dump", "-p", projectPath, "--format", "ndsl",
			"--type", bsonType, "--object", qname}
		out, err := runMxcli(mxcliPath, args...)
		out = StripBanner(out)
		title := fmt.Sprintf("BSON: %s", qname)
		if err != nil {
			return overlayContentMsg{Title: title, Content: "Error: " + out}
		}
		return overlayContentMsg{Title: title, Content: HighlightNDSL(out)}
	}
}

func (ov OverlayView) runMDLReload() tea.Cmd {
	mxcliPath := ov.mxcliPath
	projectPath := ov.projectPath
	qname := ov.qname
	nodeType := ov.nodeType
	return func() tea.Msg {
		out, err := runMxcli(mxcliPath, "-p", projectPath, "-c", buildDescribeCmd(nodeType, qname))
		out = StripBanner(out)
		title := fmt.Sprintf("MDL: %s", qname)
		if err != nil {
			return overlayContentMsg{Title: title, Content: "Error: " + out}
		}
		return overlayContentMsg{Title: title, Content: DetectAndHighlight(out)}
	}
}

