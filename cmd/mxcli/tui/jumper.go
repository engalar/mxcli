package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

const jumperMaxShow = 12

// JumpToNodeMsg is emitted when the user selects a node from the jumper.
type JumpToNodeMsg struct {
	QName    string
	NodeType string
}

// JumperView is a fuzzy-search modal for jumping to any node in the project.
type JumperView struct {
	input      textinput.Model
	list       FuzzyList
	width      int
	height     int
	history    []string
	historyIdx int // -1 = not browsing history
}

// NewJumperView creates a JumperView populated with the given items.
func NewJumperView(items []PickerItem, width, height int) JumperView {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.Placeholder = "jump to... (mf: nf: wf: pg: en:)"
	ti.CharLimit = 200
	ti.Focus()

	jv := JumperView{
		input:      ti,
		list:       NewFuzzyList(items, jumperMaxShow),
		width:      width,
		height:     height,
		history:    LoadJumperHistory(),
		historyIdx: -1,
	}
	return jv
}

// --- View interface ---

// Update handles key messages for the jumper modal.
func (jv JumperView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return jv, func() tea.Msg { return PopViewMsg{} }
		case "enter":
			sel := jv.list.Selected()
			if sel.QName != "" {
				query := strings.TrimSpace(jv.input.Value())
				AddJumperHistoryEntry(query)
				qname := sel.QName
				nodeType := sel.NodeType
				return jv, func() tea.Msg { return JumpToNodeMsg{QName: qname, NodeType: nodeType} }
			}
			return jv, func() tea.Msg { return PopViewMsg{} }
		case "up", "ctrl+p":
			inputVal := strings.TrimSpace(jv.input.Value())
			if (inputVal == "" || jv.historyIdx >= 0) && len(jv.history) > 0 {
				// Navigate history upward (older entries)
				nextIdx := jv.historyIdx + 1
				if nextIdx >= len(jv.history) {
					nextIdx = len(jv.history) - 1
				}
				jv.historyIdx = nextIdx
				jv.input.SetValue(jv.history[jv.historyIdx])
				jv.input.CursorEnd()
				jv.list.Filter(jv.history[jv.historyIdx])
			} else {
				jv.list.MoveUp()
			}
		case "down", "ctrl+n":
			if jv.historyIdx >= 0 {
				// Navigate history downward (newer entries)
				nextIdx := jv.historyIdx - 1
				if nextIdx < 0 {
					// Past the newest entry — clear input, exit history mode
					jv.historyIdx = -1
					jv.input.SetValue("")
					jv.list.Filter("")
				} else {
					jv.historyIdx = nextIdx
					jv.input.SetValue(jv.history[jv.historyIdx])
					jv.input.CursorEnd()
					jv.list.Filter(jv.history[jv.historyIdx])
				}
			} else {
				jv.list.MoveDown()
			}
		default:
			// Any regular keystroke exits history browsing mode
			jv.historyIdx = -1
			var cmd tea.Cmd
			jv.input, cmd = jv.input.Update(msg)
			jv.list.Filter(jv.input.Value())
			return jv, cmd
		}

	case tea.WindowSizeMsg:
		jv.width = msg.Width
		jv.height = msg.Height
	}
	return jv, nil
}

// Render draws the jumper as a centered modal box with an LLM anchor prefix.
func (jv JumperView) Render(width, height int) string {
	selSt := kernel.SelectedItemStyle
	normSt := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	dimSt := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	typeSt := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	boxWidth := max(30, min(60, width-10))

	fl := &jv.list

	// LLM anchor embedded at the top of the box content
	query := strings.TrimSpace(jv.input.Value())
	anchor := fmt.Sprintf("[mxcli:jump] > %s  %d matches", query, len(fl.Matches))
	anchorStr := lipgloss.NewStyle().Foreground(kernel.MutedColor).Faint(true).Render(anchor)

	var sb strings.Builder
	sb.WriteString(anchorStr + "\n")
	sb.WriteString(jv.input.View() + "\n\n")

	end := fl.VisibleEnd()
	if fl.Offset > 0 {
		sb.WriteString(dimSt.Render("  ↑ more") + "\n")
	}
	for i := fl.Offset; i < end; i++ {
		it := fl.Matches[i].item
		icon := kernel.IconFor(it.NodeType)
		label := icon + " " + it.QName
		typeLabel := it.NodeType
		if i == fl.Cursor {
			sb.WriteString(selSt.Render(" "+label) + " " + typeSt.Render(typeLabel) + "\n")
		} else {
			sb.WriteString(normSt.Render(" "+label) + " " + dimSt.Render(typeLabel) + "\n")
		}
	}
	if end < len(fl.Matches) {
		sb.WriteString(dimSt.Render("  ↓ more") + "\n")
	}
	sb.WriteString("\n" + dimSt.Render(fmt.Sprintf("  %d/%d", len(fl.Matches), len(fl.Items))))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(boxWidth).
		Render(sb.String())

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
}

// Hints returns jumper-specific key hints.
func (jv JumperView) Hints() []Hint {
	return []Hint{
		{Key: "↑/↓", Label: "navigate"},
		{Key: "Enter", Label: "jump"},
		{Key: "Esc", Label: "cancel"},
	}
}

// StatusInfo returns match count information.
func (jv JumperView) StatusInfo() StatusInfo {
	return StatusInfo{
		Mode:     "Jump",
		Position: fmt.Sprintf("%d/%d", len(jv.list.Matches), len(jv.list.Items)),
	}
}

// Mode returns ModeJumper.
func (jv JumperView) Mode() ViewMode {
	return ModeJumper
}
