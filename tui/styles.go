package tui

import "github.com/charmbracelet/lipgloss"

// Yazi-style borderless, terminal-adaptive styles.
// No hardcoded colors — relies on Bold, Dim, Reverse, Italic, Underline.

var (
	// Column separator: dim vertical bar between panels.
	SeparatorChar  = "│"
	SeparatorStyle = lipgloss.NewStyle().Faint(true)

	// Tabs
	ActiveTabStyle   = lipgloss.NewStyle().Bold(true).Underline(true)
	InactiveTabStyle = lipgloss.NewStyle().Faint(true)

	// Column title (e.g. "Entities", "Attributes")
	ColumnTitleStyle = lipgloss.NewStyle().Bold(true)

	// List items
	SelectedItemStyle = lipgloss.NewStyle().Reverse(true)
	DirectoryStyle    = lipgloss.NewStyle().Bold(true)
	LeafStyle         = lipgloss.NewStyle()

	// Breadcrumb
	BreadcrumbDimStyle     = lipgloss.NewStyle().Faint(true)
	BreadcrumbCurrentStyle = lipgloss.NewStyle()

	// Loading / status
	LoadingStyle  = lipgloss.NewStyle().Italic(true).Faint(true)
	PositionStyle = lipgloss.NewStyle().Faint(true)

	// Preview mode label (MDL / NDSL toggle)
	PreviewModeStyle = lipgloss.NewStyle().Bold(true)

	// Hint bar: key name bold, description dim
	HintKeyStyle   = lipgloss.NewStyle().Bold(true)
	HintLabelStyle = lipgloss.NewStyle().Faint(true)

	// Status bar (bottom line)
	StatusBarStyle = lipgloss.NewStyle().Faint(true)

	// Command bar
	CmdBarStyle = lipgloss.NewStyle().Bold(true)
)
