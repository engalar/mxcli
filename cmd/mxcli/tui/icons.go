package tui

// typeIconMap maps Mendix node types to display icons.
var typeIconMap = map[string]string{
	// Project-level nodes
	"systemoverview":  "🗺",
	"navigation":      "🧭",
	"projectsecurity": "🛡",

	// Modules & structure
	"module":   "⬡",
	"folder":   "📁",
	"category": "⊟",

	// Domain model
	"domainmodel":    "⊞",
	"entity":         "▣",
	"externalentity": "⊡",
	"association":    "↔",
	"enumeration":    "≡",

	// Logic
	"microflow": "⚙",
	"nanoflow":  "⚡",
	"workflow":  "🔀",

	// UI
	"page":    "▤",
	"snippet": "⬔",
	"layout":  "⬕",

	// Images
	"imagecollection": "🖼️",

	// Constants & events
	"constant":       "π",
	"scheduledevent": "⏰",

	// Actions
	"javaaction":       "☕",
	"javascriptaction": "JS",

	// Security
	"security":   "🔒",
	"modulerole": "👤",
	"userrole":   "👥",

	// Services & integrations
	"businesseventservice": "📡",
	"databaseconnection":   "🗄",
	"odataservice":         "🌐",
	"odataclient":          "🔗",
	"publishedrestservice": "🌍",
	"consumedrestservice":  "🔌",

	// Navigation sub-types
	"navprofile":  "⊕",
	"navhome":     "⌂",
	"navmenu":     "☰",
	"navmenuitem": "→",
}

// IconFor returns the icon for a Mendix node type, or "·" if unknown.
func IconFor(nodeType string) string {
	if icon, ok := typeIconMap[nodeType]; ok {
		return icon
	}
	return "·"
}
