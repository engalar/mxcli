package panels

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ElementsPanel is the middle column: children of the selected module node.
type ElementsPanel struct {
	list    list.Model
	nodes   []*TreeNode
	focused bool
	width   int
	height  int
}

func NewElementsPanel(width, height int) ElementsPanel {
	delegate := newCustomDelegate(false)
	delegate.ShowDescription = true
	l := list.New(nil, delegate, width, height)
	l.Title = "Elements"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return ElementsPanel{list: l, width: width, height: height}
}

func (p *ElementsPanel) SetNodes(nodes []*TreeNode) {
	p.nodes = nodes
	items := make([]list.Item, len(nodes))
	for i, n := range nodes {
		label := n.Label
		if len(n.Children) > 0 {
			label += " ▶"
		}
		items[i] = nodeItem{node: &TreeNode{
			Label:         label,
			Type:          n.Type,
			QualifiedName: n.QualifiedName,
			Children:      n.Children,
		}}
	}
	p.list.SetItems(items)
	p.list.ResetSelected()
}

func (p ElementsPanel) SelectedNode() *TreeNode {
	selectedItem, ok := p.list.SelectedItem().(nodeItem)
	if !ok {
		return nil
	}
	return selectedItem.node
}

func (p *ElementsPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.list.SetWidth(w)
	p.list.SetHeight(h)
}

func (p *ElementsPanel) SetFocused(f bool) {
	p.focused = f
	p.list.SetDelegate(newCustomDelegate(f))
}

func (p ElementsPanel) Update(msg tea.Msg) (ElementsPanel, tea.Cmd) {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p ElementsPanel) View() string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(p.focused))
	return border.Render(p.list.View())
}
