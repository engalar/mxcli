package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders a bottom status line with breadcrumb and position info.
type StatusBar struct {
	breadcrumb []string
	position   string // e.g. "3/4"
	mode       string // e.g. "MDL" or "NDSL"
}

// NewStatusBar creates a status bar.
func NewStatusBar() StatusBar {
	return StatusBar{}
}

// SetBreadcrumb sets the breadcrumb path segments.
func (s *StatusBar) SetBreadcrumb(segments []string) {
	s.breadcrumb = segments
}

// SetPosition sets the position indicator (e.g. "3/4").
func (s *StatusBar) SetPosition(pos string) {
	s.position = pos
}

// SetMode sets the preview mode label.
func (s *StatusBar) SetMode(mode string) {
	s.mode = mode
}

// View renders the status bar to fit the given width.
func (s *StatusBar) View(width int) string {
	// Build breadcrumb: all segments dim except last one normal.
	var crumbParts []string
	sep := BreadcrumbDimStyle.Render(" > ")
	for i, seg := range s.breadcrumb {
		if i == len(s.breadcrumb)-1 {
			crumbParts = append(crumbParts, BreadcrumbCurrentStyle.Render(seg))
		} else {
			crumbParts = append(crumbParts, BreadcrumbDimStyle.Render(seg))
		}
	}
	left := " " + strings.Join(crumbParts, sep)

	// Build right side: position + mode
	var rightParts []string
	if s.position != "" {
		rightParts = append(rightParts, PositionStyle.Render(s.position))
	}
	if s.mode != "" {
		rightParts = append(rightParts, PreviewModeStyle.Render(s.mode))
	}
	right := strings.Join(rightParts, "  ") + " "

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	gap := max(width-leftWidth-rightWidth, 0)

	return left + strings.Repeat(" ", gap) + right
}
