package who

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "214", Dark: "214"})
	keyStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "62", Dark: "63"})
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "245", Dark: "243"})
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "241", Dark: "241"})
)

type Overlay struct {
	path   []ChordNode
	node   *ChordNode
	width  int
	height int
}

func NewOverlay(path []ChordNode, node *ChordNode) Overlay {
	return Overlay{path: path, node: node}
}

func (o Overlay) SetSize(w, h int) {
	o.width = w
	o.height = h
}

func (o Overlay) Render(width, height int) string {
	prefix := "SPC"
	for _, n := range o.path {
		prefix += " " + n.Key
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(prefix+" ▸ "+LeaderLabel(o.path)) + "\n\n")

	node := o.node
	if node == nil {
		node = &ChordNode{Children: nil}
	}

	for _, child := range node.Children {
		hasSub := len(child.Children) > 0
		suffix := ""
		if hasSub {
			suffix = dimStyle.Render(" ▸")
		}
		line := fmt.Sprintf("  %s  %s%s", keyStyle.Render(child.Key), labelStyle.Render(child.Label), suffix)
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n" + dimStyle.Render(fmt.Sprintf("  %d items", len(node.Children))))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "214", Dark: "214"}).
		Padding(1, 2).
		Render(sb.String())

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
