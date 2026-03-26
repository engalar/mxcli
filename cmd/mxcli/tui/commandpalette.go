package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const paletteMaxVisible = 16

// PaletteExecMsg is sent when the user selects a command from the palette.
// App should re-dispatch this as a synthetic KeyMsg.
type PaletteExecMsg struct {
	Key string
}

// PaletteCommand describes a single entry in the command palette.
type PaletteCommand struct {
	Name     string // display name, e.g. "Compare View"
	Key      string // shortcut key, e.g. "c"
	Category string // grouping label, e.g. "Navigation", "View"
}

// CommandPaletteView is a VS Code-style command palette with fuzzy search.
type CommandPaletteView struct {
	input       textinput.Model
	commands    []PaletteCommand
	filtered    []paletteEntry // filtered entries (commands + category headers)
	selectedIdx int            // cursor position within selectable items
	width       int
	height      int
}

// paletteEntry is a render-time item: either a category header or a command.
type paletteEntry struct {
	isHeader bool
	category string
	command  PaletteCommand
	cmdIndex int // index into the original filtered commands (for cursor tracking)
}

// BrowserPaletteCommands returns the default command list for browser mode.
func BrowserPaletteCommands() []PaletteCommand {
	return []PaletteCommand{
		{Name: "Back", Key: "h", Category: "Navigation"},
		{Name: "Open / Drill In", Key: "l", Category: "Navigation"},
		{Name: "Fuzzy Jump", Key: " ", Category: "Navigation"},
		{Name: "Filter", Key: "/", Category: "Navigation"},

		{Name: "BSON Dump", Key: "b", Category: "View"},
		{Name: "Compare View", Key: "c", Category: "View"},
		{Name: "Diagram in Browser", Key: "d", Category: "View"},
		{Name: "Zen Mode", Key: "z", Category: "View"},
		{Name: "Toggle MDL/NDSL", Key: "Tab", Category: "View"},

		{Name: "Execute MDL Script", Key: "x", Category: "Action"},
		{Name: "Refresh Tree", Key: "r", Category: "Action"},
		{Name: "Copy to Clipboard", Key: "y", Category: "Action"},

		{Name: "Show Check Results", Key: "!", Category: "Check"},
		{Name: "Next Error Document", Key: "]e", Category: "Check"},
		{Name: "Prev Error Document", Key: "[e", Category: "Check"},

		{Name: "New Tab (same project)", Key: "t", Category: "Tab"},
		{Name: "New Tab (pick project)", Key: "T", Category: "Tab"},
		{Name: "Close Tab", Key: "W", Category: "Tab"},
		{Name: "Switch Tab", Key: "1-9", Category: "Tab"},

		{Name: "Help", Key: "?", Category: "Other"},
	}
}

// NewCommandPaletteView creates a command palette with browser-mode commands.
func NewCommandPaletteView(width, height int) CommandPaletteView {
	return NewCommandPaletteViewWithCommands(BrowserPaletteCommands(), width, height)
}

// NewCommandPaletteViewWithCommands creates a command palette with the given commands.
func NewCommandPaletteViewWithCommands(commands []PaletteCommand, width, height int) CommandPaletteView {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.Placeholder = "type to filter..."
	ti.CharLimit = 100
	ti.Focus()

	cp := CommandPaletteView{
		input:    ti,
		commands: commands,
		width:    width,
		height:   height,
	}
	cp.refilter()
	return cp
}

// Mode returns ModeCommandPalette.
func (cp CommandPaletteView) Mode() ViewMode {
	return ModeCommandPalette
}

// Hints returns palette-specific key hints.
func (cp CommandPaletteView) Hints() []Hint {
	return []Hint{
		{Key: "↑/↓", Label: "navigate"},
		{Key: "Enter", Label: "execute"},
		{Key: "Esc", Label: "close"},
	}
}

// StatusInfo returns display data for the status bar.
func (cp CommandPaletteView) StatusInfo() StatusInfo {
	selectableCount := cp.countSelectable()
	return StatusInfo{
		Breadcrumb: []string{"Command Palette"},
		Mode:       "Palette",
		Position:   fmt.Sprintf("%d/%d", selectableCount, len(cp.commands)),
	}
}

// Update handles input for the command palette.
func (cp CommandPaletteView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return cp, func() tea.Msg { return PopViewMsg{} }
		case "enter":
			cmd := cp.selectedCommand()
			if cmd != nil {
				key := cmd.Key
				return cp, func() tea.Msg { return PaletteExecMsg{Key: key} }
			}
			return cp, nil
		case "up", "ctrl+p", "k":
			cp.moveUp()
			return cp, nil
		case "down", "ctrl+n", "j":
			cp.moveDown()
			return cp, nil
		default:
			var cmd tea.Cmd
			cp.input, cmd = cp.input.Update(msg)
			cp.refilter()
			return cp, cmd
		}

	case tea.WindowSizeMsg:
		cp.width = msg.Width
		cp.height = msg.Height
	}
	return cp, nil
}

