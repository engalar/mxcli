package tui

// PanelVisibility controls how many panels are shown.
type PanelVisibility int

const (
	ShowOnePanel  PanelVisibility = iota // modules 100%
	ShowTwoPanels                        // modules 35%, elements 65%
	ShowZoomed                           // zoomed panel 100%
)

// PanelRect describes a panel's position, size, and visibility.
type PanelRect struct {
	X, Y, Width, Height int
	Visible             bool
}

// panelWidths2 returns [modulesW, elementsW] based on visibility mode.
func panelWidths2(totalW int, vis PanelVisibility, zoomedPanel Focus) (int, int) {
	available := totalW - 4 // 2 borders × 2 panels
	if available < 30 {
		available = 30
	}

	switch vis {
	case ShowOnePanel:
		return available, 0
	case ShowTwoPanels:
		modulesW := available * 35 / 100
		return modulesW, available - modulesW
	case ShowZoomed:
		if zoomedPanel == FocusElements {
			return 0, available
		}
		return available, 0
	}
	return available, 0
}
