package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

// BrowserView wraps MillerView and absorbs action keys from the normal browsing mode.
// It implements the View interface.
type BrowserView struct {
	miller        MillerView
	tab           *Tab
	allNodes      []*TreeNode
	mxcliPath     string
	projectPath   string
	previewEngine *PreviewEngine
}

// NewBrowserView creates a BrowserView wrapping the Miller view from the given tab.
func NewBrowserView(tab *Tab, mxcliPath string, engine *PreviewEngine) BrowserView {
	return BrowserView{
		miller:        tab.Miller,
		tab:           tab,
		allNodes:      tab.AllNodes,
		mxcliPath:     mxcliPath,
		projectPath:   tab.ProjectPath,
		previewEngine: engine,
	}
}

// Mode returns ModeBrowser.
func (bv BrowserView) Mode() ViewMode {
	return ModeBrowser
}

// Hints returns context-sensitive hints for the browser view.
func (bv BrowserView) Hints() []Hint {
	if bv.miller.focusedColumn().IsFilterActive() {
		return FilterActiveHints
	}
	return BrowserHints
}

// StatusInfo builds status bar data from the Miller view state.
func (bv BrowserView) StatusInfo() StatusInfo {
	crumbs := bv.miller.Breadcrumb()

	mode := "MDL"
	if bv.miller.preview.mode == PreviewNDSL {
		mode = "NDSL"
	}

	col := bv.miller.current
	position := fmt.Sprintf("%d/%d", col.cursor+1, col.ItemCount())

	return StatusInfo{
		Breadcrumb: crumbs,
		Position:   position,
		Mode:       mode,
	}
}

// Render sets the miller size and returns its rendered output with an LLM anchor prefix.
func (bv BrowserView) Render(width, height int) string {
	bv.miller.SetSize(width, height)
	rendered := bv.miller.View()

	// Embed LLM anchor as muted prefix on the first line
	info := bv.StatusInfo()
	anchor := fmt.Sprintf("[mxcli:browse] %s  %s  %s",
		strings.Join(info.Breadcrumb, " > "), info.Position, info.Mode)
	anchorStr := lipgloss.NewStyle().Foreground(kernel.MutedColor).Faint(true).Render(anchor)

	if idx := strings.IndexByte(rendered, '\n'); idx >= 0 {
		rendered = anchorStr + rendered[idx:]
	} else {
		rendered = anchorStr
	}
	return rendered
}

// Update handles messages for the browser view.
// Keys not handled here (q, ?, t, T, W, 1-9, [, ], ctrl+c) return (bv, nil)
// so App can handle them.
func (bv BrowserView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case PreviewReadyMsg, PreviewLoadingMsg, CursorChangedMsg, animTickMsg, previewDebounceMsg:
		var cmd tea.Cmd
		bv.miller, cmd = bv.miller.Update(msg)
		return bv, cmd

	case tea.MouseMsg:
		var cmd tea.Cmd
		bv.miller, cmd = bv.miller.Update(msg)
		return bv, cmd

	case tea.KeyMsg:
		return bv.handleKey(msg)
	}

	return bv, nil
}

func (bv BrowserView) handleKey(msg tea.KeyMsg) (View, tea.Cmd) {
	// If filter is active, forward all keys to miller
	if bv.miller.focusedColumn().IsFilterActive() {
		var cmd tea.Cmd
		bv.miller, cmd = bv.miller.Update(msg)
		return bv, cmd
	}

	// Navigation keys: forward to miller
	switch msg.String() {
	case "j", "k", "g", "G", "h", "l", "left", "right", "up", "down",
		"enter", "tab", "/", "n", "N":
		var cmd tea.Cmd
		bv.miller, cmd = bv.miller.Update(msg)
		return bv, cmd
	}

	// Action keys no longer handled here — all go through Space leader menu.
	// Keys not handled: let App handle (q, ?, :, Space, etc.)
	return bv, nil
}

// --- Load helpers (moved from app.go) ---

func (bv BrowserView) overlayOpts(qname, nodeType string, isNDSL bool) OverlayViewOpts {
	return OverlayViewOpts{
		QName:       qname,
		NodeType:    nodeType,
		IsNDSL:      isNDSL,
		Switchable:  true,
		MxcliPath:   bv.mxcliPath,
		ProjectPath: bv.projectPath,
	}
}

