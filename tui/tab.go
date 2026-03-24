package tui

import "encoding/json"

// LoadTreeMsg carries parsed tree nodes from project-tree output.
type LoadTreeMsg struct {
	Nodes []*TreeNode
	Err   error
}

// OpenOverlayMsg requests that the overlay be shown with highlighted full content.
type OpenOverlayMsg struct {
	Title   string
	Content string
}

// OpenImageOverlayMsg requests a full-size image overlay for a list of image paths.
type OpenImageOverlayMsg struct {
	Title string
	Paths []string
}

// ParseTree parses JSON from mxcli project-tree output.
func ParseTree(jsonStr string) ([]*TreeNode, error) {
	var nodes []*TreeNode
	if err := json.Unmarshal([]byte(jsonStr), &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// NavState stores one level of navigation for back/forward in a tab.
type NavState struct {
	Path       []string
	ParentNode *TreeNode
	CursorIdx  int
}

// Tab represents a single workspace tab with its own Miller view and navigation.
type Tab struct {
	ID          int
	Label       string
	ProjectPath string
	Miller      MillerView
	AllNodes    []*TreeNode
}

// NewTab creates a tab for the given project.
func NewTab(id int, projectPath string, previewEngine *PreviewEngine, allNodes []*TreeNode) Tab {
	miller := NewMillerView(previewEngine)
	if allNodes != nil {
		miller.SetRootNodes(allNodes)
	}
	label := "Tab " + itoa(id)
	if len(allNodes) > 0 {
		label = "Project"
	}
	return Tab{
		ID:          id,
		Label:       label,
		ProjectPath: projectPath,
		Miller:      miller,
		AllNodes:    allNodes,
	}
}

// CloneTab creates a new tab at the same location.
func (t *Tab) CloneTab(newID int, previewEngine *PreviewEngine) Tab {
	newTab := NewTab(newID, t.ProjectPath, previewEngine, t.AllNodes)
	return newTab
}

// UpdateLabel derives the tab label from the Miller breadcrumb.
func (t *Tab) UpdateLabel() {
	crumbs := t.Miller.Breadcrumb()
	if len(crumbs) == 0 {
		t.Label = "Project"
		return
	}
	if len(crumbs) == 1 {
		t.Label = crumbs[0]
		return
	}
	// Show deepest 2 levels
	t.Label = crumbs[len(crumbs)-2] + "/" + crumbs[len(crumbs)-1]
}

func itoa(n int) string {
	return string(rune('0'+n%10)) + ""
}

// flattenQualifiedNames collects all qualified names from the tree for fuzzy picking.
func flattenQualifiedNames(nodes []*TreeNode) []PickerItem {
	var items []PickerItem
	var walk func([]*TreeNode)
	walk = func(ns []*TreeNode) {
		for _, n := range ns {
			if n.QualifiedName != "" {
				items = append(items, PickerItem{QName: n.QualifiedName, NodeType: n.Type})
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(nodes)
	return items
}
