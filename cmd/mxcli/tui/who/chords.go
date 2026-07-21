package who

import tea "github.com/charmbracelet/bubbletea"

type ChordNode struct {
	Key      string
	Label    string
	Children []ChordNode
	Action   func() tea.Cmd
}

type ChordPath struct {
	Nodes []ChordNode
	Chord string
}

func (n *ChordNode) FindChild(key string) *ChordNode {
	for i := range n.Children {
		if n.Children[i].Key == key {
			return &n.Children[i]
		}
	}
	return nil
}

func LeaderLabel(path []ChordNode) string {
	if len(path) == 0 {
		return "Menu"
	}
	return path[len(path)-1].Label
}

func NextKeys(node *ChordNode) []string {
	if node == nil || len(node.Children) == 0 {
		return nil
	}
	keys := make([]string, 0, len(node.Children))
	for _, c := range node.Children {
		keys = append(keys, c.Key)
	}
	return keys
}
