package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ScrollListItem is the interface for items displayed in a ScrollList.
type ScrollListItem interface {
	Label() string
	Icon() string
	Description() string
	FilterValue() string
}

// ScrollList is a scrollable list with filtering, mouse support, and a visual scrollbar.
type ScrollList struct {
	items         []ScrollListItem
	filteredItems []int // indices into items
	cursor        int
	scrollOffset  int

	filterInput  textinput.Model
	filterActive bool

	title        string
	width        int
	height       int
	focused      bool
	headerHeight int // lines consumed by title + filter bar
}

// NewScrollList creates a ScrollList with the given title.
func NewScrollList(title string) ScrollList {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.CharLimit = 200

	return ScrollList{
		title:        title,
		filterInput:  ti,
		headerHeight: 1, // title line only
	}
}

// SetItems replaces the item list and resets cursor/scroll state.
func (s *ScrollList) SetItems(items []ScrollListItem) {
	s.items = items
	s.cursor = 0
	s.scrollOffset = 0
	s.rebuildFiltered()
}

// SelectedIndex returns the index of the selected item in the original items slice, or -1.
func (s ScrollList) SelectedIndex() int {
	if len(s.filteredItems) == 0 {
		return -1
	}
	if s.cursor >= len(s.filteredItems) {
		return -1
	}
	return s.filteredItems[s.cursor]
}

// SelectedItem returns the currently selected item, or nil.
func (s ScrollList) SelectedItem() ScrollListItem {
	idx := s.SelectedIndex()
	if idx < 0 || idx >= len(s.items) {
		return nil
	}
	return s.items[idx]
}

// SetSize updates the list dimensions.
func (s *ScrollList) SetSize(w, h int) {
	s.width = w
	s.height = h
}

// SetFocused sets the focus state for visual styling.
func (s *ScrollList) SetFocused(f bool) {
	s.focused = f
}

// MaxVisible returns the number of visible item rows given current height.
func (s ScrollList) MaxVisible() int {
	hdr := s.headerHeight
	if s.filterActive {
		hdr++ // filter input line
	}
	visible := s.height - hdr
	if visible < 1 {
		return 1
	}
	return visible
}

// IsFilterActive returns true if the filter input is active.
func (s ScrollList) IsFilterActive() bool { return s.filterActive }

// ToggleFilter activates or deactivates the filter input.
func (s *ScrollList) ToggleFilter() {
	if s.filterActive {
		s.deactivateFilter()
	} else {
		s.filterActive = true
		s.filterInput.SetValue("")
		s.filterInput.Focus()
		s.rebuildFiltered()
	}
}

// ClearFilter deactivates filter and shows all items.
func (s *ScrollList) ClearFilter() {
	s.deactivateFilter()
}

func (s *ScrollList) deactivateFilter() {
	s.filterActive = false
	s.filterInput.SetValue("")
	s.filterInput.Blur()
	s.rebuildFiltered()
}

func (s *ScrollList) rebuildFiltered() {
	s.filteredItems = s.filteredItems[:0]
	query := strings.ToLower(strings.TrimSpace(s.filterInput.Value()))
	for i, item := range s.items {
		if query == "" || strings.Contains(strings.ToLower(item.FilterValue()), query) {
			s.filteredItems = append(s.filteredItems, i)
		}
	}
	if s.cursor >= len(s.filteredItems) {
		s.cursor = max(0, len(s.filteredItems)-1)
	}
	s.clampScroll()
}

