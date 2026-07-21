package chrome

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mendixlabs/mxcli/cmd/mxcli/tui/kernel"
)

type breadcrumbZone struct {
	startX int
	endX   int
	depth  int
}

type StatusBar struct {
	breadcrumb []string
	position   string
	mode       string
	checkBadge string
	agentBadge string
	viewDepth  int
	viewModes  []string
	zones      []breadcrumbZone
}

func NewStatusBar() StatusBar {
	return StatusBar{}
}

func (s *StatusBar) SetBreadcrumb(segments []string) {
	s.breadcrumb = segments
}

func (s *StatusBar) SetPosition(pos string) {
	s.position = pos
}

func (s *StatusBar) SetMode(mode string) {
	s.mode = mode
}

func (s *StatusBar) SetCheckBadge(badge string) {
	s.checkBadge = badge
}

func (s *StatusBar) SetAgentBadge(badge string) {
	s.agentBadge = badge
}

func (s *StatusBar) SetViewDepth(depth int, modes []string) {
	s.viewDepth = depth
	s.viewModes = modes
}

func (s *StatusBar) View(width int) string {
	s.zones = nil

	sep := kernel.BreadcrumbDimStyle.Render(" › ")
	sepWidth := lipgloss.Width(sep)

	xPos := 1

	var crumbParts []string

	if s.viewDepth > 1 && len(s.viewModes) > 0 {
		var modeParts []string
		for _, m := range s.viewModes {
			modeParts = append(modeParts, m)
		}
		depthLabel := strings.Join(modeParts, " › ")
		rendered := kernel.BreadcrumbDimStyle.Render("[" + depthLabel + "]")
		crumbParts = append(crumbParts, rendered)
		xPos += lipgloss.Width(rendered)
		if len(s.breadcrumb) > 0 {
			xPos += sepWidth
		}
	}

	for i, seg := range s.breadcrumb {
		var rendered string
		if i == len(s.breadcrumb)-1 {
			rendered = kernel.BreadcrumbCurrentStyle.Render(seg)
		} else {
			rendered = kernel.BreadcrumbDimStyle.Render(seg)
		}

		segWidth := lipgloss.Width(rendered)
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

	var rightParts []string
	if s.agentBadge != "" {
		rightParts = append(rightParts, s.agentBadge)
	}
	if s.checkBadge != "" {
		rightParts = append(rightParts, s.checkBadge)
	}
	if s.position != "" {
		rightParts = append(rightParts, kernel.PositionStyle.Render(s.position))
	}
	if s.mode != "" {
		rightParts = append(rightParts, kernel.BreadcrumbDimStyle.Render("⎸")+kernel.PreviewModeStyle.Render(s.mode))
	}
	right := strings.Join(rightParts, "  ") + " "

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	gap := max(width-leftWidth-rightWidth, 0)

	return left + strings.Repeat(" ", gap) + right
}

func (s *StatusBar) HitTest(x int) (int, bool) {
	for _, z := range s.zones {
		if x >= z.startX && x < z.endX {
			return z.depth, true
		}
	}
	return 0, false
}
