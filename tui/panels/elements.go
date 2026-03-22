package panels

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// elementItem wraps TreeNode with a display label for the elements panel.
type elementItem struct {
	node         *TreeNode
	displayLabel string
}

func (e elementItem) Label() string       { return e.displayLabel }
func (e elementItem) Icon() string        { return iconFor(e.node.Type) }
func (e elementItem) Description() string { return e.node.Type }
func (e elementItem) FilterValue() string { return e.node.Label }

// ElementsPanel is the middle column: children of the selected module node.
type ElementsPanel struct {
	scrollList ScrollList
	nodes      []*TreeNode
	focused    bool
	width      int
	height     int
}

func NewElementsPanel(width, height int) ElementsPanel {
	sl := NewScrollList("Elements")
	sl.SetSize(width-2, height-2)
	return ElementsPanel{scrollList: sl, width: width, height: height}
}

func (p *ElementsPanel) SetNodes(nodes []*TreeNode) {
	p.nodes = nodes
	items := make([]ScrollListItem, len(nodes))
	for i, n := range nodes {
		label := n.Label
		if len(n.Children) > 0 {
			label += " ▶"
		}
		items[i] = elementItem{
			node:         n,
			displayLabel: label,
		}
	}
	p.scrollList.SetItems(items)
}

func (p ElementsPanel) SelectedNode() *TreeNode {
	selected := p.scrollList.SelectedItem()
	if selected == nil {
		return nil
	}
	return selected.(elementItem).node
}

func (p *ElementsPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.scrollList.SetSize(w-2, h-2)
}

func (p ElementsPanel) IsFilterActive() bool { return p.scrollList.IsFilterActive() }

func (p *ElementsPanel) SetFocused(f bool) {
	p.focused = f
	p.scrollList.SetFocused(f)
}

func (p ElementsPanel) Update(msg tea.Msg) (ElementsPanel, tea.Cmd) {
	var cmd tea.Cmd
	p.scrollList, cmd = p.scrollList.Update(msg)
	return p, cmd
}

func (p ElementsPanel) View() string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(p.focused))
	return border.Render(p.scrollList.View())
}
