package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// breadcrumbZone tracks the X range for a clickable breadcrumb segment.
type breadcrumbZone struct {
	startX int
	endX   int
	depth  int // navigation depth this segment corresponds to
}

// StatusBar renders a bottom status line with breadcrumb and position info.
type StatusBar struct {
	breadcrumb []string
	position   string // e.g. "3/4"
	mode       string // e.g. "MDL" or "NDSL"
	checkBadge string // e.g. "✗ 3E 2W" or "✓" (pre-styled)
	viewDepth  int
	viewModes  []string
	zones      []breadcrumbZone // clickable breadcrumb zones
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

// SetCheckBadge sets the mx check status badge (pre-styled string).
func (s *StatusBar) SetCheckBadge(badge string) {
	s.checkBadge = badge
}

// SetViewDepth sets the view stack depth and mode names for breadcrumb display.
func (s *StatusBar) SetViewDepth(depth int, modes []string) {
	s.viewDepth = depth
	s.viewModes = modes
}

// View renders the status bar to fit the given width, tracking breadcrumb click zones.
func (s *StatusBar) View(width int) string {
	s.zones = nil

	// Build breadcrumb: all segments dim except last one normal.
	// Track X positions for click zones.
	sep := BreadcrumbDimStyle.Render(" › ")
	sepWidth := lipgloss.Width(sep)

	xPos := 1 // leading space

	var crumbParts []string

	// Prepend view depth breadcrumb if deeper than 1
	if s.viewDepth > 1 && len(s.viewModes) > 0 {
		var modeParts []string
		for _, m := range s.viewModes {
			modeParts = append(modeParts, m)
		}
		depthLabel := strings.Join(modeParts, " › ")
		rendered := BreadcrumbDimStyle.Render("[" + depthLabel + "]")
		crumbParts = append(crumbParts, rendered)
		xPos += lipgloss.Width(rendered)
		if len(s.breadcrumb) > 0 {
			xPos += sepWidth
		}
	}

	for i, seg := range s.breadcrumb {
		var rendered string
		if i == len(s.breadcrumb)-1 {
			rendered = BreadcrumbCurrentStyle.Render(seg)
		} else {
			rendered = BreadcrumbDimStyle.Render(seg)
		}

		segWidth := lipgloss.Width(rendered)
		// Record clickable zone: depth = i (how many levels deep from root)
		s.zones = append(s.zones, breadcrumbZone{
			startX: xPos,
			endX:   xPos + segWidth,
			depth:  i,
		})

		crumbParts = append(crumbParts, rendered)
		xPos += segWidth
		if i < len(s.breadcrumb)-1 {
			xPos += sepWidth
		}
	}
	left := " " + strings.Join(crumbParts, sep)

	// Build right side: check badge + position + mode
	var rightParts []string
	if s.checkBadge != "" {
		rightParts = append(rightParts, s.checkBadge)
	}
	if s.position != "" {
		rightParts = append(rightParts, PositionStyle.Render(s.position))
	}
	if s.mode != "" {
		rightParts = append(rightParts, BreadcrumbDimStyle.Render("⎸")+PreviewModeStyle.Render(s.mode))
	}
	right := strings.Join(rightParts, "  ") + " "

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	gap := max(width-leftWidth-rightWidth, 0)

	return left + strings.Repeat(" ", gap) + right
}

// HitTest checks if x falls within a clickable breadcrumb zone.
// Returns the navigation depth and true if a zone was hit.
func (s *StatusBar) HitTest(x int) (int, bool) {
	for _, z := range s.zones {
		if x >= z.startX && x < z.endX {
			return z.depth, true
		}
	}
	return 0, false
}