func (s *ScrollList) clampScroll() {
	maxVis := s.MaxVisible()
	total := len(s.filteredItems)
	if s.scrollOffset > total-maxVis {
		s.scrollOffset = max(0, total-maxVis)
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
}

func (s *ScrollList) cursorDown() {
	total := len(s.filteredItems)
	if total == 0 {
		return
	}
	s.cursor++
	if s.cursor >= total {
		s.cursor = 0
		s.scrollOffset = 0
		return
	}
	maxVis := s.MaxVisible()
	if s.cursor >= s.scrollOffset+maxVis {
		s.scrollOffset = s.cursor - maxVis + 1
	}
}

func (s *ScrollList) cursorUp() {
	total := len(s.filteredItems)
	if total == 0 {
		return
	}
	s.cursor--
	if s.cursor < 0 {
		s.cursor = total - 1
		s.scrollOffset = max(0, s.cursor-s.MaxVisible()+1)
		return
	}
	if s.cursor < s.scrollOffset {
		s.scrollOffset = s.cursor
	}
}

// Update handles keyboard and mouse messages.
func (s ScrollList) Update(msg tea.Msg) (ScrollList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.filterActive {
			switch msg.String() {
			case "esc":
				s.deactivateFilter()
				return s, nil
			case "enter":
				// Lock in filter, exit filter mode but keep results
				s.filterActive = false
				s.filterInput.Blur()
				return s, nil
			case "up":
				s.cursorUp()
				return s, nil
			case "down":
				s.cursorDown()
				return s, nil
			default:
				var cmd tea.Cmd
				s.filterInput, cmd = s.filterInput.Update(msg)
				s.rebuildFiltered()
				return s, cmd
			}
		}

		switch msg.String() {
		case "j", "down":
			s.cursorDown()
		case "k", "up":
			s.cursorUp()
		case "/":
			s.ToggleFilter()
		case "G":
			total := len(s.filteredItems)
			if total > 0 {
				s.cursor = total - 1
				maxVis := s.MaxVisible()
				s.scrollOffset = max(0, total-maxVis)
			}
		case "g":
			s.cursor = 0
			s.scrollOffset = 0
		}

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				s.scrollUp(3)
			case tea.MouseButtonWheelDown:
				s.scrollDown(3)
			case tea.MouseButtonLeft:
				topOffset := s.headerHeight
				if s.filterActive {
					topOffset++
				}
				clicked := s.scrollOffset + (msg.Y - topOffset)
				if clicked >= 0 && clicked < len(s.filteredItems) {
					s.cursor = clicked
				}
			}
		}
	}
	return s, nil
}

func (s *ScrollList) scrollUp(n int) {
	s.scrollOffset -= n
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	if s.cursor >= s.scrollOffset+s.MaxVisible() {
		s.cursor = s.scrollOffset + s.MaxVisible() - 1
	}
}

func (s *ScrollList) scrollDown(n int) {
	total := len(s.filteredItems)
	maxVis := s.MaxVisible()
	s.scrollOffset += n
	if s.scrollOffset > total-maxVis {
		s.scrollOffset = max(0, total-maxVis)
	}
	if s.cursor < s.scrollOffset {
		s.cursor = s.scrollOffset
	}
}

// View renders the list with scrollbar.
func (s ScrollList) View() string {
	var sb strings.Builder

	titleSt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	selectedSt := s.selectedStyle()
	normalSt := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	descSt := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	trackSt := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	thumbSt := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	filterLabelSt := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	// Title
	sb.WriteString(titleSt.Render(s.title) + "\n")

	// Filter bar
	if s.filterActive {
		sb.WriteString(s.filterInput.View())
		sb.WriteString("\n")
	} else if s.filterInput.Value() != "" {
		sb.WriteString(filterLabelSt.Render("Filter: " + s.filterInput.Value()))
		sb.WriteString("\n")
	}

	total := len(s.filteredItems)
	maxVis := s.MaxVisible()

	// Content width for items (reserve 1 col for scrollbar if needed)
	contentWidth := s.width - 2 // subtract padding
	showScrollbar := total > maxVis
	if showScrollbar {
		contentWidth-- // reserve 1 for scrollbar
	}
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Scrollbar geometry
	var thumbStart, thumbEnd int
	if showScrollbar {
		trackHeight := maxVis
		if total <= maxVis {
			thumbStart = 0
			thumbEnd = trackHeight
		} else {
			thumbSize := max(1, trackHeight*maxVis/total)
			maxOffset := total - maxVis
			thumbStart = s.scrollOffset * (trackHeight - thumbSize) / maxOffset
			thumbEnd = thumbStart + thumbSize
		}
	}

	for vi := range maxVis {
		idx := s.scrollOffset + vi
		var line string
		if idx < total {
			itemIdx := s.filteredItems[idx]
			item := s.items[itemIdx]
			icon := item.Icon()
			if icon == "" {
				icon = "·"
			}
			label := icon + "  " + item.Label()

			desc := item.Description()
			if desc != "" {
				label += " " + descSt.Render(desc)
			}

			// Truncate to fit
			if lipgloss.Width(label) > contentWidth-4 {
				label = label[:contentWidth-4]
			}

			if idx == s.cursor {
				line = selectedSt.Render("> " + label)
			} else {
				line = normalSt.Render("  " + label)
			}
		} else {
			line = ""
		}

		// Pad line to contentWidth
		lineWidth := lipgloss.Width(line)
		if lineWidth < contentWidth {
			line += strings.Repeat(" ", contentWidth-lineWidth)
		}

		// Append scrollbar character
		if showScrollbar {
			if vi >= thumbStart && vi < thumbEnd {
				line += thumbSt.Render("█")
			} else {
				line += trackSt.Render("│")
			}
		}

		sb.WriteString(line)
		if vi < maxVis-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (s ScrollList) selectedStyle() lipgloss.Style {
	if s.focused {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
}
