package panels

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeNode mirrors cmd/mxcli.TreeNode for JSON parsing.
type TreeNode struct {
	Label         string      `json:"label"`
	Type          string      `json:"type"`
	QualifiedName string      `json:"qualifiedName,omitempty"`
	Children      []*TreeNode `json:"children,omitempty"`
}

// nodeItem wraps TreeNode to implement ScrollListItem.
type nodeItem struct{ node *TreeNode }

func (n nodeItem) Label() string       { return n.node.Label }
func (n nodeItem) Icon() string        { return iconFor(n.node.Type) }
func (n nodeItem) Description() string { return "" }
func (n nodeItem) FilterValue() string { return n.node.Label }

// ModulesPanel is the left column: a list of top-level tree nodes (modules + special nodes).
type ModulesPanel struct {
	scrollList      ScrollList
	nodes           []*TreeNode
	navigationStack [][]*TreeNode
	breadcrumb      Breadcrumb
	focused         bool
	width           int
	height          int
}

func NewModulesPanel(width, height int) ModulesPanel {
	sl := NewScrollList("Project")
	sl.SetSize(width, height-2) // reserve space for border
	return ModulesPanel{scrollList: sl, width: width, height: height}
}

// LoadTreeMsg carries parsed tree nodes from project-tree output.
type LoadTreeMsg struct {
	Nodes []*TreeNode
	Err   error
}

// ParseTree parses JSON from mxcli project-tree output.
func ParseTree(jsonStr string) ([]*TreeNode, error) {
	var nodes []*TreeNode
	if err := json.Unmarshal([]byte(jsonStr), &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (p *ModulesPanel) SetNodes(nodes []*TreeNode) {
	p.nodes = nodes
	p.navigationStack = nil
	p.breadcrumb = Breadcrumb{}
	p.breadcrumb.SetWidth(p.width - 4)
	p.setScrollListNodes(nodes)
}

func (p *ModulesPanel) setScrollListNodes(nodes []*TreeNode) {
	items := make([]ScrollListItem, len(nodes))
	for i, n := range nodes {
		items[i] = nodeItem{node: n}
	}
	p.scrollList.SetItems(items)
}

// DrillIn pushes current nodes onto the stack and displays children.
func (p *ModulesPanel) DrillIn(parentLabel string, children []*TreeNode) {
	p.navigationStack = append(p.navigationStack, p.currentNodes())
	p.breadcrumb.Push(parentLabel)
	p.setScrollListNodes(children)
}

// DrillBack pops the navigation stack and returns true if it went back, false if already at root.
func (p *ModulesPanel) DrillBack() bool {
	depth := len(p.navigationStack)
	if depth == 0 {
		return false
	}
	prev := p.navigationStack[depth-1]
	p.navigationStack = p.navigationStack[:depth-1]
	p.breadcrumb.PopTo(p.breadcrumb.Depth() - 1)
	p.setScrollListNodes(prev)
	return true
}

func (p *ModulesPanel) currentNodes() []*TreeNode {
	total := len(p.scrollList.items)
	nodes := make([]*TreeNode, total)
	for i, item := range p.scrollList.items {
		nodes[i] = item.(nodeItem).node
	}
	return nodes
}

func (p ModulesPanel) SelectedNode() *TreeNode {
	selected := p.scrollList.SelectedItem()
	if selected == nil {
		return nil
	}
	return selected.(nodeItem).node
}

func (p *ModulesPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
	bcHeight := 0
	if p.breadcrumb.Depth() > 0 {
		bcHeight = 1
	}
	p.scrollList.SetSize(w-2, h-2-bcHeight) // subtract border + optional breadcrumb
	p.breadcrumb.SetWidth(w - 4)
}

func (p ModulesPanel) IsFilterActive() bool { return p.scrollList.IsFilterActive() }

func (p *ModulesPanel) SetFocused(f bool) {
	p.focused = f
	p.scrollList.SetFocused(f)
}

func (p ModulesPanel) Update(msg tea.Msg) (ModulesPanel, tea.Cmd) {
	// Adjust mouse Y for breadcrumb height before forwarding to scroll list
	if mouseMsg, ok := msg.(tea.MouseMsg); ok && p.breadcrumb.Depth() > 0 {
		mouseMsg.Y -= 1 // breadcrumb line
		msg = mouseMsg
	}
	var cmd tea.Cmd
	p.scrollList, cmd = p.scrollList.Update(msg)
	return p, cmd
}

func (p ModulesPanel) View() string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(p.focused))

	var content string
	if p.breadcrumb.Depth() > 0 {
		content = p.breadcrumb.View() + "\n" + p.scrollList.View()
	} else {
		content = p.scrollList.View()
	}
	return border.Render(content)
}

func borderColor(focused bool) lipgloss.Color {
	if focused {
		return lipgloss.Color("63")
	}
	return lipgloss.Color("240")
}
