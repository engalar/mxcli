package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// overlayFlashClearMsg clears the "Copied!" flash after a delay.
type overlayFlashClearMsg struct{}

// Overlay is a fullscreen modal with scrollable content, line numbers,
// scrollbar, vim navigation, and mouse support.
type Overlay struct {
	content     ContentView
	title       string
	visible     bool
	copiedFlash bool
	switchable  bool // Tab key switches between NDSL and MDL
	width       int
	height      int
}

func NewOverlay() Overlay {
	return Overlay{}
}

func (o *Overlay) Show(title, content string, w, h int) {
	o.visible = true
	o.title = title
	o.width = w
	o.height = h

	innerW := w - 4 // border + padding
	innerH := h - 4 // title + hint + borders
	if innerW < 40 {
		innerW = 40
	}
	if innerH < 10 {
		innerH = 10
	}

	o.content = NewContentView(innerW, innerH)
	o.content.SetContent(content)
}

func (o *Overlay) Hide()          { o.visible = false }
func (o Overlay) IsVisible() bool { return o.visible }

func (o Overlay) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if !o.visible {
		return o, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When searching, forward all keys to content (including esc to close search)
		if o.content.IsSearching() {
			var cmd tea.Cmd
			o.content, cmd = o.content.Update(msg)
			return o, cmd
		}
		switch msg.String() {
		case "esc", "q":
			o.visible = false
			return o, nil
		case "y":
			_ = writeClipboard(o.content.PlainText())
			o.copiedFlash = true
			return o, func() tea.Msg {
				time.Sleep(time.Second)
				return overlayFlashClearMsg{}
			}
		default:
			o.copiedFlash = false
		}
	}

	var cmd tea.Cmd
	o.content, cmd = o.content.Update(msg)
	return o, cmd
}

func (o Overlay) View() string {
	if !o.visible {
		return ""
	}

	titleSt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	dimSt := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	keySt := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)

	titleBar := titleSt.Render(o.title)

	// Scroll info
	pct := fmt.Sprintf("%d%%", int(o.content.ScrollPercent()*100))
	lineInfo := fmt.Sprintf("L%d/%d", o.content.YOffset()+1, o.content.TotalLines())
	scrollInfo := dimSt.Render(lineInfo + " " + pct)

	// Hints
	activeSt := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	successSt := lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Bold(true)

	var hints []string
	hints = append(hints, keySt.Render("j/k")+" "+dimSt.Render("scroll"))
	hints = append(hints, keySt.Render("/")+" "+dimSt.Render("search"))
	if si := o.content.SearchInfo(); si != "" {
		hints = append(hints, keySt.Render("n/N")+" "+activeSt.Render(si))
	}
	hints = append(hints, keySt.Render("g/G")+" "+dimSt.Render("top/end"))
	if o.switchable {
		hints = append(hints, keySt.Render("Tab")+" "+dimSt.Render("switch"))
	}
	if o.copiedFlash {
		hints = append(hints, successSt.Render("✓ Copied!"))
	} else {
		hints = append(hints, keySt.Render("y")+" "+dimSt.Render("copy"))
	}
	hints = append(hints, keySt.Render("Esc")+" "+dimSt.Render("close"))
	hintBar := strings.Join(hints, "  ") + "  " + scrollInfo

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Render(titleBar + "\n" + o.content.View() + "\n" + hintBar)

	return lipgloss.Place(o.width, o.height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
}
