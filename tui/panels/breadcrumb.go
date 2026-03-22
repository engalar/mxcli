package panels

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	breadcrumbSeparator     = " > "
	breadcrumbNormalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	breadcrumbActiveStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	breadcrumbSepStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	breadcrumbEllipsisStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// BreadcrumbSegment represents a single level in the navigation path.
type BreadcrumbSegment struct {
	Label string
}

// Breadcrumb renders a navigable path like "Project > MyModule > Entities".
type Breadcrumb struct {
	segments []BreadcrumbSegment
	width    int
}

// Push appends a segment to the breadcrumb trail.
func (b *Breadcrumb) Push(label string) {
	b.segments = append(b.segments, BreadcrumbSegment{Label: label})
}

// PopTo truncates the breadcrumb to the given depth (0 = empty, 1 = first segment only).
func (b *Breadcrumb) PopTo(level int) {
	if level < 0 {
		level = 0
	}
	if level < len(b.segments) {
		b.segments = b.segments[:level]
	}
}

// Depth returns the current number of segments.
func (b *Breadcrumb) Depth() int {
	return len(b.segments)
}

// SetWidth sets the available width for truncation.
func (b *Breadcrumb) SetWidth(w int) {
	b.width = w
}

// View renders the breadcrumb as a styled string.
func (b *Breadcrumb) View() string {
	if len(b.segments) == 0 {
		return ""
	}

	sep := breadcrumbSepStyle.Render(breadcrumbSeparator)
	lastIdx := len(b.segments) - 1

	var parts []string
	for i, seg := range b.segments {
		if i == lastIdx {
			parts = append(parts, breadcrumbActiveStyle.Render(seg.Label))
		} else {
			parts = append(parts, breadcrumbNormalStyle.Render(seg.Label))
		}
	}

	rendered := strings.Join(parts, sep)

	if b.width > 0 && lipgloss.Width(rendered) > b.width {
		ellipsis := breadcrumbEllipsisStyle.Render("...")
		for len(parts) > 2 {
			parts = append(parts[:1], parts[2:]...)
			candidate := parts[0] + sep + ellipsis + sep + strings.Join(parts[1:], sep)
			if lipgloss.Width(candidate) <= b.width {
				return candidate
			}
		}
	}

	return rendered
}

// ClickedSegment returns the segment index at the given x position, or -1 if none.
// Uses plain-text character counting (labels + " > " separators).
func (b *Breadcrumb) ClickedSegment(x int) int {
	pos := 0
	for i, seg := range b.segments {
		end := pos + len(seg.Label)
		if x >= pos && x < end {
			return i
		}
		pos = end + len(breadcrumbSeparator) // skip " > "
	}
	return -1
}
