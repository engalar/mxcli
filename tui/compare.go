package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CompareKind determines the comparison layout.
type CompareKind int

const (
	CompareNDSL    CompareKind = iota // NDSL | NDSL
	CompareNDSLMDL                    // NDSL | MDL
	CompareMDL                        // MDL | MDL
)

// CompareFocus indicates which pane has focus.
type CompareFocus int

const (
	CompareFocusLeft CompareFocus = iota
	CompareFocusRight
)

// CompareLoadMsg carries loaded content for a compare pane.
type CompareLoadMsg struct {
	Side     CompareFocus
	Title    string
	NodeType string
	Content  string
	Err      error
}

// ComparePickMsg is emitted when user selects a qname from the fuzzy picker.
type ComparePickMsg struct {
	Side     CompareFocus
	QName    string
	NodeType string // tree node type (e.g. "Microflow", "Page")
	Kind     CompareKind
}

// CompareReloadMsg requests that both panes reload with the current kind.
type CompareReloadMsg struct {
	Kind CompareKind
}

// PickerItem holds a qualified name with its type for the fuzzy picker.
type PickerItem struct {
	QName    string
	NodeType string // e.g. "Microflow", "Workflow", "Page"
}

// comparePane is one side of the comparison view.
type comparePane struct {
	content  ContentView
	title    string
	qname    string
	nodeType string
	loading  bool
}

func (p comparePane) scrollPercent() int {
	return int(p.content.ScrollPercent() * 100)
}

func (p comparePane) lineInfo() string {
	return fmt.Sprintf("L%d/%d", p.content.YOffset()+1, p.content.TotalLines())
}

// CompareView is a side-by-side comparison overlay (lazygit-style).
type CompareView struct {
	visible      bool
	kind         CompareKind
	focus        CompareFocus
	left         comparePane
	right        comparePane
	sync         bool // synchronized scrolling
	copiedFlash  bool

	// Fuzzy picker
	picker        bool
	pickerInput   textinput.Model
	pickerItems   []PickerItem
	pickerMatches []pickerMatch
	pickerCursor  int
	pickerOffset  int
	pickerSide    CompareFocus

	width  int
	height int
}

type pickerMatch struct {
	item  PickerItem
	score int
}

const pickerMaxShow = 12

func NewCompareView() CompareView {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.Placeholder = "type to search..."
	ti.CharLimit = 200
	return CompareView{pickerInput: ti}
}

func (c *CompareView) Show(kind CompareKind, w, h int) {
	c.visible = true
	c.kind = kind
	c.focus = CompareFocusLeft
	c.width = w
	c.height = h
	c.picker = false
	c.sync = false
	pw, ph := c.paneDimensions()
	c.left.content = NewContentView(pw, ph)
	c.right.content = NewContentView(pw, ph)
}

func (c CompareView) paneDimensions() (int, int) {
	pw := (c.width - 6) / 2 // borders + gap
	ph := c.height - 4       // header + footer + borders
	if pw < 20 {
		pw = 20
	}
	if ph < 5 {
		ph = 5
	}
	return pw, ph
}

func (c *CompareView) Hide()              { c.visible = false; c.picker = false }
func (c CompareView) IsVisible() bool     { return c.visible }
func (c *CompareView) SetItems(items []PickerItem) { c.pickerItems = items }

func (c *CompareView) SetContent(side CompareFocus, title, nodeType, content string) {
	p := c.pane(side)
	p.title = title
	p.qname = title
	p.nodeType = nodeType
	p.loading = false
	p.content.SetContent(content)
	p.content.GotoTop()
}

func (c *CompareView) SetLoading(side CompareFocus) {
	p := c.pane(side)
	p.loading = true
	p.content.SetContent("Loading...")
}

func (c CompareView) emitReload() tea.Cmd {
	kind := c.kind
	return func() tea.Msg {
		return CompareReloadMsg{Kind: kind}
	}
}

func (c *CompareView) pane(side CompareFocus) *comparePane {
	if side == CompareFocusRight {
		return &c.right
	}
	return &c.left
}

func (c *CompareView) focusedPane() *comparePane { return c.pane(c.focus) }

// --- Picker ---