// Render draws the palette as a centered modal box.
func (cp CommandPaletteView) Render(width, height int) string {
	dimStyle := lipgloss.NewStyle().Foreground(MutedColor)
	catStyle := lipgloss.NewStyle().Foreground(MutedColor).Bold(true)
	selStyle := lipgloss.NewStyle().Bold(true).Foreground(AccentColor)
	normStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	keyStyle := lipgloss.NewStyle().Foreground(MutedColor)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(AccentColor)

	contentWidth := max(30, min(56, width-14)) // inner content width (box adds border+padding)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Commands") + "\n")
	sb.WriteString(cp.input.View() + "\n\n")

	// Determine scroll window
	maxVisibleLines := max(6, min(paletteMaxVisible, height-10))
	entries := cp.filtered

	// Find scroll offset to keep selected item visible
	scrollOffset := cp.computeScrollOffset(maxVisibleLines)

	selectableIdx := 0
	visibleLines := 0

	for i, entry := range entries {
		if i < scrollOffset {
			if !entry.isHeader {
				selectableIdx++
			}
			continue
		}
		if visibleLines >= maxVisibleLines {
			break
		}

		if entry.isHeader {
			sb.WriteString(catStyle.Render("  "+entry.category) + "\n")
			visibleLines++
			continue
		}

		shortcut := entry.command.Key
		name := entry.command.Name

		// Calculate padding between name and shortcut
		shortcutWidth := len(shortcut)
		nameMaxWidth := contentWidth - 6 - shortcutWidth // "  > " prefix + spacing
		if len(name) > nameMaxWidth {
			name = name[:nameMaxWidth-1] + "~"
		}
		pad := contentWidth - 4 - len(name) - shortcutWidth
		if pad < 1 {
			pad = 1
		}

		if selectableIdx == cp.selectedIdx {
			sb.WriteString(selStyle.Render("  > "+name) + strings.Repeat(" ", pad) + selStyle.Render(shortcut) + "\n")
		} else {
			sb.WriteString("    " + normStyle.Render(name) + strings.Repeat(" ", pad) + keyStyle.Render(shortcut) + "\n")
		}
		selectableIdx++
		visibleLines++
	}

	selectableCount := cp.countSelectable()
	sb.WriteString("\n" + dimStyle.Render(fmt.Sprintf("  %d commands", selectableCount)))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(AccentColor).
		Padding(1, 2).
		Render(sb.String())

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		box)
}

// computeScrollOffset returns the first entry index to render, keeping selectedIdx visible.
func (cp *CommandPaletteView) computeScrollOffset(maxVisible int) int {
	if len(cp.filtered) <= maxVisible {
		return 0
	}

	// Find the line index of the selected command
	targetLine := 0
	selectableIdx := 0
	for i, entry := range cp.filtered {
		if !entry.isHeader {
			if selectableIdx == cp.selectedIdx {
				targetLine = i
				break
			}
			selectableIdx++
		}
	}

	// Ensure selected item is within visible window
	// Show a few lines of context above
	offset := targetLine - maxVisible/3
	if offset < 0 {
		offset = 0
	}
	if offset+maxVisible > len(cp.filtered) {
		offset = max(0, len(cp.filtered)-maxVisible)
	}
	return offset
}

// refilter rebuilds the filtered entries based on the current input.
func (cp *CommandPaletteView) refilter() {
	query := strings.ToLower(strings.TrimSpace(cp.input.Value()))

	var matchedCommands []PaletteCommand
	for _, cmd := range cp.commands {
		if query == "" || fuzzyMatch(strings.ToLower(cmd.Name), query) {
			matchedCommands = append(matchedCommands, cmd)
		}
	}

	// Group by category, preserving order
	var entries []paletteEntry
	lastCategory := ""
	for i, cmd := range matchedCommands {
		if cmd.Category != lastCategory {
			entries = append(entries, paletteEntry{
				isHeader: true,
				category: cmd.Category,
			})
			lastCategory = cmd.Category
		}
		entries = append(entries, paletteEntry{
			command:  cmd,
			cmdIndex: i,
		})
	}

	cp.filtered = entries

	// Clamp cursor
	selectableCount := cp.countSelectable()
	if cp.selectedIdx >= selectableCount {
		cp.selectedIdx = max(0, selectableCount-1)
	}
}

// fuzzyMatch performs case-insensitive substring matching.
func fuzzyMatch(text, query string) bool {
	return strings.Contains(text, query)
}

// countSelectable returns the number of selectable (non-header) entries.
func (cp *CommandPaletteView) countSelectable() int {
	count := 0
	for _, entry := range cp.filtered {
		if !entry.isHeader {
			count++
		}
	}
	return count
}

// selectedCommand returns the currently selected command, or nil if none.
func (cp *CommandPaletteView) selectedCommand() *PaletteCommand {
	selectableIdx := 0
	for _, entry := range cp.filtered {
		if entry.isHeader {
			continue
		}
		if selectableIdx == cp.selectedIdx {
			return &entry.command
		}
		selectableIdx++
	}
	return nil
}

// moveUp moves the cursor up, skipping category headers.
func (cp *CommandPaletteView) moveUp() {
	selectableCount := cp.countSelectable()
	if selectableCount == 0 {
		return
	}
	cp.selectedIdx--
	if cp.selectedIdx < 0 {
		cp.selectedIdx = selectableCount - 1
	}
}

// moveDown moves the cursor down, skipping category headers.
func (cp *CommandPaletteView) moveDown() {
	selectableCount := cp.countSelectable()
	if selectableCount == 0 {
		return
	}
	cp.selectedIdx++
	if cp.selectedIdx >= selectableCount {
		cp.selectedIdx = 0
	}
}
