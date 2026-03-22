package panels

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PreviewPanel is the right column: DESCRIBE output in a scrollable viewport.
type PreviewPanel struct {
	viewport viewport.Model
	focused  bool
	loading  bool
	width    int
	height   int
}

// DescribeResultMsg carries DESCRIBE output.
type DescribeResultMsg struct {
	Content string
	Err     error
}

func NewPreviewPanel(width, height int) PreviewPanel {
	vp := viewport.New(width, height)
	vp.SetContent("Select an element to preview its DESCRIBE output.")
	return PreviewPanel{viewport: vp, width: width, height: height}
}

func (p *PreviewPanel) SetContent(content string) {
	p.loading = false
	p.viewport.SetContent(content)
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

func (p PreviewPanel) Update(msg tea.Msg) (PreviewPanel, tea.Cmd) {
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
