package kernel

import "github.com/charmbracelet/lipgloss"

var (
	FocusColor   = lipgloss.AdaptiveColor{Light: "62", Dark: "63"}
	AccentColor  = lipgloss.AdaptiveColor{Light: "214", Dark: "214"}
	MutedColor   = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	AddedColor   = lipgloss.AdaptiveColor{Light: "28", Dark: "114"}
	RemovedColor = lipgloss.AdaptiveColor{Light: "124", Dark: "210"}
)

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
	SeparatorChar  = "│"
	SeparatorStyle = lipgloss.NewStyle().Foreground(MutedColor)

	ActiveTabStyle   = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(FocusColor)
	InactiveTabStyle = lipgloss.NewStyle().Foreground(MutedColor)

	ColumnTitleStyle = lipgloss.NewStyle().Bold(true)

	SelectedItemStyle = lipgloss.NewStyle().Reverse(true)
	DirectoryStyle    = lipgloss.NewStyle().Bold(true)
	LeafStyle         = lipgloss.NewStyle()

	BreadcrumbDimStyle     = lipgloss.NewStyle().Foreground(MutedColor)
	BreadcrumbCurrentStyle = lipgloss.NewStyle()

	LoadingStyle  = lipgloss.NewStyle().Italic(true).Foreground(MutedColor)
	PositionStyle = lipgloss.NewStyle().Foreground(MutedColor)

	PreviewModeStyle = lipgloss.NewStyle().Bold(true)

	HintKeyStyle   = lipgloss.NewStyle().Bold(true)
	HintLabelStyle = lipgloss.NewStyle().Foreground(MutedColor)

	StatusBarStyle = lipgloss.NewStyle().Foreground(MutedColor)

	CmdBarStyle = lipgloss.NewStyle().Bold(true)

	FocusedTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(FocusColor)
	FocusedEdgeChar   = "▎"
	FocusedEdgeStyle  = lipgloss.NewStyle().Foreground(FocusColor)
	AccentStyle       = lipgloss.NewStyle().Foreground(AccentColor)

	CheckErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "196"})
	CheckWarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "172", Dark: "214"})
	CheckDeprecStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "67", Dark: "103"})
	CheckPassStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "114"})
	CheckLocStyle     = lipgloss.NewStyle().Foreground(MutedColor)
	CheckHeaderStyle  = lipgloss.NewStyle().Bold(true)
	CheckRunningStyle = lipgloss.NewStyle().Foreground(MutedColor).Italic(true)

	AgentBadgeStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "33", Dark: "39"}).Bold(true)

	MutedStyle = lipgloss.NewStyle().Foreground(MutedColor)
)