func (c *CompareView) openPicker() {
	c.picker = true
	c.pickerSide = c.focus
	c.pickerInput.SetValue("")
	c.pickerInput.Focus()
	c.pickerCursor = 0
	c.pickerOffset = 0
	c.filterPicker()
}

func (c *CompareView) closePicker() { c.picker = false; c.pickerInput.Blur() }

func (c *CompareView) filterPicker() {
	query := strings.TrimSpace(c.pickerInput.Value())
	c.pickerMatches = nil
	for _, it := range c.pickerItems {
		if query == "" {
			c.pickerMatches = append(c.pickerMatches, pickerMatch{item: it})
			continue
		}
		if ok, sc := fuzzyScore(it.QName, query); ok {
			c.pickerMatches = append(c.pickerMatches, pickerMatch{item: it, score: sc})
		}
	}
	// Sort by score descending (insertion sort, small n)
	for i := 1; i < len(c.pickerMatches); i++ {
		for j := i; j > 0 && c.pickerMatches[j].score > c.pickerMatches[j-1].score; j-- {
			c.pickerMatches[j], c.pickerMatches[j-1] = c.pickerMatches[j-1], c.pickerMatches[j]
		}
	}
	if c.pickerCursor >= len(c.pickerMatches) {
		c.pickerCursor = max(0, len(c.pickerMatches)-1)
	}
	c.pickerOffset = 0
}

func (c *CompareView) pickerDown() {
	if len(c.pickerMatches) == 0 {
		return
	}
	c.pickerCursor++
	if c.pickerCursor >= len(c.pickerMatches) {
		c.pickerCursor = 0
		c.pickerOffset = 0
	} else if c.pickerCursor >= c.pickerOffset+pickerMaxShow {
		c.pickerOffset = c.pickerCursor - pickerMaxShow + 1
	}
}

func (c *CompareView) pickerUp() {
	if len(c.pickerMatches) == 0 {
		return
	}
	c.pickerCursor--
	if c.pickerCursor < 0 {
		c.pickerCursor = len(c.pickerMatches) - 1
		c.pickerOffset = max(0, c.pickerCursor-pickerMaxShow+1)
	} else if c.pickerCursor < c.pickerOffset {
		c.pickerOffset = c.pickerCursor
	}
}

func (c CompareView) pickerSelected() PickerItem {
	if len(c.pickerMatches) == 0 || c.pickerCursor >= len(c.pickerMatches) {
		return PickerItem{}
	}
	return c.pickerMatches[c.pickerCursor].item
}

// --- Update ---

func (c CompareView) Update(msg tea.Msg) (CompareView, tea.Cmd) {
	if !c.visible {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if c.picker {
			return c.updatePicker(msg)
		}
		return c.updateNormal(msg)

	case tea.MouseMsg:
		if c.picker {
			return c, nil
		}
		return c.updateMouse(msg)

	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		pw, ph := c.paneDimensions()
		c.left.content.SetSize(pw, ph)
		c.right.content.SetSize(pw, ph)
	}
	return c, nil
}

func (c CompareView) updatePicker(msg tea.KeyMsg) (CompareView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		c.closePicker()
		return c, nil
	case "enter":
		selected := c.pickerSelected()
		c.closePicker()
		if selected.QName != "" {
			return c, func() tea.Msg {
				return ComparePickMsg{Side: c.pickerSide, QName: selected.QName, NodeType: selected.NodeType, Kind: c.kind}
			}
		}
		return c, nil
	case "up", "ctrl+p":
		c.pickerUp()
	case "down", "ctrl+n":
		c.pickerDown()
	default:
		var cmd tea.Cmd
		c.pickerInput, cmd = c.pickerInput.Update(msg)
		c.filterPicker()
		return c, cmd
	}
	return c, nil
}

