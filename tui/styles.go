package tui

import "github.com/charmbracelet/lipgloss"

var (
	colFocusedBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	colNormalBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	statusBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	cmdBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	typeIconMap = map[string]string{
		"module":               "⬡",
		"domainmodel":          "⊞",
		"entity":               "▣",
		"externalentity":       "⊡",
		"association":          "↔",
		"enumeration":          "≡",
		"microflow":            "⚙",
		"nanoflow":             "⚡",
		"page":                 "▤",
		"snippet":              "⬔",
		"layout":               "⬕",
		"constant":             "π",
		"javaaction":           "☕",
		"javascriptaction":     "JS",
		"scheduledevent":       "⏰",
		"folder":               "📁",
		"security":             "🔒",
		"modulerole":           "👤",
		"userrole":             "👥",
		"projectsecurity":      "🛡",
		"navigation":           "🧭",
		"systemoverview":       "🗺",
		"businesseventservice": "📡",
		"databaseconnection":   "🗄",
		"odataservice":         "🌐",
		"odataclient":          "🔗",
		"publishedrestservice": "REST",
		"workflow":             "🔀",
	}
)

func iconFor(nodeType string) string {
	if icon, ok := typeIconMap[nodeType]; ok {
		return icon
	}
	return "·"
}
