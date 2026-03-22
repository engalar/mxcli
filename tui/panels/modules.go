package panels

import (
	"encoding/json"

	"github.com/charmbracelet/bubbles/list"
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

// nodeItem wraps TreeNode for the bubbles list.
type nodeItem struct{ node *TreeNode }

func (n nodeItem) Title() string       { return n.node.Label }
func (n nodeItem) Description() string { return n.node.Type }
func (n nodeItem) FilterValue() string { return n.node.Label }

// ModulesPanel is the left column: a list of top-level tree nodes (modules + special nodes).
type ModulesPanel struct {
	list    list.Model
	nodes   []*TreeNode
	focused bool
	width   int
	height  int
}

func NewModulesPanel(width, height int) ModulesPanel {
	delegate := newCustomDelegate(false)
	delegate.ShowDescription = false
	l := list.New(nil, delegate, width, height)
	l.SetShowTitle(true)
	l.Title = "Project"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return ModulesPanel{list: l, width: width, height: height}
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
	items := make([]list.Item, len(nodes))
	for i, n := range nodes {
		items[i] = nodeItem{node: n}
	}
	p.list.SetItems(items)
}

func (p ModulesPanel) SelectedNode() *TreeNode {
	selectedItem, ok := p.list.SelectedItem().(nodeItem)
	if !ok {
		return nil
	}
	return selectedItem.node
}

func (p *ModulesPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.list.SetWidth(w)
	p.list.SetHeight(h)
}

func (p *ModulesPanel) SetFocused(f bool) {
	p.focused = f
	p.list.SetDelegate(newCustomDelegate(f))
}

func (p ModulesPanel) Update(msg tea.Msg) (ModulesPanel, tea.Cmd) {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p ModulesPanel) View() string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(p.focused))
	return border.Render(p.list.View())
}

func newCustomDelegate(focused bool) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	if focused {
		d.Styles.SelectedTitle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("63")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 0, 0, 1)
		d.Styles.SelectedDesc = d.Styles.SelectedTitle.
			Foreground(lipgloss.Color("63"))
	} else {
		d.Styles.SelectedTitle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("240")).
			Foreground(lipgloss.Color("245")).
			Padding(0, 0, 0, 1)
		d.Styles.SelectedDesc = d.Styles.SelectedTitle.
			Foreground(lipgloss.Color("240"))
	}
	return d
}

func borderColor(focused bool) lipgloss.Color {
	if focused {
		return lipgloss.Color("63")
	}
	return lipgloss.Color("240")
}