func (c CompareView) updateNormal(msg tea.KeyMsg) (CompareView, tea.Cmd) {
	// When content is searching, forward all keys to it
	if c.focusedPane().content.IsSearching() {
		p := c.focusedPane()
		var cmd tea.Cmd
		p.content, cmd = p.content.Update(msg)
		return c, cmd
	}

	switch msg.String() {
	case "esc", "q":
		c.visible = false
		return c, nil

	// Focus switching — lazygit style: Tab only
	case "tab":
		if c.focus == CompareFocusLeft {
			c.focus = CompareFocusRight
		} else {
			c.focus = CompareFocusLeft
		}
		return c, nil

	// Fuzzy picker
	case "/":
		c.openPicker()
		return c, nil

	// Mode switching — reload both panes with new kind
	case "1":
		c.kind = CompareNDSL
		return c, c.emitReload()
	case "2":
		c.kind = CompareNDSLMDL
		return c, c.emitReload()
	case "3":
		c.kind = CompareMDL
		return c, c.emitReload()

	// Refresh both panes
	case "r":
		return c, c.emitReload()

	// Sync scroll toggle
	case "s":
		c.sync = !c.sync
		return c, nil

	// Copy focused pane content to clipboard
	case "y":
		_ = writeClipboard(c.focusedPane().content.PlainText())
		c.copiedFlash = true
		return c, func() tea.Msg {
			time.Sleep(time.Second)
			return compareFlashClearMsg{}
		}

	// Scroll — forward j/k/arrows/pgup/pgdn/g/G to focused viewport
	default:
		c.copiedFlash = false
		p := c.focusedPane()
		var cmd tea.Cmd
		p.content, cmd = p.content.Update(msg)

		if c.sync {
			c.syncOtherPane()
		}
		return c, cmd
	}
}

func (c CompareView) updateMouse(msg tea.MouseMsg) (CompareView, tea.Cmd) {
	// Determine which pane the mouse is in
	pw, _ := c.paneDimensions()
	leftEnd := pw + 3 // border + padding
	if msg.X < leftEnd {
		c.focus = CompareFocusLeft
	} else {
		c.focus = CompareFocusRight
	}

	// Forward mouse to focused pane's ContentView
	p := c.focusedPane()
	p.content, _ = p.content.Update(msg)
	if c.sync {
		c.syncOtherPane()
	}
	return c, nil
}

func (c *CompareView) syncOtherPane() {
	src := c.focusedPane()
	other := &c.left
	if c.focus == CompareFocusLeft {
		other = &c.right
	}
	pct := src.content.ScrollPercent()
	otherMax := other.content.maxOffset()
	if otherMax > 0 {
		other.content.SetYOffset(int(pct * float64(otherMax)))
	}
}

// --- View ---

func (c CompareView) View() string {
	if !c.visible {
		return ""
	}

	pw, _ := c.paneDimensions()

	// Pane rendering
	leftView := c.renderPane(&c.left, CompareFocusLeft, pw)
	rightView := c.renderPane(&c.right, CompareFocusRight, pw)
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)

	// Status bar (lazygit-style)
	statusBar := c.renderStatusBar()
	result := content + "\n" + statusBar

	// Picker overlay
	if c.picker {
		result = lipgloss.Place(c.width, c.height,
			lipgloss.Center, lipgloss.Center,
			c.renderPicker(),
			lipgloss.WithWhitespaceBackground(lipgloss.Color("0")))
	}

	return result
}

func (c CompareView) renderPane(p *comparePane, side CompareFocus, pw int) string {
	focused := c.focus == side
	bc := lipgloss.Color("240")
	if focused {
		bc = lipgloss.Color("63")
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bc).
		Width(pw)

	titleSt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	dimSt := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	loadSt := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	// Title line: name [kind] scroll%
	title := p.title
	if title == "" {
		title = "—"
	}
	if p.loading {
		title += loadSt.Render(" ⏳")
	}
	kindTag := c.kindLabel(side)
	scrollInfo := fmt.Sprintf("%s %d%%", p.lineInfo(), p.scrollPercent())

	header := titleSt.Render(title) + " " + dimSt.Render(kindTag) +
		strings.Repeat(" ", max(1, pw-lipgloss.Width(title)-lipgloss.Width(kindTag)-len(scrollInfo)-4)) +
		dimSt.Render(scrollInfo)

	return border.Render(header + "\n" + p.content.View())
}

