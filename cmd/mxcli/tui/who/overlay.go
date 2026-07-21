package who

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

type ExecuteMsg struct {
	Chord string
}

type PopMsg struct{}

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "214", Dark: "214"})
	categoryStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "62", Dark: "63"})
	actionStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "255", Dark: "255"})
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "241", Dark: "243"})
	hintKeyStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "214", Dark: "214"})
	hintDescStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "245", Dark: "243"})
)

type Overlay struct {
	rootNode *ChordNode
	node     *ChordNode
	path     []ChordNode
}

func NewOverlay(root ChordNode) Overlay {
	r := new(ChordNode)
	*r = root
	return Overlay{
		rootNode: r,
		node:     r,
	}
}

func (o Overlay) Mode() kernel.ViewMode  { return kernel.ModeCommandPalette }
func (o Overlay) Hints() []kernel.Hint   { return nil }

func (o Overlay) StatusInfo() kernel.StatusInfo {
	label := "Menu"
	if len(o.path) > 0 {
		label = o.path[len(o.path)-1].Label
	}
	return kernel.StatusInfo{
		Breadcrumb: []string{"SPC", label},
		Mode:       "LEADER",
	}
}

func (o Overlay) Update(msg tea.Msg) (interface{}, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if len(o.path) > 0 {
				o.path = o.path[:len(o.path)-1]
				if len(o.path) > 0 {
					o.node = &o.path[len(o.path)-1]
				} else {
					o.node = o.rootNode
				}
				return o, nil
			}
			return o, func() tea.Msg { return PopMsg{} }

		case "q":
			return o, func() tea.Msg { return PopMsg{} }

		default:
			child := o.node.FindChild(msg.String())
			if child == nil {
				return o, nil
			}
			o.path = append(o.path, *child)
			o.node = child
			if child.Action != nil {
				return o, func() tea.Msg { return ExecuteMsg{Chord: o.chordString()} }
			}
			return o, nil
		}
	}
	return o, nil
}

func (o Overlay) chordString() string {
	var parts []string
	for _, n := range o.path {
		parts = append(parts, n.Key)
	}
	return strings.Join(parts, "")
}

func (o Overlay) Render(width, height int) string {
	boxW := min(width-8, 80)
	innerW := boxW - 4

	var sb strings.Builder

	prefix := "SPC"
	for _, n := range o.path {
		prefix += " " + n.Key
	}
	label := "Menu"
	if len(o.path) > 0 {
		label = o.path[len(o.path)-1].Label
	}

	sb.WriteString(titleStyle.Render(fmt.Sprintf(" %s ▸ %s", prefix, label)) + "\n\n")

	for _, child := range o.node.Children {
		isCat := len(child.Children) > 0
		suffix := ""
		if isCat {
			suffix = dimStyle.Render("  ▸")
		}
		pad := innerW - len(child.Key) - len(child.Label) - 2
		if pad < 1 {
			pad = 1
		}
		keyStr := categoryStyle.Render(" " + child.Key)
		nameStr := actionStyle.Render(" " + child.Label)
		line := keyStr + nameStr + strings.Repeat(" ", pad) + suffix
		sb.WriteString("  " + line + "\n")
	}

	sb.WriteString("\n")
	navHint := hintKeyStyle.Render("Esc") + hintDescStyle.Render("=cancel ") + hintKeyStyle.Render("q") + hintDescStyle.Render("=quit")
	sb.WriteString("  " + navHint)

	content := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "214", Dark: "214"}).
		Padding(1, 2).
		Width(boxW).
		Render(sb.String())

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
