package tui

import "github.com/charmbracelet/lipgloss"

// Semantic color tokens — AdaptiveColor picks Light/Dark based on terminal background.
var (
	FocusColor   = lipgloss.AdaptiveColor{Light: "62", Dark: "63"}
	AccentColor  = lipgloss.AdaptiveColor{Light: "214", Dark: "214"}
	MutedColor   = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	AddedColor   = lipgloss.AdaptiveColor{Light: "28", Dark: "114"}
	RemovedColor = lipgloss.AdaptiveColor{Light: "124", Dark: "210"}
)

// Diff view color palette — centralized so the entire diff color scheme can be
// adjusted in one place. AdaptiveColor picks Light/Dark based on terminal background.
var (
	DiffAddedFg        = lipgloss.AdaptiveColor{Light: "#00875f", Dark: "#00D787"}
	DiffAddedChangedFg = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"}
	DiffAddedChangedBg = lipgloss.AdaptiveColor{Light: "#005F00", Dark: "#005F00"}

	DiffRemovedFg        = lipgloss.AdaptiveColor{Light: "#AF005F", Dark: "#FF5F87"}
	DiffRemovedChangedFg = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"}
	DiffRemovedChangedBg = lipgloss.AdaptiveColor{Light: "#5F0000", Dark: "#5F0000"}

	DiffEqualGutter     = lipgloss.AdaptiveColor{Light: "241", Dark: "241"}
	DiffGutterAddedFg   = lipgloss.AdaptiveColor{Light: "#00875f", Dark: "#00D787"}
	DiffGutterRemovedFg = lipgloss.AdaptiveColor{Light: "#AF005F", Dark: "#FF5F87"}
)

var (
	// Column separator: dim vertical bar between panels.
	SeparatorChar  = "│"
	SeparatorStyle = lipgloss.NewStyle().Foreground(MutedColor)

	// Tabs
	ActiveTabStyle   = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(FocusColor)
	InactiveTabStyle = lipgloss.NewStyle().Foreground(MutedColor)

	// Column title (e.g. "Entities", "Attributes")
	ColumnTitleStyle = lipgloss.NewStyle().Bold(true)

	// List items
	SelectedItemStyle = lipgloss.NewStyle().Reverse(true)
	DirectoryStyle    = lipgloss.NewStyle().Bold(true)
	LeafStyle         = lipgloss.NewStyle()

	// Breadcrumb
	BreadcrumbDimStyle     = lipgloss.NewStyle().Foreground(MutedColor)
	BreadcrumbCurrentStyle = lipgloss.NewStyle()

	// Loading / status
	LoadingStyle  = lipgloss.NewStyle().Italic(true).Foreground(MutedColor)
	PositionStyle = lipgloss.NewStyle().Foreground(MutedColor)

	// Preview mode label (MDL / NDSL toggle)
	PreviewModeStyle = lipgloss.NewStyle().Bold(true)

	// Hint bar: key name bold, description dim
	HintKeyStyle   = lipgloss.NewStyle().Bold(true)
	HintLabelStyle = lipgloss.NewStyle().Foreground(MutedColor)

	// Status bar (bottom line)
	StatusBarStyle = lipgloss.NewStyle().Foreground(MutedColor)

	// Command bar
	CmdBarStyle = lipgloss.NewStyle().Bold(true)

	// Focus indicator styles (Phase 2)
	FocusedTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(FocusColor)
	FocusedEdgeChar   = "▎"
	FocusedEdgeStyle  = lipgloss.NewStyle().Foreground(FocusColor)
	AccentStyle       = lipgloss.NewStyle().Foreground(AccentColor)

	// mx check result styles
	CheckErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "196"})
	CheckWarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "172", Dark: "214"})
	CheckDeprecStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "67", Dark: "103"})
	CheckPassStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "114"})
	CheckLocStyle     = lipgloss.NewStyle().Foreground(MutedColor)
	CheckHeaderStyle  = lipgloss.NewStyle().Bold(true)
	CheckRunningStyle = lipgloss.NewStyle().Foreground(MutedColor).Italic(true)
)