func (c CompareView) renderStatusBar() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	active := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	success := lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Bold(true)

	kindNames := []string{"NDSL|NDSL", "NDSL|MDL", "MDL|MDL"}

	var parts []string
	parts = append(parts, key.Render("/")+" "+dim.Render("pick"))
	parts = append(parts, key.Render("Tab")+" "+dim.Render("switch"))

	// Mode indicators
	for i, name := range kindNames {
		k := fmt.Sprintf("%d", i+1)
		if CompareKind(i) == c.kind {
			parts = append(parts, active.Render(k+" "+name))
		} else {
			parts = append(parts, key.Render(k)+" "+dim.Render(name))
		}
	}

	// Sync indicator
	syncLabel := "sync"
	if c.sync {
		parts = append(parts, active.Render("s "+syncLabel))
	} else {
		parts = append(parts, key.Render("s")+" "+dim.Render(syncLabel))
	}

	parts = append(parts, key.Render("/")+" "+dim.Render("search"))
	if si := c.focusedPane().content.SearchInfo(); si != "" {
		parts = append(parts, key.Render("n/N")+" "+active.Render(si))
	}
	parts = append(parts, key.Render("r")+" "+dim.Render("reload"))
	parts = append(parts, key.Render("j/k")+" "+dim.Render("scroll"))
	if c.copiedFlash {
		parts = append(parts, success.Render("✓ Copied!"))
	} else {
		parts = append(parts, key.Render("y")+" "+dim.Render("copy"))
	}
	parts = append(parts, key.Render("Esc")+" "+dim.Render("close"))

	return lipgloss.NewStyle().Width(c.width).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Render(" " + strings.Join(parts, "  "))
}

func (c CompareView) kindLabel(side CompareFocus) string {
	switch c.kind {
	case CompareNDSL:
		return "[NDSL]"
	case CompareNDSLMDL:
		if side == CompareFocusLeft {
			return "[NDSL]"
		}
		return "[MDL]"
	case CompareMDL:
		return "[MDL]"
	}
	return ""
}

func (c CompareView) renderPicker() string {
	selSt := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	normSt := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	dimSt := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleSt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))

	sideLabel := "LEFT"
	if c.pickerSide == CompareFocusRight {
		sideLabel = "RIGHT"
	}

	var sb strings.Builder
	sb.WriteString(titleSt.Render(fmt.Sprintf("Pick object (%s)", sideLabel)) + "\n\n")
	sb.WriteString(c.pickerInput.View() + "\n\n")

	end := min(c.pickerOffset+pickerMaxShow, len(c.pickerMatches))
	if c.pickerOffset > 0 {
		sb.WriteString(dimSt.Render("  ↑ more") + "\n")
	}
	typeSt := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	for i := c.pickerOffset; i < end; i++ {
		it := c.pickerMatches[i].item
		if i == c.pickerCursor {
			sb.WriteString(selSt.Render("▸ "+it.QName) + " " + typeSt.Render(it.NodeType) + "\n")
		} else {
			sb.WriteString(normSt.Render("  "+it.QName) + " " + dimSt.Render(it.NodeType) + "\n")
		}
	}
	if end < len(c.pickerMatches) {
		sb.WriteString(dimSt.Render("  ↓ more") + "\n")
	}
	sb.WriteString("\n" + dimSt.Render(fmt.Sprintf("  %d/%d matches", len(c.pickerMatches), len(c.pickerItems))))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(min(60, c.width-10)).
		Render(sb.String())
}

// --- Utilities ---

// fuzzyScore checks if query chars appear in order within target (fzf-style).
func fuzzyScore(target, query string) (bool, int) {
	tLower := strings.ToLower(target)
	qLower := strings.ToLower(query)
	if len(qLower) == 0 {
		return true, 0
	}
	if len(qLower) > len(tLower) {
		return false, 0
	}
	score := 0
	qi := 0
	prevMatched := false
	for ti := range len(tLower) {
		if qi >= len(qLower) {
			break
		}
		if tLower[ti] == qLower[qi] {
			qi++
			if ti == 0 {
				score += 7
			} else if target[ti-1] == '.' || target[ti-1] == '_' {
				score += 5
			} else if target[ti] >= 'A' && target[ti] <= 'Z' {
				score += 5
			}
			if prevMatched {
				score += 3
			}
			prevMatched = true
		} else {
			prevMatched = false
		}
	}
	if qi < len(qLower) {
		return false, 0
	}
	score += max(0, 50-len(tLower))
	return true, score
}

