package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HighlightFunc is a function that applies syntax highlighting to content.
// Set by the tui package to avoid circular imports.
var HighlightFunc func(string) string

// PreviewPanel is the right column: DESCRIBE output in a scrollable viewport.
type PreviewPanel struct {
	viewport       viewport.Model
	focused        bool
	loading        bool
	width          int
	height         int
	summaryContent string
	fullContent    string
	selectedNode   *TreeNode
}

// DescribeResultMsg carries DESCRIBE output.
type DescribeResultMsg struct {
	Content string
	Err     error
}

// OpenOverlayMsg requests that the overlay be shown with highlighted full content.
type OpenOverlayMsg struct {
	Title   string
	Content string
}

func NewPreviewPanel(width, height int) PreviewPanel {
	vp := viewport.New(width, height)
	vp.SetContent("Select an element to preview its DESCRIBE output.")
	return PreviewPanel{viewport: vp, width: width, height: height}
}

func (p *PreviewPanel) SetContent(content string) {
	p.loading = false
	p.fullContent = content
	p.summaryContent = buildSummary(content)
	p.viewport.SetContent(p.summaryContent)
	p.viewport.GotoTop()
}

func (p *PreviewPanel) SetLoading() {
	p.loading = true
	p.viewport.SetContent("Loading...")
}

func (p *PreviewPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.viewport.Width = w
	p.viewport.Height = h
}

func (p *PreviewPanel) SetFocused(f bool) { p.focused = f }

func (p *PreviewPanel) SetSelectedNode(node *TreeNode) {
	p.selectedNode = node
}

// FullContent returns the full DESCRIBE output for overlay display.
func (p PreviewPanel) FullContent() string {
	return p.fullContent
}

func (p PreviewPanel) Update(msg tea.Msg) (PreviewPanel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" && p.fullContent != "" {
		title := "Preview"
		if p.selectedNode != nil {
			title = fmt.Sprintf("%s %s", p.selectedNode.Type, p.selectedNode.QualifiedName)
		}
		content := p.fullContent
		if HighlightFunc != nil {
			content = HighlightFunc(content)
		}
		return p, func() tea.Msg {
			return OpenOverlayMsg{Title: title, Content: content}
		}
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p PreviewPanel) View() string {
	title := "Preview"
	if p.loading {
		title = "Preview (loading...)"
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(title)
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(p.focused))
	return border.Render(header + "\n" + p.viewport.View())
}

// buildSummary extracts key metadata lines from DESCRIBE output.
func buildSummary(content string) string {
	if content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	var summary []string
	attrCount := 0
	inAttributes := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Keep header lines (Type, Name, Module, etc.)
		for _, prefix := range []string{
			"-- Type:", "-- Name:", "-- Module:", "-- Qualified",
			"CREATE ", "ALTER ", "-- Documentation",
			"-- Return", "-- Persistent", "-- Generalization",
		} {
			if strings.HasPrefix(trimmed, prefix) {
				summary = append(summary, line)
				break
			}
		}

		// Count attributes/parameters
		if strings.Contains(trimmed, "ATTRIBUTE") || strings.Contains(trimmed, "PARAMETER") {
			if !inAttributes {
				inAttributes = true
			}
			attrCount++
		}
	}

	if attrCount > 0 {
		summary = append(summary, fmt.Sprintf("\n  %d attribute(s)/parameter(s)", attrCount))
	}

	if len(summary) == 0 {
		// Fallback: show first 15 lines
		limit := 15
		if len(lines) < limit {
			limit = len(lines)
		}
		return strings.Join(lines[:limit], "\n")
	}

	summary = append(summary, "\n  [Enter] full view")
	return strings.Join(summary, "\n")
}