func (bv BrowserView) runBsonOverlay(bsonType, qname, nodeType string) tea.Cmd {
	mxcliPath := bv.mxcliPath
	projectPath := bv.projectPath
	opts := bv.overlayOpts(qname, nodeType, true)
	return func() tea.Msg {
		args := []string{"bson", "dump", "-p", projectPath, "--format", "ndsl",
			"--type", bsonType, "--object", qname}
		out, err := runMxcli(mxcliPath, args...)
		out = StripBanner(out)
		title := fmt.Sprintf("BSON: %s", qname)
		if err != nil {
			return OpenOverlayMsg{Title: title, Content: "Error: " + out, Opts: opts}
		}
		return OpenOverlayMsg{Title: title, Content: HighlightNDSL(out), Opts: opts}
	}
}

func (bv BrowserView) runMDLOverlay(nodeType, qname string) tea.Cmd {
	mxcliPath := bv.mxcliPath
	projectPath := bv.projectPath
	opts := bv.overlayOpts(qname, nodeType, false)
	return func() tea.Msg {
		out, err := runMxcli(mxcliPath, "-p", projectPath, "-c", buildDescribeCmd(nodeType, qname))
		out = StripBanner(out)
		title := fmt.Sprintf("MDL: %s", qname)
		if err != nil {
			return OpenOverlayMsg{Title: title, Content: "Error: " + out, Opts: opts}
		}
		return OpenOverlayMsg{Title: title, Content: DetectAndHighlight(out), Opts: opts}
	}
}

func (bv BrowserView) loadBsonNDSL(qname, nodeType string, side CompareFocus) tea.Cmd {
	return loadBsonNDSL(bv.mxcliPath, bv.projectPath, qname, nodeType, side)
}

// navigateToNode resets the miller view to root and drills down to the node
// matching the given qualified name. Returns a preview request command.
func (bv *BrowserView) navigateToNode(qname string) tea.Cmd {
	path := findNodePath(bv.allNodes, qname)
	if len(path) == 0 {
		return nil
	}

	// Reset to root
	bv.miller.SetRootNodes(bv.allNodes)

	// Drill in for each intermediate node in the path (all except the last)
	for i := 0; i < len(path)-1; i++ {
		node := path[i]
		// Find this node's index in the current column
		idx := -1
		for j, item := range bv.miller.current.items {
			if item.Node == node {
				idx = j
				break
			}
		}
		if idx < 0 {
			return nil
		}
		bv.miller.current.SetCursor(idx)
		bv.miller, _ = bv.miller.drillIn()
	}

	// Select the final node
	target := path[len(path)-1]
	for j, item := range bv.miller.current.items {
		if item.Node == target {
			bv.miller.current.SetCursor(j)
			break
		}
	}

	// Request preview for the selected node
	if target.QualifiedName != "" && target.Type != "" && len(target.Children) == 0 {
		return bv.miller.previewEngine.RequestPreview(target.Type, target.QualifiedName, bv.miller.preview.mode)
	}
	return nil
}

// findNodePath walks the tree to find the chain of nodes from root to the node
// with the matching qualified name. Returns nil if not found.
func findNodePath(nodes []*TreeNode, qname string) []*TreeNode {
	for _, n := range nodes {
		if n.QualifiedName == qname {
			return []*TreeNode{n}
		}
		if len(n.Children) > 0 {
			if sub := findNodePath(n.Children, qname); sub != nil {
				return append([]*TreeNode{n}, sub...)
			}
		}
	}
	return nil
}

func (bv BrowserView) openDiagram(nodeType, qualifiedName string) tea.Cmd {
	mxcliPath := bv.mxcliPath
	projectPath := bv.projectPath
	return func() tea.Msg {
		out, err := runMxcli(mxcliPath, "describe", "-p", projectPath,
			"--format", "elk", nodeType, qualifiedName)
		if err != nil {
			return CmdResultMsg{Output: out, Err: err}
		}
		htmlContent := buildDiagramHTML(out, nodeType, qualifiedName)
		tmpFile, err := os.CreateTemp("", "mxcli-diagram-*.html")
		if err != nil {
			return CmdResultMsg{Err: err}
		}
		if _, err := tmpFile.WriteString(htmlContent); err != nil {
			tmpFile.Close()
			return CmdResultMsg{Err: fmt.Errorf("writing diagram HTML: %w", err)}
		}
		tmpFile.Close()
		tmpPath := tmpFile.Name()
		openBrowser(tmpPath)
		time.AfterFunc(30*time.Second, func() { os.Remove(tmpPath) })
		return CmdResultMsg{Output: fmt.Sprintf("Opened diagram: %s", tmpPath)}
	}
}
